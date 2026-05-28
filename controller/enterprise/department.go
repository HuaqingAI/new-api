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
	items, err := serviceenterprise.NewDepartmentService(model.DB).GetDepartmentTree()
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
