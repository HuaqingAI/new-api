package router

import (
	controlleragentplatform "github.com/QuantumNous/new-api/controller/agentplatform"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAgentPlatformRouter(apiRouter *gin.RouterGroup) {
	agentPlatformRoute := apiRouter.Group("/agent-platform")
	agentPlatformRoute.Use(middleware.AdminAuth())
	{
		agentPlatformRoute.GET("/resources", controlleragentplatform.ListResources)
		agentPlatformRoute.POST("/resources", controlleragentplatform.CreateResource)
		agentPlatformRoute.GET("/resources/:id", controlleragentplatform.GetResource)
		agentPlatformRoute.POST("/resources/:id/versions", controlleragentplatform.CreateResourceVersion)
		agentPlatformRoute.GET("/resources/:id/versions/:version", controlleragentplatform.GetResourceVersion)
	}
}
