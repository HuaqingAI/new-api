package enterprise

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateBudgetDelegation(c *gin.Context) {
	var req dtoenterprise.CreateBudgetDelegationRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	var item entservice.BudgetDelegationItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = entservice.NewBudgetDelegationService(tx).Create(entservice.CreateBudgetDelegationInput{
			TenantId:           tenantId,
			SourceDepartmentId: req.SourceDepartmentId,
			SourceBudgetId:     req.SourceBudgetId,
			TargetDepartmentId: req.TargetDepartmentId,
			TargetBudgetId:     req.TargetBudgetId,
			ActorId:            c.GetInt("id"),
			CommittedQuota:     int64Value(req.CommittedQuota),
			Reason:             req.Reason,
		})
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionBudgetDelegationCreate,
			ObjectType:  entservice.AdminObjectBudgetDelegation,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: "Created department budget delegation",
			Payload: map[string]any{
				"source_department_id": item.SourceDepartmentId,
				"source_budget_id":     item.SourceBudgetId,
				"target_department_id": item.TargetDepartmentId,
				"target_budget_id":     item.TargetBudgetId,
				"committed_quota":      item.CommittedQuota,
				"budget_type_snapshot": item.BudgetTypeSnapshot,
				"delegation_status":    item.Status,
				"actor_id":             item.ActorId,
				"reason":               item.Reason,
			},
		})
	})
	if err != nil {
		if shouldAuditBudgetDelegationFailure(err) {
			_ = writeAdminAction(model.DB, c, entservice.AdminActionInput{
				TenantId:    tenantId,
				ActorId:     c.GetInt("id"),
				ActionType:  entservice.AdminActionBudgetDelegationReject,
				ObjectType:  entservice.AdminObjectBudgetDelegation,
				ObjectId:    strconv.Itoa(req.SourceBudgetId),
				DiffSummary: "Rejected department budget delegation",
				Payload: map[string]any{
					"source_department_id": req.SourceDepartmentId,
					"source_budget_id":     req.SourceBudgetId,
					"target_department_id": req.TargetDepartmentId,
					"target_budget_id":     req.TargetBudgetId,
					"committed_quota":      int64Value(req.CommittedQuota),
					"error":                err.Error(),
				},
			})
		}
		writeBudgetDelegationError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.BudgetDelegationResponse{
		Item: mapBudgetDelegationItemDTO(item),
	})
}

func SupersedeBudgetDelegation(c *gin.Context) {
	delegationId, err := strconv.Atoi(c.Param("id"))
	if err != nil || delegationId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	var req dtoenterprise.SupersedeBudgetDelegationRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	var item entservice.BudgetDelegationItem
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = entservice.NewBudgetDelegationService(tx).Supersede(entservice.SupersedeBudgetDelegationInput{
			TenantId:           tenantId,
			DelegationId:       delegationId,
			SourceDepartmentId: req.SourceDepartmentId,
			ActorId:            c.GetInt("id"),
			NewCommittedQuota:  int64Value(req.NewCommittedQuota),
			Reason:             req.Reason,
		})
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionBudgetDelegationSupersede,
			ObjectType:  entservice.AdminObjectBudgetDelegation,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: "Superseded department budget delegation",
			Payload: map[string]any{
				"source_department_id": item.SourceDepartmentId,
				"source_budget_id":     item.SourceBudgetId,
				"target_department_id": item.TargetDepartmentId,
				"target_budget_id":     item.TargetBudgetId,
				"committed_quota":      item.CommittedQuota,
				"budget_type_snapshot": item.BudgetTypeSnapshot,
				"delegation_status":    item.Status,
				"superseded_by_id":     item.SupersededById,
				"actor_id":             item.ActorId,
				"reason":               item.Reason,
			},
		})
	})
	if err != nil {
		writeBudgetDelegationError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.BudgetDelegationResponse{
		Item: mapBudgetDelegationItemDTO(item),
	})
}

func ListBudgetDelegations(c *gin.Context) {
	var req dtoenterprise.BudgetDelegationListQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}
	items, err := entservice.NewBudgetDelegationService(model.DB).List(entservice.BudgetDelegationListQuery{
		TenantId:     tenantId,
		DepartmentId: req.DepartmentId,
	})
	if err != nil {
		writeBudgetDelegationError(c, err)
		return
	}
	result := make([]dtoenterprise.BudgetDelegationItem, 0, len(items))
	for _, item := range items {
		result = append(result, *mapBudgetDelegationItemDTO(item))
	}
	common.ApiSuccess(c, dtoenterprise.BudgetDelegationListResponse{
		Items: result,
	})
}

