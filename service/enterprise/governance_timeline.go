package enterprise

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const (
	GovernanceSourceAdminAction      = "admin_action"
	GovernanceSourceBudgetDelegation = "budget_delegation"
	GovernanceSourceQuotaAllocation  = "quota_allocation"
	GovernanceSourceQuotaRequest     = "quota_request"
)

type GovernanceTimelineService struct {
	db *gorm.DB
}

type GovernanceTimelineQuery struct {
	TenantId     int
	DepartmentId *int
	SourceId     *int
	ActorId      *int
	ActionType   string
	SourceType   string
	Status       string
	StartAt      *int64
	EndAt        *int64
	Page         int
	PageSize     int
	ViewerId     int
}

type GovernanceTimelineTarget struct {
	DepartmentId   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
	UserId         int    `json:"user_id"`
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`
	ObjectType     string `json:"object_type"`
	ObjectId       string `json:"object_id"`
}

type GovernanceTimelineItem struct {
	TraceId       string                   `json:"trace_id"`
	SourceType    string                   `json:"source_type"`
	SourceId      int                      `json:"source_id"`
	ActionType    string                   `json:"action_type"`
	TenantId      int                      `json:"tenant_id"`
	ActorId       int                      `json:"actor_id"`
	ActorName     string                   `json:"actor_name"`
	Target        GovernanceTimelineTarget `json:"target"`
	QuotaDelta    int64                    `json:"quota_delta"`
	BeforeQuota   int64                    `json:"before_quota"`
	AfterQuota    int64                    `json:"after_quota"`
	Status        string                   `json:"status"`
	OccurredAt    int64                    `json:"occurred_at"`
	DetailRoute   string                   `json:"detail_route"`
	DetailAPIPath string                   `json:"detail_api_path"`
	Summary       string                   `json:"summary"`
}

type GovernanceTimelineResult struct {
	Items    []GovernanceTimelineItem `json:"items"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

func NewGovernanceTimelineService(db *gorm.DB) *GovernanceTimelineService {
	if db == nil {
		db = model.DB
	}
	return &GovernanceTimelineService{db: db}
}

func (s *GovernanceTimelineService) List(query GovernanceTimelineQuery) (GovernanceTimelineResult, error) {
	if s == nil || s.db == nil {
		return GovernanceTimelineResult{Items: []GovernanceTimelineItem{}}, nil
	}
	if query.TenantId < 0 || query.ViewerId <= 0 {
		return GovernanceTimelineResult{}, ErrInvalidGovernanceTimelineQuery
	}
	page, pageSize := normalizeGovernancePage(query.Page, query.PageSize)
	scope, err := s.buildViewerScope(query)
	if err != nil {
		return GovernanceTimelineResult{}, err
	}
	items := make([]GovernanceTimelineItem, 0)
	if query.SourceType == "" || query.SourceType == GovernanceSourceAdminAction {
		adminItems, err := s.listAdminActionItems(query, scope)
		if err != nil {
			return GovernanceTimelineResult{}, err
		}
		items = append(items, adminItems...)
	}
	if query.SourceType == "" || query.SourceType == GovernanceSourceBudgetDelegation {
		delegationItems, err := s.listBudgetDelegationItems(query, scope)
		if err != nil {
			return GovernanceTimelineResult{}, err
		}
		items = append(items, delegationItems...)
	}
	if query.SourceType == "" || query.SourceType == GovernanceSourceQuotaAllocation {
		allocationItems, err := s.listQuotaAllocationItems(query, scope)
		if err != nil {
			return GovernanceTimelineResult{}, err
		}
		items = append(items, allocationItems...)
	}
	if query.SourceType == "" || query.SourceType == GovernanceSourceQuotaRequest {
		requestItems, err := s.listQuotaRequestItems(query, scope)
		if err != nil {
			return GovernanceTimelineResult{}, err
		}
		items = append(items, requestItems...)
	}
	if query.SourceType != "" && len(items) == 0 && !isKnownGovernanceSource(query.SourceType) {
		return GovernanceTimelineResult{}, ErrInvalidGovernanceTimelineQuery
	}
	items = filterGovernanceItems(items, query)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].OccurredAt != items[j].OccurredAt {
			return items[i].OccurredAt > items[j].OccurredAt
		}
		if items[i].SourceType != items[j].SourceType {
			return items[i].SourceType < items[j].SourceType
		}
		return items[i].SourceId > items[j].SourceId
	})
	total := len(items)
	start := (page - 1) * pageSize
	if start >= len(items) {
		items = []GovernanceTimelineItem{}
	} else {
		end := start + pageSize
		if end > len(items) {
			end = len(items)
		}
		items = items[start:end]
	}
	return GovernanceTimelineResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

