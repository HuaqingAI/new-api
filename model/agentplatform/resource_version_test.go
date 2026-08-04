package agentplatform

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
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

func TestResourceVersionResourceRelationUsesBelongsToDirection(t *testing.T) {
	parsed, err := schema.Parse(&ResourceVersion{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)

	relation, ok := parsed.Relationships.Relations["Resource"]
	require.True(t, ok)
	require.Equal(t, schema.BelongsTo, relation.Type)
	require.Len(t, relation.References, 1)
	require.Equal(t, "agent_platform_resources", relation.References[0].PrimaryKey.Schema.Table)
	require.Equal(t, "resource_id", relation.References[0].PrimaryKey.DBName)
	require.Equal(t, "agent_platform_resource_versions", relation.References[0].ForeignKey.Schema.Table)
	require.Equal(t, "resource_id", relation.References[0].ForeignKey.DBName)
	require.False(t, relation.References[0].OwnPrimaryKey)
}

func TestResourceVersionAndDefinitionsPersistRetainedFields(t *testing.T) {
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
	}
	knowledgeVersion := ResourceVersion{
		ResourceId:      knowledge.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          ResourceStatusDraft,
		CreatedBy:       1,
	}
	agentVersion := ResourceVersion{
		ResourceId:      agent.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          ResourceStatusDraft,
		CreatedBy:       1,
	}
	require.NoError(t, db.Create(&skillVersion).Error)
	require.NoError(t, db.Create(&knowledgeVersion).Error)
	require.NoError(t, db.Create(&agentVersion).Error)

	require.NoError(t, db.Create(&SkillDef{
		ResourceId: skill.ResourceId,
		FileName:   "skill.zip",
		FilePath:   "oss://bucket/skill.zip",
		Sha256:     strings.Repeat("a", 64),
		SizeBytes:  10,
	}).Error)
	require.NoError(t, db.Create(&KnowledgeDef{
		ResourceId:          knowledge.ResourceId,
		ExternalKnowledgeId: "kb_demo",
	}).Error)
	require.NoError(t, db.Create(&AgentDef{
		ResourceId:      agent.ResourceId,
		ResourceVersion: "1.0.0",
		CliType:         AgentCliTypeOpenCode,
		Name:            "Agent",
	}).Error)

	var skillDetail SkillDef
	var knowledgeDetail KnowledgeDef
	var agentDetail AgentDef
	require.NoError(t, db.Where("resource_id = ?", skill.ResourceId).First(&skillDetail).Error)
	require.NoError(t, db.Where("resource_id = ?", knowledge.ResourceId).First(&knowledgeDetail).Error)
	require.NoError(t, db.Where("resource_id = ? AND resource_version = ?", agent.ResourceId, "1.0.0").First(&agentDetail).Error)
	require.Equal(t, "skill.zip", skillDetail.FileName)
	require.Equal(t, "kb_demo", knowledgeDetail.ExternalKnowledgeId)
	require.Equal(t, "Agent", agentDetail.Name)
}
