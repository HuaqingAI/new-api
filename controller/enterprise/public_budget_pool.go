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

func ListPublicBudgetPools(c *gin.Context) {
	tenantID, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return
	}
	includeInactive := c.Query("include_inactive") == "true"
	items, err := entservice.NewDepartmentBudgetService(model.DB).ListPublic(tenantID, includeInactive)
	if err != nil {
		writePublicBudgetError(c, err)
		return
	}
	result := make([]dtoenterprise.DepartmentBudgetItem, 0, len(items))
	for _, item := range items {
		result = append(result, *mapDepartmentBudgetItemDTO(item))
	}
	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetListResponse{Items: result})
}

func GetPublicBudgetPool(c *gin.Context) {
	budgetID, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	tenantID, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return
	}
	item, err := entservice.NewDepartmentBudgetService(model.DB).GetPublic(tenantID, budgetID)
	if err != nil {
		writePublicBudgetError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{Item: mapDepartmentBudgetItemDTO(*item)})
}

func CreatePublicBudgetPool(c *gin.Context) {
	var req dtoenterprise.CreatePublicBudgetRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantID, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}
	input := entservice.CreateDepartmentBudgetInput{
		TenantId:       tenantID,
		Type:           req.Type,
		TotalQuota:     req.TotalQuota,
		CycleQuota:     req.CycleQuota,
		CycleType:      req.CycleType,
		CycleStartedAt: req.CycleStartedAt,
		CustomSeconds:  req.CustomSeconds,
		ExpiresAt:      req.ExpiresAt,
	}
	var item entservice.DepartmentBudgetItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = entservice.NewDepartmentBudgetService(tx).CreatePublic(input, req.Name)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionPublicBudgetCreate,
			ObjectType:  entservice.AdminObjectDepartmentBudget,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: "Created public budget pool",
			Payload: map[string]any{
				"budget_id":   item.Id,
				"scope_type":  item.ScopeType,
				"name":        item.Name,
				"type":        item.Type,
				"total_quota": item.TotalQuota,
				"cycle_quota": item.CycleQuota,
				"cycle_type":  item.CycleType,
				"expires_at":  item.ExpiresAt,
			},
		})
	})
	if err != nil {
		writePublicBudgetError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{Item: mapDepartmentBudgetItemDTO(item)})
}

func PausePublicBudgetPool(c *gin.Context) {
	mutatePublicBudgetStatus(c, entservice.AdminActionPublicBudgetPause, entmodelStatusActive(), entmodelStatusPaused(), "Paused public budget pool")
}

func ResumePublicBudgetPool(c *gin.Context) {
	mutatePublicBudgetStatus(c, entservice.AdminActionPublicBudgetResume, entmodelStatusPaused(), entmodelStatusActive(), "Resumed public budget pool")
}

func mutatePublicBudgetStatus(c *gin.Context, actionType string, fromStatus string, toStatus string, summary string) {
	budgetID, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	var req dtoenterprise.DepartmentBudgetLifecycleRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantID, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}
	var item entservice.DepartmentBudgetItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = entservice.NewDepartmentBudgetService(tx).UpdatePublicStatus(tenantID, budgetID, fromStatus, toStatus)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  actionType,
			ObjectType:  entservice.AdminObjectDepartmentBudget,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: summary,
			Payload: map[string]any{
				"budget_id":     item.Id,
				"scope_type":    item.ScopeType,
				"before_status": fromStatus,
				"after_status":  toStatus,
			},
		})
	})
	if err != nil {
		writePublicBudgetError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{Item: mapDepartmentBudgetItemDTO(item)})
}

func ResizePublicBudgetPool(c *gin.Context) {
	budgetID, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	var req dtoenterprise.ResizeDepartmentBudgetRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantID, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}
	var item entservice.DepartmentBudgetItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		svc := entservice.NewDepartmentBudgetService(tx)
		before, err := svc.GetPublic(tenantID, budgetID)
		if err != nil {
			return err
		}
		item, err = svc.ResizePublic(tenantID, budgetID, entservice.ResizeDepartmentBudgetInput{
			TotalQuota: req.TotalQuota,
			CycleQuota: req.CycleQuota,
		})
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionPublicBudgetResize,
			ObjectType:  entservice.AdminObjectDepartmentBudget,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: "Resized public budget pool",
			Payload:     buildDepartmentBudgetResizePayload(0, budgetID, before, item, nil),
		})
	})
	if err != nil {
		writePublicBudgetError(c, err)
		return
	}
	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{Item: mapDepartmentBudgetItemDTO(item)})
}

func writePublicBudgetError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrPublicBudgetInvalidInput),
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidType),
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidQuota),
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleQuota),
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleType),
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCustomSeconds),
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleStartedAt):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	case errors.Is(err, entservice.ErrPublicBudgetNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetNotFound)
	case errors.Is(err, entservice.ErrDepartmentBudgetStatusTransitionInvalid):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetStatusTransition)
	case errors.Is(err, entservice.ErrDepartmentBudgetTypeImmutable),
		errors.Is(err, entservice.ErrDepartmentBudgetResizeBelowCommitted):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}
