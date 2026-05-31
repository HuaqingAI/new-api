package agentplatform

import (
	"encoding/json"
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newResourceVersionServiceForTest(t *testing.T) (*ResourceVersionService, *gorm.DB, apmodel.Resource, apmodel.Resource, apmodel.Resource) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	skill := apmodel.Resource{ResourceType: apmodel.ResourceTypeSkill, DisplayName: "Skill", OwnerUserId: 100}
	knowledge := apmodel.Resource{ResourceType: apmodel.ResourceTypeKnowledge, DisplayName: "Knowledge", OwnerUserId: 100}
	agent := apmodel.Resource{ResourceType: apmodel.ResourceTypeAgent, DisplayName: "Agent", OwnerUserId: 100}
	require.NoError(t, db.Create(&skill).Error)
	require.NoError(t, db.Create(&knowledge).Error)
	require.NoError(t, db.Create(&agent).Error)

	return NewResourceVersionService(db), db, skill, knowledge, agent
}

func TestResourceVersionServiceCreatesTypedDetailsForAllResourceTypes(t *testing.T) {
	svc, _, skill, knowledge, agent := newResourceVersionServiceForTest(t)

	skillVersion, err := svc.Create(skill.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Skill: &SkillDetailInput{
			InvokeSchema:   json.RawMessage(`{"type":"object"}`),
			OutputSchema:   json.RawMessage(`{"type":"object"}`),
			InvokeMode:     "sync",
			TimeoutSeconds: intPtr(30),
			BindingConfig:  json.RawMessage(`{"provider":"demo"}`),
		},
		CreatedBy: 100,
	})
	require.NoError(t, err)
	require.Equal(t, skill.ResourceId, skillVersion.ResourceId)
	require.NotNil(t, skillVersion.Skill)

	knowledgeVersion, err := svc.Create(knowledge.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Knowledge: &KnowledgeDetailInput{
			KnowledgeMode:        "retrieval",
			ProviderType:         "http",
			ProviderAdapterKey:   "http_retrieval",
			ProviderConfig:       json.RawMessage(`{"endpoint":"https://example.com"}`),
			QuerySchema:          json.RawMessage(`{"type":"object"}`),
			CitationSchema:       json.RawMessage(`{"type":"array"}`),
			FreshnessRules:       json.RawMessage(`{"ttl":300}`),
			ProviderCapabilities: json.RawMessage(`{"freshness":true}`),
		},
		CreatedBy: 100,
	})
	require.NoError(t, err)
	require.NotNil(t, knowledgeVersion.Knowledge)
	require.Equal(t, "retrieval", knowledgeVersion.Knowledge.KnowledgeMode)

	agentVersion, err := svc.Create(agent.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Agent: &AgentDetailInput{
			Manifest:          json.RawMessage(`{"name":"agent"}`),
			Dependencies:      json.RawMessage(`["skill","knowledge"]`),
			PromptMetadata:    json.RawMessage(`{"template":"default"}`),
			CompatibilityMeta: json.RawMessage(`{"clients":["demo"]}`),
		},
		CreatedBy: 100,
	})
	require.NoError(t, err)
	require.NotNil(t, agentVersion.Agent)
}

func TestResourceVersionServiceRejectsMismatchedOrMissingTypedDetails(t *testing.T) {
	svc, _, skill, _, _ := newResourceVersionServiceForTest(t)

	_, err := svc.Create(skill.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Knowledge: &KnowledgeDetailInput{
			KnowledgeMode:      "retrieval",
			ProviderType:       "http",
			ProviderAdapterKey: "http_retrieval",
		},
		CreatedBy: 100,
	})
	require.ErrorIs(t, err, ErrInvalidResourceVersionInput)

	_, err = svc.Create(skill.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.1",
		ContractVersion: "2026-06",
		CreatedBy:       100,
	})
	require.ErrorIs(t, err, ErrInvalidResourceVersionInput)
}

func TestResourceVersionServiceRejectsInvalidSkillContract(t *testing.T) {
	svc, _, skill, _, _ := newResourceVersionServiceForTest(t)

	_, err := svc.Create(skill.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Skill: &SkillDetailInput{
			InvokeSchema:   json.RawMessage(`{"type":"object"}`),
			OutputSchema:   json.RawMessage(`{"type":"object"}`),
			InvokeMode:     "stream",
			TimeoutSeconds: intPtr(30),
			BindingConfig:  json.RawMessage(`{"provider":"demo"}`),
		},
		CreatedBy: 100,
	})
	require.ErrorIs(t, err, ErrSkillContractInvalid)

	_, err = svc.Create(skill.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.1",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Skill: &SkillDetailInput{
			InvokeSchema:   json.RawMessage(`{"type":"object"}`),
			OutputSchema:   json.RawMessage(`{"type":"object"}`),
			InvokeMode:     "sync",
			TimeoutSeconds: intPtr(0),
			BindingConfig:  json.RawMessage(`{"provider":"demo"}`),
		},
		CreatedBy: 100,
	})
	require.ErrorIs(t, err, ErrSkillContractInvalid)

	_, err = svc.Create(skill.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.2",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Skill: &SkillDetailInput{
			OutputSchema:   json.RawMessage(`{"type":"object"}`),
			InvokeMode:     "sync",
			TimeoutSeconds: intPtr(30),
			BindingConfig:  json.RawMessage(`{"provider":"demo"}`),
		},
		CreatedBy: 100,
	})
	require.ErrorIs(t, err, ErrSkillContractInvalid)
}

func intPtr(v int) *int {
	return &v
}
