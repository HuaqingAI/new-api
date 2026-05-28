package middleware

import (
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

		allowed, err := entservice.NewPermissionService(model.DB).CanManageDepartment(c.GetInt("id"), 0, departmentId)
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
