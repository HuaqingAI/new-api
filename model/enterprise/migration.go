package enterprise

import "gorm.io/gorm"

type dingTalkConfigAutoSyncColumn struct {
	AutoSyncOnLogin bool `gorm:"column:auto_sync_on_login;type:boolean;not null;default:false"`
}

type dingTalkScheduleConfigColumns struct {
	ScheduledFullSyncEnabled    bool   `gorm:"column:scheduled_full_sync_enabled;type:boolean;not null;default:false"`
	ScheduledFullSyncCron       string `gorm:"column:scheduled_full_sync_cron;type:varchar(128);not null;default:'0 0 * * *'"`
	ScheduledFullSyncTimezone   string `gorm:"column:scheduled_full_sync_timezone;type:varchar(64);not null;default:'Asia/Shanghai'"`
	ScheduledFullSyncNextRunAt  int64  `gorm:"column:scheduled_full_sync_next_run_at;type:bigint;not null;default:0"`
	ScheduledFullSyncLastRunAt  int64  `gorm:"column:scheduled_full_sync_last_run_at;type:bigint;not null;default:0"`
	ScheduledFullSyncLastTaskId int    `gorm:"column:scheduled_full_sync_last_task_id;type:int;not null;default:0"`
	ScheduledFullSyncLastStatus string `gorm:"column:scheduled_full_sync_last_status;type:varchar(32);not null;default:''"`
	ScheduledFullSyncLastError  string `gorm:"column:scheduled_full_sync_last_error;type:varchar(1024);not null;default:''"`
	ScheduledFullSyncRevision   int64  `gorm:"column:scheduled_full_sync_revision;type:bigint;not null;default:1"`
	ScheduledFullSyncUpdatedAt  int64  `gorm:"column:scheduled_full_sync_updated_at;type:bigint;not null;default:0"`
}

func (dingTalkScheduleConfigColumns) TableName() string { return DingTalkConfig{}.TableName() }

type dingTalkSyncTaskScheduleColumns struct {
	TriggerSource    string  `gorm:"column:trigger_source;type:varchar(32);not null;default:'manual'"`
	ScheduledFor     *int64  `gorm:"column:scheduled_for;type:bigint"`
	ScheduleRevision int64   `gorm:"column:schedule_revision;type:bigint;not null;default:0"`
	ActiveKey        *string `gorm:"column:active_key;type:varchar(32)"`
	RunnerId         string  `gorm:"column:runner_id;type:varchar(128);not null;default:''"`
}

func (dingTalkSyncTaskScheduleColumns) TableName() string { return DingTalkSyncTask{}.TableName() }

func (dingTalkConfigAutoSyncColumn) TableName() string {
	return DingTalkConfig{}.TableName()
}

type dingTalkSyncConflictTriggerSourceColumn struct {
	TriggerSource string `gorm:"column:trigger_source;type:varchar(32);not null;default:'full_sync'"`
}

func (dingTalkSyncConflictTriggerSourceColumn) TableName() string {
	return DingTalkSyncConflict{}.TableName()
}

func Migrate(db *gorm.DB) error {
	if err := ensureDepartmentRoleOwnerColumnsAndIndexes(db); err != nil {
		return err
	}
	if err := ensureAlertEventDepartmentTokensColumn(db); err != nil {
		return err
	}
	if err := ensureAlertDeliveryColumns(db); err != nil {
		return err
	}
	if err := ensureQuotaAllocationLifecycleColumns(db); err != nil {
		return err
	}
	if err := ensureQuotaRequestColumns(db); err != nil {
		return err
	}
	if err := ensureGovernanceNotificationDeliveryColumns(db); err != nil {
		return err
	}
	dingTalkColumns, err := ensureDingTalkAutoSyncColumns(db)
	if err != nil {
		return err
	}
	if err := ensureDingTalkScheduleColumns(db); err != nil {
		return err
	}
	// Existing DingTalk tables use explicit additive migrations above. SQLite
	// rebuilds them when AutoMigrate sees the new defaults, dropping backfilled
	// values during the copy.
	models := []any{
		&Department{},
		&DepartmentBudget{},
		&BudgetDelegation{},
		&QuotaAllocation{},
		&QuotaRequest{},
		&UserDepartment{},
		&DepartmentRole{},
		&AdminAction{},
		&AlertEvent{},
		&AlertRule{},
		&AlertDelivery{},
		&GovernanceNotificationDelivery{},
		&UsageSnapshot{},
		&UsageScopeSnapshot{},
		&UsageReportJob{},
		&DingTalkIdentity{},
		&DingTalkSyncTask{},
		&DingTalkSyncLog{},
	}
	if !dingTalkColumns.configTableExists {
		models = append(models, &DingTalkConfig{})
	}
	if !dingTalkColumns.conflictTableExists {
		models = append(models, &DingTalkSyncConflict{})
	}
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	if err := backfillDingTalkAutoSyncColumns(db, dingTalkColumns); err != nil {
		return err
	}
	if err := backfillAlertEventDepartmentTokens(db); err != nil {
		return err
	}
	return normalizeDepartmentBudgetRemaining(db)
}

