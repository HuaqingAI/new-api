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
	)
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}
