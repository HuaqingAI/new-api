package enterprise

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const (
	AdminActionMembershipReplace      = "enterprise.organization.membership.replace"
	AdminActionMembershipAdd          = "enterprise.organization.membership.add"
	AdminActionMembershipDisable      = "enterprise.organization.membership.disable"
	AdminActionMembershipRestore      = "enterprise.organization.membership.restore"
	AdminActionDeptAdminGrant         = "enterprise.organization.department_admin.grant"
	AdminActionDeptAdminRevoke        = "enterprise.organization.department_admin.revoke"
	AdminActionDingTalkConfigSet      = "enterprise.dingtalk.config.set"
	AdminActionDingTalkTest           = "enterprise.dingtalk.connectivity.test"
	AdminActionDingTalkSyncStart      = "enterprise.dingtalk.sync.start"
	AdminActionUsageReportSet         = "enterprise.usage.report.set"
	AdminActionDepartmentBudgetCreate = "enterprise.organization.department_budget.create"
	AdminActionDepartmentBudgetReject = "enterprise.organization.department_budget.reject"

	AdminObjectUserDepartment   = "enterprise_user_department"
	AdminObjectDepartmentMember = "enterprise_department_member"
	AdminObjectDepartmentRole   = "enterprise_department_role"
	AdminObjectDingTalkConfig   = "enterprise_dingtalk_config"
	AdminObjectDingTalkSyncTask = "enterprise_dingtalk_sync_task"
	AdminObjectUsageReportJob   = "enterprise_usage_report_job"
	AdminObjectDepartmentBudget = "enterprise_department_budget"
)

type AdminActionService struct {
	db *gorm.DB
}

type AdminActionInput struct {
	TenantId    int
	ActorId     int
	ActionType  string
	ObjectType  string
	ObjectId    string
	DiffSummary string
	Payload     map[string]any
}

type AdminActionQuery struct {
	TenantId   *int
	ActorId    *int
	ActionType string
	ObjectType string
	ObjectId   string
	StartAt    *int64
	EndAt      *int64
	Page       int
	PageSize   int
}

type AdminActionItem struct {
	ActionId    int    `json:"action_id"`
	TenantId    int    `json:"tenant_id"`
	ActorId     int    `json:"actor_id"`
	ActionType  string `json:"action_type"`
	ObjectType  string `json:"object_type"`
	ObjectId    string `json:"object_id"`
	CreatedAt   int64  `json:"created_at"`
	DiffSummary string `json:"diff_summary"`
	Payload     string `json:"payload,omitempty"`
}

type AdminActionsResult struct {
	Items    []AdminActionItem `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

func NewAdminActionService(db *gorm.DB) *AdminActionService {
	return &AdminActionService{db: db}
}

func (s *AdminActionService) Write(input AdminActionInput) error {
	if input.ActorId <= 0 || strings.TrimSpace(input.ActionType) == "" || strings.TrimSpace(input.ObjectType) == "" {
		return ErrInvalidAdminActionInput
	}
	payload := sanitizeAdminActionPayload(input.Payload)
	payloadBytes, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	action := entmodel.AdminAction{
		TenantId:    input.TenantId,
		ActorId:     input.ActorId,
		ActionType:  truncateAdminActionText(input.ActionType, 96),
		ObjectType:  truncateAdminActionText(input.ObjectType, 64),
		ObjectId:    truncateAdminActionText(input.ObjectId, 128),
		DiffSummary: truncateAdminActionText(sanitizeAdminActionString(input.DiffSummary), 1024),
		Payload:     string(payloadBytes),
	}
	return s.db.Create(&action).Error
}

func (s *AdminActionService) List(query AdminActionQuery) (AdminActionsResult, error) {
	db := s.db.Model(&entmodel.AdminAction{})
	db = applyAdminActionQuery(db, query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return AdminActionsResult{Items: []AdminActionItem{}}, err
	}

	page, pageSize := normalizeAdminActionPage(query.Page, query.PageSize)
	var actions []entmodel.AdminAction
	if err := db.Order("created_at DESC").Order("action_id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&actions).Error; err != nil {
		return AdminActionsResult{Items: []AdminActionItem{}}, err
	}

	items := make([]AdminActionItem, 0, len(actions))
	for _, action := range actions {
		items = append(items, mapAdminAction(action))
	}
	return AdminActionsResult{
		Items:    items,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AdminActionService) Get(actionId int) (AdminActionItem, error) {
	if actionId <= 0 {
		return AdminActionItem{}, ErrInvalidAdminActionInput
	}
	var action entmodel.AdminAction
	if err := s.db.Where("action_id = ?", actionId).First(&action).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AdminActionItem{}, ErrAdminActionNotFound
		}
		return AdminActionItem{}, err
	}
	return mapAdminAction(action), nil
}

func applyAdminActionQuery(db *gorm.DB, query AdminActionQuery) *gorm.DB {
	if query.TenantId != nil {
		db = db.Where("tenant_id = ?", *query.TenantId)
	}
	if query.ActorId != nil {
		db = db.Where("actor_id = ?", *query.ActorId)
	}
	if query.ActionType != "" {
		db = db.Where("action_type = ?", query.ActionType)
	}
	if query.ObjectType != "" {
		db = db.Where("object_type = ?", query.ObjectType)
	}
	if query.ObjectId != "" {
		db = db.Where("object_id = ?", query.ObjectId)
	}
	if query.StartAt != nil {
		db = db.Where("created_at >= ?", *query.StartAt)
	}
	if query.EndAt != nil {
		db = db.Where("created_at <= ?", *query.EndAt)
	}
	return db
}

func normalizeAdminActionPage(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func mapAdminAction(action entmodel.AdminAction) AdminActionItem {
	return AdminActionItem{
		ActionId:    action.ActionId,
		TenantId:    action.TenantId,
		ActorId:     action.ActorId,
		ActionType:  action.ActionType,
		ObjectType:  action.ObjectType,
		ObjectId:    action.ObjectId,
		CreatedAt:   action.CreatedAt,
		DiffSummary: action.DiffSummary,
		Payload:     action.Payload,
	}
}

func sanitizeAdminActionPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	sanitized := make(map[string]any, len(payload))
	for key, value := range payload {
		if isSensitiveAdminActionKey(key) {
			sanitized[key] = "[REDACTED]"
			continue
		}
		sanitized[key] = sanitizeAdminActionValue(value)
	}
	return sanitized
}

func sanitizeAdminActionValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return sanitizeAdminActionPayload(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeAdminActionValue(item))
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeAdminActionPayload(item))
		}
		return out
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeAdminActionString(item))
		}
		return out
	case string:
		return sanitizeAdminActionString(typed)
	default:
		return value
	}
}

func sanitizeAdminActionString(value string) string {
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"secret", "token", "password", "credential", "webhook", "private_key", "app_key"} {
		if strings.Contains(lower, marker) {
			return "[REDACTED]"
		}
	}
	return value
}

func isSensitiveAdminActionKey(key string) bool {
	key = strings.ToLower(key)
	for _, marker := range []string{"secret", "token", "password", "credential", "webhook", "private_key", "app_key"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func truncateAdminActionText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func MembershipObjectId(departmentId int, userId int) string {
	return fmt.Sprintf("%d:%d", departmentId, userId)
}
