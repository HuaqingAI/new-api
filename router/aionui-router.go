package router

import (
	"github.com/QuantumNous/new-api/controller"
	controlleraionui "github.com/QuantumNous/new-api/controller/aionui"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAionUiRouter(apiRouter *gin.RouterGroup) {
	aionUiRoute := apiRouter.Group("/aionui")
	{
		aionUiRoute.GET("/agent-configs", middleware.AionUiDesktopAuth(), controlleraionui.GetAgentConfigs)
		aionUiRoute.GET("/pricing", middleware.AionUiDesktopAuth(), controller.GetAionUiPricing)
		aionUiRoute.GET("/quota-summary", middleware.AionUiDesktopAuth(), controlleraionui.GetQuotaSummary)
		aionUiRoute.POST("/clients/heartbeat", middleware.AionUiDesktopAuth(), middleware.UserCriticalRateLimit("aionui-client-heartbeat"), controlleraionui.ReportClientHeartbeat)
		aionUiRoute.GET("/client-packages/latest", controlleraionui.ListLatestClientPackages)
		aionUiRoute.GET("/client-packages/:id/download", controlleraionui.DownloadClientPackage)
		aionUiRoute.POST("/client-updates/access", middleware.AionUiOptionalDesktopAuth(), controlleraionui.PrepareClientUpdateAccess)
		aionUiRoute.GET("/client-updates/latest.yml", controlleraionui.GetClientUpdateFeed)
		aionUiRoute.GET("/client-updates/latest-mac.yml", controlleraionui.GetClientUpdateFeed)
		aionUiRoute.GET("/client-updates/latest-arm64-mac.yml", controlleraionui.GetClientUpdateFeed)
		aionUiRoute.GET("/client-updates/:version/:file", controlleraionui.DownloadClientUpdateArtifact)

		adminClientPackageRoute := aionUiRoute.Group("/client-packages")
		adminClientPackageRoute.Use(middleware.AdminAuth())
		{
			adminClientPackageRoute.GET("", controlleraionui.AdminListClientPackages)
			adminClientPackageRoute.GET("/:id/download-url", controlleraionui.AdminGetClientPackageDownloadURL)
			adminClientPackageRoute.POST("", controlleraionui.AdminUploadClientPackage)
			adminClientPackageRoute.POST("/uploads/init", controlleraionui.AdminCreateClientPackageUpload)
			adminClientPackageRoute.POST("/uploads/complete", controlleraionui.AdminCompleteClientPackageUpload)
			adminClientPackageRoute.PATCH("/:id/status", controlleraionui.AdminUpdateClientPackageStatus)
			adminClientPackageRoute.GET("/:id/rollout", controlleraionui.AdminGetClientPackageRollout)
			adminClientPackageRoute.PUT("/:id/rollout", controlleraionui.AdminUpdateClientPackageRollout)
			adminClientPackageRoute.DELETE("/:id", controlleraionui.AdminDeleteClientPackage)
		}

		adminClientRoute := aionUiRoute.Group("/clients")
		adminClientRoute.Use(middleware.AdminAuth())
		{
			adminClientRoute.GET("", controlleraionui.AdminListClientInstallations)
		}
	}
}

func RegisterAionUiDesktopAuthRouter(apiRouter *gin.RouterGroup) {
	aionUiRoute := apiRouter.Group("/aionui")
	{
		aionUiRoute.GET("/desktop/login", controlleraionui.DesktopLogin)
		aionUiRoute.POST("/desktop/token", controlleraionui.DesktopToken)
	}
}
