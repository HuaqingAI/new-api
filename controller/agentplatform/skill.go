package agentplatform

import (
	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func ListSkills(c *gin.Context) {
	var query dtoagentplatform.SkillQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := skillService().List(apservice.SkillQuery{
		OwnerUserId: query.OwnerUserId,
		TenantId:    query.TenantId,
		Page:        valueOrZero(query.Page),
		PageSize:    valueOrZero(query.PageSize),
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	items := make([]dtoagentplatform.SkillItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapResourceItem(item.ResourceItem))
	}
	common.ApiSuccess(c, dtoagentplatform.SkillListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func GetSkill(c *gin.Context) {
	item, err := skillService().Get(c.Param("id"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func CreateSkill(c *gin.Context) {
	var req dtoagentplatform.CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := skillService().Create(apservice.SkillCreateInput{
		DisplayName: req.DisplayName,
		OwnerUserId: req.OwnerUserId,
		TenantId:    valueOrZero(req.TenantId),
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func UpdateSkill(c *gin.Context) {
	var req dtoagentplatform.UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := skillService().Update(c.Param("id"), apservice.SkillUpdateInput{
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func skillService() *apservice.SkillService {
	return apservice.NewSkillService(model.DB)
}
