package enterprise

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	serviceenterprise "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
)

func GetDepartmentTree(c *gin.Context) {
	departmentService := serviceenterprise.NewDepartmentService(model.DB)
	var (
		items any
		err   error
	)
	if c.GetInt("role") >= common.RoleAdminUser {
		items, err = departmentService.GetDepartmentTree()
	} else {
		manageableDepartmentIds, permissionErr := serviceenterprise.NewPermissionService(model.DB).ListManageableDepartmentIds(c.GetInt("id"), 0)
		if permissionErr != nil {
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
			return
		}
		if len(manageableDepartmentIds) == 0 {
			membershipResult, membershipErr := serviceenterprise.NewDepartmentMembershipService(model.DB).ListUserDepartments(c.GetInt("id"), serviceenterprise.MembershipQuery{})
			if membershipErr != nil {
				common.ApiErrorI18n(c, i18n.MsgDatabaseError)
				return
			}
			memberDepartmentIds := make([]int, 0, len(membershipResult.Items))
			for _, item := range membershipResult.Items {
				memberDepartmentIds = append(memberDepartmentIds, item.DepartmentId)
			}
			if len(memberDepartmentIds) == 0 {
				common.ApiErrorI18n(c, i18n.MsgEnterprisePermissionDeptAdminRequired)
				return
			}
			items, err = departmentService.GetDepartmentTreeByIds(memberDepartmentIds)
		} else {
			items, err = departmentService.GetDepartmentTreeByIds(manageableDepartmentIds)
		}
	}
	if err == nil {
		common.ApiSuccess(c, items)
		return
	}
	if errors.Is(err, serviceenterprise.ErrDepartmentNameHistoryInvalid) {
		common.ApiErrorI18n(c, i18n.MsgEnterpriseOrganizationInvalidNameHistory)
		return
	}
	if errors.Is(err, serviceenterprise.ErrDepartmentNotFound) {
		common.ApiErrorI18n(c, i18n.MsgEnterpriseOrganizationDepartmentNotFound)
		return
	}
	common.ApiErrorI18n(c, i18n.MsgDatabaseError)
}
