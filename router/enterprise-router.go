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
		enterpriseRoute.GET("/users/:id/departments", controllerenterprise.ListUserDepartments)
		enterpriseRoute.PUT("/users/:id/departments", controllerenterprise.ReplaceUserDepartments)
		enterpriseRoute.GET("/departments/:id/members", controllerenterprise.ListDepartmentMembers)
		enterpriseRoute.POST("/departments/:id/members", controllerenterprise.AddDepartmentMember)
		enterpriseRoute.DELETE("/departments/:id/members/:user_id", controllerenterprise.DeactivateDepartmentMember)
		enterpriseRoute.POST("/departments/:id/members/:user_id/restore", controllerenterprise.RestoreDepartmentMember)
	}
}
