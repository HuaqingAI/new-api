package enterprise

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Department{},
		&UserDepartment{},
	)
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}
