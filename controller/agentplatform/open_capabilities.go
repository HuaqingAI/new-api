package agentplatform

import (
	"io"

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
		ContractCompatible:  detail.ContractCompatible,
		Diagnostics:         mapOpenCapabilityDiagnostics(detail.Diagnostics),
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
	result, err := discoveryService().Refresh(apservice.RefreshInput{
		ClientID:                claims.ClientId,
		ResourceID:              req.ResourceId,
		ObservedETag:            req.ObservedETag,
		ObservedResourceVersion: req.ObservedResourceVersion,
		ObservedAtUnix:          req.ObservedAt,
	})
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
		ContractCompatible:  result.ContractCompatible,
		Diagnostics:         mapOpenCapabilityDiagnostics(result.Diagnostics),
	})
}

func OpenCapabilityModelDiscovery(c *gin.Context) {
	claims, ok := openCapabilityClaims(c)
	if !ok {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityPermissionDenied, "", "")
		return
	}
	result, err := modelDiscoveryService().List(claims.ClientId)
	if err != nil {
		writeOpenCapabilityError(c, err, "", "")
		return
	}
	items := make([]dtoagentplatform.OpenCapabilityModelDiscoveryItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dtoagentplatform.OpenCapabilityModelDiscoveryItem{
			ModelID:          item.ModelID,
			ProviderStableID: item.ProviderStableID,
			DisplayName:      item.DisplayName,
			IsDefault:        item.IsDefault,
			Status:           item.Status,
			DisabledReason:   item.DisabledReason,
			Capabilities:     jsonTextToRawMessage(item.CapabilitiesJSON),
			AccountID:        item.AccountID,
			TenantID:         item.TenantID,
		})
	}
	common.ApiSuccess(c, dtoagentplatform.OpenCapabilityModelDiscoveryResponse{
		ContractVersion: result.ContractVersion,
		DefaultState:    result.DefaultState,
		Items:           items,
		Total:           result.Total,
	})
}

func OpenCapabilitySkillInvoke(c *gin.Context) {
	claims, ok := openCapabilityClaims(c)
	if !ok {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityPermissionDenied, c.Param("id"), "")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityContractInvalid, c.Param("id"), "")
		return
	}
	result, err := skillInvokeService().Invoke(apservice.SkillInvokeInput{
		ClientID:   claims.ClientId,
		ResourceID: c.Param("id"),
		Payload:    body,
	})
	if err != nil {
		writeOpenCapabilityError(c, err, c.Param("id"), "")
		return
	}
	var output any
	if err := common.Unmarshal(result.Output, &output); err != nil {
		output = gin.H{"raw": string(result.Output)}
	}
	common.ApiSuccess(c, dtoagentplatform.OpenCapabilitySkillInvokeResponse{
		ResourceId:      result.ResourceID,
		ResourceVersion: result.ResourceVersion,
		ContractVersion: result.ContractVersion,
		Output:          output,
	})
}

func OpenCapabilityKnowledgeQuery(c *gin.Context) {
	claims, ok := openCapabilityClaims(c)
	if !ok {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityPermissionDenied, c.Param("id"), "")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeOpenCapabilityError(c, apservice.ErrOpenCapabilityContractInvalid, c.Param("id"), "")
		return
	}
	result, err := knowledgeQueryService().Query(apservice.KnowledgeQueryInput{
		ClientID:   claims.ClientId,
		ResourceID: c.Param("id"),
		Payload:    body,
	})
	if err != nil {
		writeOpenCapabilityError(c, err, c.Param("id"), "")
		return
	}
	items := make([]dtoagentplatform.OpenCapabilityKnowledgeResultItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dtoagentplatform.OpenCapabilityKnowledgeResultItem{
			ID:       item.ID,
			Score:    item.Score,
			Snippet:  item.Snippet,
			Metadata: item.Metadata,
		})
	}
	citations := make([]dtoagentplatform.OpenCapabilityKnowledgeCitation, 0, len(result.Citations))
	for _, citation := range result.Citations {
		citations = append(citations, dtoagentplatform.OpenCapabilityKnowledgeCitation{
			SourceID: citation.SourceID,
			Title:    citation.Title,
			URL:      citation.URL,
			Metadata: citation.Metadata,
		})
	}
	common.ApiSuccess(c, dtoagentplatform.OpenCapabilityKnowledgeQueryResponse{
		ResourceId:      result.ResourceID,
		ResourceVersion: result.ResourceVersion,
		ContractVersion: result.ContractVersion,
		Items:           items,
		Citations:       citations,
	})
}

func OpenCapabilityAgentDetail(c *gin.Context) {
	OpenCapabilityResourceDetail(c)
}

var discoveryService = func() *apservice.DiscoveryService {
	return apservice.NewDiscoveryService(model.DB)
}

var skillInvokeService = func() *apservice.SkillInvokeService {
	return apservice.NewSkillInvokeService(model.DB)
}

var knowledgeQueryService = func() *apservice.KnowledgeQueryService {
	return apservice.NewKnowledgeQueryService(model.DB)
}

var modelDiscoveryService = func() *apservice.ModelDiscoveryService {
	return apservice.NewModelDiscoveryService(model.DB)
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

func mapOpenCapabilityDiagnostics(input apservice.CapabilityDiagnostics) dtoagentplatform.OpenCapabilityDiagnostics {
	return dtoagentplatform.OpenCapabilityDiagnostics{
		Reason:             input.Reason,
		Converged:          input.Converged,
		ClientNonCompliant: input.ClientNonCompliant,
		ObservedETag:       input.ObservedETag,
		ObservedVersion:    input.ObservedVersion,
	}
}
