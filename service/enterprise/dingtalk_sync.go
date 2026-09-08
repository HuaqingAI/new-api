package enterprise

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
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

type DingTalkSyncConflictItem struct {
	Id              int    `json:"id"`
	TenantId        int    `json:"tenant_id"`
	TaskId          int    `json:"task_id"`
	TriggerSource   string `json:"trigger_source"`
	ExternalUserId  string `json:"external_user_id"`
	UnionId         string `json:"union_id"`
	Mobile          string `json:"mobile"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	ConflictType    string `json:"conflict_type"`
	CandidateUserId int    `json:"candidate_user_id"`
	Details         string `json:"details"`
	Status          string `json:"status"`
	LastTaskId      int    `json:"last_task_id"`
	ResolvedBy      int    `json:"resolved_by"`
	ResolvedAt      int64  `json:"resolved_at"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
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

type DingTalkSyncConflictQuery struct {
	TenantId *int
	Status   string
	Page     int
	PageSize int
}

type DingTalkSyncConflictResolveInput struct {
	TenantId                int
	ConflictId              int
	ExpectedCandidateUserId *int
	ActorId                 int
}

type DingTalkSyncLogsResult struct {
	Items    []DingTalkSyncLogItem `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type DingTalkSyncConflictsResult struct {
	Items    []DingTalkSyncConflictItem `json:"items"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}

type dingTalkSyncSnapshot struct {
	seenDepartmentExternalIds map[string]struct{}
	seenMembershipKeys        map[string]struct{}
	seenOwnerKeys             map[string]struct{}
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
	defer ClearDingTalkScopeCache(config.TenantId)
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
		seenOwnerKeys:             map[string]struct{}{},
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
		s.syncDepartmentTree(ctx, taskId, config.TenantId, config.CorpId, accessToken, rootDepartmentId, nil, snapshot)
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

func (s *DingTalkSyncService) ListConflicts(ctx context.Context, query DingTalkSyncConflictQuery) (DingTalkSyncConflictsResult, error) {
	db := s.db.WithContext(ctx).Model(&entmodel.DingTalkSyncConflict{})
	if query.TenantId != nil {
		db = db.Where("tenant_id = ?", *query.TenantId)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return DingTalkSyncConflictsResult{Items: []DingTalkSyncConflictItem{}}, err
	}
	page, pageSize := normalizeDingTalkSyncLogPage(query.Page, query.PageSize)
	var conflicts []entmodel.DingTalkSyncConflict
	if err := db.Order("updated_at DESC").Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&conflicts).Error; err != nil {
		return DingTalkSyncConflictsResult{Items: []DingTalkSyncConflictItem{}}, err
	}
	items := make([]DingTalkSyncConflictItem, 0, len(conflicts))
	for _, conflict := range conflicts {
		items = append(items, mapDingTalkSyncConflict(conflict))
	}
	return DingTalkSyncConflictsResult{Items: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *DingTalkSyncService) ResolveConflictByCandidate(ctx context.Context, input DingTalkSyncConflictResolveInput) (DingTalkSyncConflictItem, error) {
	if input.ConflictId <= 0 {
		return DingTalkSyncConflictItem{}, ErrDingTalkSyncConflictNotFound
	}
	if input.TenantId < 0 {
		input.TenantId = 0
	}

	var resolved entmodel.DingTalkSyncConflict
	now := time.Now().Unix()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conflict entmodel.DingTalkSyncConflict
		if err := tx.Where("id = ? AND tenant_id = ?", input.ConflictId, input.TenantId).First(&conflict).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDingTalkSyncConflictNotFound
			}
			return err
		}
		if conflict.Status != constant.DingTalkSyncConflictStatusPending {
			return ErrDingTalkSyncConflictNotPending
		}
		if conflict.CandidateUserId <= 0 {
			return ErrDingTalkSyncConflictNoCandidate
		}
		if input.ExpectedCandidateUserId != nil && *input.ExpectedCandidateUserId != conflict.CandidateUserId {
			return ErrDingTalkSyncConflictNoCandidate
		}
		identityKey := dingtalkSyncConflictIdentityKey(conflict)
		if identityKey == "" {
			return ErrDingTalkOAuthIdentityMissing
		}
		if err := validateDingTalkSyncConflictCandidate(tx, input.TenantId, conflict); err != nil {
			return err
		}

		var user model.User
		if err := tx.Where("id = ?", conflict.CandidateUserId).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if user.Status != common.UserStatusEnabled {
			return ErrDingTalkOAuthUserDisabled
		}

		var byIdentity entmodel.DingTalkIdentity
		identityErr := tx.Where("tenant_id = ? AND identity_key = ?", input.TenantId, identityKey).First(&byIdentity).Error
		if identityErr != nil && !errors.Is(identityErr, gorm.ErrRecordNotFound) {
			return identityErr
		}
		if identityErr == nil && byIdentity.UserId != conflict.CandidateUserId {
			return ErrDingTalkOAuthBindingConflict
		}

		var binding entmodel.DingTalkIdentity
		userBindingErr := tx.Where("tenant_id = ? AND user_id = ?", input.TenantId, conflict.CandidateUserId).First(&binding).Error
		if userBindingErr != nil && !errors.Is(userBindingErr, gorm.ErrRecordNotFound) {
			return userBindingErr
		}
		if userBindingErr == nil && identityErr == nil && binding.Id != byIdentity.Id {
			return ErrDingTalkOAuthBindingConflict
		}

		bindingUpdate := map[string]any{
			"identity_key":     identityKey,
			"union_id":         strings.TrimSpace(conflict.UnionId),
			"external_user_id": strings.TrimSpace(conflict.ExternalUserId),
			"mobile":           strings.TrimSpace(conflict.Mobile),
			"user_id":          conflict.CandidateUserId,
			"status":           DingTalkIdentityStatusActive,
		}
		if userBindingErr == nil {
			if err := tx.Model(&binding).Updates(bindingUpdate).Error; err != nil {
				return err
			}
		} else if identityErr == nil {
			if err := tx.Model(&byIdentity).Updates(bindingUpdate).Error; err != nil {
				return err
			}
		} else {
			binding = entmodel.DingTalkIdentity{
				TenantId:       input.TenantId,
				IdentityKey:    identityKey,
				UnionId:        strings.TrimSpace(conflict.UnionId),
				ExternalUserId: strings.TrimSpace(conflict.ExternalUserId),
				Mobile:         strings.TrimSpace(conflict.Mobile),
				UserId:         conflict.CandidateUserId,
				Status:         DingTalkIdentityStatusActive,
			}
			if err := tx.Create(&binding).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&conflict).Updates(map[string]any{
			"status":      constant.DingTalkSyncConflictStatusResolved,
			"resolved_by": input.ActorId,
			"resolved_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", conflict.Id).First(&resolved).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return DingTalkSyncConflictItem{}, err
	}
	return mapDingTalkSyncConflict(resolved), nil
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