type governanceViewerScope struct {
	admin            bool
	departmentIds    map[int]struct{}
	requesterUserId  int
	filterDepartment *int
	allowOwnRequests bool
}

func (s *GovernanceTimelineService) buildViewerScope(query GovernanceTimelineQuery) (governanceViewerScope, error) {
	scope := governanceViewerScope{filterDepartment: query.DepartmentId, departmentIds: map[int]struct{}{}}
	if model.IsAdmin(query.ViewerId) {
		scope.admin = true
		return scope, nil
	}
	ids, err := NewPermissionService(s.db).ListManageableDepartmentIds(query.ViewerId, query.TenantId)
	if err != nil {
		return scope, err
	}
	for _, id := range ids {
		scope.departmentIds[id] = struct{}{}
	}
	scope.requesterUserId = query.ViewerId
	scope.allowOwnRequests = true
	return scope, nil
}

func (s *GovernanceTimelineService) listAdminActionItems(query GovernanceTimelineQuery, scope governanceViewerScope) ([]GovernanceTimelineItem, error) {
	db := s.db.Model(&entmodel.AdminAction{}).Where("tenant_id = ?", query.TenantId)
	if query.ActorId != nil {
		db = db.Where("actor_id = ?", *query.ActorId)
	}
	if query.ActionType != "" {
		db = db.Where("action_type = ?", query.ActionType)
	}
	if query.StartAt != nil {
		db = db.Where("created_at >= ?", *query.StartAt)
	}
	if query.EndAt != nil {
		db = db.Where("created_at <= ?", *query.EndAt)
	}
	var actions []entmodel.AdminAction
	if err := db.Find(&actions).Error; err != nil {
		return nil, err
	}
	names := s.userDisplayNames(collectAdminActorIds(actions))
	items := make([]GovernanceTimelineItem, 0, len(actions))
	for _, action := range actions {
		target := targetFromAdminAction(action)
		if !scopeAllowsTimelineTarget(scope, target, 0) {
			continue
		}
		status := "recorded"
		if parsed := parseAdminPayload(action.Payload); parsed != nil {
			if value := stringFromMap(parsed, "status"); value != "" {
				status = value
			}
		}
		items = append(items, GovernanceTimelineItem{
			TraceId:       fmt.Sprintf("admin_action:%d", action.ActionId),
			SourceType:    GovernanceSourceAdminAction,
			SourceId:      action.ActionId,
			ActionType:    action.ActionType,
			TenantId:      action.TenantId,
			ActorId:       action.ActorId,
			ActorName:     names[action.ActorId],
			Target:        target,
			QuotaDelta:    int64FromAdminPayload(action.Payload, "quota_delta", "approved_quota", "requested_quota", "committed_quota"),
			Status:        status,
			OccurredAt:    action.CreatedAt,
			DetailRoute:   buildGovernanceDetailRoute(GovernanceSourceAdminAction, action.ActionId, action.TenantId),
			DetailAPIPath: buildGovernanceDetailAPIPath(GovernanceSourceAdminAction, action.ActionId, action.TenantId),
			Summary:       action.DiffSummary,
		})
	}
	return items, nil
}

