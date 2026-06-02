package agentplatform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResourceVersionAndTypedDetailTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	models := []any{
		ResourceVersion{},
		SkillDef{},
		KnowledgeDef{},
		AgentDef{},
	}

	for _, model := range models {
		modelType := reflect.TypeOf(model)
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			gormTag := field.Tag.Get("gorm")
			if strings.Contains(gormTag, "type:text") {
				require.NotContainsf(t, gormTag, "default:", "%s.%s text field must not declare a DB default", modelType.Name(), field.Name)
			}
		}
	}
}

func TestMigrateCreatesResourceVersionAndTypedDetailTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&ResourceVersion{}))
	require.True(t, db.Migrator().HasTable(&SkillDef{}))
	require.True(t, db.Migrator().HasTable(&KnowledgeDef{}))
	require.True(t, db.Migrator().HasTable(&AgentDef{}))
}

func TestResourceVersionAndTypedDetailsPersistForEachResourceType(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	skill := Resource{ResourceType: ResourceTypeSkill, DisplayName: "Skill", OwnerUserId: 1}
	knowledge := Resource{ResourceType: ResourceTypeKnowledge, DisplayName: "Knowledge", OwnerUserId: 1}
	agent := Resource{ResourceType: ResourceTypeAgent, DisplayName: "Agent", OwnerUserId: 1}
	require.NoError(t, db.Create(&skill).Error)
	require.NoError(t, db.Create(&knowledge).Error)
	require.NoError(t, db.Create(&agent).Error)

	skillVersion := ResourceVersion{
		ResourceId:      skill.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          ResourceStatusDraft,
		CreatedBy:       1,
		SchemaJSON:      "{\"in\":1}",
		DetailJSON:      "{\"kind\":\"skill\"}",
	}
	knowledgeVersion := ResourceVersion{
		ResourceId:      knowledge.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          ResourceStatusDraft,
		CreatedBy:       1,
		SchemaJSON:      "{\"query\":1}",
		DetailJSON:      "{\"kind\":\"knowledge\"}",
	}
	agentVersion := ResourceVersion{
		ResourceId:      agent.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          ResourceStatusDraft,
		CreatedBy:       1,
		SchemaJSON:      "{\"agent\":1}",
		DetailJSON:      "{\"kind\":\"agent\"}",
	}
	require.NoError(t, db.Create(&skillVersion).Error)
	require.NoError(t, db.Create(&knowledgeVersion).Error)
	require.NoError(t, db.Create(&agentVersion).Error)

	require.NoError(t, db.Create(&SkillDef{
		ResourceId:        skill.ResourceId,
		ResourceVersion:   "1.0.0",
		InvokeSchemaJSON:  "{\"type\":\"object\"}",
		OutputSchemaJSON:  "{\"type\":\"object\"}",
		InvokeMode:        "sync",
		TimeoutSeconds:    30,
		BindingConfigJSON: "{\"provider\":\"demo\"}",
	}).Error)
	require.NoError(t, db.Create(&KnowledgeDef{
		ResourceId:               knowledge.ResourceId,
		ResourceVersion:          "1.0.0",
		KnowledgeMode:            "retrieval",
		ProviderType:             "http",
		ProviderAdapterKey:       "http_retrieval",
		ProviderConfigJSON:       "{\"endpoint\":\"https://example.com\"}",
		QuerySchemaJSON:          "{\"type\":\"object\"}",
		CitationSchemaJSON:       "{\"type\":\"array\"}",
		FreshnessRulesJSON:       "{\"ttl\":300}",
		ProviderCapabilitiesJSON: "{\"freshness\":true}",
	}).Error)
	require.NoError(t, db.Create(&AgentDef{
		ResourceId:            agent.ResourceId,
		ResourceVersion:       "1.0.0",
		ManifestJSON:          "{\"name\":\"agent\"}",
		DependenciesJSON:      "[\"skill\",\"knowledge\"]",
		PromptMetadataJSON:    "{\"template\":\"default\"}",
		CompatibilityMetaJSON: "{\"clients\":[\"demo\"]}",
	}).Error)

	var skillDetail SkillDef
	var knowledgeDetail KnowledgeDef
	var agentDetail AgentDef
	require.NoError(t, db.Where("resource_id = ? AND resource_version = ?", skill.ResourceId, "1.0.0").First(&skillDetail).Error)
	require.NoError(t, db.Where("resource_id = ? AND resource_version = ?", knowledge.ResourceId, "1.0.0").First(&knowledgeDetail).Error)
	require.NoError(t, db.Where("resource_id = ? AND resource_version = ?", agent.ResourceId, "1.0.0").First(&agentDetail).Error)
	require.Equal(t, "sync", skillDetail.InvokeMode)
	require.Equal(t, "retrieval", knowledgeDetail.KnowledgeMode)
	require.Contains(t, agentDetail.DependenciesJSON, "skill")
}
