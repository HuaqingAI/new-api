package enterprise

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const (
	GovernanceActionQuotaRequestSubmitted = "enterprise.organization.quota_request.submit"
	GovernanceActionQuotaRequestApproved  = "enterprise.organization.quota_request.approve"
	GovernanceActionQuotaRequestRejected  = "enterprise.organization.quota_request.reject"
	GovernanceActionAllocationCreated     = "enterprise.organization.quota_allocation.create"
	GovernanceActionAllocationReclaim     = "enterprise.organization.quota_allocation.reclaim"
	GovernanceActionAllocationCancel      = "enterprise.organization.quota_allocation.cancel"
	GovernanceActionAllocationRevoke      = "enterprise.organization.quota_allocation.revoke"
)

type GovernanceNotificationService struct {
	db  *gorm.DB
	now func() time.Time
}

type GovernanceNotificationQuery struct {
	TenantId        int
	DepartmentId    *int
	RecipientUserId *int
	SourceType      string
	SourceId        *int
	ActionType      string
	Status          string
	Page            int
	PageSize        int
	ViewerId        int
}

type GovernanceNotificationItem struct {
	Id              int
	TenantId        int
	SourceType      string
	SourceId        int
	TraceId         string
	ActionType      string
	RecipientUserId int
	RecipientKind   string
	ChannelType     string
	Status          string
	AttemptCount    int
	MaxAttempts     int
	NextRetryAt     int64
	LastAttemptAt   int64
	SentAt          int64
	FinalFailedAt   int64
	ErrorReason     string
	DedupeKey       string
	TriggerSource   string
	ManualParentId  *int
	TraceSummary    string
	Trace           *entmodel.GovernanceNotificationTracePayload
	CreatedAt       int64
	UpdatedAt       int64
}

type GovernanceNotificationListResult struct {
	Items    []GovernanceNotificationItem
	Total    int
	Page     int
	PageSize int
}

type GovernanceNotificationResendResult struct {
	Item         GovernanceNotificationItem
	Created      bool
	AuditSummary string
	AuditPayload map[string]any
}

type governanceNotificationRecipient struct {
	UserId int
	Kind   string
}

func NewGovernanceNotificationService(db *gorm.DB) *GovernanceNotificationService {
	if db == nil {
		db = model.DB
	}
	return &GovernanceNotificationService{db: db, now: time.Now}
}

func (s *GovernanceNotificationService) EnqueueQuotaRequestSubmitted(item QuotaRequestItem, actorId int) {
	s.enqueueQuotaRequestBestEffort(item, actorId, GovernanceActionQuotaRequestSubmitted)
}

func (s *GovernanceNotificationService) EnqueueQuotaRequestDecision(result QuotaRequestDecisionResult, actorId int, action string) {
	actionType := GovernanceActionQuotaRequestApproved
	if action == QuotaRequestActionReject {
		actionType = GovernanceActionQuotaRequestRejected
	}
	s.enqueueQuotaRequestBestEffort(result.Request, actorId, actionType)
	if actionType == GovernanceActionQuotaRequestApproved && result.Allocation != nil {
		s.enqueueAllocationBestEffort(*result.Allocation, actorId, GovernanceActionAllocationCreated)
	}
}

func (s *GovernanceNotificationService) EnqueueAllocationGovernance(item QuotaAllocationItem, actorId int, actionType string) {
	s.enqueueAllocationBestEffort(item, actorId, actionType)
}

func (s *GovernanceNotificationService) enqueueQuotaRequestBestEffort(item QuotaRequestItem, actorId int, actionType string) {
	if s == nil || s.db == nil || item.Id <= 0 {
		return
	}
	if err := s.enqueueQuotaRequest(item, actorId, actionType); err != nil {
		common.SysLog(fmt.Sprintf("enterprise governance notification enqueue failed: source=quota_request id=%d action=%s error=%v", item.Id, actionType, err))
	}
}

