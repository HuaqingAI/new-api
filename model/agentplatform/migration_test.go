package agentplatform

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrateDropsRemovedColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	createLegacyAgentPlatformTables(t, db)

	for _, migration := range removedAgentPlatformColumns {
		for _, column := range migration.columns {
			require.True(t, db.Migrator().HasColumn(migration.model, column), column)
		}
	}
	require.True(t, db.Migrator().HasColumn(&AgentDef{}, "resource_version"))

	require.NoError(t, Migrate(db))

	for _, migration := range removedAgentPlatformColumns {
		for _, column := range migration.columns {
			require.False(t, db.Migrator().HasColumn(migration.model, column), column)
		}
	}
	require.True(t, db.Migrator().HasColumn(&AgentDef{}, "resource_version"))
}

func TestMigrateDropsRemovedTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(removedAgentPlatformTables...))

	for _, table := range removedAgentPlatformTables {
		require.True(t, db.Migrator().HasTable(table))
	}

	require.NoError(t, Migrate(db))

	for _, table := range removedAgentPlatformTables {
		require.False(t, db.Migrator().HasTable(table))
	}
}

func createLegacyAgentPlatformTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&legacyAgentDef{},
		&legacyKnowledgeDef{},
		&legacySkillDef{},
		&legacyResourceVersion{},
	))
}

type legacyAgentDef struct {
	Id                    int    `gorm:"primaryKey"`
	ResourceId            string `gorm:"type:varchar(40);uniqueIndex:idx_ap_agent_def_version;not null"`
	ResourceVersion       string `gorm:"type:varchar(64);uniqueIndex:idx_ap_agent_def_version;not null"`
	ManifestJSON          string `gorm:"type:text"`
	DependenciesJSON      string `gorm:"type:text"`
	PromptMetadataJSON    string `gorm:"type:text"`
	CompatibilityMetaJSON string `gorm:"type:text"`
}

func (legacyAgentDef) TableName() string {
	return AgentDef{}.TableName()
}

type legacyKnowledgeDef struct {
	Id                       int    `gorm:"primaryKey"`
	ResourceId               string `gorm:"type:varchar(40);uniqueIndex:idx_ap_knowledge_def_resource;not null"`
	ExternalKnowledgeId      string `gorm:"type:varchar(128);index"`
	ProviderConfigJSON       string `gorm:"type:text"`
	QuerySchemaJSON          string `gorm:"type:text"`
	CitationSchemaJSON       string `gorm:"type:text"`
	FreshnessRulesJSON       string `gorm:"type:text"`
	ProviderCapabilitiesJSON string `gorm:"type:text"`
	ResourceVersion          string `gorm:"type:varchar(64)"`
	KnowledgeMode            string `gorm:"type:varchar(32)"`
	ProviderType             string `gorm:"type:varchar(32)"`
	ProviderAdapterKey       string `gorm:"type:varchar(128)"`
}

func (legacyKnowledgeDef) TableName() string {
	return KnowledgeDef{}.TableName()
}

type legacySkillDef struct {
	Id                int    `gorm:"primaryKey"`
	ResourceId        string `gorm:"type:varchar(40);uniqueIndex:idx_ap_skill_def_resource;not null"`
	FileName          string `gorm:"type:varchar(255)"`
	FilePath          string `gorm:"type:text"`
	Sha256            string `gorm:"type:varchar(64)"`
	SizeBytes         int64  `gorm:"not null;default:0"`
	ResourceVersion   string `gorm:"type:varchar(64)"`
	InvokeSchemaJSON  string `gorm:"type:text"`
	OutputSchemaJSON  string `gorm:"type:text"`
	InvokeMode        string `gorm:"type:varchar(32)"`
	TimeoutSeconds    int    `gorm:"not null;default:0"`
	BindingConfigJSON string `gorm:"type:text"`
}

func (legacySkillDef) TableName() string {
	return SkillDef{}.TableName()
}

type legacyResourceVersion struct {
	Id              int    `gorm:"primaryKey"`
	ResourceId      string `gorm:"type:varchar(40);uniqueIndex:idx_ap_resource_version;not null"`
	Version         string `gorm:"column:version;type:varchar(64);uniqueIndex:idx_ap_resource_version;not null"`
	ContractVersion string `gorm:"type:varchar(64);not null"`
	SchemaJSON      string `gorm:"type:text"`
	DetailJSON      string `gorm:"type:text"`
}

func (legacyResourceVersion) TableName() string {
	return ResourceVersion{}.TableName()
}
