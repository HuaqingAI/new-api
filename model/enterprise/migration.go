package enterprise

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := ensureDepartmentRoleOwnerColumnsAndIndexes(db); err != nil {
		return err
	}
	if err := ensureAlertEventDepartmentTokensColumn(db); err != nil {
		return err
	}
	if err := ensureAlertDeliveryColumns(db); err != nil {
		return err
	}
	if err := ensureQuotaAllocationLifecycleColumns(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&Department{},
		&DepartmentBudget{},
		&BudgetDelegation{},
		&QuotaAllocation{},
		&UserDepartment{},
		&DepartmentRole{},
		&AdminAction{},
		&AlertEvent{},
		&AlertRule{},
		&AlertDelivery{},
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

func ensureDepartmentRoleOwnerColumnsAndIndexes(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&DepartmentRole{}) {
		return nil
	}
	for _, column := range []string{"source", "effect", "external_source"} {
		if db.Migrator().HasColumn(&DepartmentRole{}, column) {
			continue
		}
		if err := db.Migrator().AddColumn(&DepartmentRole{}, column); err != nil {
			return err
		}
	}
	if err := db.Model(&DepartmentRole{}).
		Where("source = '' OR source IS NULL").
		Updates(map[string]any{
			"source":          "manual_grant",
			"effect":          "allow",
			"external_source": "",
		}).Error; err != nil {
		return err
	}
	if db.Migrator().HasIndex(&DepartmentRole{}, "uq_ent_dept_roles_role") {
		if err := db.Migrator().DropIndex(&DepartmentRole{}, "uq_ent_dept_roles_role"); err != nil {
			return err
		}
	}
	return nil
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

func ensureAlertDeliveryColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&AlertDelivery{}) {
		return nil
	}
	columns := []string{
		"tenant_id",
		"event_id",
		"rule_id",
		"channel_type",
		"status",
		"attempt_count",
		"max_attempts",
		"next_retry_at",
		"last_attempt_at",
		"sent_at",
		"final_failed_at",
		"error_reason",
		"dedupe_key",
		"trace_payload",
		"trigger_source",
		"manual_parent_id",
		"created_at",
		"updated_at",
	}
	for _, column := range columns {
		if db.Migrator().HasColumn(&AlertDelivery{}, column) {
			continue
		}
		if err := db.Migrator().AddColumn(&AlertDelivery{}, column); err != nil {
			return err
		}
	}
	return nil
}

func ensureQuotaAllocationLifecycleColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&QuotaAllocation{}) {
		return nil
	}
	columns := []string{
		"superseded_by_id",
		"supersedes_allocation_id",
		"revoke_reason",
		"reclaimed_quota",
		"processed_source",
	}
	for _, column := range columns {
		if db.Migrator().HasColumn(&QuotaAllocation{}, column) {
			continue
		}
		if err := db.Migrator().AddColumn(&QuotaAllocation{}, column); err != nil {
			return err
		}
	}
	return nil
}
