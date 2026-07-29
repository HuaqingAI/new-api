package aionui

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
)

func GetAgentConfigs(c *gin.Context) {
	cliType := c.Query("cli_type")
	if err := serviceaionui.ValidateAgentConfigRequest(cliType); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	email := c.GetString("aionui_email")
	result, err := serviceaionui.NewDefaultAgentConfigService().ListForUser(c.GetInt("aionui_user_id"), email, cliType)
	if err != nil {
		if errors.Is(err, serviceaionui.ErrUnsupportedCliType) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}
