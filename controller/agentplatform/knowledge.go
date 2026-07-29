package agentplatform

import (
	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
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
	items := make([]dtoagentplatform.KnowledgeDetailItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapKnowledgeDetailItem(item))
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
	common.ApiSuccess(c, mapKnowledgeDetailItem(item))
}

func CreateKnowledge(c *gin.Context) {
	var req dtoagentplatform.CreateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := knowledgeService().Create(apservice.KnowledgeCreateInput{
		DisplayName:         req.DisplayName,
		Description:         req.Description,
		ExternalKnowledgeId: req.ExternalKnowledgeId,
		OwnerUserId:         c.GetInt("id"),
		TenantId:            0,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapKnowledgeDetailItem(item))
}

func UpdateKnowledge(c *gin.Context) {
	var req dtoagentplatform.UpdateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := knowledgeService().Update(c.Param("id"), apservice.KnowledgeUpdateInput{
		DisplayName:         req.DisplayName,
		Description:         req.Description,
		ExternalKnowledgeId: req.ExternalKnowledgeId,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapKnowledgeDetailItem(item))
}

func knowledgeService() *apservice.KnowledgeService {
	return apservice.NewKnowledgeService(model.DB)
}

func EnableKnowledge(c *gin.Context) {
	item, err := knowledgeService().SetStatus(c.Param("id"), apmodel.ResourceStatusPublished)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapKnowledgeDetailItem(item))
}

func DisableKnowledge(c *gin.Context) {
	item, err := knowledgeService().SetStatus(c.Param("id"), apmodel.ResourceStatusDisabled)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapKnowledgeDetailItem(item))
}

func DeleteKnowledge(c *gin.Context) {
	if err := knowledgeService().Delete(c.Param("id")); err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"resource_id": c.Param("id")})
}

func mapKnowledgeDetailItem(item apservice.KnowledgeItem) dtoagentplatform.KnowledgeDetailItem {
	return dtoagentplatform.KnowledgeDetailItem{
		ResourceItem:        mapResourceItem(item.ResourceItem),
		ExternalKnowledgeId: item.ExternalKnowledgeId,
	}
}
