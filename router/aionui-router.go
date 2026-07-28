package router

import (
	controlleraionui "github.com/QuantumNous/new-api/controller/aionui"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAionUiRouter(apiRouter *gin.RouterGroup) {
	aionUiRoute := apiRouter.Group("/aionui")
	{
		aionUiRoute.GET("/desktop/login", middleware.CriticalRateLimit(), controlleraionui.DesktopLogin)
		aionUiRoute.POST("/desktop/token", middleware.CriticalRateLimit(), controlleraionui.DesktopToken)
		aionUiRoute.GET("/agent-configs", middleware.AionUiDesktopAuth(), controlleraionui.GetAgentConfigs)
	}
}
