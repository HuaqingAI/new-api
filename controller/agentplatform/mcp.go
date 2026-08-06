package agentplatform

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func ListMcps(c *gin.Context) {
	var query dtoagentplatform.McpQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := mcpService().List(apservice.McpQuery{
		OwnerUserId: query.OwnerUserId,
		TenantId:    query.TenantId,
		Page:        valueOrZero(query.Page),
		PageSize:    valueOrZero(query.PageSize),
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	items := make([]dtoagentplatform.McpItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapMcpItem(item))
	}
	common.ApiSuccess(c, dtoagentplatform.McpListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func GetMcp(c *gin.Context) {
	item, err := mcpService().Get(c.Param("id"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapMcpItem(item))
}

func CreateMcp(c *gin.Context) {
	var req dtoagentplatform.CreateMcpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := mcpService().Create(apservice.McpCreateInput{
		DisplayName: req.DisplayName,
		Description: req.Description,
		Config:      req.Config,
		OwnerUserId: c.GetInt("id"),
		TenantId:    0,
	})
	if err != nil {
		writeMcpError(c, err)
		return
	}
	common.ApiSuccess(c, mapMcpItem(item))
}

func UpdateMcp(c *gin.Context) {
	var req dtoagentplatform.UpdateMcpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := mcpService().Update(c.Param("id"), apservice.McpUpdateInput{
		DisplayName: req.DisplayName,
		Description: req.Description,
		Config:      req.Config,
	})
	if err != nil {
		writeMcpError(c, err)
		return
	}
	common.ApiSuccess(c, mapMcpItem(item))
}

func EnableMcp(c *gin.Context) {
	item, err := mcpService().SetStatus(c.Param("id"), apmodel.ResourceStatusPublished)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapMcpItem(item))
}

func DisableMcp(c *gin.Context) {
	item, err := mcpService().SetStatus(c.Param("id"), apmodel.ResourceStatusDisabled)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapMcpItem(item))
}

func DeleteMcp(c *gin.Context) {
	if err := mcpService().Delete(c.Param("id")); err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"resource_id": c.Param("id")})
}

func mcpService() *apservice.McpService {
	return apservice.NewMcpService(model.DB)
}

func writeMcpError(c *gin.Context, err error) {
	if errors.Is(err, apservice.ErrInvalidResourceInput) {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	writeResourceError(c, err)
}

func mapMcpItem(item apservice.McpItem) dtoagentplatform.McpItem {
	return dtoagentplatform.McpItem{
		ResourceItem: mapResourceItem(item.ResourceItem),
		Config:       jsonTextToRawMessage(item.ConfigJSON),
	}
}