func (s *GovernanceTimelineService) listBudgetDelegationItems(query GovernanceTimelineQuery, scope governanceViewerScope) ([]GovernanceTimelineItem, error) {
	type row struct {
		entmodel.BudgetDelegation
		SourceDepartmentName string
		TargetDepartmentName string
	}
	db := s.db.Table(entmodel.BudgetDelegation{}.TableName()+" AS delegations").
		Select("delegations.*, src.name AS source_department_name, dst.name AS target_department_name").
		Joins("LEFT JOIN enterprise_departments AS src ON src.id = delegations.source_department_id").
		Joins("LEFT JOIN enterprise_departments AS dst ON dst.id = delegations.target_department_id").
		Where("delegations.tenant_id = ?", query.TenantId)
	if query.ActorId != nil {
		db = db.Where("delegations.actor_id = ?", *query.ActorId)
	}
	if query.Status != "" {
		db = db.Where("delegations.status = ?", query.Status)
	}
	if query.StartAt != nil {
		db = db.Where("delegations.created_at >= ?", *query.StartAt)
	}
	if query.EndAt != nil {
		db = db.Where("delegations.created_at <= ?", *query.EndAt)
	}
	var rows []row
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}
	actorIds := map[int]struct{}{}
	for _, row := range rows {
		if row.ActorId > 0 {
			actorIds[row.ActorId] = struct{}{}
		}
	}
	names := s.userDisplayNames(mapKeysInt(actorIds))
	items := make([]GovernanceTimelineItem, 0, len(rows))
	for _, row := range rows {
		target := GovernanceTimelineTarget{
			DepartmentId:   row.TargetDepartmentId,
			DepartmentName: row.TargetDepartmentName,
			ObjectType:     "enterprise_budget_delegation",
			ObjectId:       strconv.Itoa(row.Id),
		}
		if !scopeAllowsDelegation(scope, row.SourceDepartmentId, row.TargetDepartmentId) {
			continue
		}
		actionType := AdminActionBudgetDelegationCreate
		if row.SupersededById > 0 || row.Status == entmodel.BudgetDelegationStatusSuperseded {
			actionType = AdminActionBudgetDelegationSupersede
		}
		items = append(items, GovernanceTimelineItem{
			TraceId:       fmt.Sprintf("budget_delegation:%d", row.Id),
			SourceType:    GovernanceSourceBudgetDelegation,
			SourceId:      row.Id,
			ActionType:    actionType,
			TenantId:      row.TenantId,
			ActorId:       row.ActorId,
			ActorName:     names[row.ActorId],
			Target:        target,
			QuotaDelta:    row.CommittedQuota,
			Status:        row.Status,
			OccurredAt:    nonZeroInt64(row.ProcessedAt, row.CreatedAt),
			DetailRoute:   buildGovernanceDetailRoute(GovernanceSourceBudgetDelegation, row.Id, row.TenantId),
			DetailAPIPath: buildGovernanceDetailAPIPath(GovernanceSourceBudgetDelegation, row.Id, row.TenantId),
			Summary:       row.Reason,
		})
	}
	return items, nil
}

