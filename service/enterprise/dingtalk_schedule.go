package enterprise

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const (
	DefaultDingTalkScheduleCron     = "0 0 * * *"
	DefaultDingTalkScheduleTimezone = "Asia/Shanghai"
	DingTalkScheduledTrigger        = "scheduled"
	DingTalkManualTrigger           = "manual"
)

var dingTalkCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func nextDingTalkScheduleTime(expression, timezone string, from time.Time) (time.Time, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		expression = DefaultDingTalkScheduleCron
	}
	if len(expression) > 128 || strings.HasPrefix(expression, "@") {
		return time.Time{}, ErrDingTalkScheduleCronInvalid
	}
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return time.Time{}, ErrDingTalkScheduleTimezoneInvalid
	}
	schedule, err := dingTalkCronParser.Parse(expression)
	if err != nil {
		return time.Time{}, ErrDingTalkScheduleCronInvalid
	}
	return schedule.Next(from.In(location)).UTC(), nil
}

type DingTalkScheduleDispatchResult struct {
	Scanned   int `json:"scanned"`
	Due       int `json:"due"`
	Enqueued  int `json:"enqueued"`
	Skipped   int `json:"skipped_active"`
	Recovered int `json:"recovered"`
}

type DingTalkScheduleService struct {
	db *gorm.DB
}

func NewDingTalkScheduleService(db *gorm.DB) *DingTalkScheduleService {
	return &DingTalkScheduleService{db: db}
}

// DispatchDueFullSyncs advances each due occurrence exactly once under a row
// lock, then starts the shared full-sync runner. It is safe to call from every
// application instance; the transaction and unique task indexes provide the
// cross-instance guard.
func (s *DingTalkScheduleService) DispatchDueFullSyncs(ctx context.Context, now time.Time) (result DingTalkScheduleDispatchResult, err error) {
	if s == nil || s.db == nil {
		return result, errors.New("dingtalk schedule database is nil")
	}
	var configs []entmodel.DingTalkConfig
	if err := s.db.WithContext(ctx).Where("scheduled_full_sync_enabled = ? AND sync_enabled = ? AND scheduled_full_sync_next_run_at > 0 AND scheduled_full_sync_next_run_at <= ?", true, true, now.Unix()).Find(&configs).Error; err != nil {
		return result, err
	}
	result.Scanned = len(configs)
	for _, candidate := range configs {
		result.Due++
		created, skipped, recovered, dispatchErr := s.dispatchTenant(ctx, candidate.TenantId, now)
		if dispatchErr != nil {
			continue
		}
		if created {
			result.Enqueued++
		}
		if skipped {
			result.Skipped++
		}
		result.Recovered += recovered
	}
	return result, nil
}

func (s *DingTalkScheduleService) dispatchTenant(ctx context.Context, tenantID int, now time.Time) (bool, bool, int, error) {
	var task *entmodel.DingTalkSyncTask
	var config entmodel.DingTalkConfig
	created := false
	skipped := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := model.LockForUpdate(tx).Where("tenant_id = ?", tenantID).First(&config).Error; err != nil {
			return err
		}
		if !config.ScheduledFullSyncEnabled || !config.SyncEnabled || config.ScheduledFullSyncNextRunAt <= 0 || config.ScheduledFullSyncNextRunAt > now.Unix() {
			return nil
		}
		occurrence := config.ScheduledFullSyncNextRunAt
		next, err := nextDingTalkScheduleTime(config.ScheduledFullSyncCron, config.ScheduledFullSyncTimezone, time.Unix(occurrence, 0))
		if err != nil {
			return tx.Model(&config).Updates(map[string]any{"scheduled_full_sync_last_status": "invalid_config", "scheduled_full_sync_last_error": err.Error(), "scheduled_full_sync_next_run_at": 0}).Error
		}
		var active entmodel.DingTalkSyncTask
		activeErr := tx.Where("tenant_id = ? AND mode = ? AND status IN ?", tenantID, constant.DingTalkSyncTaskModeFull, []string{constant.DingTalkSyncTaskStatusPending, constant.DingTalkSyncTaskStatusRunning}).Order("id desc").First(&active).Error
		if activeErr == nil {
			skipped = true
		} else if !errors.Is(activeErr, gorm.ErrRecordNotFound) {
			return activeErr
		} else {
			activeKey := "full"
			task = &entmodel.DingTalkSyncTask{TenantId: tenantID, Mode: constant.DingTalkSyncTaskModeFull, Status: constant.DingTalkSyncTaskStatusPending, TriggerSource: DingTalkScheduledTrigger, ScheduledFor: &occurrence, ScheduleRevision: config.ScheduledFullSyncRevision, ActiveKey: &activeKey}
			if err := tx.Create(task).Error; err != nil {
				var existing entmodel.DingTalkSyncTask
				if readErr := tx.Where("tenant_id = ? AND scheduled_for = ?", tenantID, occurrence).First(&existing).Error; readErr == nil {
					task = &existing
				} else {
					return err
				}
			} else {
				created = true
			}
		}
		updates := map[string]any{"scheduled_full_sync_next_run_at": next.Unix(), "scheduled_full_sync_last_run_at": occurrence, "scheduled_full_sync_last_status": "skipped_active", "scheduled_full_sync_last_error": "active_task"}
		if created && task != nil {
			updates["scheduled_full_sync_last_task_id"] = task.Id
			updates["scheduled_full_sync_last_status"] = "scheduled"
			updates["scheduled_full_sync_last_error"] = ""
		}
		return tx.Model(&config).Updates(updates).Error
	})
	if err != nil {
		return false, false, 0, err
	}
	if task != nil {
		go func(taskID, tenantID int) {
			runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			var cfg entmodel.DingTalkConfig
			if err := s.db.Where("tenant_id = ?", tenantID).First(&cfg).Error; err == nil {
				_ = NewDingTalkSyncService(s.db, nil).RunTask(runCtx, taskID, cfg)
			}
		}(task.Id, tenantID)
	}
	return created, skipped, 0, nil
}

// RecoverPending starts scheduled tasks left behind by a process crash. RunTask
// atomically transitions pending to running, so repeated recovery scans cannot
// execute the same task twice.
func (s *DingTalkScheduleService) RecoverPending(ctx context.Context) (int, error) {
	var tasks []entmodel.DingTalkSyncTask
	if err := s.db.WithContext(ctx).Where("trigger_source = ? AND status = ?", DingTalkScheduledTrigger, "pending").Limit(100).Find(&tasks).Error; err != nil {
		return 0, err
	}
	for _, task := range tasks {
		var config entmodel.DingTalkConfig
		if err := s.db.WithContext(ctx).Where("tenant_id = ?", task.TenantId).First(&config).Error; err != nil {
			continue
		}
		go func(t entmodel.DingTalkSyncTask, cfg entmodel.DingTalkConfig) {
			runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			_ = NewDingTalkSyncService(s.db, nil).RunTask(runCtx, t.Id, cfg)
		}(task, config)
	}
	return len(tasks), nil
}
