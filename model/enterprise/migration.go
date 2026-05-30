package enterprise

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := ensureAlertEventDepartmentTokensColumn(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&Department{},
		&DepartmentBudget{},
		&QuotaAllocation{},
		&UserDepartment{},
		&DepartmentRole{},
		&AdminAction{},
		&AlertEvent{},
		&AlertRule{},
		&UsageSnapshot{},
		&UsageReportJob{},
		&DingTalkConfig{},
		&DingTalkIdentity{},
		&DingTalkSyncTask{},
		&DingTalkSyncLog{},
		&DingTalkSyncConflict{},
	); err != nil {
		return err
	}
	return backfillAlertEventDepartmentTokens(db)
}

func AutoMigrate(db *gorm.DB) error {
	return Migrate(db)
}

func ensureAlertEventDepartmentTokensColumn(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&AlertEvent{}) || db.Migrator().HasColumn(&AlertEvent{}, "department_tokens") {
		return nil
	}
	return db.Exec("ALTER TABLE enterprise_alert_events ADD COLUMN department_tokens TEXT").Error
}

func backfillAlertEventDepartmentTokens(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&AlertEvent{}) || !db.Migrator().HasColumn(&AlertEvent{}, "department_tokens") {
		return nil
	}

	var batch []AlertEvent
	return db.Where("department_tokens = '' OR department_tokens IS NULL").
		FindInBatches(&batch, 100, func(tx *gorm.DB, batchIndex int) error {
			for _, event := range batch {
				snapshot, err := event.ParsedDepartmentSnapshot()
				if err != nil {
					return err
				}
				if err := tx.Model(&AlertEvent{}).
					Where("id = ?", event.Id).
					UpdateColumn("department_tokens", buildAlertEventDepartmentTokens(snapshot)).
					Error; err != nil {
					return err
				}
			}
			return nil
		}).Error
}
