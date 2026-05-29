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

func CreateDepartmentBudget(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	var req dtoenterprise.CreateDepartmentBudgetRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	input := entservice.CreateDepartmentBudgetInput{
		TenantId:       tenantId,
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
		item, err = entservice.NewDepartmentBudgetService(tx).Create(departmentId, input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionDepartmentBudgetCreate,
			ObjectType:  entservice.AdminObjectDepartmentBudget,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: "Created department budget pool",
			Payload: map[string]any{
				"department_id":    item.DepartmentId,
				"type":             item.Type,
				"total_quota":      item.TotalQuota,
				"cycle_quota":      item.CycleQuota,
				"cycle_type":       item.CycleType,
				"cycle_started_at": item.CycleStartedAt,
				"custom_seconds":   item.CustomSeconds,
				"expires_at":       item.ExpiresAt,
			},
		})
	})
	if err != nil {
		if shouldAuditDepartmentBudgetFailure(err) {
			_ = writeAdminAction(model.DB, c, entservice.AdminActionInput{
				TenantId:    input.TenantId,
				ActorId:     c.GetInt("id"),
				ActionType:  entservice.AdminActionDepartmentBudgetReject,
				ObjectType:  entservice.AdminObjectDepartmentBudget,
				ObjectId:    strconv.Itoa(departmentId),
				DiffSummary: "Rejected department budget creation",
				Payload: map[string]any{
					"department_id":    departmentId,
					"type":             input.Type,
					"total_quota":      int64PtrValueOrZero(input.TotalQuota),
					"cycle_quota":      int64PtrValueOrZero(input.CycleQuota),
					"cycle_type":       input.CycleType,
					"cycle_started_at": int64PtrValueOrZero(input.CycleStartedAt),
					"custom_seconds":   int64PtrValueOrZero(input.CustomSeconds),
					"expires_at":       int64PtrValueOrZero(input.ExpiresAt),
					"error":            err.Error(),
				},
			})
		}
		writeDepartmentBudgetError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{
		Item: mapDepartmentBudgetItemDTO(item),
	})
}

func GetDepartmentBudget(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	tenantId, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return
	}

	item, err := entservice.NewDepartmentBudgetService(model.DB).GetByDepartment(departmentId, tenantId)
	if err != nil {
		writeDepartmentBudgetError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{
		Item: mapDepartmentBudgetItemDTOOrNil(item),
	})
}

func writeDepartmentBudgetError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrDepartmentNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentNotFound)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidType):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidType)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidQuota):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidQuota)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleQuota):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidCycleQuota)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleType):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidCycleType)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidCustomSeconds):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidCustomSeconds)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleStartedAt):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidCycleStartedAt)
	case errors.Is(err, entservice.ErrDepartmentBudgetTypeImmutable):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetTypeImmutable)
	case errors.Is(err, entservice.ErrInvalidDepartmentBudgetInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func shouldAuditDepartmentBudgetFailure(err error) bool {
	return errors.Is(err, entservice.ErrDepartmentBudgetInvalidType) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidQuota) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleQuota) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleType) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCustomSeconds) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleStartedAt) ||
		errors.Is(err, entservice.ErrDepartmentBudgetTypeImmutable)
}

func mapDepartmentBudgetItemDTO(item entservice.DepartmentBudgetItem) *dtoenterprise.DepartmentBudgetItem {
	return &dtoenterprise.DepartmentBudgetItem{
		Id:             item.Id,
		TenantId:       item.TenantId,
		DepartmentId:   item.DepartmentId,
		Type:           item.Type,
		Status:         item.Status,
		TotalQuota:     item.TotalQuota,
		Remaining:      item.Remaining,
		CycleQuota:     item.CycleQuota,
		CycleType:      item.CycleType,
		CycleStartedAt: item.CycleStartedAt,
		CustomSeconds:  item.CustomSeconds,
		ExpiresAt:      item.ExpiresAt,
		ParentStatus:   item.ParentStatus,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func mapDepartmentBudgetItemDTOOrNil(item *entservice.DepartmentBudgetItem) *dtoenterprise.DepartmentBudgetItem {
	if item == nil {
		return nil
	}
	return mapDepartmentBudgetItemDTO(*item)
}

func int64PtrValueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
