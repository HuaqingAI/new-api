package agentplatform

import (
	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func ListAgents(c *gin.Context) {
	var query dtoagentplatform.AgentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := agentService().List(apservice.AgentQuery{
		OwnerUserId: query.OwnerUserId,
		TenantId:    query.TenantId,
		Page:        valueOrZero(query.Page),
		PageSize:    valueOrZero(query.PageSize),
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	items := make([]dtoagentplatform.AgentItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapResourceItem(item.ResourceItem))
	}
	common.ApiSuccess(c, dtoagentplatform.AgentListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func GetAgent(c *gin.Context) {
	item, err := agentService().Get(c.Param("id"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func CreateAgent(c *gin.Context) {
	var req dtoagentplatform.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := agentService().Create(apservice.AgentCreateInput{
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

func UpdateAgent(c *gin.Context) {
	var req dtoagentplatform.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := agentService().Update(c.Param("id"), apservice.AgentUpdateInput{
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapResourceItem(item.ResourceItem))
}

func agentService() *apservice.AgentService {
	return apservice.NewAgentService(model.DB)
}