type dingTalkAutoSyncMigrationColumns struct {
	autoSyncOnLogin     bool
	triggerSource       bool
	configTableExists   bool
	conflictTableExists bool
}

func ensureDingTalkAutoSyncColumns(db *gorm.DB) (dingTalkAutoSyncMigrationColumns, error) {
	if db == nil {
		return dingTalkAutoSyncMigrationColumns{}, nil
	}
	columns := dingTalkAutoSyncMigrationColumns{}
	columns.configTableExists = db.Migrator().HasTable(&DingTalkConfig{})
	if columns.configTableExists && !db.Migrator().HasColumn(&DingTalkConfig{}, "auto_sync_on_login") {
		columns.autoSyncOnLogin = true
		if err := db.Migrator().AddColumn(&dingTalkConfigAutoSyncColumn{}, "auto_sync_on_login"); err != nil {
			return dingTalkAutoSyncMigrationColumns{}, err
		}
	}
	columns.conflictTableExists = db.Migrator().HasTable(&DingTalkSyncConflict{})
	if columns.conflictTableExists && !db.Migrator().HasColumn(&DingTalkSyncConflict{}, "trigger_source") {
		columns.triggerSource = true
		if err := db.Migrator().AddColumn(&dingTalkSyncConflictTriggerSourceColumn{}, "trigger_source"); err != nil {
			return dingTalkAutoSyncMigrationColumns{}, err
		}
	}
	if columns.conflictTableExists && !db.Migrator().HasIndex(&DingTalkSyncConflict{}, "idx_enterprise_dingtalk_sync_conflicts_trigger_source") {
		if err := db.Migrator().CreateIndex(&DingTalkSyncConflict{}, "TriggerSource"); err != nil {
			return dingTalkAutoSyncMigrationColumns{}, err
		}
	}
	return columns, nil
}

