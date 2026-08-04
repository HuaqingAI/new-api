package agentplatform

import (
	"fmt"

	"gorm.io/gorm"
)

type removedColumnMigration struct {
	table   string
	columns []string
}

var removedAgentPlatformColumns = []removedColumnMigration{
	{
		table: AgentDef{}.TableName(),
		columns: []string{
			"manifest_json",
			"dependencies_json",
			"prompt_metadata_json",
			"compatibility_meta_json",
		},
	},
	{
		table: KnowledgeDef{}.TableName(),
		columns: []string{
			"provider_config_json",
			"query_schema_json",
			"citation_schema_json",
			"freshness_rules_json",
			"provider_capabilities_json",
			"resource_version",
			"knowledge_mode",
			"provider_type",
			"provider_adapter_key",
		},
	},
	{
		table: SkillDef{}.TableName(),
		columns: []string{
			"resource_version",
			"invoke_schema_json",
			"output_schema_json",
			"invoke_mode",
			"timeout_seconds",
			"binding_config_json",
		},
	},
	{
		table: ResourceVersion{}.TableName(),
		columns: []string{
			"schema_json",
			"detail_json",
		},
	},
}

func Migrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.AutoMigrate(
		&ResourceVersion{},
		&Resource{},
		&McpDef{},
		&SkillDef{},
		&KnowledgeDef{},
		&AgentDef{},
		&AgentDependency{},
		&ResourceGrant{},
		&Exposure{},
		&AdminAction{},
		&Client{},
		&AuthorizationGrant{},
		&RefreshToken{},
	); err != nil {
		return err
	}
	return dropRemovedColumns(db, removedAgentPlatformColumns)
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}

func dropRemovedColumns(db *gorm.DB, migrations []removedColumnMigration) error {
	for _, migration := range migrations {
		if !db.Migrator().HasTable(migration.table) {
			continue
		}
		for _, column := range migration.columns {
			if !db.Migrator().HasColumn(migration.table, column) {
				continue
			}
			if err := db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", migration.table, column)).Error; err != nil {
				return fmt.Errorf("drop removed agent platform column %s: %w", column, err)
			}
		}
	}
	return nil
}
