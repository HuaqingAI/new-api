package enterprise

import (
	"errors"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const (
	QuotaRequestBudgetModeDepartment = "department_budget"
	QuotaRequestActionApprove        = "approve"
	QuotaRequestActionReject         = "reject"
)

type QuotaRequestService struct {
	db *gorm.DB
}

type SubmitQuotaRequestInput struct {
	TenantId           int
	DepartmentId       int
	DepartmentBudgetId int
	BudgetMode         string
	RequesterUserId    int
	RequestedQuota     int64
	RequestReason      string
	IdempotencyKey     string
}

type DecideQuotaRequestInput struct {
	TenantId       int
	RequestId      int
	ActorId        int
	Action         string
	ApprovedQuota  *int64
	ApprovalReason string
	RejectedReason string
}

type QuotaRequestListQuery struct {
	TenantId        int
	DepartmentId    int
	RequesterUserId int
	ActorId         int
	IncludePending  bool
	Limit           int
}

type QuotaRequestItem struct {
	Id                   int    `json:"id"`
	TenantId             int    `json:"tenant_id"`
	DepartmentId         int    `json:"department_id"`
	DepartmentName       string `json:"department_name"`
	DepartmentBudgetId   int    `json:"department_budget_id"`
	BudgetMode           string `json:"budget_mode"`
	RequesterUserId      int    `json:"requester_user_id"`
	RequesterUsername    string `json:"requester_username"`
	RequesterDisplayName string `json:"requester_display_name"`
	RequestedQuota       int64  `json:"requested_quota"`
	ApprovedQuota        int64  `json:"approved_quota"`
	Status               string `json:"status"`
	ApproverUserId       int    `json:"approver_user_id"`
	ApproverUsername     string `json:"approver_username"`
	ApprovalReason       string `json:"approval_reason"`
	RequestReason        string `json:"request_reason"`
	AllocationId         int    `json:"allocation_id"`
	OwnerCountSnapshot   int    `json:"owner_count_snapshot"`
	Fallback             string `json:"fallback"`
	SubmittedAt          int64  `json:"submitted_at"`
	ApprovedAt           int64  `json:"approved_at"`
	RejectedAt           int64  `json:"rejected_at"`
	FulfilledAt          int64  `json:"fulfilled_at"`
	ProcessedAt          int64  `json:"processed_at"`
	ExpiresAt            int64  `json:"expires_at"`
	CreatedAt            int64  `json:"created_at"`
	UpdatedAt            int64  `json:"updated_at"`
}

type QuotaRequestDecisionResult struct {
	Request    QuotaRequestItem     `json:"request"`
	Allocation *QuotaAllocationItem `json:"allocation,omitempty"`
}

type QuotaRequestCapability struct {
	CanSubmit bool                   `json:"can_submit"`
	CanGovern bool                   `json:"can_govern"`
	Budgets   []DepartmentBudgetItem `json:"budgets"`
}

func NewQuotaRequestService(db *gorm.DB) *QuotaRequestService {
	return &QuotaRequestService{db: db}
}

func (s *QuotaRequestService) Submit(input SubmitQuotaRequestInput) (QuotaRequestItem, error) {
	if input.DepartmentId <= 0 || input.DepartmentBudgetId <= 0 || input.RequesterUserId <= 0 || input.RequestedQuota <= 0 {
		return QuotaRequestItem{}, ErrQuotaRequestInvalidInput
	}
	mode := strings.TrimSpace(input.BudgetMode)
	if mode == "" {
		return QuotaRequestItem{}, ErrQuotaRequestBudgetModeRequired
	}
	if mode != QuotaRequestBudgetModeDepartment {
		return QuotaRequestItem{}, ErrQuotaRequestBudgetModeRequired
	}
	if err := s.ensureRequesterContext(input); err != nil {
		return QuotaRequestItem{}, err
	}

	var result QuotaRequestItem
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if input.IdempotencyKey != "" {
			var existing entmodel.QuotaRequest
			err := tx.Where("tenant_id = ? AND requester_user_id = ? AND idempotency_key = ?", input.TenantId, input.RequesterUserId, input.IdempotencyKey).First(&existing).Error
			if err == nil {
				item, mapErr := s.getByIDTx(tx, existing.Id)
				if mapErr != nil {
					return mapErr
				}
				result = item
				return nil
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		ownerCount, fallback, err := s.resolveApprovalRouteSnapshotTx(tx, input.TenantId, input.DepartmentId)
		if err != nil {
			return err
		}
		record := entmodel.QuotaRequest{
			TenantId:           input.TenantId,
			DepartmentId:       input.DepartmentId,
			DepartmentBudgetId: input.DepartmentBudgetId,
			BudgetMode:         mode,
			RequesterUserId:    input.RequesterUserId,
			RequestedQuota:     input.RequestedQuota,
			RequestReason:      strings.TrimSpace(input.RequestReason),
			Status:             entmodel.QuotaRequestStatusSubmitted,
			IdempotencyKey:     strings.TrimSpace(input.IdempotencyKey),
			OwnerCountSnapshot: ownerCount,
			Fallback:           fallback,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		item, err := s.getByIDTx(tx, record.Id)
		if err != nil {
			return err
		}
		result = item
		return nil
	})
	return result, err
}

func (s *QuotaRequestService) GetByID(tenantId int, requestId int, actorId int) (QuotaRequestItem, error) {
	if requestId <= 0 {
		return QuotaRequestItem{}, ErrQuotaRequestInvalidInput
	}
	item, err := s.getByIDTx(s.db, requestId)
	if err != nil {
		return QuotaRequestItem{}, err
	}
	if item.TenantId != tenantId {
		return QuotaRequestItem{}, ErrQuotaRequestNotFound
	}
	if err := s.ensureActorCanView(item, actorId); err != nil {
		return QuotaRequestItem{}, err
	}
	return item, nil
}

func (s *QuotaRequestService) List(query QuotaRequestListQuery) ([]QuotaRequestItem, error) {
	db := s.db.Model(&entmodel.QuotaRequest{})
	if query.TenantId > 0 {
		db = db.Where("enterprise_quota_requests.tenant_id = ?", query.TenantId)
	}
	if query.DepartmentId > 0 {
		db = db.Where("enterprise_quota_requests.department_id = ?", query.DepartmentId)
	}
	if query.RequesterUserId > 0 {
		db = db.Where("enterprise_quota_requests.requester_user_id = ?", query.RequesterUserId)
	}
	if query.IncludePending {
		db = db.Where("enterprise_quota_requests.status IN ?", []string{
			entmodel.QuotaRequestStatusSubmitted,
			entmodel.QuotaRequestStatusApproved,
			entmodel.QuotaRequestStatusRejected,
			entmodel.QuotaRequestStatusFulfilled,
			entmodel.QuotaRequestStatusCanceled,
			entmodel.QuotaRequestStatusExpired,
		})
	}
	if query.ActorId > 0 && !model.IsAdmin(query.ActorId) {
		manageable, err := NewPermissionService(s.db).ListManageableDepartmentIds(query.ActorId, query.TenantId)
		if err != nil {
			return nil, err
		}
		if len(manageable) == 0 {
			db = db.Where("enterprise_quota_requests.requester_user_id = ?", query.ActorId)
		} else {
			db = db.Where("(enterprise_quota_requests.requester_user_id = ? OR enterprise_quota_requests.department_id IN ?)", query.ActorId, manageable)
		}
	}
	limit := query.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.listRows(db.Order("enterprise_quota_requests.id DESC").Limit(limit))
	if err != nil {
		return nil, err
	}
	items := make([]QuotaRequestItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapQuotaRequestItem(row))
	}
	return items, nil
}

func (s *QuotaRequestService) Approve(input DecideQuotaRequestInput) (QuotaRequestDecisionResult, error) {
	input.Action = QuotaRequestActionApprove
	return s.decide(input)
}

func (s *QuotaRequestService) Reject(input DecideQuotaRequestInput) (QuotaRequestDecisionResult, error) {
	input.Action = QuotaRequestActionReject
	return s.decide(input)
}

func (s *QuotaRequestService) GetCapability(tenantId int, departmentId int, actorId int) (QuotaRequestCapability, error) {
	if departmentId <= 0 || actorId <= 0 {
		return QuotaRequestCapability{Budgets: []DepartmentBudgetItem{}}, ErrQuotaRequestInvalidInput
	}
	capability := QuotaRequestCapability{Budgets: []DepartmentBudgetItem{}}
	if err := ensureEnterpriseDepartmentExists(s.db, tenantId, departmentId); err != nil {
		return capability, err
	}
	var membership entmodel.UserDepartment
	if err := s.db.Where(
		"tenant_id = ? AND user_id = ? AND department_id = ? AND status = ?",
		tenantId,
		actorId,
		departmentId,
		constant.EnterpriseMembershipStatusActive,
	).First(&membership).Error; err == nil {
		capability.CanSubmit = true
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return capability, err
	}

	if model.IsAdmin(actorId) {
		capability.CanGovern = true
	} else {
		allowed, err := NewPermissionService(s.db).CanGovernDepartment(actorId, tenantId, departmentId)
		if err != nil && !errors.Is(err, ErrDepartmentOwnerNotFound) {
			return capability, err
		}
		capability.CanGovern = allowed
	}
	if !capability.CanSubmit && !capability.CanGovern {
		return capability, nil
	}

	budgets, err := NewDepartmentBudgetService(s.db).ListByDepartment(departmentId, tenantId, DepartmentBudgetListQuery{})
	if err != nil {
		return capability, err
	}
	for _, item := range budgets.Items {
		if item.DepartmentId != departmentId {
			continue
		}
		if item.Status != entmodel.DepartmentBudgetStatusActive {
			continue
		}
		capability.Budgets = append(capability.Budgets, item)
	}
	return capability, nil
}

func (s *QuotaRequestService) decide(input DecideQuotaRequestInput) (QuotaRequestDecisionResult, error) {
	if input.RequestId <= 0 || input.ActorId <= 0 {
		return QuotaRequestDecisionResult{}, ErrQuotaRequestInvalidInput
	}
	action := strings.TrimSpace(input.Action)
	if action != QuotaRequestActionApprove && action != QuotaRequestActionReject {
		return QuotaRequestDecisionResult{}, ErrQuotaRequestInvalidInput
	}

	var result QuotaRequestDecisionResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		record, err := s.lockRequest(tx, input.TenantId, input.RequestId)
		if err != nil {
			return err
		}
		if err := s.ensureActorCanApproveTx(tx, record, input.ActorId); err != nil {
			return err
		}
		switch record.Status {
		case entmodel.QuotaRequestStatusRejected, entmodel.QuotaRequestStatusCanceled, entmodel.QuotaRequestStatusExpired:
			return ErrQuotaRequestAlreadyProcessed
		case entmodel.QuotaRequestStatusApproved, entmodel.QuotaRequestStatusFulfilled:
			allocation, err := s.findAllocationTx(tx, record.AllocationId)
			if err != nil {
				return err
			}
			item, err := s.getByIDTx(tx, record.Id)
			if err != nil {
				return err
			}
			result.Request = item
			if allocation != nil {
				mapped := *allocation
				result.Allocation = &mapped
			}
			return nil
		}

		switch action {
		case QuotaRequestActionReject:
			reason := strings.TrimSpace(input.RejectedReason)
			if reason == "" {
				return ErrQuotaRequestRejectedReasonRequired
			}
			now := common.GetTimestamp()
			if err := tx.Model(&entmodel.QuotaRequest{}).
				Where("id = ?", record.Id).
				Updates(map[string]any{
					"status":           entmodel.QuotaRequestStatusRejected,
					"approver_user_id": input.ActorId,
					"approval_reason":  reason,
					"approved_quota":   int64(0),
					"rejected_at":      now,
					"processed_at":     now,
				}).Error; err != nil {
				return err
			}
			item, err := s.getByIDTx(tx, record.Id)
			if err != nil {
				return err
			}
			result.Request = item
			return nil
		default:
			approvedQuota := record.RequestedQuota
			if input.ApprovedQuota != nil {
				approvedQuota = *input.ApprovedQuota
			}
			if approvedQuota <= 0 || approvedQuota > record.RequestedQuota {
				return ErrQuotaRequestApprovedQuotaInvalid
			}
			reason := strings.TrimSpace(input.ApprovalReason)
			if reason == "" {
				return ErrQuotaRequestApprovalReasonRequired
			}
			allocationItem, err := NewQuotaAllocationService(tx).createTx(tx, CreateQuotaAllocationInput{
				TenantId:           record.TenantId,
				DepartmentBudgetId: record.DepartmentBudgetId,
				DepartmentId:       record.DepartmentId,
				TargetUserId:       record.RequesterUserId,
				ActorId:            input.ActorId,
				CommittedQuota:     approvedQuota,
				Reason:             reason,
			})
			if err != nil {
				return err
			}
			now := common.GetTimestamp()
			if err := tx.Model(&entmodel.QuotaRequest{}).
				Where("id = ?", record.Id).
				Updates(map[string]any{
					"status":           entmodel.QuotaRequestStatusFulfilled,
					"approver_user_id": input.ActorId,
					"approval_reason":  reason,
					"approved_quota":   approvedQuota,
					"allocation_id":    allocationItem.Id,
					"approved_at":      now,
					"fulfilled_at":     now,
					"processed_at":     now,
				}).Error; err != nil {
				return err
			}
			item, err := s.getByIDTx(tx, record.Id)
			if err != nil {
				return err
			}
			result.Request = item
			result.Allocation = &allocationItem
			return nil
		}
	})
	return result, err
}

