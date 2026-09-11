package enterprise

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SubmitQuotaRequest(c *gin.Context) {
	var req dtoenterprise.SubmitQuotaRequestRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	var item entservice.QuotaRequestItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = entservice.NewQuotaRequestService(tx).Submit(entservice.SubmitQuotaRequestInput{
			TenantId:           tenantId,
			DepartmentId:       req.DepartmentId,
			DepartmentBudgetId: req.DepartmentBudgetId,
			BudgetMode:         req.BudgetMode,
			RequesterUserId:    c.GetInt("id"),
			RequestedQuota:     int64Value(req.RequestedQuota),
			RequestReason:      req.RequestReason,
			IdempotencyKey:     req.IdempotencyKey,
		})
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionQuotaRequestSubmit,
			ObjectType:  entservice.AdminObjectQuotaRequest,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: "Submitted employee quota request",
			Payload: map[string]any{
				"department_id":        item.DepartmentId,
				"department_budget_id": item.DepartmentBudgetId,
				"budget_mode":          item.BudgetMode,
				"requested_quota":      item.RequestedQuota,
				"requester_user_id":    item.RequesterUserId,
				"status":               item.Status,
			},
		})
	})
	if err != nil {
		writeQuotaRequestError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.QuotaRequestResponse{
		Item: mapQuotaRequestItemDTO(item),
	})
}

func ListQuotaRequests(c *gin.Context) {
	var query dtoenterprise.QuotaRequestListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, query.TenantId)
	if !ok {
		return
	}
	page := valueOrZero(query.Page)
	if page <= 0 {
		page = 1
	}
	if page > 1_000_000 {
		page = 1_000_000
	}
	pageSize := valueOrZero(query.PageSize)
	if pageSize <= 0 {
		pageSize = valueOrZero(query.Limit)
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 100
	}
	pageResult, err := entservice.NewQuotaRequestService(model.DB).ListPage(entservice.QuotaRequestListQuery{
		TenantId:        tenantId,
		DepartmentId:    valueOrZero(query.DepartmentId),
		RequesterUserId: valueOrZero(query.RequesterUserId),
		ActorId:         c.GetInt("id"),
		IncludePending:  query.IncludePending == nil || *query.IncludePending,
		Limit:           pageSize,
		View:            query.View,
		Status:          query.Status,
		Page:            page,
		PageSize:        pageSize,
	})
	if err != nil {
		writeQuotaRequestError(c, err)
		return
	}
	result := make([]dtoenterprise.QuotaRequestItem, 0, len(pageResult.Items))
	for _, item := range pageResult.Items {
		result = append(result, *mapQuotaRequestItemDTO(item))
	}
	common.ApiSuccess(c, dtoenterprise.QuotaRequestListResponse{Items: result, Total: pageResult.Total, Page: page, PageSize: pageSize, Scope: query.View})
}

func GetQuotaRequest(c *gin.Context) {
	requestId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	tenantId, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return
	}
	item, err := entservice.NewQuotaRequestService(model.DB).GetByID(tenantId, requestId, c.GetInt("id"))
	if err != nil {
		writeQuotaRequestError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.QuotaRequestResponse{
		Item: mapQuotaRequestItemDTO(item),
	})
}

func GetQuotaRequestCapability(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "department_id")
	if !ok {
		return
	}
	tenantId, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return
	}
	result, err := entservice.NewQuotaRequestService(model.DB).GetCapability(tenantId, departmentId, c.GetInt("id"))
	if err != nil {
		writeQuotaRequestError(c, err)
		return
	}
	budgets := make([]dtoenterprise.QuotaRequestCapabilityBudgetItem, 0, len(result.Budgets))
	for _, item := range result.Budgets {
		budgets = append(budgets, dtoenterprise.QuotaRequestCapabilityBudgetItem{
			Id:             item.Id,
			TenantId:       item.TenantId,
			DepartmentId:   item.DepartmentId,
			DepartmentName: item.DepartmentName,
			ScopeType:      item.ScopeType,
			Name:           item.Name,
			IsPublic:       item.IsPublic,
			Type:           item.Type,
			Status:         item.Status,
			TotalQuota:     item.TotalQuota,
			Remaining:      item.Remaining,
			AllocatedTotal: item.AllocatedTotal,
			CycleQuota:     item.CycleQuota,
			CycleType:      item.CycleType,
			CycleStartedAt: item.CycleStartedAt,
			CustomSeconds:  item.CustomSeconds,
			ExpiresAt:      item.ExpiresAt,
			ParentStatus:   item.ParentStatus,
			UsageRatio:     item.UsageRatio,
			ThresholdState: string(item.ThresholdState),
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		})
	}
	common.ApiSuccess(c, dtoenterprise.QuotaRequestCapabilityResponse{
		CanSubmit: result.CanSubmit,
		CanGovern: result.CanGovern,
		Budgets:   budgets,
	})
}

