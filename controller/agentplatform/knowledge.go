package agentplatform

import (
	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func ListKnowledge(c *gin.Context) {
	var query dtoagentplatform.KnowledgeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := knowledgeService().List(apservice.KnowledgeQuery{
		OwnerUserId: query.OwnerUserId,
		TenantId:    query.TenantId,
		Page:        valueOrZero(query.Page),
		PageSize:    valueOrZero(query.PageSize),
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	items := make([]dtoagentplatform.KnowledgeItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapResourceItem(item.ResourceItem))
	}
	common.ApiSuccess(c, dtoagentplatform.KnowledgeListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func GetKnowledge(c *gin.Context) {
	item, err := knowledgeService().Get(c.Param("id"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func CreateKnowledge(c *gin.Context) {
	var req dtoagentplatform.CreateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := knowledgeService().Create(apservice.KnowledgeCreateInput{
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

func UpdateKnowledge(c *gin.Context) {
	var req dtoagentplatform.UpdateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := knowledgeService().Update(c.Param("id"), apservice.KnowledgeUpdateInput{
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func knowledgeService() *apservice.KnowledgeService {
	return apservice.NewKnowledgeService(model.DB)
}
