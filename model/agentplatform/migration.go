package agentplatform

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(
		&ResourceVersion{},
		&Resource{},
		&SkillDef{},
		&KnowledgeDef{},
		&AgentDef{},
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