func (s *GovernanceNotificationService) enqueueAllocationBestEffort(item QuotaAllocationItem, actorId int, actionType string) {
	if s == nil || s.db == nil || item.Id <= 0 {
		return
	}
	if err := s.enqueueAllocation(item, actorId, actionType); err != nil {
		common.SysLog(fmt.Sprintf("enterprise governance notification enqueue failed: source=quota_allocation id=%d action=%s error=%v", item.Id, actionType, err))
	}
}

func (s *GovernanceNotificationService) enqueueQuotaRequest(item QuotaRequestItem, actorId int, actionType string) error {
	recipients, err := s.resolveQuotaRequestRecipients(item, actionType)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		recipients = []governanceNotificationRecipient{{UserId: 0, Kind: entmodel.GovernanceNotificationRecipientAdmin}}
	}
	for _, recipient := range recipients {
		payload := entmodel.GovernanceNotificationTracePayload{
			TraceId:           fmt.Sprintf("quota_request:%d", item.Id),
			SourceType:        GovernanceSourceQuotaRequest,
			SourceId:          item.Id,
			ActionType:        actionType,
			TenantId:          item.TenantId,
			DepartmentId:      item.DepartmentId,
			DepartmentName:    item.DepartmentName,
			BudgetId:          item.DepartmentBudgetId,
			RequestId:         item.Id,
			AllocationId:      item.AllocationId,
			ActorId:           actorId,
			TargetUserId:      item.RequesterUserId,
			TargetUsername:    item.RequesterUsername,
			TargetDisplayName: item.RequesterDisplayName,
			QuotaDelta:        quotaRequestTimelineDelta(entmodel.QuotaRequest{RequestedQuota: item.RequestedQuota, ApprovedQuota: item.ApprovedQuota}),
			RequestedQuota:    item.RequestedQuota,
			ApprovedQuota:     item.ApprovedQuota,
			Status:            item.Status,
			Fallback:          item.Fallback,
			OccurredAt:        quotaRequestItemOccurredAt(item),
			DetailRoute:       buildGovernanceDetailRoute(GovernanceSourceQuotaRequest, item.Id, item.TenantId),
			DetailAPIPath:     buildGovernanceDetailAPIPath(GovernanceSourceQuotaRequest, item.Id, item.TenantId),
			Summary:           firstNonEmptyGovernance(item.ApprovalReason, item.RequestReason, item.Fallback),
			RecipientUserId:   recipient.UserId,
			RecipientKind:     recipient.Kind,
		}
		if err := s.createDeliveryIfMissing(payload, recipient); err != nil {
			return err
		}
	}
	return nil
}

func (s *GovernanceNotificationService) enqueueAllocation(item QuotaAllocationItem, actorId int, actionType string) error {
	recipients := []governanceNotificationRecipient{
		{UserId: item.TargetUserId, Kind: entmodel.GovernanceNotificationRecipientMember},
	}
	owners, err := s.resolveDepartmentOwnerRecipients(item.TenantId, item.DepartmentId)
	if err != nil {
		return err
	}
	recipients = append(recipients, owners...)
	for _, recipient := range dedupeGovernanceRecipients(recipients) {
		payload := entmodel.GovernanceNotificationTracePayload{
			TraceId:           fmt.Sprintf("quota_allocation:%d", item.Id),
			SourceType:        GovernanceSourceQuotaAllocation,
			SourceId:          item.Id,
			ActionType:        actionType,
			TenantId:          item.TenantId,
			DepartmentId:      item.DepartmentId,
			BudgetId:          item.DepartmentBudgetId,
			AllocationId:      item.Id,
			ActorId:           actorId,
			TargetUserId:      item.TargetUserId,
			TargetUsername:    item.TargetUsername,
			TargetDisplayName: item.TargetDisplayName,
			QuotaDelta:        item.CommittedQuota,
			CommittedQuota:    item.CommittedQuota,
			Status:            item.Status,
			OccurredAt:        nonZeroInt64(item.ProcessedAt, item.CreatedAt),
			DetailRoute:       buildGovernanceDetailRoute(GovernanceSourceQuotaAllocation, item.Id, item.TenantId),
			DetailAPIPath:     buildGovernanceDetailAPIPath(GovernanceSourceQuotaAllocation, item.Id, item.TenantId),
			Summary:           firstNonEmptyGovernance(item.RevokeReason, item.Reason, item.ProcessedSource),
			RecipientUserId:   recipient.UserId,
			RecipientKind:     recipient.Kind,
		}
		if actionType == GovernanceActionAllocationReclaim || actionType == GovernanceActionAllocationCancel || actionType == GovernanceActionAllocationRevoke {
			payload.QuotaDelta = -item.ReclaimedQuota
			if payload.QuotaDelta == 0 {
				payload.QuotaDelta = -item.CommittedQuota
			}
		}
		if err := s.createDeliveryIfMissing(payload, recipient); err != nil {
			return err
		}
	}
	return nil
}