func (s *QuotaRequestService) ensureRequesterContext(input SubmitQuotaRequestInput) error {
	if err := ensureEnterpriseDepartmentExists(s.db, input.TenantId, input.DepartmentId); err != nil {
		return err
	}
	if err := ensureEnterpriseUserExists(s.db, input.RequesterUserId); err != nil {
		return err
	}
	var membership entmodel.UserDepartment
	if err := s.db.Where(
		"tenant_id = ? AND user_id = ? AND department_id = ? AND status = ?",
		input.TenantId,
		input.RequesterUserId,
		input.DepartmentId,
		constant.EnterpriseMembershipStatusActive,
	).First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQuotaRequestDepartmentMembershipRequired
		}
		return err
	}
	var budget entmodel.DepartmentBudget
	if err := s.db.Where(
		"id = ? AND tenant_id = ? AND department_id = ?",
		input.DepartmentBudgetId,
		input.TenantId,
		input.DepartmentId,
	).First(&budget).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQuotaRequestBudgetPoolRequired
		}
		return err
	}
	if budget.Status != entmodel.DepartmentBudgetStatusActive {
		return ErrQuotaAllocationBudgetInactive
	}
	return nil
}

func (s *QuotaRequestService) resolveApprovalRouteSnapshotTx(tx *gorm.DB, tenantId int, departmentId int) (int, string, error) {
	resolution, err := NewPermissionService(tx).ResolveEffectiveDepartmentOwners(tenantId, departmentId)
	if err != nil && !errors.Is(err, ErrDepartmentOwnerNotFound) {
		return 0, "", err
	}
	fallback := resolution.Fallback
	if resolution.OwnerCount == 0 {
		fallback = "admin"
	}
	return resolution.OwnerCount, fallback, nil
}

