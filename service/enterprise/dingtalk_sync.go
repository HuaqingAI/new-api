package enterprise

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type DingTalkSyncClient interface {
	GetAccessToken(ctx context.Context, appKey string, appSecret string) (string, error)
	ListSubDepartments(ctx context.Context, accessToken string, departmentId int64) ([]DingTalkDepartmentInfo, error)
	ListDepartmentUsers(ctx context.Context, accessToken string, departmentId int64) ([]DingTalkDepartmentUserInfo, error)
}

type DingTalkSyncService struct {
	db     *gorm.DB
	client DingTalkSyncClient
}

type DingTalkSyncStartInput struct {
	TenantId  int
	ActorId   int
	RunInline bool
}

type DingTalkSyncTaskItem struct {
	Id                  int    `json:"id"`
	TenantId            int    `json:"tenant_id"`
	Mode                string `json:"mode"`
	Status              string `json:"status"`
	Progress            int    `json:"progress"`
	DepartmentsCreated  int    `json:"departments_created"`
	DepartmentsUpdated  int    `json:"departments_updated"`
	DepartmentsDisabled int    `json:"departments_disabled"`
	UsersCreated        int    `json:"users_created"`
	UsersUpdated        int    `json:"users_updated"`
	MembershipsCreated  int    `json:"memberships_created"`
	MembershipsUpdated  int    `json:"memberships_updated"`
	MembershipsDisabled int    `json:"memberships_disabled"`
	SkippedCount        int    `json:"skipped_count"`
	FailedCount         int    `json:"failed_count"`
	ErrorSummary        string `json:"error_summary"`
	CreatedBy           int    `json:"created_by"`
	StartedAt           int64  `json:"started_at"`
	FinishedAt          int64  `json:"finished_at"`
	CreatedAt           int64  `json:"created_at"`
	UpdatedAt           int64  `json:"updated_at"`
}

type DingTalkSyncLogItem struct {
	Id               int    `json:"id"`
	TaskId           int    `json:"task_id"`
	TenantId         int    `json:"tenant_id"`
	ObjectType       string `json:"object_type"`
	ObjectExternalId string `json:"object_external_id"`
	Action           string `json:"action"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	CreatedAt        int64  `json:"created_at"`
}

type DingTalkSyncLogQuery struct {
	TenantId   *int
	TaskId     *int
	Status     string
	ObjectType string
	StartAt    *int64
	EndAt      *int64
	Page       int
	PageSize   int
}

type DingTalkSyncLogsResult struct {
	Items    []DingTalkSyncLogItem `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type dingTalkSyncSnapshot struct {
	seenDepartmentExternalIds map[string]struct{}
	seenMembershipKeys        map[string]struct{}
}

func NewDingTalkSyncService(db *gorm.DB, client DingTalkSyncClient) *DingTalkSyncService {
	if client == nil {
		client = NewDingTalkClient()
	}
	return &DingTalkSyncService{db: db, client: client}
}

func (s *DingTalkSyncService) StartFullSync(ctx context.Context, input DingTalkSyncStartInput) (DingTalkSyncTaskItem, error) {
	if input.TenantId < 0 {
		input.TenantId = 0
	}
	config, err := s.getSyncConfig(input.TenantId)
	if err != nil {
		return DingTalkSyncTaskItem{}, err
	}
	task := entmodel.DingTalkSyncTask{
		TenantId:  input.TenantId,
		Mode:      constant.DingTalkSyncTaskModeFull,
		Status:    constant.DingTalkSyncTaskStatusPending,
		Progress:  0,
		CreatedBy: input.ActorId,
	}
	if err := s.db.WithContext(ctx).Create(&task).Error; err != nil {
		return DingTalkSyncTaskItem{}, err
	}
	if input.RunInline {
		if err := s.RunTask(ctx, task.Id, config); err != nil {
			return DingTalkSyncTaskItem{}, err
		}
		return s.GetTask(ctx, task.Id)
	}
	go func() {
		runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := s.RunTask(runCtx, task.Id, config); err != nil {
			common.SysLog(fmt.Sprintf("dingtalk sync task %d failed: %s", task.Id, err.Error()))
		}
	}()
	return mapDingTalkSyncTask(task), nil
}

func (s *DingTalkSyncService) RunTask(ctx context.Context, taskId int, config entmodel.DingTalkConfig) error {
	if taskId <= 0 {
		return ErrDingTalkSyncTaskNotFound
	}
	now := time.Now().Unix()
	if err := s.db.WithContext(ctx).Model(&entmodel.DingTalkSyncTask{}).Where("id = ?", taskId).Updates(map[string]any{
		"status":     constant.DingTalkSyncTaskStatusRunning,
		"progress":   5,
		"started_at": now,
	}).Error; err != nil {
		return err
	}

	snapshot := &dingTalkSyncSnapshot{
		seenDepartmentExternalIds: map[string]struct{}{},
		seenMembershipKeys:        map[string]struct{}{},
	}
	accessToken, err := s.client.GetAccessToken(ctx, config.AppKey, config.AppSecret)
	if err != nil {
		s.finishTaskFailed(ctx, taskId, "access_token_failed")
		return nil
	}

	rootDepartmentIds := parseDingTalkSyncScope(config.SyncScope)
	if len(rootDepartmentIds) == 0 {
		rootDepartmentIds = []int64{1}
	}
	for _, rootDepartmentId := range rootDepartmentIds {
		s.syncDepartmentTree(ctx, taskId, config.TenantId, accessToken, rootDepartmentId, nil, snapshot)
	}
	s.disableStaleRecords(ctx, taskId, config.TenantId, snapshot)
	return s.finishTask(ctx, taskId)
}

func (s *DingTalkSyncService) GetTask(ctx context.Context, taskId int) (DingTalkSyncTaskItem, error) {
	var task entmodel.DingTalkSyncTask
	if err := s.db.WithContext(ctx).Where("id = ?", taskId).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DingTalkSyncTaskItem{}, ErrDingTalkSyncTaskNotFound
		}
		return DingTalkSyncTaskItem{}, err
	}
	return mapDingTalkSyncTask(task), nil
}

