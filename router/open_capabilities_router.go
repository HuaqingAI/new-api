package router

import (
	controlleragentplatform "github.com/QuantumNous/new-api/controller/agentplatform"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterOpenCapabilitiesRouter(apiRouter *gin.RouterGroup) {
	openCapabilitiesRoute := apiRouter.Group("/open-capabilities")
	{
		openCapabilitiesRoute.GET("/discovery", middleware.AgentPlatformBearer("ap.resources.read"), controlleragentplatform.OpenCapabilityDiscovery)
		openCapabilitiesRoute.GET("/resources/:id", middleware.AgentPlatformBearer("ap.resources.read"), controlleragentplatform.OpenCapabilityResourceDetail)
		openCapabilitiesRoute.GET("/models", middleware.AgentPlatformBearer("ap.resources.read"), controlleragentplatform.OpenCapabilityModelDiscovery)
		openCapabilitiesRoute.POST("/refresh", middleware.AgentPlatformBearer("ap.resources.read"), controlleragentplatform.OpenCapabilityRefresh)
		openCapabilitiesRoute.POST("/skills/:id/invoke", middleware.AgentPlatformBearer("ap.skills.invoke"), controlleragentplatform.OpenCapabilitySkillInvoke)
		openCapabilitiesRoute.POST("/knowledge-bases/:id/query", middleware.AgentPlatformBearer("ap.knowledge.query"), controlleragentplatform.OpenCapabilityKnowledgeQuery)
		openCapabilitiesRoute.GET("/agents/:id", middleware.AgentPlatformBearer("ap.agents.read"), controlleragentplatform.OpenCapabilityAgentDetail)
	}
}