func (s *GovernanceNotificationService) createDeliveryIfMissing(payload entmodel.GovernanceNotificationTracePayload, recipient governanceNotificationRecipient) error {
	nowUnix := s.now().Unix()
	dedupeKey := buildGovernanceNotificationDedupeKey(payload, recipient)
	delivery := entmodel.GovernanceNotificationDelivery{
		TenantId:        payload.TenantId,
		SourceType:      payload.SourceType,
		SourceId:        payload.SourceId,
		TraceId:         payload.TraceId,
		ActionType:      payload.ActionType,
		RecipientUserId: recipient.UserId,
		RecipientKind:   recipient.Kind,
		ChannelType:     entmodel.GovernanceNotificationChannelDingTalkRobot,
		Status:          entmodel.GovernanceNotificationStatusPending,
		AttemptCount:    0,
		MaxAttempts:     entmodel.GovernanceNotificationDefaultMaxAttempts,
		NextRetryAt:     nowUnix,
		DedupeKey:       dedupeKey,
		TriggerSource:   entmodel.GovernanceNotificationTriggerGovernanceAction,
		CreatedAt:       nowUnix,
		UpdatedAt:       nowUnix,
	}
	if err := delivery.SetTracePayload(&payload); err != nil {
		return err
	}
	var existing entmodel.GovernanceNotificationDelivery
	err := s.db.Where("dedupe_key = ?", dedupeKey).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return s.db.Create(&delivery).Error
}

