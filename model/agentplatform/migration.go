package agentplatform

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
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
		&ClientPackageScope{},
	); err != nil {
		return err
	}
	if err := backfillAgentDefCategories(db); err != nil {
		return err
	}
	if err := backfillAgentDefRecommendedPrompts(db); err != nil {
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

func backfillAgentDefCategories(db *gorm.DB) error {
	if !db.Migrator().HasTable(&AgentDef{}) || !db.Migrator().HasColumn(&AgentDef{}, "categories_json") {
		return nil
	}
	data, err := common.Marshal(DefaultAgentCategories())
	if err != nil {
		return err
	}
	return db.Model(&AgentDef{}).
		Where("categories_json = ? OR categories_json IS NULL", "").
		Update("categories_json", string(data)).Error
}

var legacyOpenRemarkBlockPattern = regexp.MustCompile(`(?is)<open-remark>(.*?)</open-remark>`)

func backfillAgentDefRecommendedPrompts(db *gorm.DB) error {
	if !db.Migrator().HasTable(&AgentDef{}) || !db.Migrator().HasColumn(&AgentDef{}, "recommended_prompts_json") {
		return nil
	}

	resourceDescriptions := map[string]string{}
	if db.Migrator().HasTable(&Resource{}) {
		var resources []Resource
		if err := db.Where("resource_type = ?", ResourceTypeAgent).Find(&resources).Error; err != nil {
			return err
		}
		for _, resource := range resources {
			resourceDescriptions[resource.ResourceId] = resource.Description
		}
	}

	var definitions []AgentDef
	if err := db.Find(&definitions).Error; err != nil {
		return err
	}
	for _, definition := range definitions {
		legacyDescription := definition.Description
		legacyPrompts := extractLegacyRecommendedPrompts(legacyDescription)
		if definition.ResourceVersion == ResourceStatusDraft {
			legacyPrompts = append(legacyPrompts, extractLegacyRecommendedPrompts(resourceDescriptions[definition.ResourceId])...)
		}
		cleanedDescription := removeLegacyRecommendedPromptBlock(legacyDescription)
		if cleanedDescription == legacyDescription && len(legacyPrompts) == 0 {
			continue
		}

		updates := map[string]any{}
		if cleanedDescription != legacyDescription {
			updates["description"] = cleanedDescription
		}
		if len(legacyPrompts) > 0 {
			prompts := append(definition.RecommendedPrompts(), legacyPrompts...)
			data, err := common.Marshal(NormalizeAgentRecommendedPrompts(prompts))
			if err != nil {
				return err
			}
			updates["recommended_prompts_json"] = string(data)
		}
		if len(updates) > 0 {
			if err := db.Model(&AgentDef{}).Where("id = ?", definition.Id).Updates(updates).Error; err != nil {
				return err
			}
		}
	}

	if len(resourceDescriptions) == 0 {
		return nil
	}
	for resourceID, description := range resourceDescriptions {
		cleanedDescription := removeLegacyRecommendedPromptBlock(description)
		if cleanedDescription == description {
			continue
		}
		if err := db.Model(&Resource{}).Where("resource_id = ?", resourceID).Update("description", cleanedDescription).Error; err != nil {
			return err
		}
	}
	return nil
}

func extractLegacyRecommendedPrompts(description string) []string {
	matches := legacyOpenRemarkBlockPattern.FindAllStringSubmatch(description, -1)
	if len(matches) == 0 {
		return []string{}
	}
	prompts := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			prompts = append(prompts, strings.Split(match[1], "\n")...)
		}
	}
	return NormalizeAgentRecommendedPrompts(prompts)
}

func removeLegacyRecommendedPromptBlock(description string) string {
	if !legacyOpenRemarkBlockPattern.MatchString(description) {
		return description
	}
	return strings.TrimSpace(legacyOpenRemarkBlockPattern.ReplaceAllString(description, ""))
}
