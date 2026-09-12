package middleware

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/gin-gonic/gin"
)

func EnterpriseAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetInt("role") < common.RoleAdminUser {
			common.ApiErrorI18n(c, i18n.MsgEnterprisePermissionAdminRequired)
			c.Abort()
			return
		}
		c.Next()
	}
}
