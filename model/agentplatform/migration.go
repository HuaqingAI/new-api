package agentplatform

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(
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
	)
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}