func (s *GovernanceNotificationService) ListDeliveries(query GovernanceNotificationQuery) (GovernanceNotificationListResult, error) {
	if s == nil || s.db == nil {
		return GovernanceNotificationListResult{Items: []GovernanceNotificationItem{}}, nil
	}
	if query.TenantId < 0 || query.ViewerId <= 0 {
		return GovernanceNotificationListResult{}, ErrInvalidGovernanceNotificationQuery
	}
	page, pageSize := normalizeGovernancePage(query.Page, query.PageSize)
	db := s.db.Model(&entmodel.GovernanceNotificationDelivery{}).Where("tenant_id = ?", query.TenantId)
	if query.RecipientUserId != nil {
		db = db.Where("recipient_user_id = ?", *query.RecipientUserId)
	}
	if query.SourceType != "" {
		if !isKnownGovernanceNotificationSource(query.SourceType) {
			return GovernanceNotificationListResult{}, ErrInvalidGovernanceNotificationQuery
		}
		db = db.Where("source_type = ?", query.SourceType)
	}
	if query.SourceId != nil {
		db = db.Where("source_id = ?", *query.SourceId)
	}
	if query.ActionType != "" {
		db = db.Where("action_type = ?", query.ActionType)
	}
	if query.Status != "" {
		if !isKnownGovernanceNotificationStatus(query.Status) {
			return GovernanceNotificationListResult{}, ErrInvalidGovernanceNotificationQuery
		}
		db = db.Where("status = ?", query.Status)
	}
	allowedDepartments, err := s.allowedDepartmentsForViewer(query)
	if err != nil {
		return GovernanceNotificationListResult{}, err
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return GovernanceNotificationListResult{}, err
	}
	var deliveries []entmodel.GovernanceNotificationDelivery
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&deliveries).Error; err != nil {
		return GovernanceNotificationListResult{}, err
	}
	items := make([]GovernanceNotificationItem, 0, len(deliveries))
	for _, delivery := range deliveries {
		item, err := mapGovernanceNotificationItem(delivery)
		if err != nil {
			return GovernanceNotificationListResult{}, err
		}
		if !deliveryVisibleToViewer(item, query.ViewerId, allowedDepartments) {
			continue
		}
		if query.DepartmentId != nil && (item.Trace == nil || item.Trace.DepartmentId != *query.DepartmentId) {
			continue
		}
		items = append(items, item)
	}
	if items == nil {
		items = []GovernanceNotificationItem{}
	}
	return GovernanceNotificationListResult{Items: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *GovernanceNotificationService) ResendDelivery(tenantId int, deliveryId int, actorId int) (GovernanceNotificationResendResult, error) {
	if s == nil || s.db == nil || tenantId < 0 || deliveryId <= 0 || actorId <= 0 {
		return GovernanceNotificationResendResult{}, ErrInvalidGovernanceNotificationQuery
	}
	var parent entmodel.GovernanceNotificationDelivery
	if err := s.db.Where("tenant_id = ? AND id = ?", tenantId, deliveryId).First(&parent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GovernanceNotificationResendResult{}, ErrGovernanceNotificationDeliveryNotFound
		}
		return GovernanceNotificationResendResult{}, err
	}
	parentItem, err := mapGovernanceNotificationItem(parent)
	if err != nil {
		return GovernanceNotificationResendResult{}, err
	}
	allowed, err := s.canResendDelivery(parentItem, actorId)
	if err != nil {
		return GovernanceNotificationResendResult{}, err
	}
	if !allowed || parent.Status != entmodel.GovernanceNotificationStatusFinalFailed {
		return GovernanceNotificationResendResult{}, ErrGovernanceNotificationResendNotAllowed
	}
	var existing entmodel.GovernanceNotificationDelivery
	err = s.db.Where(
		"tenant_id = ? AND manual_parent_id = ? AND trigger_source = ? AND status IN ?",
		tenantId,
		parent.Id,
		entmodel.GovernanceNotificationTriggerManual,
		[]string{entmodel.GovernanceNotificationStatusPending, entmodel.GovernanceNotificationStatusFailed},
	).Order("created_at DESC, id DESC").First(&existing).Error
	if err == nil {
		item, mapErr := mapGovernanceNotificationItem(existing)
		if mapErr != nil {
			return GovernanceNotificationResendResult{}, mapErr
		}
		return GovernanceNotificationResendResult{
			Item:         item,
			Created:      false,
			AuditSummary: buildGovernanceNotificationResendSummary(parent, existing, false),
			AuditPayload: buildGovernanceNotificationResendPayload(parent, existing, actorId, false),
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return GovernanceNotificationResendResult{}, err
	}
	round, err := nextManualGovernanceNotificationRound(s.db, parent.Id)
	if err != nil {
		return GovernanceNotificationResendResult{}, err
	}
	nowUnix := s.now().Unix()
	child := parent
	child.Id = 0
	child.Status = entmodel.GovernanceNotificationStatusPending
	child.AttemptCount = 0
	child.NextRetryAt = nowUnix
	child.LastAttemptAt = 0
	child.SentAt = 0
	child.FinalFailedAt = 0
	child.ErrorReason = ""
	child.TriggerSource = entmodel.GovernanceNotificationTriggerManual
	child.ManualParentId = &parent.Id
	child.DedupeKey = buildManualGovernanceNotificationDedupeKey(parent, round)
	child.CreatedAt = nowUnix
	child.UpdatedAt = nowUnix
	if err := s.db.Create(&child).Error; err != nil {
		return GovernanceNotificationResendResult{}, err
	}
	item, err := mapGovernanceNotificationItem(child)
	if err != nil {
		return GovernanceNotificationResendResult{}, err
	}
	return GovernanceNotificationResendResult{
		Item:         item,
		Created:      true,
		AuditSummary: buildGovernanceNotificationResendSummary(parent, child, true),
		AuditPayload: buildGovernanceNotificationResendPayload(parent, child, actorId, true),
	}, nil
}

