package agentplatform

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(
		&Resource{},
		&ResourceVersion{},
		&SkillDef{},
		&KnowledgeDef{},
		&AgentDef{},
		&Exposure{},
		&AdminAction{},
	)
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}
