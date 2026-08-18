package agentplatform

import (
	"encoding/json"
	"testing"
	"time"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newResourceDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))
	return db
}

func TestDeleteTypedResourcePhysicallyDeletesMcpWhenUnused(t *testing.T) {
	db := newResourceDeleteTestDB(t)
	mcpService := NewMcpService(db)

	mcp, err := mcpService.Create(McpCreateInput{
		DisplayName: "MCP A",
		Config:      json.RawMessage(`{"mcpServers":{"local":{"command":"node"}}}`),
		OwnerUserId: 100,
	})
	require.NoError(t, err)

	err = mcpService.Delete(mcp.ResourceId)

	require.NoError(t, err)
	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", mcp.ResourceId, 0)
	requireTableCount(t, db, &apmodel.McpDef{}, "resource_id = ?", mcp.ResourceId, 0)
	_, err = mcpService.Get(mcp.ResourceId)
	require.ErrorIs(t, err, ErrResourceNotFound)
}

func TestDeleteTypedResourceAllowsDraftOnlyDependencyTarget(t *testing.T) {
	db := newResourceDeleteTestDB(t)
	mcpService := NewMcpService(db)

	mcp, err := mcpService.Create(McpCreateInput{
		DisplayName: "MCP A",
		Config:      json.RawMessage(`{"mcpServers":{"local":{"command":"node"}}}`),
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	_, err = NewAgentService(db).Create(AgentCreateInput{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "Agent A",
		Categories:  []string{apmodel.AgentCategoryGeneral},
		McpIds:      []string{mcp.ResourceId},
		OwnerUserId: 100,
	})
	require.NoError(t, err)

	err = mcpService.Delete(mcp.ResourceId)

	require.NoError(t, err)
	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", mcp.ResourceId, 0)
	requireTableCount(t, db, &apmodel.McpDef{}, "resource_id = ?", mcp.ResourceId, 0)
	requireTableCount(t, db, &apmodel.AgentDependency{}, "target_resource_id = ?", mcp.ResourceId, 0)
}

func TestDeleteTypedResourceRejectsDependencyTargetReferencedByLatestAgentVersion(t *testing.T) {
	db := newResourceDeleteTestDB(t)
	mcpService := NewMcpService(db)

	mcp, err := mcpService.Create(McpCreateInput{
		DisplayName: "MCP A",
		Config:      json.RawMessage(`{"mcpServers":{"local":{"command":"node"}}}`),
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	agent, err := NewAgentService(db).Create(AgentCreateInput{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "Agent A",
		Categories:  []string{apmodel.AgentCategoryGeneral},
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	requireLatestAgentDependency(t, db, agent.ResourceId, "1.0.0", apmodel.AgentDependencyTypeMCP, mcp.ResourceId)

	err = mcpService.Delete(mcp.ResourceId)

	require.ErrorIs(t, err, ErrResourceInUse)
	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", mcp.ResourceId, 1)
	requireTableCount(t, db, &apmodel.McpDef{}, "resource_id = ?", mcp.ResourceId, 1)
}

func TestDeleteTypedResourceAllowsOlderPublishedDependencyTarget(t *testing.T) {
	db := newResourceDeleteTestDB(t)
	mcpService := NewMcpService(db)

	mcp, err := mcpService.Create(McpCreateInput{
		DisplayName: "MCP A",
		Config:      json.RawMessage(`{"mcpServers":{"local":{"command":"node"}}}`),
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	agent, err := NewAgentService(db).Create(AgentCreateInput{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "Agent A",
		Categories:  []string{apmodel.AgentCategoryGeneral},
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	requireLatestAgentDependency(t, db, agent.ResourceId, "1.0.0", apmodel.AgentDependencyTypeMCP, mcp.ResourceId)
	require.NoError(t, db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", agent.ResourceId).
		Update("latest_version", "1.0.1").Error)

	err = mcpService.Delete(mcp.ResourceId)

	require.NoError(t, err)
	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", mcp.ResourceId, 0)
	requireTableCount(t, db, &apmodel.AgentDependency{}, "target_resource_id = ?", mcp.ResourceId, 0)
}

func TestDeleteTypedResourceRejectsSkillAndKnowledgeReferencedByLatestAgentVersion(t *testing.T) {
	db := newResourceDeleteTestDB(t)
	skill, err := NewSkillService(db).Create(SkillCreateInput{
		DisplayName: "Skill A",
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	knowledge, err := NewKnowledgeService(db).Create(KnowledgeCreateInput{
		DisplayName:         "Knowledge A",
		ExternalKnowledgeId: "kb-001",
		OwnerUserId:         100,
	})
	require.NoError(t, err)
	agent, err := NewAgentService(db).Create(AgentCreateInput{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "Agent A",
		Categories:  []string{apmodel.AgentCategoryGeneral},
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	requireLatestAgentDependency(t, db, agent.ResourceId, "1.0.0", apmodel.AgentDependencyTypeSkill, skill.ResourceId)
	requireLatestAgentDependency(t, db, agent.ResourceId, "1.0.0", apmodel.AgentDependencyTypeKnowledge, knowledge.ResourceId)

	err = NewSkillService(db).Delete(skill.ResourceId)
	require.ErrorIs(t, err, ErrResourceInUse)
	err = NewKnowledgeService(db).Delete(knowledge.ResourceId)
	require.ErrorIs(t, err, ErrResourceInUse)
	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", skill.ResourceId, 1)
	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", knowledge.ResourceId, 1)
}

func TestDeleteTypedResourcePhysicallyDeletesSkillAndKnowledgeDetails(t *testing.T) {
	db := newResourceDeleteTestDB(t)
	skill, err := NewSkillService(db).Create(SkillCreateInput{
		DisplayName: "Skill A",
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	require.NoError(t, db.Create(&apmodel.SkillDef{
		ResourceId: skill.ResourceId,
		FileName:   "skill.zip",
		FilePath:   "oss://bucket/skill.zip",
		Sha256:     "0123456789abcdef",
		SizeBytes:  10,
	}).Error)
	knowledge, err := NewKnowledgeService(db).Create(KnowledgeCreateInput{
		DisplayName:         "Knowledge A",
		ExternalKnowledgeId: "kb-001",
		OwnerUserId:         100,
	})
	require.NoError(t, err)

	require.NoError(t, NewSkillService(db).Delete(skill.ResourceId))
	require.NoError(t, NewKnowledgeService(db).Delete(knowledge.ResourceId))

	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", skill.ResourceId, 0)
	requireTableCount(t, db, &apmodel.SkillDef{}, "resource_id = ?", skill.ResourceId, 0)
	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", knowledge.ResourceId, 0)
	requireTableCount(t, db, &apmodel.KnowledgeDef{}, "resource_id = ?", knowledge.ResourceId, 0)
}

func TestDeleteTypedResourcePhysicallyDeletesAgentGraph(t *testing.T) {
	db := newResourceDeleteTestDB(t)
	mcp, err := NewMcpService(db).Create(McpCreateInput{
		DisplayName: "MCP A",
		Config:      json.RawMessage(`{"mcpServers":{"local":{"command":"node"}}}`),
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	agent, err := NewAgentService(db).Create(AgentCreateInput{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "Agent A",
		Categories:  []string{apmodel.AgentCategoryGeneral},
		McpIds:      []string{mcp.ResourceId},
		OwnerUserId: 100,
	})
	require.NoError(t, err)
	publishedAt := time.Now().UTC()
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      agent.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "agent-platform/v1",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
		PublishedAt:     &publishedAt,
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceGrant{
		ResourceId:      agent.ResourceId,
		ResourceVersion: "1.0.0",
		SubjectType:     apmodel.GrantSubjectTypeUser,
		SubjectId:       "100",
		GrantedBy:       100,
	}).Error)

	err = NewAgentService(db).Delete(agent.ResourceId)

	require.NoError(t, err)
	requireTableCount(t, db, &apmodel.Resource{}, "resource_id = ?", agent.ResourceId, 0)
	requireTableCount(t, db, &apmodel.AgentDef{}, "resource_id = ?", agent.ResourceId, 0)
	requireTableCount(t, db, &apmodel.AgentDependency{}, "agent_resource_id = ?", agent.ResourceId, 0)
	requireTableCount(t, db, &apmodel.ResourceVersion{}, "resource_id = ?", agent.ResourceId, 0)
	requireTableCount(t, db, &apmodel.ResourceGrant{}, "resource_id = ?", agent.ResourceId, 0)
}

func requireLatestAgentDependency(t *testing.T, db *gorm.DB, agentResourceID string, version string, targetType string, targetResourceID string) {
	t.Helper()

	require.NoError(t, db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", agentResourceID).
		Update("latest_version", version).Error)
	var count int64
	require.NoError(t, db.Model(&apmodel.ResourceVersion{}).
		Where("resource_id = ? AND version = ?", agentResourceID, version).
		Count(&count).Error)
	if count == 0 {
		publishedAt := time.Now().UTC()
		require.NoError(t, db.Create(&apmodel.ResourceVersion{
			ResourceId:      agentResourceID,
			Version:         version,
			ContractVersion: "agent-platform/v1",
			Status:          apmodel.ResourceStatusPublished,
			CreatedBy:       100,
			PublishedAt:     &publishedAt,
		}).Error)
	}
	require.NoError(t, db.Create(&apmodel.AgentDependency{
		AgentResourceId:  agentResourceID,
		ResourceVersion:  version,
		TargetType:       targetType,
		TargetResourceId: targetResourceID,
	}).Error)
}

func requireTableCount(t *testing.T, db *gorm.DB, model any, query string, arg any, expected int64) {
	t.Helper()

	var count int64
	require.NoError(t, db.Model(model).Where(query, arg).Count(&count).Error)
	require.Equal(t, expected, count)
}
