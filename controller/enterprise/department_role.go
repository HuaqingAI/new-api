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
			ActionType:  entservice.AdminActionDeptOwnerManualGrant,
			ObjectType:  entservice.AdminObjectDepartmentRole,
			ObjectId:    strconv.Itoa(role.Id),
			DiffSummary: "Granted department owner role",
			Payload: map[string]any{
				"user_id":       role.UserId,
				"department_id": role.DepartmentId,
				"role":          role.Role,
				"source":        role.Source,
				"effect":        role.Effect,
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
			ActionType:  entservice.AdminActionDeptOwnerManualGrantRevoke,
			ObjectType:  entservice.AdminObjectDepartmentRole,
			ObjectId:    entservice.MembershipObjectId(departmentId, userId),
			DiffSummary: "Revoked department owner manual grant",
			Payload: map[string]any{
				"user_id":       userId,
				"department_id": departmentId,
				"source":        "manual_grant",
			},
		})
	})
	if err != nil {
		writeDepartmentRoleError(c, err)
		return
	}

	common.ApiSuccess(c, nil)
}

func ListDepartmentOwners(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	tenantId, ok := parseDepartmentRoleTenantId(c)
	if !ok {
		return
	}

	resolution, err := entservice.NewPermissionService(model.DB).ResolveEffectiveDepartmentOwners(tenantId, departmentId)
	if err != nil && !errors.Is(err, entservice.ErrDepartmentOwnerNotFound) {
		writeDepartmentRoleError(c, err)
		return
	}
	common.ApiSuccess(c, mapDepartmentOwnersResponse(resolution))
}

func GrantDepartmentOwner(c *gin.Context) {
	mutateDepartmentOwner(c, constantOwnerMutationGrant)
}

func DenyDepartmentOwner(c *gin.Context) {
	mutateDepartmentOwner(c, constantOwnerMutationDeny)
}

func RevokeDepartmentOwnerGrant(c *gin.Context) {
	mutateDepartmentOwner(c, constantOwnerMutationRevokeGrant)
}

func RevokeDepartmentOwnerDeny(c *gin.Context) {
	mutateDepartmentOwner(c, constantOwnerMutationRevokeDeny)
}

const (
	constantOwnerMutationGrant       = "grant"
	constantOwnerMutationDeny        = "deny"
	constantOwnerMutationRevokeGrant = "revoke_grant"
	constantOwnerMutationRevokeDeny  = "revoke_deny"
)

func mutateDepartmentOwner(c *gin.Context, mutation string) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	req, ok := parseDepartmentOwnerMutationRequest(c, mutation)
	if !ok {
		return
	}
	tenantId := valueOrZero(req.TenantId)
	input := entservice.DepartmentAdminRoleInput{TenantId: tenantId, UserId: req.UserId, DepartmentId: departmentId}

	var item dtoenterprise.DepartmentRoleItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		permission := entservice.NewPermissionService(tx)
		action := entservice.AdminActionInput{
			TenantId:   tenantId,
			ActorId:    c.GetInt("id"),
			ObjectType: entservice.AdminObjectDepartmentRole,
			ObjectId:   entservice.MembershipObjectId(departmentId, req.UserId),
			Payload: map[string]any{
				"actor_id":      c.GetInt("id"),
				"user_id":       req.UserId,
				"department_id": departmentId,
			},
		}
		switch mutation {
		case constantOwnerMutationGrant:
			role, err := permission.GrantDepartmentAdmin(input)
			if err != nil {
				return err
			}
			item = mapDepartmentRole(role)
			action.ActionType = entservice.AdminActionDeptOwnerManualGrant
			action.ObjectId = strconv.Itoa(role.Id)
			action.DiffSummary = "Granted department owner manual grant"
			action.Payload["source"] = role.Source
			action.Payload["effect"] = role.Effect
			action.Payload["after"] = mapDepartmentRole(role)
		case constantOwnerMutationDeny:
			role, err := permission.DenyDepartmentOwner(input)
			if err != nil {
				return err
			}
			item = mapDepartmentRole(role)
			action.ActionType = entservice.AdminActionDeptOwnerManualDeny
			action.ObjectId = strconv.Itoa(role.Id)
			action.DiffSummary = "Applied department owner manual deny override"
			action.Payload["source"] = role.Source
			action.Payload["effect"] = role.Effect
			action.Payload["after"] = mapDepartmentRole(role)
		case constantOwnerMutationRevokeGrant:
			if err := permission.RevokeDepartmentAdmin(input); err != nil {
				return err
			}
			action.ActionType = entservice.AdminActionDeptOwnerManualGrantRevoke
			action.DiffSummary = "Revoked department owner manual grant"
			action.Payload["source"] = "manual_grant"
			action.Payload["effect"] = "allow"
		case constantOwnerMutationRevokeDeny:
			if err := permission.RevokeDepartmentOwnerDeny(input); err != nil {
				return err
			}
			action.ActionType = entservice.AdminActionDeptOwnerManualDenyRevoke
			action.DiffSummary = "Revoked department owner manual deny override"
			action.Payload["source"] = "manual_deny_override"
			action.Payload["effect"] = "deny"
		}
		return entservice.NewAdminActionService(tx).Write(action)
	})
	if err != nil {
		writeDepartmentRoleError(c, err)
		return
	}
	if mutation == constantOwnerMutationRevokeGrant || mutation == constantOwnerMutationRevokeDeny {
		common.ApiSuccess(c, nil)
		return
	}
	common.ApiSuccess(c, item)
}

