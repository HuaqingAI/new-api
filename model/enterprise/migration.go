package enterprise

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Department{},
		&DepartmentBudget{},
		&QuotaAllocation{},
		&UserDepartment{},
		&DepartmentRole{},
		&AdminAction{},
		&DingTalkConfig{},
		&DingTalkIdentity{},
		&DingTalkSyncTask{},
		&DingTalkSyncLog{},
		&DingTalkSyncConflict{},
	)
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}