func (s *QuotaRequestService) ensureActorCanApproveTx(tx *gorm.DB, record entmodel.QuotaRequest, actorId int) error {
	var actor model.User
	if err := tx.Select("id", "role").Where("id = ?", actorId).First(&actor).Error; err == nil && actor.Role >= common.RoleAdminUser {
		return nil
	}
	permissionService := NewPermissionService(tx)
	allowed, err := permissionService.CanGovernDepartment(actorId, record.TenantId, record.DepartmentId)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}
	resolution, err := permissionService.ResolveEffectiveDepartmentOwners(record.TenantId, record.DepartmentId)
	if err != nil {
		if errors.Is(err, ErrDepartmentOwnerNotFound) {
			return ErrQuotaRequestApprovalNotAllowed
		}
		return err
	}
	if resolution.OwnerCount == 0 {
		return ErrQuotaRequestApprovalNotAllowed
	}
	return ErrQuotaRequestApprovalNotAllowed
}

func (s *QuotaRequestService) ensureActorCanView(item QuotaRequestItem, actorId int) error {
	if actorId == item.RequesterUserId || model.IsAdmin(actorId) {
		return nil
	}
	allowed, err := NewPermissionService(s.db).CanGovernDepartment(actorId, item.TenantId, item.DepartmentId)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrQuotaRequestApprovalNotAllowed
	}
	return nil
}