func ensureDingTalkScheduleColumns(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if db.Migrator().HasTable(&DingTalkConfig{}) {
		for _, column := range []string{"scheduled_full_sync_enabled", "scheduled_full_sync_cron", "scheduled_full_sync_timezone", "scheduled_full_sync_next_run_at", "scheduled_full_sync_last_run_at", "scheduled_full_sync_last_task_id", "scheduled_full_sync_last_status", "scheduled_full_sync_last_error", "scheduled_full_sync_revision", "scheduled_full_sync_updated_at"} {
			if db.Migrator().HasColumn(&DingTalkConfig{}, column) {
				continue
			}
			if err := db.Migrator().AddColumn(&dingTalkScheduleConfigColumns{}, column); err != nil {
				return err
			}
		}
	}
	if db.Migrator().HasTable(&DingTalkSyncTask{}) {
		for _, column := range []string{"trigger_source", "scheduled_for", "schedule_revision", "active_key", "runner_id"} {
			if db.Migrator().HasColumn(&DingTalkSyncTask{}, column) {
				continue
			}
			if err := db.Migrator().AddColumn(&dingTalkSyncTaskScheduleColumns{}, column); err != nil {
				return err
			}
		}
	}
	if db.Migrator().HasTable(&DingTalkConfig{}) {
		if err := db.Table(DingTalkConfig{}.TableName()).Where("scheduled_full_sync_cron = '' OR scheduled_full_sync_cron IS NULL").UpdateColumn("scheduled_full_sync_cron", "0 0 * * *").Error; err != nil {
			return err
		}
		if err := db.Table(DingTalkConfig{}.TableName()).Where("scheduled_full_sync_timezone = '' OR scheduled_full_sync_timezone IS NULL").UpdateColumn("scheduled_full_sync_timezone", "Asia/Shanghai").Error; err != nil {
			return err
		}
		if err := db.Table(DingTalkConfig{}.TableName()).Where("scheduled_full_sync_revision IS NULL OR scheduled_full_sync_revision <= 0").UpdateColumn("scheduled_full_sync_revision", 1).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasTable(&DingTalkSyncTask{}) {
		if err := db.Table(DingTalkSyncTask{}.TableName()).Where("trigger_source = '' OR trigger_source IS NULL").UpdateColumn("trigger_source", "manual").Error; err != nil {
			return err
		}
		var duplicateTenants []struct{ TenantId int }
		if err := db.Model(&DingTalkSyncTask{}).Select("tenant_id").Where("active_key IS NOT NULL AND status IN ?", []string{"pending", "running"}).Group("tenant_id").Having("COUNT(*) > 1").Scan(&duplicateTenants).Error; err != nil {
			return err
		}
		for _, duplicate := range duplicateTenants {
			var activeTasks []DingTalkSyncTask
			if err := db.Where("tenant_id = ? AND active_key IS NOT NULL AND status IN ?", duplicate.TenantId, []string{"pending", "running"}).Order("updated_at DESC, id DESC").Find(&activeTasks).Error; err != nil {
				return err
			}
			for _, stale := range activeTasks[1:] {
				if err := db.Model(&DingTalkSyncTask{}).Where("id = ?", stale.Id).Updates(map[string]any{"status": "failed", "active_key": nil, "error_summary": "duplicate active task during migration"}).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func backfillDingTalkAutoSyncColumns(db *gorm.DB, columns dingTalkAutoSyncMigrationColumns) error {
	if columns.autoSyncOnLogin {
		var configs []struct {
			Id          int
			SyncEnabled bool
		}
		if err := db.Table(DingTalkConfig{}.TableName()).Select("id", "sync_enabled").Find(&configs).Error; err != nil {
			return err
		}
		for _, config := range configs {
			if err := db.Table(DingTalkConfig{}.TableName()).
				Where("id = ?", config.Id).
				UpdateColumn("auto_sync_on_login", config.SyncEnabled).Error; err != nil {
				return err
			}
		}
	}
	if columns.triggerSource {
		if err := db.Model(&DingTalkSyncConflict{}).
			Where("tenant_id >= ?", 0).
			UpdateColumn("trigger_source", "full_sync").Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureGovernanceNotificationDeliveryColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&GovernanceNotificationDelivery{}) {
		return nil
	}
	columns := []string{
		"tenant_id",
		"source_type",
		"source_id",
		"trace_id",
		"action_type",
		"recipient_user_id",
		"recipient_kind",
		"channel_type",
		"status",
		"attempt_count",
		"max_attempts",
		"next_retry_at",
		"last_attempt_at",
		"sent_at",
		"final_failed_at",
		"error_reason",
		"dedupe_key",
		"trace_payload",
		"trigger_source",
		"manual_parent_id",
		"created_at",
		"updated_at",
	}
	for _, column := range columns {
		if db.Migrator().HasColumn(&GovernanceNotificationDelivery{}, column) {
			continue
		}
		if err := db.Migrator().AddColumn(&GovernanceNotificationDelivery{}, column); err != nil {
			return err
		}
	}
	return nil
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}

func ensureDepartmentRoleOwnerColumnsAndIndexes(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&DepartmentRole{}) {
		return nil
	}
	for _, column := range []string{"source", "effect", "external_source"} {
		if db.Migrator().HasColumn(&DepartmentRole{}, column) {
			continue
		}
		if err := db.Migrator().AddColumn(&DepartmentRole{}, column); err != nil {
			return err
		}
	}
	if err := db.Model(&DepartmentRole{}).
		Where("source = '' OR source IS NULL").
		Updates(map[string]any{
			"source":          "manual_grant",
			"effect":          "allow",
			"external_source": "",
		}).Error; err != nil {
		return err
	}
	if db.Migrator().HasIndex(&DepartmentRole{}, "uq_ent_dept_roles_role") {
		if err := db.Migrator().DropIndex(&DepartmentRole{}, "uq_ent_dept_roles_role"); err != nil {
			return err
		}
	}
	return nil
}

func ensureAlertEventDepartmentTokensColumn(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&AlertEvent{}) || db.Migrator().HasColumn(&AlertEvent{}, "department_tokens") {
		return nil
	}
	return db.Exec("ALTER TABLE enterprise_alert_events ADD COLUMN department_tokens TEXT").Error
}

func backfillAlertEventDepartmentTokens(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&AlertEvent{}) || !db.Migrator().HasColumn(&AlertEvent{}, "department_tokens") {
		return nil
	}

	var batch []AlertEvent
	return db.Where("department_tokens = '' OR department_tokens IS NULL").
		FindInBatches(&batch, 100, func(tx *gorm.DB, batchIndex int) error {
			for _, event := range batch {
				snapshot, err := event.ParsedDepartmentSnapshot()
				if err != nil {
					return err
				}
				if err := tx.Model(&AlertEvent{}).
					Where("id = ?", event.Id).
					UpdateColumn("department_tokens", buildAlertEventDepartmentTokens(snapshot)).
					Error; err != nil {
					return err
				}
			}
			return nil
		}).Error
}

func normalizeDepartmentBudgetRemaining(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&DepartmentBudget{}) {
		return nil
	}

	var batch []DepartmentBudget
	return db.FindInBatches(&batch, 100, func(tx *gorm.DB, batchIndex int) error {
		for _, budget := range batch {
			updates := map[string]any{}
			switch budget.Type {
			case DepartmentBudgetTypeSubscription:
				allocatedTotal := budget.AllocatedTotal
				if allocatedTotal < 0 {
					allocatedTotal = 0
					updates["allocated_total"] = allocatedTotal
				}
				cycleQuota := budget.CycleQuota
				if cycleQuota < 0 {
					cycleQuota = 0
					updates["cycle_quota"] = cycleQuota
				}
				remaining := cycleQuota - allocatedTotal
				if remaining < 0 {
					remaining = 0
				}
				if budget.Remaining != remaining {
					updates["remaining"] = remaining
				}
			default:
				totalQuota := budget.TotalQuota
				if totalQuota < 0 {
					totalQuota = 0
					updates["total_quota"] = totalQuota
				}
				remaining := budget.Remaining
				if remaining < 0 {
					remaining = 0
				}
				if totalQuota > 0 && remaining > totalQuota {
					remaining = totalQuota
				}
				if budget.Remaining != remaining {
					updates["remaining"] = remaining
				}
			}
			if len(updates) == 0 {
				continue
			}
			if err := tx.Model(&DepartmentBudget{}).
				Where("id = ?", budget.Id).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	}).Error
}

func ensureAlertDeliveryColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&AlertDelivery{}) {
		return nil
	}
	columns := []string{
		"tenant_id",
		"event_id",
		"rule_id",
		"channel_type",
		"status",
		"attempt_count",
		"max_attempts",
		"next_retry_at",
		"last_attempt_at",
		"sent_at",
		"final_failed_at",
		"error_reason",
		"dedupe_key",
		"trace_payload",
		"trigger_source",
		"manual_parent_id",
		"created_at",
		"updated_at",
	}
	for _, column := range columns {
		if db.Migrator().HasColumn(&AlertDelivery{}, column) {
			continue
		}
		if err := db.Migrator().AddColumn(&AlertDelivery{}, column); err != nil {
			return err
		}
	}
	return nil
}

func ensureQuotaAllocationLifecycleColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&QuotaAllocation{}) {
		return nil
	}
	columns := []string{
		"superseded_by_id",
		"supersedes_allocation_id",
		"revoke_reason",
		"reclaimed_quota",
		"processed_source",
	}
	for _, column := range columns {
		if db.Migrator().HasColumn(&QuotaAllocation{}, column) {
			continue
		}
		if err := db.Migrator().AddColumn(&QuotaAllocation{}, column); err != nil {
			return err
		}
	}
	return nil
}

func ensureQuotaRequestColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&QuotaRequest{}) {
		return nil
	}
	columns := []string{
		"tenant_id",
		"department_id",
		"department_budget_id",
		"budget_mode",
		"requester_user_id",
		"requested_quota",
		"approved_quota",
		"status",
		"approver_user_id",
		"request_reason",
		"approval_reason",
		"allocation_id",
		"idempotency_key",
		"owner_count_snapshot",
		"fallback",
		"submitted_at",
		"approved_at",
		"rejected_at",
		"fulfilled_at",
		"processed_at",
		"expires_at",
		"created_at",
		"updated_at",
	}
	for _, column := range columns {
		if db.Migrator().HasColumn(&QuotaRequest{}, column) {
			continue
		}
		if err := db.Migrator().AddColumn(&QuotaRequest{}, column); err != nil {
			return err
		}
	}
	return nil
}