func (s *GovernanceNotificationService) resolveQuotaRequestRecipients(item QuotaRequestItem, actionType string) ([]governanceNotificationRecipient, error) {
	switch actionType {
	case GovernanceActionQuotaRequestSubmitted:
		owners, err := s.resolveDepartmentOwnerRecipients(item.TenantId, item.DepartmentId)
		if err != nil {
			return nil, err
		}
		if len(owners) == 0 {
			return []governanceNotificationRecipient{{UserId: 0, Kind: entmodel.GovernanceNotificationRecipientAdmin}}, nil
		}
		return owners, nil
	default:
		return []governanceNotificationRecipient{{UserId: item.RequesterUserId, Kind: entmodel.GovernanceNotificationRecipientRequester}}, nil
	}
}

func (s *GovernanceNotificationService) resolveDepartmentOwnerRecipients(tenantId int, departmentId int) ([]governanceNotificationRecipient, error) {
	resolution, err := NewPermissionService(s.db).ResolveEffectiveDepartmentOwners(tenantId, departmentId)
	if err != nil && !errors.Is(err, ErrDepartmentOwnerNotFound) {
		return nil, err
	}
	recipients := make([]governanceNotificationRecipient, 0, len(resolution.EffectiveOwners))
	for _, owner := range resolution.EffectiveOwners {
		recipients = append(recipients, governanceNotificationRecipient{UserId: owner.UserId, Kind: entmodel.GovernanceNotificationRecipientOwner})
	}
	return dedupeGovernanceRecipients(recipients), nil
}

func (s *GovernanceNotificationService) allowedDepartmentsForViewer(query GovernanceNotificationQuery) (map[int]struct{}, error) {
	if model.IsAdmin(query.ViewerId) {
		return nil, nil
	}
	ids, err := NewPermissionService(s.db).ListManageableDepartmentIds(query.ViewerId, query.TenantId)
	if err != nil {
		return nil, err
	}
	out := map[int]struct{}{}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out, nil
}

func (s *GovernanceNotificationService) canResendDelivery(item GovernanceNotificationItem, actorId int) (bool, error) {
	if model.IsAdmin(actorId) {
		return true, nil
	}
	if item.Trace == nil || item.Trace.DepartmentId <= 0 {
		return false, nil
	}
	return NewPermissionService(s.db).CanGovernDepartment(actorId, item.TenantId, item.Trace.DepartmentId)
}

func deliveryVisibleToViewer(item GovernanceNotificationItem, viewerId int, allowedDepartments map[int]struct{}) bool {
	if allowedDepartments == nil {
		return true
	}
	if item.RecipientUserId == viewerId {
		return true
	}
	if item.Trace != nil {
		_, ok := allowedDepartments[item.Trace.DepartmentId]
		return ok
	}
	return false
}

func mapGovernanceNotificationItem(delivery entmodel.GovernanceNotificationDelivery) (GovernanceNotificationItem, error) {
	trace, err := delivery.ParsedTracePayload()
	if err != nil {
		return GovernanceNotificationItem{}, err
	}
	return GovernanceNotificationItem{
		Id:              delivery.Id,
		TenantId:        delivery.TenantId,
		SourceType:      delivery.SourceType,
		SourceId:        delivery.SourceId,
		TraceId:         delivery.TraceId,
		ActionType:      delivery.ActionType,
		RecipientUserId: delivery.RecipientUserId,
		RecipientKind:   delivery.RecipientKind,
		ChannelType:     delivery.ChannelType,
		Status:          delivery.Status,
		AttemptCount:    delivery.AttemptCount,
		MaxAttempts:     delivery.MaxAttempts,
		NextRetryAt:     delivery.NextRetryAt,
		LastAttemptAt:   delivery.LastAttemptAt,
		SentAt:          delivery.SentAt,
		FinalFailedAt:   delivery.FinalFailedAt,
		ErrorReason:     delivery.ErrorReason,
		DedupeKey:       delivery.DedupeKey,
		TriggerSource:   delivery.TriggerSource,
		ManualParentId:  delivery.ManualParentId,
		TraceSummary:    buildGovernanceNotificationTraceSummary(trace),
		Trace:           trace,
		CreatedAt:       delivery.CreatedAt,
		UpdatedAt:       delivery.UpdatedAt,
	}, nil
}

