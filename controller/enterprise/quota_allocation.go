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
)

func CreateQuotaAllocation(c *gin.Context) {
	var req dtoenterprise.CreateQuotaAllocationRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}
	item, err := entservice.NewQuotaAllocationService(model.DB).Create(entservice.CreateQuotaAllocationInput{
		TenantId:           tenantId,
		DepartmentBudgetId: req.DepartmentBudgetId,
		DepartmentId:       req.DepartmentId,
		TargetUserId:       req.TargetUserId,
		ActorId:            c.GetInt("id"),
		CommittedQuota:     int64Value(req.CommittedQuota),
		Reason:             req.Reason,
	})
	if err != nil {
		writeQuotaAllocationError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.QuotaAllocationResponse{
		Item: mapQuotaAllocationItemDTO(item),
	})
}

func ListQuotaAllocations(c *gin.Context) {
	budgetId, err := strconv.Atoi(c.Query("department_budget_id"))
	if err != nil || budgetId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return
	}
	departmentId, err := entservice.NewQuotaAllocationService(model.DB).GetBudgetDepartment(tenantId, budgetId)
	if err != nil {
		writeQuotaAllocationError(c, err)
		return
	}
	if rawDepartmentID := c.Query("department_id"); rawDepartmentID != "" {
		requestDepartmentId, convErr := strconv.Atoi(rawDepartmentID)
		if convErr != nil || requestDepartmentId <= 0 || requestDepartmentId != departmentId {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
	}
	items, err := entservice.NewQuotaAllocationService(model.DB).ListByBudget(tenantId, budgetId)
	if err != nil {
		writeQuotaAllocationError(c, err)
		return
	}
	result := make([]dtoenterprise.QuotaAllocationItem, 0, len(items))
	for _, item := range items {
		result = append(result, *mapQuotaAllocationItemDTO(item))
	}
	common.ApiSuccess(c, dtoenterprise.QuotaAllocationListResponse{
		Items: result,
	})
}

func mapQuotaAllocationItemDTO(item entservice.QuotaAllocationItem) *dtoenterprise.QuotaAllocationItem {
	return &dtoenterprise.QuotaAllocationItem{
		Id:                     item.Id,
		TenantId:               item.TenantId,
		DepartmentBudgetId:     item.DepartmentBudgetId,
		DepartmentId:           item.DepartmentId,
		TargetUserId:           item.TargetUserId,
		WalletId:               item.WalletId,
		ActorId:                item.ActorId,
		CommittedQuota:         item.CommittedQuota,
		BudgetTypeSnapshot:     item.BudgetTypeSnapshot,
		CycleTypeSnapshot:      item.CycleTypeSnapshot,
		CycleStartedAtSnapshot: item.CycleStartedAtSnapshot,
		CustomSecondsSnapshot:  item.CustomSecondsSnapshot,
		ExpiresAtSnapshot:      item.ExpiresAtSnapshot,
		Reason:                 item.Reason,
		Status:                 item.Status,
		CreatedAt:              item.CreatedAt,
		UpdatedAt:              item.UpdatedAt,
	}
}

func writeQuotaAllocationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrQuotaAllocationInvalidInput):
		common.ApiErrorMsg(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrUserNotFound):
		common.ApiErrorMsg(c, i18n.MsgUserNotExists)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetNotFound):
		common.ApiErrorMsg(c, i18n.MsgEnterpriseDepartmentNotFound)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetInactive):
		common.ApiErrorMsg(c, i18n.MsgEnterpriseQuotaAllocationBudgetInactive)
	case errors.Is(err, entservice.ErrQuotaAllocationQuotaInvalid):
		common.ApiErrorMsg(c, i18n.MsgEnterpriseQuotaAllocationQuotaInvalid)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetInsufficient):
		writeQuotaAllocationBudgetError(c, quotaAllocationBudgetReasonKey(err))
	case errors.Is(err, entservice.ErrQuotaAllocationUserOutOfDepartment):
		common.ApiErrorMsg(c, i18n.MsgEnterpriseQuotaAllocationUserOutOfDepartment)
	default:
		common.ApiErrorMsg(c, i18n.MsgDatabaseError)
	}
}

func writeQuotaAllocationBudgetError(c *gin.Context, reasonKey string) {
	data := gin.H{}
	if reasonKey != "" {
		data["reason"] = reasonKey
	}
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": i18n.MsgEnterpriseBudgetInsufficient,
		"data":    data,
	})
}

func quotaAllocationBudgetReasonKey(err error) string {
	switch {
	case errors.Is(err, entservice.ErrQuotaAllocationBalanceRemainingInsufficient):
		return i18n.MsgEnterpriseBalanceRemainingInsufficient
	case errors.Is(err, entservice.ErrQuotaAllocationSubscriptionCycleAllocatedExceeded):
		return i18n.MsgEnterpriseSubscriptionCycleAllocatedExceeded
	default:
		return ""
	}
}