func DecideQuotaRequest(c *gin.Context) {
	requestId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	var req dtoenterprise.DecideQuotaRequestRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	var result entservice.QuotaRequestDecisionResult
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		service := entservice.NewQuotaRequestService(tx)
		switch req.Action {
		case entservice.QuotaRequestActionApprove:
			result, err = service.Approve(entservice.DecideQuotaRequestInput{
				TenantId:       tenantId,
				RequestId:      requestId,
				ActorId:        c.GetInt("id"),
				Action:         req.Action,
				ApprovedQuota:  req.ApprovedQuota,
				ApprovalReason: req.ApprovalReason,
			})
		case entservice.QuotaRequestActionReject:
			result, err = service.Reject(entservice.DecideQuotaRequestInput{
				TenantId:       tenantId,
				RequestId:      requestId,
				ActorId:        c.GetInt("id"),
				Action:         req.Action,
				RejectedReason: req.RejectedReason,
			})
		default:
			return entservice.ErrQuotaRequestInvalidInput
		}
		if err != nil {
			return err
		}
		actionType := entservice.AdminActionQuotaRequestApprove
		diffSummary := "Approved employee quota request"
		payload := map[string]any{
			"request_id":        result.Request.Id,
			"department_id":     result.Request.DepartmentId,
			"approved_quota":    result.Request.ApprovedQuota,
			"status":            result.Request.Status,
			"allocation_id":     result.Request.AllocationId,
			"requester_user_id": result.Request.RequesterUserId,
		}
		if req.Action == entservice.QuotaRequestActionReject {
			actionType = entservice.AdminActionQuotaRequestReject
			diffSummary = "Rejected employee quota request"
			payload["rejected_reason"] = result.Request.ApprovalReason
		} else {
			payload["approval_reason"] = result.Request.ApprovalReason
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    result.Request.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  actionType,
			ObjectType:  entservice.AdminObjectQuotaRequest,
			ObjectId:    strconv.Itoa(result.Request.Id),
			DiffSummary: diffSummary,
			Payload:     payload,
		})
	})
	if err != nil {
		writeQuotaRequestError(c, err)
		return
	}
	response := dtoenterprise.QuotaRequestResponse{
		Item: mapQuotaRequestItemDTO(result.Request),
	}
	if result.Allocation != nil {
		response.Allocation = mapQuotaAllocationItemDTO(*result.Allocation)
	}
	common.ApiSuccess(c, response)
}

func mapQuotaRequestItemDTO(item entservice.QuotaRequestItem) *dtoenterprise.QuotaRequestItem {
	return &dtoenterprise.QuotaRequestItem{
		Id:                   item.Id,
		TenantId:             item.TenantId,
		DepartmentId:         item.DepartmentId,
		DepartmentName:       item.DepartmentName,
		DepartmentBudgetId:   item.DepartmentBudgetId,
		BudgetScopeType:      item.BudgetScopeType,
		BudgetName:           item.BudgetName,
		BudgetMode:           item.BudgetMode,
		RequesterUserId:      item.RequesterUserId,
		RequesterUsername:    item.RequesterUsername,
		RequesterDisplayName: item.RequesterDisplayName,
		RequestedQuota:       item.RequestedQuota,
		ApprovedQuota:        item.ApprovedQuota,
		Status:               item.Status,
		ApproverUserId:       item.ApproverUserId,
		ApproverUsername:     item.ApproverUsername,
		ApprovalReason:       item.ApprovalReason,
		RequestReason:        item.RequestReason,
		AllocationId:         item.AllocationId,
		OwnerCountSnapshot:   item.OwnerCountSnapshot,
		Fallback:             item.Fallback,
		SubmittedAt:          item.SubmittedAt,
		ApprovedAt:           item.ApprovedAt,
		RejectedAt:           item.RejectedAt,
		FulfilledAt:          item.FulfilledAt,
		ProcessedAt:          item.ProcessedAt,
		ExpiresAt:            item.ExpiresAt,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func writeQuotaRequestError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrQuotaRequestInvalidInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrQuotaRequestNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestNotFound)
	case errors.Is(err, entservice.ErrQuotaRequestBudgetModeRequired):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestBudgetModeRequired)
	case errors.Is(err, entservice.ErrQuotaRequestBudgetPoolRequired):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestBudgetPoolRequired)
	case errors.Is(err, entservice.ErrQuotaRequestBudgetScopeMismatch):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestBudgetPoolRequired)
	case errors.Is(err, entservice.ErrQuotaRequestDepartmentMembershipRequired):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestDepartmentMembershipRequired)
	case errors.Is(err, entservice.ErrQuotaRequestApprovalNotAllowed):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestApprovalNotAllowed)
	case errors.Is(err, entservice.ErrQuotaRequestAlreadyProcessed):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestAlreadyProcessed)
	case errors.Is(err, entservice.ErrQuotaRequestApprovalReasonRequired):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestApprovalReasonRequired)
	case errors.Is(err, entservice.ErrQuotaRequestApprovedQuotaInvalid):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestApprovedQuotaInvalid)
	case errors.Is(err, entservice.ErrQuotaRequestRejectedReasonRequired):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaRequestRejectedReasonRequired)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetNotFound)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetInactive):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaAllocationBudgetInactive)
	case errors.Is(err, entservice.ErrQuotaAllocationQuotaInvalid):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseQuotaAllocationQuotaInvalid)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetInsufficient):
		writeQuotaAllocationBudgetError(c, quotaAllocationBudgetReasonKey(err))
	case errors.Is(err, entservice.ErrDepartmentOwnerDeniedByLocalRule):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentOwnerDeniedByLocalRule)
	case errors.Is(err, entservice.ErrDepartmentNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentNotFound)
	case errors.Is(err, entservice.ErrUserNotFound):
		common.ApiErrorI18n(c, i18n.MsgUserNotExists)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}
