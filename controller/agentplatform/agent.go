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
	items := make([]dtoagentplatform.AgentDetailItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapAgentDetailItem(item))
	}
	common.ApiSuccess(c, gin.H{
		"items":     items,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

func GetAgent(c *gin.Context) {
	item, err := agentService().Get(c.Param("id"))
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapAgentDetailItem(item))
}

func CreateAgent(c *gin.Context) {
	var req dtoagentplatform.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := agentService().Create(apservice.AgentCreateInput{
		CliType:      req.CliType,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Avatar:       req.Avatar,
		Instructions: req.Instructions,
		Categories:   req.Categories,
		McpIds:       req.McpIds,
		SkillIds:     req.SkillIds,
		KnowledgeIds: req.KnowledgeIds,
		OwnerUserId:  c.GetInt("id"),
		TenantId:     0,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapAgentDetailItem(item))
}

func UpdateAgent(c *gin.Context) {
	var req dtoagentplatform.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	item, err := agentService().Update(c.Param("id"), apservice.AgentUpdateInput{
		CliType:      req.CliType,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Avatar:       req.Avatar,
		Instructions: req.Instructions,
		Categories:   req.Categories,
		McpIds:       req.McpIds,
		SkillIds:     req.SkillIds,
		KnowledgeIds: req.KnowledgeIds,
	})
	if err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, mapAgentDetailItem(item))
}

func agentService() *apservice.AgentService {
	return apservice.NewAgentService(model.DB)
}

func DeleteAgent(c *gin.Context) {
	if err := agentService().Delete(c.Param("id")); err != nil {
		writeResourceError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"resource_id": c.Param("id")})
}

func PublishAgent(c *gin.Context) {
	var req dtoagentplatform.PublishAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	grants := make([]apservice.PublishAgentGrantInput, 0, len(req.Grants.Users)+len(req.Grants.Departments))
	for _, userID := range req.Grants.Users {
		grants = append(grants, apservice.PublishAgentGrantInput{
			SubjectType: "user",
			SubjectId:   userID,
		})
	}
	for _, departmentID := range req.Grants.Departments {
		grants = append(grants, apservice.PublishAgentGrantInput{
			SubjectType: "department",
			SubjectId:   departmentID,
		})
	}
	result, err := apservice.NewAgentPublishService(model.DB).Publish(apservice.PublishAgentInput{
		ResourceId:  c.Param("id"),
		Summary:     req.Summary,
		Grants:      grants,
		ActorUserId: c.GetInt("id"),
	})
	if err != nil {
		writeResourceVersionError(c, err)
		return
	}
	common.ApiSuccess(c, mapPublishAgentResponse(result))
}

func ListAgentVersions(c *gin.Context) {
	result, err := apservice.NewAgentPublishService(model.DB).ListVersions(c.Param("id"))
	if err != nil {
		writeResourceVersionError(c, err)
		return
	}
	items := make([]dtoagentplatform.AgentVersionResponse, 0, len(result))
	for _, item := range result {
		items = append(items, mapAgentVersionResponse(item))
	}
	common.ApiSuccess(c, gin.H{"items": items})
}

func ListAgentVersionGrants(c *gin.Context) {
	result, err := apservice.NewAgentPublishService(model.DB).ListVersionGrants(c.Param("id"), c.Param("version"))
	if err != nil {
		writeResourceVersionError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": mapAgentGrantResponses(result)})
}

func GetAgentPublishDefaults(c *gin.Context) {
	result, err := apservice.NewAgentPublishService(model.DB).PublishDefaults(c.Param("id"))
	if err != nil {
		writeResourceVersionError(c, err)
		return
	}
	common.ApiSuccess(c, dtoagentplatform.PublishAgentDefaultsResponse{
		ResourceId:    result.ResourceId,
		LatestVersion: result.LatestVersion,
		NextVersion:   result.NextVersion,
		Grants:        mapAgentGrantResponses(result.Grants),
	})
}

func mapAgentDetailItem(item apservice.AgentItem) dtoagentplatform.AgentDetailItem {
	return dtoagentplatform.AgentDetailItem{
		ResourceItem: mapResourceItem(item.ResourceItem),
		CliType:      item.CliType,
		Instructions: item.Instructions,
		Categories:   item.Categories,
		McpIds:       item.McpIds,
		SkillIds:     item.SkillIds,
		KnowledgeIds: item.KnowledgeIds,
	}
}

func mapPublishAgentResponse(result apservice.PublishAgentResult) dtoagentplatform.PublishAgentResponse {
	return dtoagentplatform.PublishAgentResponse{
		ResourceId: result.ResourceId,
		Version:    result.Version,
		Status:     result.Status,
		Artifact: dtoagentplatform.AgentArtifactResponse{
			CliType:      result.Artifact.CliType,
			ArtifactKey:  result.Artifact.ArtifactKey,
			Url:          result.Artifact.Url,
			UrlType:      result.Artifact.UrlType,
			UrlExpiresAt: result.Artifact.UrlExpiresAt,
			Sha256:       result.Artifact.Sha256,
			Size:         result.Artifact.Size,
		},
		Grants: mapAgentGrantResponses(result.Grants),
	}
}

func mapAgentGrantResponses(items []apservice.PublishAgentGrantResult) []dtoagentplatform.AgentGrantResponse {
	grants := make([]dtoagentplatform.AgentGrantResponse, 0, len(items))
	for _, item := range items {
		grants = append(grants, dtoagentplatform.AgentGrantResponse{
			GrantId:     item.GrantId,
			SubjectType: item.SubjectType,
			SubjectId:   item.SubjectId,
			SubjectName: item.SubjectName,
		})
	}
	return grants
}

func mapAgentVersionResponse(item apservice.AgentVersionItem) dtoagentplatform.AgentVersionResponse {
	return dtoagentplatform.AgentVersionResponse{
		ResourceId:    item.ResourceId,
		Version:       item.Version,
		Summary:       item.Summary,
		Status:        item.Status,
		PackagePath:   item.PackagePath,
		PackageSha256: item.PackageSha256,
		PackageSize:   item.PackageSize,
		PublishedAt:   item.PublishedAt,
		CreatedAt:     item.CreatedAt,
	}
}
