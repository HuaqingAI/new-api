package enterprise

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GrantDepartmentAdmin(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	var req dtoenterprise.DepartmentAdminRoleRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	var item dtoenterprise.DepartmentRoleItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		role, err := entservice.NewPermissionService(tx).GrantDepartmentAdmin(entservice.DepartmentAdminRoleInput{
			TenantId:     valueOrZero(req.TenantId),
			UserId:       req.UserId,
			DepartmentId: departmentId,
		})
		if err != nil {
			return err
		}
		item = mapDepartmentRole(role)
		return entservice.NewAdminActionService(tx).Write(entservice.AdminActionInput{
			TenantId:    role.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionDeptAdminGrant,
			ObjectType:  entservice.AdminObjectDepartmentRole,
			ObjectId:    strconv.Itoa(role.Id),
			DiffSummary: "Granted department administrator role",
			Payload: map[string]any{
				"user_id":       role.UserId,
				"department_id": role.DepartmentId,
				"role":          role.Role,
				"status":        role.Status,
			},
		})
	})
	if err != nil {
		writeDepartmentRoleError(c, err)
		return
	}

	common.ApiSuccess(c, item)
}

func RevokeDepartmentAdmin(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	userId, ok := parsePathInt(c, "user_id")
	if !ok {
		return
	}

	tenantId, ok := parseDepartmentRoleTenantId(c)
	if !ok {
		return
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := entservice.NewPermissionService(tx).RevokeDepartmentAdmin(entservice.DepartmentAdminRoleInput{
			TenantId:     tenantId,
			UserId:       userId,
			DepartmentId: departmentId,
		}); err != nil {
			return err
		}
		return entservice.NewAdminActionService(tx).Write(entservice.AdminActionInput{
			TenantId:    tenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionDeptAdminRevoke,
			ObjectType:  entservice.AdminObjectDepartmentRole,
			ObjectId:    entservice.MembershipObjectId(departmentId, userId),
			DiffSummary: "Revoked department administrator role",
			Payload: map[string]any{
				"user_id":       userId,
				"department_id": departmentId,
			},
		})
	})
	if err != nil {
		writeDepartmentRoleError(c, err)
		return
	}

	common.ApiSuccess(c, nil)
}

func parseDepartmentRoleTenantId(c *gin.Context) (int, bool) {
	if c.Request.Method == http.MethodGet || c.Query("tenant_id") != "" {
		raw := c.Query("tenant_id")
		if raw == "" {
			return 0, true
		}
		tenantId, err := strconv.Atoi(raw)
		if err != nil || tenantId < 0 {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return 0, false
		}
		return tenantId, true
	}
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return 0, true
	}
	var req struct {
		TenantId *int `json:"tenant_id,omitempty"`
	}
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return 0, false
	}
	if c.Query("tenant_id") != "" {
		return 0, true
	}
	if req.TenantId == nil {
		return 0, true
	}
	if *req.TenantId < 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return 0, false
	}
	return *req.TenantId, true
}

func writeDepartmentRoleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrDepartmentNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentNotFound)
	case errors.Is(err, entservice.ErrUserNotFound):
		common.ApiErrorI18n(c, i18n.MsgUserNotExists)
	case errors.Is(err, entservice.ErrDepartmentRoleNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentRoleNotFound)
	case errors.Is(err, entservice.ErrInvalidAdminActionInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func mapDepartmentRole(role entmodel.DepartmentRole) dtoenterprise.DepartmentRoleItem {
	return dtoenterprise.DepartmentRoleItem{
		Id:           role.Id,
		TenantId:     role.TenantId,
		UserId:       role.UserId,
		DepartmentId: role.DepartmentId,
		Role:         role.Role,
		Status:       role.Status,
		CreatedAt:    role.CreatedAt,
		UpdatedAt:    role.UpdatedAt,
	}
}