func buildGovernanceNotificationDedupeKey(payload entmodel.GovernanceNotificationTracePayload, recipient governanceNotificationRecipient) string {
	return strings.Join([]string{
		strconv.Itoa(payload.TenantId),
		payload.SourceType,
		strconv.Itoa(payload.SourceId),
		payload.ActionType,
		strconv.Itoa(recipient.UserId),
		recipient.Kind,
	}, ":")
}

func buildManualGovernanceNotificationDedupeKey(parent entmodel.GovernanceNotificationDelivery, round int64) string {
	return strings.Join([]string{
		strconv.Itoa(parent.TenantId),
		parent.SourceType,
		strconv.Itoa(parent.SourceId),
		parent.ActionType,
		parent.ChannelType,
		"manual",
		strconv.Itoa(parent.Id),
		strconv.FormatInt(round, 10),
	}, ":")
}

func nextManualGovernanceNotificationRound(db *gorm.DB, parentId int) (int64, error) {
	var count int64
	if err := db.Model(&entmodel.GovernanceNotificationDelivery{}).
		Where("manual_parent_id = ? AND trigger_source = ?", parentId, entmodel.GovernanceNotificationTriggerManual).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count + 1, nil
}

func buildGovernanceNotificationTraceSummary(trace *entmodel.GovernanceNotificationTracePayload) string {
	if trace == nil {
		return ""
	}
	parts := []string{trace.TraceId, trace.ActionType, trace.Status}
	if trace.DepartmentName != "" {
		parts = append(parts, trace.DepartmentName)
	}
	return strings.Join(parts, " · ")
}

func buildGovernanceNotificationResendSummary(parent entmodel.GovernanceNotificationDelivery, child entmodel.GovernanceNotificationDelivery, created bool) string {
	if created {
		return fmt.Sprintf("Manual governance notification resend created delivery #%d from final_failed delivery #%d (%s)", child.Id, parent.Id, parent.TraceId)
	}
	return fmt.Sprintf("Manual governance notification resend reused delivery #%d for final_failed delivery #%d (%s)", child.Id, parent.Id, parent.TraceId)
}

func buildGovernanceNotificationResendPayload(parent entmodel.GovernanceNotificationDelivery, child entmodel.GovernanceNotificationDelivery, actorId int, created bool) map[string]any {
	return map[string]any{
		"actor_id":             actorId,
		"original_delivery_id": parent.Id,
		"new_delivery_id":      child.Id,
		"trace_id":             parent.TraceId,
		"source_type":          parent.SourceType,
		"source_id":            parent.SourceId,
		"action_type":          parent.ActionType,
		"channel_type":         parent.ChannelType,
		"trigger_source":       child.TriggerSource,
		"manual_parent_id":     child.ManualParentId,
		"created":              created,
	}
}

func quotaRequestItemOccurredAt(item QuotaRequestItem) int64 {
	return nonZeroInt64(item.ProcessedAt, item.FulfilledAt, item.ApprovedAt, item.RejectedAt, item.SubmittedAt, item.CreatedAt)
}

func dedupeGovernanceRecipients(recipients []governanceNotificationRecipient) []governanceNotificationRecipient {
	seen := map[string]struct{}{}
	out := make([]governanceNotificationRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		key := strconv.Itoa(recipient.UserId) + ":" + recipient.Kind
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, recipient)
	}
	return out
}

func isKnownGovernanceNotificationSource(sourceType string) bool {
	return sourceType == GovernanceSourceQuotaRequest || sourceType == GovernanceSourceQuotaAllocation
}

func isKnownGovernanceNotificationStatus(status string) bool {
	switch status {
	case entmodel.GovernanceNotificationStatusPending,
		entmodel.GovernanceNotificationStatusSent,
		entmodel.GovernanceNotificationStatusFailed,
		entmodel.GovernanceNotificationStatusFinalFailed,
		entmodel.GovernanceNotificationStatusResent,
		entmodel.GovernanceNotificationStatusUnconfigured:
		return true
	default:
		return false
	}
}
