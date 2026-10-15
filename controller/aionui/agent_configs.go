package aionui

import (
	"github.com/QuantumNous/new-api/common"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
)

func GetAgentConfigs(c *gin.Context) {
	email := c.GetString("aionui_email")
	result, err := serviceaionui.NewDefaultAgentConfigService().ListForUser(c.GetInt("aionui_user_id"), email)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}