func parseDepartmentOwnerMutationRequest(c *gin.Context, mutation string) (dtoenterprise.DepartmentOwnerMutationRequest, bool) {
	if mutation != constantOwnerMutationRevokeGrant && mutation != constantOwnerMutationRevokeDeny {
		var req dtoenterprise.DepartmentOwnerMutationRequest
		if err := common.UnmarshalBodyReusable(c, &req); err != nil {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return dtoenterprise.DepartmentOwnerMutationRequest{}, false
		}
		return req, true
	}
	userId, ok := parsePathInt(c, "user_id")
	if !ok {
		return dtoenterprise.DepartmentOwnerMutationRequest{}, false
	}
	tenantId, ok := parseDepartmentRoleTenantId(c)
	if !ok {
		return dtoenterprise.DepartmentOwnerMutationRequest{}, false
	}
	return dtoenterprise.DepartmentOwnerMutationRequest{TenantId: &tenantId, UserId: userId}, true
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
	case errors.Is(err, entservice.ErrDepartmentOwnerDeniedByLocalRule):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentOwnerDeniedByLocalRule)
	case errors.Is(err, entservice.ErrDepartmentOwnerNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentOwnerNotFound)
	case errors.Is(err, entservice.ErrInvalidAdminActionInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func mapDepartmentRole(role entmodel.DepartmentRole) dtoenterprise.DepartmentRoleItem {
	return dtoenterprise.DepartmentRoleItem{
		Id:             role.Id,
		TenantId:       role.TenantId,
		UserId:         role.UserId,
		DepartmentId:   role.DepartmentId,
		Role:           role.Role,
		Source:         role.Source,
		Effect:         role.Effect,
		ExternalSource: role.ExternalSource,
		Status:         role.Status,
		CreatedAt:      role.CreatedAt,
		UpdatedAt:      role.UpdatedAt,
	}
}

func mapDepartmentOwnersResponse(resolution entservice.DepartmentOwnerResolution) dtoenterprise.DepartmentOwnersResponse {
	facts := make([]dtoenterprise.DepartmentOwnerFactItem, 0, len(resolution.Facts))
	for _, fact := range resolution.Facts {
		facts = append(facts, dtoenterprise.DepartmentOwnerFactItem{
			Id:                        fact.Id,
			TenantId:                  fact.TenantId,
			UserId:                    fact.UserId,
			DepartmentId:              fact.DepartmentId,
			Role:                      fact.Role,
			Source:                    fact.Source,
			Effect:                    fact.Effect,
			ExternalSource:            fact.ExternalSource,
			Status:                    fact.Status,
			InheritedFromDepartmentId: fact.InheritedFromDepartmentId,
			CreatedAt:                 fact.CreatedAt,
			UpdatedAt:                 fact.UpdatedAt,
		})
	}
	effectiveOwners := make([]dtoenterprise.EffectiveDepartmentOwnerItem, 0, len(resolution.EffectiveOwners))
	for _, owner := range resolution.EffectiveOwners {
		effectiveOwners = append(effectiveOwners, dtoenterprise.EffectiveDepartmentOwnerItem{
			UserId:                    owner.UserId,
			DepartmentId:              owner.DepartmentId,
			Source:                    owner.Source,
			Effect:                    owner.Effect,
			InheritedFromDepartmentId: owner.InheritedFromDepartmentId,
			RoleFactId:                owner.RoleFactId,
		})
	}
	return dtoenterprise.DepartmentOwnersResponse{
		Facts:           facts,
		EffectiveOwners: effectiveOwners,
		OwnerCount:      resolution.OwnerCount,
		Fallback:        resolution.Fallback,
	}
}