func (s *GovernanceTimelineService) listQuotaAllocationItems(query GovernanceTimelineQuery, scope governanceViewerScope) ([]GovernanceTimelineItem, error) {
	type row struct {
		entmodel.QuotaAllocation
		DepartmentName    string
		TargetUsername    string
		TargetDisplayName string
	}
	db := s.db.Model(&entmodel.QuotaAllocation{}).
		Select("enterprise_quota_allocations.*, enterprise_departments.name AS department_name, users.username AS target_username, users.display_name AS target_display_name").
		Joins("LEFT JOIN enterprise_departments ON enterprise_departments.id = enterprise_quota_allocations.department_id").
		Joins("LEFT JOIN users ON users.id = enterprise_quota_allocations.target_user_id").
		Where("enterprise_quota_allocations.tenant_id = ?", query.TenantId)
	if query.ActorId != nil {
		db = db.Where("enterprise_quota_allocations.actor_id = ?", *query.ActorId)
	}
	if query.Status != "" {
		db = db.Where("enterprise_quota_allocations.status = ?", query.Status)
	}
	if query.StartAt != nil {
		db = db.Where("enterprise_quota_allocations.created_at >= ? OR enterprise_quota_allocations.processed_at >= ?", *query.StartAt, *query.StartAt)
	}
	if query.EndAt != nil {
		db = db.Where("enterprise_quota_allocations.created_at <= ? OR enterprise_quota_allocations.processed_at <= ?", *query.EndAt, *query.EndAt)
	}
	var rows []row
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}
	actorIds := map[int]struct{}{}
	for _, row := range rows {
		if row.ActorId > 0 {
			actorIds[row.ActorId] = struct{}{}
		}
	}
	names := s.userDisplayNames(mapKeysInt(actorIds))
	items := make([]GovernanceTimelineItem, 0, len(rows))
	for _, row := range rows {
		target := GovernanceTimelineTarget{
			DepartmentId:   row.DepartmentId,
			DepartmentName: row.DepartmentName,
			UserId:         row.TargetUserId,
			Username:       row.TargetUsername,
			DisplayName:    row.TargetDisplayName,
			ObjectType:     "enterprise_quota_allocation",
			ObjectId:       strconv.Itoa(row.Id),
		}
		if !scopeAllowsTimelineTarget(scope, target, row.TargetUserId) {
			continue
		}
		items = append(items, GovernanceTimelineItem{
			TraceId:       fmt.Sprintf("quota_allocation:%d", row.Id),
			SourceType:    GovernanceSourceQuotaAllocation,
			SourceId:      row.Id,
			ActionType:    quotaAllocationTimelineAction(row.QuotaAllocation),
			TenantId:      row.TenantId,
			ActorId:       row.ActorId,
			ActorName:     names[row.ActorId],
			Target:        target,
			QuotaDelta:    quotaAllocationTimelineDelta(row.QuotaAllocation),
			Status:        row.Status,
			OccurredAt:    nonZeroInt64(row.ProcessedAt, row.CreatedAt),
			DetailRoute:   buildGovernanceDetailRoute(GovernanceSourceQuotaAllocation, row.Id, row.TenantId),
			DetailAPIPath: buildGovernanceDetailAPIPath(GovernanceSourceQuotaAllocation, row.Id, row.TenantId),
			Summary:       firstNonEmptyGovernance(row.RevokeReason, row.Reason, row.ProcessedSource),
		})
	}
	return items, nil
}

func (s *GovernanceTimelineService) listQuotaRequestItems(query GovernanceTimelineQuery, scope governanceViewerScope) ([]GovernanceTimelineItem, error) {
	type row struct {
		entmodel.QuotaRequest
		DepartmentName       string
		RequesterUsername    string
		RequesterDisplayName string
		ApproverUsername     string
	}
	db := s.db.Model(&entmodel.QuotaRequest{}).
		Select("enterprise_quota_requests.*, enterprise_departments.name AS department_name, requester.username AS requester_username, requester.display_name AS requester_display_name, approver.username AS approver_username").
		Joins("LEFT JOIN enterprise_departments ON enterprise_departments.id = enterprise_quota_requests.department_id").
		Joins("LEFT JOIN users AS requester ON requester.id = enterprise_quota_requests.requester_user_id").
		Joins("LEFT JOIN users AS approver ON approver.id = enterprise_quota_requests.approver_user_id").
		Where("enterprise_quota_requests.tenant_id = ?", query.TenantId)
	if query.ActorId != nil {
		db = db.Where("(enterprise_quota_requests.requester_user_id = ? OR enterprise_quota_requests.approver_user_id = ?)", *query.ActorId, *query.ActorId)
	}
	if query.Status != "" {
		db = db.Where("enterprise_quota_requests.status = ?", query.Status)
	}
	if query.StartAt != nil {
		db = db.Where("enterprise_quota_requests.created_at >= ? OR enterprise_quota_requests.processed_at >= ?", *query.StartAt, *query.StartAt)
	}
	if query.EndAt != nil {
		db = db.Where("enterprise_quota_requests.created_at <= ? OR enterprise_quota_requests.processed_at <= ?", *query.EndAt, *query.EndAt)
	}
	var rows []row
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]GovernanceTimelineItem, 0, len(rows))
	for _, row := range rows {
		target := GovernanceTimelineTarget{
			DepartmentId:   row.DepartmentId,
			DepartmentName: row.DepartmentName,
			UserId:         row.RequesterUserId,
			Username:       row.RequesterUsername,
			DisplayName:    row.RequesterDisplayName,
			ObjectType:     "enterprise_quota_request",
			ObjectId:       strconv.Itoa(row.Id),
		}
		if !scopeAllowsTimelineTarget(scope, target, row.RequesterUserId) {
			continue
		}
		actorId := row.RequesterUserId
		actorName := row.RequesterUsername
		if row.ApproverUserId > 0 && row.Status != entmodel.QuotaRequestStatusSubmitted {
			actorId = row.ApproverUserId
			actorName = row.ApproverUsername
		}
		items = append(items, GovernanceTimelineItem{
			TraceId:       fmt.Sprintf("quota_request:%d", row.Id),
			SourceType:    GovernanceSourceQuotaRequest,
			SourceId:      row.Id,
			ActionType:    quotaRequestTimelineAction(row.QuotaRequest),
			TenantId:      row.TenantId,
			ActorId:       actorId,
			ActorName:     actorName,
			Target:        target,
			QuotaDelta:    quotaRequestTimelineDelta(row.QuotaRequest),
			Status:        row.Status,
			OccurredAt:    quotaRequestOccurredAt(row.QuotaRequest),
			DetailRoute:   buildGovernanceDetailRoute(GovernanceSourceQuotaRequest, row.Id, row.TenantId),
			DetailAPIPath: buildGovernanceDetailAPIPath(GovernanceSourceQuotaRequest, row.Id, row.TenantId),
			Summary:       firstNonEmptyGovernance(row.ApprovalReason, row.RequestReason, row.Fallback),
		})
	}
	return items, nil
}