func mapBudgetDelegationItemDTO(item entservice.BudgetDelegationItem) *dtoenterprise.BudgetDelegationItem {
	return &dtoenterprise.BudgetDelegationItem{
		Id:                         item.Id,
		TenantId:                   item.TenantId,
		SourceDepartmentId:         item.SourceDepartmentId,
		SourceDepartmentName:       item.SourceDepartmentName,
		SourceBudgetId:             item.SourceBudgetId,
		TargetDepartmentId:         item.TargetDepartmentId,
		TargetDepartmentName:       item.TargetDepartmentName,
		TargetBudgetId:             item.TargetBudgetId,
		ActorId:                    item.ActorId,
		CommittedQuota:             item.CommittedQuota,
		BudgetTypeSnapshot:         item.BudgetTypeSnapshot,
		CycleTypeSnapshot:          item.CycleTypeSnapshot,
		BeforeSourceBudgetSnapshot: item.BeforeSourceBudgetSnapshot,
		AfterSourceBudgetSnapshot:  item.AfterSourceBudgetSnapshot,
		BeforeTargetBudgetSnapshot: item.BeforeTargetBudgetSnapshot,
		AfterTargetBudgetSnapshot:  item.AfterTargetBudgetSnapshot,
		Status:                     item.Status,
		SupersededById:             item.SupersededById,
		ProcessedAt:                item.ProcessedAt,
		Reason:                     item.Reason,
		CreatedAt:                  item.CreatedAt,
		UpdatedAt:                  item.UpdatedAt,
	}
}

func writeBudgetDelegationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrBudgetDelegationInvalidInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrBudgetDelegationNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseBudgetDelegationNotFound)
	case errors.Is(err, entservice.ErrBudgetDelegationQuotaInvalid):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseBudgetDelegationQuotaInvalid)
	case errors.Is(err, entservice.ErrBudgetDelegationBudgetInactive):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseBudgetDelegationBudgetInactive)
	case errors.Is(err, entservice.ErrBudgetDelegationTargetNotDescendant):
		writeBudgetDelegationReasonError(c, i18n.MsgEnterpriseBudgetDelegationTargetNotDescendant)
	case errors.Is(err, entservice.ErrBudgetDelegationPermissionDenied):
		writeBudgetDelegationReasonError(c, i18n.MsgEnterpriseBudgetDelegationPermissionDenied)
	case errors.Is(err, entservice.ErrBudgetDelegationBudgetTypeMismatch):
		writeBudgetDelegationReasonError(c, i18n.MsgEnterpriseBudgetDelegationBudgetTypeMismatch)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetNotFound)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetInsufficient):
		writeBudgetDelegationReasonError(c, budgetDelegationReasonKey(err))
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func writeBudgetDelegationReasonError(c *gin.Context, reasonKey string) {
	data := gin.H{}
	if reasonKey != "" {
		data["reason"] = reasonKey
	}
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": i18n.MsgEnterpriseBudgetDelegationRejected,
		"data":    data,
	})
}

func budgetDelegationReasonKey(err error) string {
	switch {
	case errors.Is(err, entservice.ErrQuotaAllocationBalanceRemainingInsufficient):
		return i18n.MsgEnterpriseBalanceRemainingInsufficient
	case errors.Is(err, entservice.ErrQuotaAllocationSubscriptionCycleAllocatedExceeded):
		return i18n.MsgEnterpriseSubscriptionCycleAllocatedExceeded
	default:
		return ""
	}
}

func shouldAuditBudgetDelegationFailure(err error) bool {
	return errors.Is(err, entservice.ErrBudgetDelegationQuotaInvalid) ||
		errors.Is(err, entservice.ErrBudgetDelegationBudgetInactive) ||
		errors.Is(err, entservice.ErrBudgetDelegationTargetNotDescendant) ||
		errors.Is(err, entservice.ErrBudgetDelegationPermissionDenied) ||
		errors.Is(err, entservice.ErrBudgetDelegationBudgetTypeMismatch) ||
		errors.Is(err, entservice.ErrQuotaAllocationBudgetInsufficient)
}
