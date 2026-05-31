package agentplatform

import (
	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func OpenCapabilityDiscovery(c *gin.Context) {
	claims, ok := openCapabilityClaims(c)
	if !ok {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityPermissionDenied, "", "")
		return
	}
	var query dtoagentplatform.OpenCapabilityDiscoveryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityContractInvalid, "", "")
		return
	}
	result, err := discoveryService().Discovery(apservice.DiscoveryQuery{
		ClientID:     claims.ClientId,
		ResourceType: query.ResourceType,
	})
	if err != nil {
		writeOpenCapabilityError(c, err, "", "")
		return
	}
	items := make([]dtoagentplatform.OpenCapabilityDiscoveryItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dtoagentplatform.OpenCapabilityDiscoveryItem{
			ResourceId:          item.ResourceID,
			ResourceType:        item.ResourceType,
			DisplayName:         item.DisplayName,
			ResourceVersion:     item.ResourceVersion,
			ContractVersion:     item.ContractVersion,
			VisibilityState:     item.VisibilityState,
			CallableState:       item.CallableState,
			FreshnessTTLSeconds: item.FreshnessTTL,
			Freshness:           item.Freshness,
			ETag:                item.ETag,
			Extensions:          jsonTextToRawMessage(item.ExtensionsJSON),
		})
	}
	common.ApiSuccess(c, dtoagentplatform.OpenCapabilityDiscoveryResponse{Items: items, Total: result.Total})
}

func OpenCapabilityResourceDetail(c *gin.Context) {
	claims, ok := openCapabilityClaims(c)
	if !ok {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityPermissionDenied, c.Param("id"), "")
		return
	}
	detail, err := discoveryService().Detail(claims.ClientId, c.Param("id"))
	if err != nil {
		writeOpenCapabilityError(c, err, c.Param("id"), "")
		return
	}
	common.ApiSuccess(c, dtoagentplatform.OpenCapabilityDetailResponse{
		ResourceId:          detail.ResourceID,
		ResourceType:        detail.ResourceType,
		DisplayName:         detail.DisplayName,
		ResourceVersion:     detail.ResourceVersion,
		ContractVersion:     detail.ContractVersion,
		Status:              detail.Status,
		VisibilityState:     detail.VisibilityState,
		CallableState:       detail.CallableState,
		FreshnessTTLSeconds: detail.FreshnessTTLSeconds,
		Freshness:           detail.Freshness,
		ETag:                detail.ETag,
		Schema:              jsonTextToRawMessage(detail.SchemaJSON),
		Detail:              jsonTextToRawMessage(detail.DetailJSON),
		Extensions:          jsonTextToRawMessage(detail.ExtensionsJSON),
		SupportedExtensions: detail.SupportedExtensions,
	})
}

func OpenCapabilityRefresh(c *gin.Context) {
	claims, ok := openCapabilityClaims(c)
	if !ok {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityPermissionDenied, "", "")
		return
	}
	var req dtoagentplatform.OpenCapabilityRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityContractInvalid, "", "")
		return
	}
	result, err := discoveryService().Refresh(claims.ClientId, req.ResourceId)
	if err != nil {
		writeOpenCapabilityError(c, err, req.ResourceId, "")
		return
	}
	common.ApiSuccess(c, dtoagentplatform.OpenCapabilityRefreshResponse{
		ResourceId:          result.ResourceID,
		ResourceVersion:     result.ResourceVersion,
		ContractVersion:     result.ContractVersion,
		FreshnessTTLSeconds: result.FreshnessTTLSeconds,
		Freshness:           result.Freshness,
		ETag:                result.ETag,
		VisibilityState:     result.VisibilityState,
		CallableState:       result.CallableState,
	})
}

func OpenCapabilitySkillInvoke(c *gin.Context) {
	writeOpenCapabilityError(c, apservice.ErrOpenCapabilityContractInvalid, c.Param("id"), "")
}

func OpenCapabilityKnowledgeQuery(c *gin.Context) {
	writeOpenCapabilityError(c, apservice.ErrOpenCapabilityContractInvalid, c.Param("id"), "")
}

func OpenCapabilityAgentDetail(c *gin.Context) {
	OpenCapabilityResourceDetail(c)
}

func discoveryService() *apservice.DiscoveryService {
	return apservice.NewDiscoveryService(model.DB)
}

func writeOpenCapabilityError(c *gin.Context, err error, resourceID string, resourceVersion string) {
	requestID, _ := c.Get(common.RequestIdKey)
	c.JSON(200, apservice.MapOpenCapabilityError(err, apservice.OpenCapabilityContext{
		RequestID:       apservice.OpenCapabilityRequestIDFromGinValue(requestID),
		ResourceID:      resourceID,
		ResourceVersion: resourceVersion,
	}))
}

func openCapabilityClaims(c *gin.Context) (apservice.TokenClaims, bool) {
	value, exists := c.Get(middleware.AgentPlatformClaimsKey)
	if !exists || value == nil {
		return apservice.TokenClaims{}, false
	}
	claims, ok := value.(apservice.TokenClaims)
	return claims, ok
}
