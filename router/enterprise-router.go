package router

import (
	controllerenterprise "github.com/QuantumNous/new-api/controller/enterprise"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterEnterpriseRouter(apiRouter *gin.RouterGroup) {
	enterpriseRoute := apiRouter.Group("/enterprise")
	enterpriseRoute.Use(middleware.UserAuth())
	{
		enterpriseRoute.GET("/departments/tree", controllerenterprise.GetDepartmentTree)
		enterpriseRoute.GET("/users/:id/departments", middleware.AdminAuth(), controllerenterprise.ListUserDepartments)
		enterpriseRoute.PUT("/users/:id/departments", middleware.AdminAuth(), controllerenterprise.ReplaceUserDepartments)
		enterpriseRoute.GET("/departments/:id/members", middleware.EnterpriseDepartmentAdmin("id"), controllerenterprise.ListDepartmentMembers)
		enterpriseRoute.GET("/departments/:id/budget", middleware.EnterpriseDepartmentAdmin("id"), controllerenterprise.GetDepartmentBudget)
		enterpriseRoute.GET("/departments/:id/budgets", middleware.EnterpriseDepartmentAdmin("id"), controllerenterprise.ListDepartmentBudgets)
		enterpriseRoute.GET("/departments/:id/budgets/:budget_id", middleware.EnterpriseDepartmentAdmin("id"), controllerenterprise.GetDepartmentBudgetDetail)
		enterpriseRoute.POST("/departments/:id/budget", middleware.EnterpriseDepartmentAdmin("id"), controllerenterprise.CreateDepartmentBudget)
		enterpriseRoute.GET("/quota-allocations", middleware.EnterpriseDepartmentAdmin("department_id"), controllerenterprise.ListQuotaAllocations)
		enterpriseRoute.POST("/quota-allocations", middleware.EnterpriseDepartmentAdmin("department_id"), controllerenterprise.CreateQuotaAllocation)
		enterpriseRoute.POST("/quota-allocations/:id/revoke", middleware.EnterpriseDepartmentAdmin("department_id"), controllerenterprise.RevokeQuotaAllocation)
		enterpriseRoute.POST("/departments/:id/members", middleware.EnterpriseDepartmentAdmin("id"), controllerenterprise.AddDepartmentMember)
		enterpriseRoute.DELETE("/departments/:id/members/:user_id", middleware.EnterpriseDepartmentAdmin("id"), controllerenterprise.DeactivateDepartmentMember)
		enterpriseRoute.POST("/departments/:id/members/:user_id/restore", middleware.EnterpriseDepartmentAdmin("id"), controllerenterprise.RestoreDepartmentMember)
		enterpriseRoute.POST("/departments/:id/admins", middleware.EnterpriseAdmin(), controllerenterprise.GrantDepartmentAdmin)
		enterpriseRoute.DELETE("/departments/:id/admins/:user_id", middleware.EnterpriseAdmin(), controllerenterprise.RevokeDepartmentAdmin)
		enterpriseRoute.GET("/admin-actions", middleware.EnterpriseAdmin(), controllerenterprise.ListAdminActions)
		enterpriseRoute.GET("/admin-actions/:id", middleware.EnterpriseAdmin(), controllerenterprise.GetAdminAction)
		enterpriseRoute.GET("/dingtalk/config", middleware.RootAuth(), controllerenterprise.GetDingTalkConfig)
		enterpriseRoute.PUT("/dingtalk/config", middleware.RootAuth(), controllerenterprise.SaveDingTalkConfig)
		enterpriseRoute.POST("/dingtalk/connectivity-test", middleware.RootAuth(), controllerenterprise.TestDingTalkConnectivity)
		enterpriseRoute.POST("/dingtalk/sync/full", middleware.RootAuth(), controllerenterprise.StartDingTalkFullSync)
		enterpriseRoute.GET("/dingtalk/sync/tasks/:id", middleware.RootAuth(), controllerenterprise.GetDingTalkSyncTask)
		enterpriseRoute.GET("/dingtalk/sync/logs", middleware.RootAuth(), controllerenterprise.ListDingTalkSyncLogs)
		enterpriseRoute.GET("/dingtalk/sync/conflicts", middleware.RootAuth(), controllerenterprise.ListDingTalkSyncConflicts)
	}
}
