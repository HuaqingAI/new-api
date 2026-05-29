package middleware

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
)

func EnterpriseDepartmentAdmin(departmentParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetInt("role") >= common.RoleAdminUser {
			c.Next()
			return
		}

		departmentId, err := strconv.Atoi(c.Param(departmentParam))
		if err != nil || departmentId <= 0 {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			c.Abort()
			return
		}
		tenantId, ok := enterpriseDepartmentTenantId(c)
		if !ok {
			c.Abort()
			return
		}

		allowed, err := entservice.NewPermissionService(model.DB).CanManageDepartment(c.GetInt("id"), tenantId, departmentId)
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
			c.Abort()
			return
		}
		if !allowed {
			common.ApiErrorI18n(c, i18n.MsgEnterprisePermissionDeptAdminRequired)
			c.Abort()
			return
		}
		c.Next()
	}
}

func enterpriseDepartmentTenantId(c *gin.Context) (int, bool) {
	if c.Request.Method == http.MethodGet || c.Query("tenant_id") != "" {
		return enterpriseDepartmentTenantIdFromQuery(c)
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
	if req.TenantId == nil {
		return 0, true
	}
	if *req.TenantId < 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return 0, false
	}
	return *req.TenantId, true
}

func enterpriseDepartmentTenantIdFromQuery(c *gin.Context) (int, bool) {
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