func (s *DingTalkSyncService) ListLogs(ctx context.Context, query DingTalkSyncLogQuery) (DingTalkSyncLogsResult, error) {
	db := s.db.WithContext(ctx).Model(&entmodel.DingTalkSyncLog{})
	if query.TenantId != nil {
		db = db.Where("tenant_id = ?", *query.TenantId)
	}
	if query.TaskId != nil {
		db = db.Where("task_id = ?", *query.TaskId)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.ObjectType != "" {
		db = db.Where("object_type = ?", query.ObjectType)
	}
	if query.StartAt != nil {
		db = db.Where("created_at >= ?", *query.StartAt)
	}
	if query.EndAt != nil {
		db = db.Where("created_at <= ?", *query.EndAt)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return DingTalkSyncLogsResult{Items: []DingTalkSyncLogItem{}}, err
	}
	page, pageSize := normalizeDingTalkSyncLogPage(query.Page, query.PageSize)
	var logs []entmodel.DingTalkSyncLog
	if err := db.Order("created_at DESC").Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		return DingTalkSyncLogsResult{Items: []DingTalkSyncLogItem{}}, err
	}
	items := make([]DingTalkSyncLogItem, 0, len(logs))
	for _, log := range logs {
		items = append(items, mapDingTalkSyncLog(log))
	}
	return DingTalkSyncLogsResult{Items: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *DingTalkSyncService) getSyncConfig(tenantId int) (entmodel.DingTalkConfig, error) {
	var config entmodel.DingTalkConfig
	if err := s.db.Where("tenant_id = ?", tenantId).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entmodel.DingTalkConfig{}, ErrDingTalkConfigNotFound
		}
		return entmodel.DingTalkConfig{}, err
	}
	if !config.SyncEnabled {
		return entmodel.DingTalkConfig{}, ErrDingTalkSyncNotEnabled
	}
	if strings.TrimSpace(config.AppKey) == "" || strings.TrimSpace(config.AppSecret) == "" {
		return entmodel.DingTalkConfig{}, ErrDingTalkMissingCredentials
	}
	return config, nil
}