func normalizeGovernancePage(page int, pageSize int) (int, int) {
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

func isKnownGovernanceSource(sourceType string) bool {
	switch sourceType {
	case GovernanceSourceAdminAction, GovernanceSourceBudgetDelegation, GovernanceSourceQuotaAllocation, GovernanceSourceQuotaRequest:
		return true
	default:
		return false
	}
}

func filterGovernanceItems(items []GovernanceTimelineItem, query GovernanceTimelineQuery) []GovernanceTimelineItem {
	out := make([]GovernanceTimelineItem, 0, len(items))
	for _, item := range items {
		if query.DepartmentId != nil && item.Target.DepartmentId != *query.DepartmentId {
			continue
		}
		if query.SourceId != nil && item.SourceId != *query.SourceId {
			continue
		}
		if query.ActorId != nil && item.ActorId != *query.ActorId {
			continue
		}
		if query.ActionType != "" && item.ActionType != query.ActionType {
			continue
		}
		if query.Status != "" && item.Status != query.Status {
			continue
		}
		if query.StartAt != nil && item.OccurredAt < *query.StartAt {
			continue
		}
		if query.EndAt != nil && item.OccurredAt > *query.EndAt {
			continue
		}
		out = append(out, item)
	}
	if out == nil {
		return []GovernanceTimelineItem{}
	}
	return out
}

func scopeAllowsTimelineTarget(scope governanceViewerScope, target GovernanceTimelineTarget, requesterUserId int) bool {
	if scope.admin {
		return true
	}
	if scope.filterDepartment != nil {
		if _, ok := scope.departmentIds[*scope.filterDepartment]; !ok {
			return requesterUserId == scope.requesterUserId && target.DepartmentId == *scope.filterDepartment
		}
	}
	if _, ok := scope.departmentIds[target.DepartmentId]; ok {
		return true
	}
	return scope.allowOwnRequests && requesterUserId == scope.requesterUserId && requesterUserId > 0
}

func scopeAllowsDelegation(scope governanceViewerScope, sourceDepartmentId int, targetDepartmentId int) bool {
	if scope.admin {
		return true
	}
	if _, ok := scope.departmentIds[sourceDepartmentId]; ok {
		return true
	}
	if _, ok := scope.departmentIds[targetDepartmentId]; ok {
		return true
	}
	return false
}

func targetFromAdminAction(action entmodel.AdminAction) GovernanceTimelineTarget {
	payload := parseAdminPayload(action.Payload)
	target := GovernanceTimelineTarget{
		ObjectType: action.ObjectType,
		ObjectId:   action.ObjectId,
	}
	if payload != nil {
		target.DepartmentId = intFromMap(payload, "department_id", "source_department_id", "target_department_id")
		target.UserId = intFromMap(payload, "user_id", "requester_user_id", "target_user_id")
	}
	return target
}

func parseAdminPayload(payload string) map[string]any {
	payload = strings.TrimSpace(payload)
	if payload == "" || payload == "{}" {
		return map[string]any{}
	}
	var out map[string]any
	if err := common.UnmarshalJsonStr(payload, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func int64FromAdminPayload(payload string, keys ...string) int64 {
	parsed := parseAdminPayload(payload)
	for _, key := range keys {
		switch value := parsed[key].(type) {
		case float64:
			return int64(value)
		case int64:
			return value
		case int:
			return int64(value)
		case string:
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func intFromMap(values map[string]any, keys ...string) int {
	for _, key := range keys {
		switch value := values[key].(type) {
		case float64:
			return int(value)
		case int:
			return value
		case int64:
			return int(value)
		case string:
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func quotaAllocationTimelineAction(allocation entmodel.QuotaAllocation) string {
	if allocation.ProcessedSource != "" {
		return "enterprise.organization.quota_allocation." + strings.TrimPrefix(allocation.ProcessedSource, "manual_")
	}
	if allocation.SupersedesAllocationId > 0 {
		return "enterprise.organization.quota_allocation.supersede.create"
	}
	return "enterprise.organization.quota_allocation.create"
}

func quotaAllocationTimelineDelta(allocation entmodel.QuotaAllocation) int64 {
	if allocation.ProcessedAt > 0 {
		if allocation.ReclaimedQuota > 0 {
			return -allocation.ReclaimedQuota
		}
		return -allocation.CommittedQuota
	}
	return allocation.CommittedQuota
}

func quotaRequestTimelineAction(request entmodel.QuotaRequest) string {
	switch request.Status {
	case entmodel.QuotaRequestStatusRejected:
		return AdminActionQuotaRequestReject
	case entmodel.QuotaRequestStatusApproved, entmodel.QuotaRequestStatusFulfilled:
		return AdminActionQuotaRequestApprove
	default:
		return AdminActionQuotaRequestSubmit
	}
}

func quotaRequestTimelineDelta(request entmodel.QuotaRequest) int64 {
	if request.ApprovedQuota > 0 {
		return request.ApprovedQuota
	}
	return request.RequestedQuota
}

func quotaRequestOccurredAt(request entmodel.QuotaRequest) int64 {
	return nonZeroInt64(request.ProcessedAt, request.FulfilledAt, request.ApprovedAt, request.RejectedAt, request.SubmittedAt, request.CreatedAt)
}

func buildGovernanceDetailRoute(sourceType string, sourceId int, tenantId int) string {
	route := fmt.Sprintf("/enterprise-organization?governance_source=%s&governance_id=%d", sourceType, sourceId)
	if tenantId > 0 {
		route += fmt.Sprintf("&tenant_id=%d", tenantId)
	}
	return route
}

func buildGovernanceDetailAPIPath(sourceType string, sourceId int, tenantId int) string {
	path := fmt.Sprintf("/api/enterprise/governance/timeline?source_type=%s&page=1&page_size=20", sourceType)
	if sourceId > 0 {
		path += fmt.Sprintf("&source_id=%d", sourceId)
	}
	if tenantId > 0 {
		path += fmt.Sprintf("&tenant_id=%d", tenantId)
	}
	return path
}

func nonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmptyGovernance(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *GovernanceTimelineService) userDisplayNames(ids []int) map[int]string {
	out := map[int]string{}
	if len(ids) == 0 {
		return out
	}
	var users []model.User
	if err := s.db.Select("id", "username", "display_name").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return out
	}
	for _, user := range users {
		out[user.Id] = firstNonEmptyGovernance(user.DisplayName, user.Username, fmt.Sprintf("#%d", user.Id))
	}
	return out
}

func collectAdminActorIds(actions []entmodel.AdminAction) []int {
	ids := make(map[int]struct{})
	for _, action := range actions {
		if action.ActorId > 0 {
			ids[action.ActorId] = struct{}{}
		}
	}
	return mapKeysInt(ids)
}

func mapKeysInt(values map[int]struct{}) []int {
	ids := make([]int, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}