func (s *QuotaRequestService) lockRequest(tx *gorm.DB, tenantId int, requestId int) (entmodel.QuotaRequest, error) {
	var record entmodel.QuotaRequest
	query := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", requestId)
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if err := query.First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entmodel.QuotaRequest{}, ErrQuotaRequestNotFound
		}
		return entmodel.QuotaRequest{}, err
	}
	return record, nil
}

func (s *QuotaRequestService) findAllocationTx(tx *gorm.DB, allocationId int) (*QuotaAllocationItem, error) {
	if allocationId <= 0 {
		return nil, nil
	}
	var allocation entmodel.QuotaAllocation
	if err := tx.Where("id = ?", allocationId).First(&allocation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	item := mapQuotaAllocationItem(allocation)
	return &item, nil
}

type quotaRequestListRow struct {
	entmodel.QuotaRequest
	DepartmentName       string
	RequesterUsername    string
	RequesterDisplayName string
	ApproverUsername     string
}

func (s *QuotaRequestService) getByIDTx(tx *gorm.DB, requestId int) (QuotaRequestItem, error) {
	rows, err := s.listRows(tx.Model(&entmodel.QuotaRequest{}).Where("enterprise_quota_requests.id = ?", requestId).Limit(1))
	if err != nil {
		return QuotaRequestItem{}, err
	}
	if len(rows) == 0 {
		return QuotaRequestItem{}, ErrQuotaRequestNotFound
	}
	return mapQuotaRequestItem(rows[0]), nil
}

func (s *QuotaRequestService) listRows(db *gorm.DB) ([]quotaRequestListRow, error) {
	var rows []quotaRequestListRow
	err := db.Select(
		"enterprise_quota_requests.*",
		"enterprise_departments.name AS department_name",
		"requester.username AS requester_username",
		"requester.display_name AS requester_display_name",
		"approver.username AS approver_username",
	).
		Joins("LEFT JOIN enterprise_departments ON enterprise_departments.id = enterprise_quota_requests.department_id").
		Joins("LEFT JOIN users AS requester ON requester.id = enterprise_quota_requests.requester_user_id").
		Joins("LEFT JOIN users AS approver ON approver.id = enterprise_quota_requests.approver_user_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func mapQuotaRequestItem(row quotaRequestListRow) QuotaRequestItem {
	return QuotaRequestItem{
		Id:                   row.Id,
		TenantId:             row.TenantId,
		DepartmentId:         row.DepartmentId,
		DepartmentName:       row.DepartmentName,
		DepartmentBudgetId:   row.DepartmentBudgetId,
		BudgetMode:           row.BudgetMode,
		RequesterUserId:      row.RequesterUserId,
		RequesterUsername:    row.RequesterUsername,
		RequesterDisplayName: row.RequesterDisplayName,
		RequestedQuota:       row.RequestedQuota,
		ApprovedQuota:        row.ApprovedQuota,
		Status:               row.Status,
		ApproverUserId:       row.ApproverUserId,
		ApproverUsername:     row.ApproverUsername,
		ApprovalReason:       row.ApprovalReason,
		RequestReason:        row.RequestReason,
		AllocationId:         row.AllocationId,
		OwnerCountSnapshot:   row.OwnerCountSnapshot,
		Fallback:             row.Fallback,
		SubmittedAt:          row.SubmittedAt,
		ApprovedAt:           row.ApprovedAt,
		RejectedAt:           row.RejectedAt,
		FulfilledAt:          row.FulfilledAt,
		ProcessedAt:          row.ProcessedAt,
		ExpiresAt:            row.ExpiresAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func SortQuotaRequestsByIDAsc(items []QuotaRequestItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Id < items[j].Id
	})
}
