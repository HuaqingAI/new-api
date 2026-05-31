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
		agentPlatformRoute.GET("/resources/:id/exposures", controlleragentplatform.ListExposures)
		agentPlatformRoute.POST("/resources/:id/exposures", controlleragentplatform.CreateExposure)
		agentPlatformRoute.GET("/resources/:id/exposures/:target", controlleragentplatform.GetExposure)
		agentPlatformRoute.PUT("/resources/:id/exposures/:target", controlleragentplatform.UpdateExposure)
		agentPlatformRoute.POST("/resources/:id/publish", controlleragentplatform.PublishResource)
		agentPlatformRoute.POST("/resources/:id/disable", controlleragentplatform.DisableResource)
		agentPlatformRoute.POST("/resources/:id/revoke", controlleragentplatform.RevokeResource)
		agentPlatformRoute.POST("/resources/:id/offline", controlleragentplatform.OfflineResource)
		agentPlatformRoute.POST("/resources/:id/versions/:version/rollback", controlleragentplatform.RollbackResourceVersion)
	}
}
