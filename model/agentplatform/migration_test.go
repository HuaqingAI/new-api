package agentplatform

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
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

func TestMigrateBackfillsAgentCategories(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyAgentDef{}))
	require.NoError(t, db.Create(&legacyAgentDef{
		ResourceId:      "res_agent",
		ResourceVersion: "draft",
	}).Error)

	require.NoError(t, Migrate(db))

	var agent AgentDef
	require.NoError(t, db.Where("resource_id = ? AND resource_version = ?", "res_agent", "draft").First(&agent).Error)
	require.Equal(t, []string{AgentCategoryGeneral}, agent.Categories())
}

func TestMigrateBackfillsRecommendedPromptsAndRemovesLegacyBlocks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyAgentDef{}, &legacyResource{}))
	require.NoError(t, db.Create(&legacyResource{
		ResourceId:   "res_legacy",
		ResourceType: ResourceTypeAgent,
		DisplayName:  "Legacy Agent",
		Description:  "资源说明\n<open-remark>\n资源问题\n</open-remark>",
		OwnerUserId:  1,
		Status:       ResourceStatusDraft,
	}).Error)
	require.NoError(t, db.Create(&legacyAgentDef{
		ResourceId:      "res_legacy",
		ResourceVersion: ResourceStatusDraft,
		Description:     "负责售后。\n<open-remark>\n问题一\n\n问题二\n问题一\n</open-remark>\n继续服务",
	}).Error)
	require.NoError(t, db.Create(&legacyAgentDef{
		ResourceId:      "res_unclosed",
		ResourceVersion: ResourceStatusDraft,
		Description:     "保留说明\n<open-remark>\n未闭合问题",
	}).Error)

	require.NoError(t, Migrate(db))

	var agent AgentDef
	require.NoError(t, db.Where("resource_id = ?", "res_legacy").First(&agent).Error)
	require.Equal(t, []string{"问题一", "问题二", "资源问题"}, agent.RecommendedPrompts())
	require.Equal(t, "负责售后。\n\n继续服务", agent.Description)

	var unclosed AgentDef
	require.NoError(t, db.Where("resource_id = ?", "res_unclosed").First(&unclosed).Error)
	require.Empty(t, unclosed.RecommendedPrompts())
	require.Equal(t, "保留说明\n<open-remark>\n未闭合问题", unclosed.Description)

	var resource Resource
	require.NoError(t, db.Where("resource_id = ?", "res_legacy").First(&resource).Error)
	require.Equal(t, "资源说明", resource.Description)
}

func TestMigrateClientPackageRolloutSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	testMigrateClientPackageRollout(t, db)
}

func TestMigrateClientPackageRolloutConfiguredDatabases(t *testing.T) {
	tests := []struct {
		name      string
		env       string
		dialector func(string) gorm.Dialector
	}{
		{name: "mysql", env: "TEST_MYSQL_DSN", dialector: func(dsn string) gorm.Dialector { return mysql.Open(dsn) }},
		{name: "postgres", env: "TEST_POSTGRES_DSN", dialector: func(dsn string) gorm.Dialector {
			return postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dsn := strings.TrimSpace(os.Getenv(test.env))
			if dsn == "" {
				t.Skip(test.env + " is not configured")
			}
			db, err := gorm.Open(test.dialector(dsn), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			testMigrateClientPackageRollout(t, db)
		})
	}
}

func testMigrateClientPackageRollout(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Migrator().DropTable(&ClientPackageScope{}, &ClientPackage{}))

	t.Run("fresh", func(t *testing.T) {
		for range 2 {
			require.NoError(t, Migrate(db))
		}
		require.True(t, db.Migrator().HasColumn(&ClientPackage{}, "rollout_mode"))
		require.True(t, db.Migrator().HasTable(&ClientPackageScope{}))

		pkg := ClientPackage{
			Platform:    ClientPackagePlatformWindowsX64,
			Version:     "1.0.0",
			Status:      ClientPackageStatusPublished,
			FileName:    "AionUi-1.0.0.exe",
			FilePath:    "client-packages/1.0.0/AionUi-1.0.0.exe",
			FileSha256:  strings.Repeat("a", 64),
			FileSha512:  strings.Repeat("b", 128),
			FileSize:    1,
			ContentType: "application/octet-stream",
			CreatedBy:   1,
		}
		require.NoError(t, db.Create(&pkg).Error)
		assert.Equal(t, ClientPackageRolloutModeGlobal, pkg.RolloutMode)
	})

	require.NoError(t, db.Migrator().DropTable(&ClientPackageScope{}, &ClientPackage{}))

	t.Run("upgrade", func(t *testing.T) {
		require.NoError(t, db.AutoMigrate(&legacyClientPackage{}))
		legacy := legacyClientPackage{
			Platform:    ClientPackagePlatformWindowsX64,
			Version:     fmt.Sprintf("legacy-%d", time.Now().UnixNano()),
			Status:      ClientPackageStatusPublished,
			FileName:    "AionUi-legacy.exe",
			FilePath:    "client-packages/legacy/AionUi-legacy.exe",
			FileSha256:  strings.Repeat("c", 64),
			FileSha512:  strings.Repeat("d", 128),
			FileSize:    1,
			ContentType: "application/octet-stream",
			CreatedBy:   1,
		}
		require.NoError(t, db.Create(&legacy).Error)

		for range 2 {
			require.NoError(t, Migrate(db))
		}

		var migrated ClientPackage
		require.NoError(t, db.First(&migrated, legacy.Id).Error)
		assert.Equal(t, legacy.Version, migrated.Version)
		assert.Equal(t, legacy.FilePath, migrated.FilePath)
		assert.Equal(t, ClientPackageRolloutModeGlobal, migrated.RolloutMode)

		scope := ClientPackageScope{
			ClientPackageId: migrated.Id,
			SubjectType:     GrantSubjectTypeUser,
			SubjectId:       "1",
			CreatedBy:       1,
		}
		require.NoError(t, db.Create(&scope).Error)
		assert.Error(t, db.Create(&ClientPackageScope{
			ClientPackageId: migrated.Id,
			SubjectType:     GrantSubjectTypeUser,
			SubjectId:       "1",
			CreatedBy:       1,
		}).Error)
	})
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
	Description           string `gorm:"type:text"`
	ManifestJSON          string `gorm:"type:text"`
	DependenciesJSON      string `gorm:"type:text"`
	PromptMetadataJSON    string `gorm:"type:text"`
	CompatibilityMetaJSON string `gorm:"type:text"`
}

