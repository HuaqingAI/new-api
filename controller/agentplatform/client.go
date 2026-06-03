package agentplatform

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func CreateClient(c *gin.Context) {
	var req dtoagentplatform.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := clientService().Create(apservice.CreateClientInput{
		Slug:                   req.Slug,
		DisplayName:            req.DisplayName,
		ClientType:             req.ClientType,
		Status:                 req.Status,
		AllowedGrantTypes:      req.AllowedGrantTypes,
		RedirectURIs:           req.RedirectURIs,
		AllowedScopes:          req.AllowedScopes,
		ContractVersion:        req.ContractVersion,
		Capabilities:           req.Capabilities,
		Extensions:             req.Extensions,
		AllowClientCredentials: req.AllowClientCredentials,
	})
	if err != nil {
		writeClientError(c, err)
		return
	}
	common.ApiSuccess(c, mapClientItem(item))
}

func GetClient(c *gin.Context) {
	item, err := clientService().Get(c.Param("id"))
	if err != nil {
		writeClientError(c, err)
		return
	}
	common.ApiSuccess(c, mapClientItem(item))
}

func ListClients(c *gin.Context) {
	var query dtoagentplatform.ClientQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := clientService().List(apservice.ClientQuery{
		Status:          query.Status,
		ContractVersion: query.ContractVersion,
	})
	if err != nil {
		writeClientError(c, err)
		return
	}
	items := make([]dtoagentplatform.ClientItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapClientItem(item))
	}
	common.ApiSuccess(c, dtoagentplatform.ClientListResponse{Items: items, Total: result.Total})
}

func UpdateClient(c *gin.Context) {
	var req dtoagentplatform.UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := clientService().Update(c.Param("id"), apservice.UpdateClientInput{
		DisplayName:            req.DisplayName,
		Status:                 req.Status,
		AllowedGrantTypes:      req.AllowedGrantTypes,
		RedirectURIs:           req.RedirectURIs,
		AllowedScopes:          req.AllowedScopes,
		ContractVersion:        req.ContractVersion,
		Capabilities:           req.Capabilities,
		Extensions:             req.Extensions,
		AllowClientCredentials: req.AllowClientCredentials,
	})
	if err != nil {
		writeClientError(c, err)
		return
	}
	common.ApiSuccess(c, mapClientItem(item))
}

func clientService() *apservice.ClientService {
	return apservice.NewClientService(model.DB)
}

func writeClientError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apservice.ErrInvalidClientInput):
		common.ApiErrorMsg(c, "invalid request params")
	case errors.Is(err, apservice.ErrClientNotFound):
		common.ApiErrorMsg(c, "client not found")
	default:
		common.ApiError(c, err)
	}
}

func mapClientItem(item apservice.ClientItem) dtoagentplatform.ClientItem {
	return dtoagentplatform.ClientItem{
		Id:                     item.Id,
		ClientId:               item.ClientId,
		Slug:                   item.Slug,
		DisplayName:            item.DisplayName,
		ClientType:             item.ClientType,
		Status:                 item.Status,
		AllowedGrantTypes:      jsonTextToRawMessage(item.AllowedGrantTypesJSON),
		RedirectURIs:           jsonTextToRawMessage(item.RedirectURIsJSON),
		AllowedScopes:          jsonTextToRawMessage(item.AllowedScopesJSON),
		ContractVersion:        item.ContractVersion,
		Capabilities:           jsonTextToRawMessage(item.CapabilitiesJSON),
		Extensions:             jsonTextToRawMessage(item.ExtensionsJSON),
		AllowClientCredentials: item.AllowClientCredentials,
		CreatedAt:              item.CreatedAt,
		UpdatedAt:              item.UpdatedAt,
	}
}
