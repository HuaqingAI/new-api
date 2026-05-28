package router

import (
	controllerenterprise "github.com/QuantumNous/new-api/controller/enterprise"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterEnterpriseRouter(apiRouter *gin.RouterGroup) {
	enterpriseRoute := apiRouter.Group("/enterprise")
	enterpriseRoute.Use(middleware.AdminAuth())
	{
		enterpriseRoute.GET("/departments/tree", controllerenterprise.GetDepartmentTree)
	}
}
