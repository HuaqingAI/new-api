package enterprise_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestEnterpriseTextFieldsDoNotDeclareDatabaseDefaults(t *testing.T) {
	models := []any{
		entmodel.Department{},
		entmodel.DepartmentBudget{},
		entmodel.BudgetDelegation{},
		entmodel.UserDepartment{},
		entmodel.DepartmentRole{},
		entmodel.AdminAction{},
		entmodel.DingTalkConfig{},
		entmodel.DingTalkIdentity{},
		entmodel.DingTalkSyncTask{},
		entmodel.DingTalkSyncLog{},
		entmodel.DingTalkSyncConflict{},
	}

	for _, model := range models {
		modelType := reflect.TypeOf(model)
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			gormTag := field.Tag.Get("gorm")
			if strings.Contains(gormTag, "type:text") {
				require.NotContainsf(t, gormTag, "default:", "%s.%s text field must not declare a DB default", modelType.Name(), field.Name)
			}
		}
	}
}

func TestEnterpriseTextFieldApplicationDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	department := entmodel.Department{Name: "Root"}
	require.NoError(t, db.Create(&department).Error)
	require.JSONEq(t, "[]", department.NameHistory)

	action := entmodel.AdminAction{
		ActorId:     100,
		ActionType:  "enterprise.organization.membership.add",
		ObjectType:  "enterprise_department_member",
		ObjectId:    "1:200",
		DiffSummary: "Added department member",
	}
	require.NoError(t, db.Create(&action).Error)
	require.JSONEq(t, "{}", action.Payload)

	config := entmodel.DingTalkConfig{TenantId: 1}
	require.NoError(t, db.Create(&config).Error)
	require.Equal(t, "", config.SyncScope)
}

func TestDingTalkMigrationBackfillsAutoSyncAndConflictSource(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	testDingTalkMigrationBackfill(t, db)
}

func TestDingTalkMigrationBackfillsAutoSyncAndConflictSourceConfiguredDatabases(t *testing.T) {
	tests := []struct {
		name      string
		env       string
		dialector func(string) gorm.Dialector
	}{
		{name: "mysql", env: "TEST_MYSQL_DSN", dialector: func(dsn string) gorm.Dialector { return mysql.Open(dsn) }},
		{name: "postgres", env: "TEST_POSTGRES_DSN", dialector: func(dsn string) gorm.Dialector {
			return postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dsn := strings.TrimSpace(os.Getenv(test.env))
			if dsn == "" {
				t.Skip(test.env + " is not configured")
			}
			db, err := gorm.Open(test.dialector(dsn), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			testDingTalkMigrationBackfill(t, db)
		})
	}
}

func testDingTalkMigrationBackfill(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Migrator().DropTable(&entmodel.DingTalkSyncConflict{}, &entmodel.DingTalkConfig{}))

	for range 2 {
		require.NoError(t, entmodel.Migrate(db))
	}
	require.True(t, db.Migrator().HasColumn(&entmodel.DingTalkConfig{}, "auto_sync_on_login"))
	require.True(t, db.Migrator().HasColumn(&entmodel.DingTalkSyncConflict{}, "trigger_source"))
	require.True(t, db.Migrator().HasIndex(&entmodel.DingTalkSyncConflict{}, "idx_enterprise_dingtalk_sync_conflicts_trigger_source"))

	require.NoError(t, db.Migrator().DropTable(&entmodel.DingTalkSyncConflict{}, &entmodel.DingTalkConfig{}))
	require.NoError(t, db.AutoMigrate(&legacyDingTalkConfig{}, &legacyDingTalkSyncConflict{}))
	legacyConfig := legacyDingTalkConfig{
		TenantId:     7,
		CorpId:       "corp",
		AppKey:       "key",
		AppSecret:    "secret",
		CallbackUrl:  "https://example.com/callback",
		LoginEnabled: true,
		SyncEnabled:  true,
	}
	require.NoError(t, db.Create(&legacyConfig).Error)
	legacyConflict := legacyDingTalkSyncConflict{
		TenantId:       7,
		TaskId:         12,
		ExternalUserId: "staff-1",
		ConflictType:   "email",
		Status:         "pending",
	}
	require.NoError(t, db.Create(&legacyConflict).Error)

	for range 2 {
		require.NoError(t, entmodel.Migrate(db))
	}

	var config entmodel.DingTalkConfig
	require.NoError(t, db.Where("id = ?", legacyConfig.Id).First(&config).Error)
	require.True(t, config.AutoSyncOnLogin)
	var conflict entmodel.DingTalkSyncConflict
	require.NoError(t, db.Where("id = ?", legacyConflict.Id).First(&conflict).Error)
	require.Equal(t, "full_sync", conflict.TriggerSource)
	require.True(t, db.Migrator().HasIndex(&entmodel.DingTalkSyncConflict{}, "idx_enterprise_dingtalk_sync_conflicts_trigger_source"))
}

type legacyDingTalkConfig struct {
	Id           int    `gorm:"primaryKey"`
	TenantId     int    `gorm:"type:int;not null;default:0;uniqueIndex:uq_enterprise_dingtalk_configs_tenant"`
	CorpId       string `gorm:"type:varchar(128);not null;default:''"`
	AppKey       string `gorm:"type:varchar(128);not null;default:''"`
	AppSecret    string `gorm:"type:varchar(512);not null;default:''"`
	CallbackUrl  string `gorm:"type:varchar(1024);not null;default:''"`
	SyncScope    string `gorm:"type:text;not null"`
	LoginEnabled bool   `gorm:"not null;default:false"`
	SyncEnabled  bool   `gorm:"not null;default:false"`
	CreatedAt    int64  `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt    int64  `gorm:"autoUpdateTime;column:updated_at"`
}

func (legacyDingTalkConfig) TableName() string {
	return "enterprise_dingtalk_configs"
}

type legacyDingTalkSyncConflict struct {
	Id              int    `gorm:"primaryKey"`
	TenantId        int    `gorm:"type:int;not null;default:0;index;uniqueIndex:uq_enterprise_dingtalk_sync_conflict"`
	TaskId          int    `gorm:"type:int;not null;default:0;index"`
	ExternalUserId  string `gorm:"type:varchar(128);not null;default:'';index;uniqueIndex:uq_enterprise_dingtalk_sync_conflict"`
	UnionId         string `gorm:"type:varchar(128);not null;default:'';index"`
	Mobile          string `gorm:"type:varchar(64);not null;default:'';index"`
	Email           string `gorm:"type:varchar(255);not null;default:'';index"`
	Name            string `gorm:"type:varchar(255);not null;default:''"`
	ConflictType    string `gorm:"type:varchar(64);not null;default:'';index;uniqueIndex:uq_enterprise_dingtalk_sync_conflict"`
	CandidateUserId int    `gorm:"type:int;not null;default:0;index"`
	Details         string `gorm:"type:varchar(1024);not null;default:''"`
	Status          string `gorm:"type:varchar(32);not null;default:'pending';index"`
	LastTaskId      int    `gorm:"type:int;not null;default:0;index"`
	ResolvedBy      int    `gorm:"type:int;not null;default:0"`
	ResolvedAt      int64  `gorm:"type:bigint;not null;default:0"`
	CreatedAt       int64  `gorm:"autoCreateTime;column:created_at;index"`
	UpdatedAt       int64  `gorm:"autoUpdateTime;column:updated_at"`
}

func (legacyDingTalkSyncConflict) TableName() string {
	return "enterprise_dingtalk_sync_conflicts"
}