type legacyClientPackage struct {
	Id                     int    `gorm:"primaryKey"`
	Platform               string `gorm:"type:varchar(32);not null;index:idx_agent_platform_client_package_platform_status"`
	Version                string `gorm:"type:varchar(32);not null;index:idx_agent_platform_client_package_platform_version"`
	Status                 string `gorm:"type:varchar(32);not null;index:idx_agent_platform_client_package_platform_status"`
	FileName               string `gorm:"type:varchar(255);not null"`
	FilePath               string `gorm:"type:text;not null"`
	FileSha256             string `gorm:"type:char(64);not null"`
	FileSha512             string `gorm:"type:varchar(128);not null"`
	FileSize               int64  `gorm:"type:bigint;not null"`
	ContentType            string `gorm:"type:varchar(128);not null"`
	UpdateFileName         string `gorm:"type:varchar(255);not null;default:''"`
	UpdateFilePath         string `gorm:"type:text"`
	UpdateFileSha256       string `gorm:"type:char(64);not null;default:''"`
	UpdateFileSha512       string `gorm:"type:varchar(128);not null;default:''"`
	UpdateFileSize         int64  `gorm:"type:bigint;not null;default:0"`
	UpdateContentType      string `gorm:"type:varchar(128);not null;default:''"`
	UpdateMetadataFileName string `gorm:"type:varchar(255);not null;default:''"`
	UpdateMetadataFilePath string `gorm:"type:text"`
	UpdateMetadataSha256   string `gorm:"type:char(64);not null;default:''"`
	ReleaseNote            string `gorm:"type:text"`
	CreatedBy              int    `gorm:"type:int;not null;index:idx_agent_platform_client_package_created_by"`
	PublishedAt            *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (legacyClientPackage) TableName() string {
	return ClientPackage{}.TableName()
}

func (legacyAgentDef) TableName() string {
	return AgentDef{}.TableName()
}

type legacyResource struct {
	Id            int    `gorm:"primaryKey"`
	ResourceId    string `gorm:"type:varchar(40);uniqueIndex:idx_ap_resource_id;not null"`
	ResourceType  string `gorm:"type:varchar(16);index:idx_ap_resource_type;not null"`
	DisplayName   string `gorm:"type:varchar(255);not null"`
	Description   string `gorm:"type:text"`
	Avatar        string `gorm:"type:text"`
	OwnerUserId   int    `gorm:"index:idx_ap_resource_owner;not null"`
	Status        string `gorm:"type:varchar(16);index:idx_ap_resource_status;not null"`
	LatestVersion string `gorm:"type:varchar(64);not null"`
	TenantId      int    `gorm:"index:idx_ap_resource_tenant;not null;default:0"`
}

func (legacyResource) TableName() string {
	return Resource{}.TableName()
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
