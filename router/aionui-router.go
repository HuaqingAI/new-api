package router

import (
	controlleraionui "github.com/QuantumNous/new-api/controller/aionui"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAionUiRouter(apiRouter *gin.RouterGroup) {
	aionUiRoute := apiRouter.Group("/aionui")
	{
		aionUiRoute.GET("/agent-configs", middleware.AionUiDesktopAuth(), controlleraionui.GetAgentConfigs)
		aionUiRoute.GET("/quota-summary", middleware.AionUiDesktopAuth(), controlleraionui.GetQuotaSummary)
		aionUiRoute.GET("/client-packages/latest", controlleraionui.ListLatestClientPackages)
		aionUiRoute.GET("/client-packages/:id/download", controlleraionui.DownloadClientPackage)
		aionUiRoute.GET("/client-updates/latest.yml", controlleraionui.GetClientUpdateFeed)
		aionUiRoute.GET("/client-updates/latest-mac.yml", controlleraionui.GetClientUpdateFeed)
		aionUiRoute.GET("/client-updates/latest-arm64-mac.yml", controlleraionui.GetClientUpdateFeed)
		aionUiRoute.GET("/client-updates/:version/:file", controlleraionui.DownloadClientUpdateArtifact)

		adminClientPackageRoute := aionUiRoute.Group("/client-packages")
		adminClientPackageRoute.Use(middleware.AdminAuth())
		{
			adminClientPackageRoute.GET("", controlleraionui.AdminListClientPackages)
			adminClientPackageRoute.POST("", controlleraionui.AdminUploadClientPackage)
			adminClientPackageRoute.POST("/uploads/init", controlleraionui.AdminCreateClientPackageUpload)
			adminClientPackageRoute.POST("/uploads/complete", controlleraionui.AdminCompleteClientPackageUpload)
			adminClientPackageRoute.PATCH("/:id/status", controlleraionui.AdminUpdateClientPackageStatus)
			adminClientPackageRoute.DELETE("/:id", controlleraionui.AdminDeleteClientPackage)
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
