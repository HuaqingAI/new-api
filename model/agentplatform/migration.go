package agentplatform

import (
	"fmt"

	"gorm.io/gorm"
)

type removedColumnMigration struct {
	model   any
	table   string
	columns []string
}

var removedAgentPlatformTables = []any{
	&RefreshToken{},
	&AuthorizationGrant{},
	&Exposure{},
	&Client{},
}

var removedAgentPlatformColumns = []removedColumnMigration{
	{
		model: &AgentDef{},
		table: AgentDef{}.TableName(),
		columns: []string{
			"manifest_json",
			"dependencies_json",
			"prompt_metadata_json",
			"compatibility_meta_json",
		},
	},
	{
		model: &KnowledgeDef{},
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
		model: &SkillDef{},
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
		model: &ResourceVersion{},
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
		&AdminAction{},
		&ClientPackage{},
	); err != nil {
		return err
	}
	if err := dropRemovedColumns(db, removedAgentPlatformColumns); err != nil {
		return err
	}
	return dropRemovedTables(db, removedAgentPlatformTables)
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}

func dropRemovedColumns(db *gorm.DB, migrations []removedColumnMigration) error {
	for _, migration := range migrations {
		if !db.Migrator().HasTable(migration.model) {
			continue
		}
		for _, column := range migration.columns {
			if !db.Migrator().HasColumn(migration.model, column) {
				continue
			}
			if err := db.Migrator().DropColumn(migration.model, column); err != nil {
				return fmt.Errorf("drop removed agent platform column %s.%s: %w", migration.table, column, err)
			}
		}
	}
	return nil
}

func dropRemovedTables(db *gorm.DB, tables []any) error {
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			continue
		}
		if err := db.Migrator().DropTable(table); err != nil {
			return fmt.Errorf("drop removed agent platform table: %w", err)
		}
	}
	return nil
}
