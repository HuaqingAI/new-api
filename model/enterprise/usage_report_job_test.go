package enterprise

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type legacyUsageReportJobForMigration struct {
	Id              int    `gorm:"primaryKey"`
	TenantId        int    `gorm:"type:int;not null;default:0;uniqueIndex:uq_usage_report_job_tenant,priority:1"`
	Receivers       string `gorm:"type:text;not null"`
	Frequency       string `gorm:"type:varchar(32);not null;default:'daily'"`
	RangeType       string `gorm:"type:varchar(32);not null;default:'last7d'"`
	Enabled         bool   `gorm:"not null;default:true"`
	Status          string `gorm:"type:varchar(32);not null;default:'pending';index:idx_usage_report_jobs_status"`
	LastRunAt       int64  `gorm:"type:bigint;not null;default:0"`
	NextRunAt       int64  `gorm:"type:bigint;not null;default:0;index:idx_usage_report_jobs_next_run"`
	LastSuccessAt   int64  `gorm:"type:bigint;not null;default:0"`
	LastWindowStart int64  `gorm:"type:bigint;not null;default:0"`
	LastWindowEnd   int64  `gorm:"type:bigint;not null;default:0"`
	RunCount        int64  `gorm:"type:bigint;not null;default:0"`
	FailureCount    int64  `gorm:"type:bigint;not null;default:0"`
	ErrorReason     string `gorm:"type:text;not null"`
	LastSnapshot    string `gorm:"type:text;not null"`
	CreatedAt       int64  `gorm:"type:bigint;not null;default:0"`
	UpdatedAt       int64  `gorm:"type:bigint;not null;default:0"`
}

func (legacyUsageReportJobForMigration) TableName() string {
	return UsageReportJob{}.TableName()
}

func TestUsageReportJobJSONWrappersNormalizeEmptyValues(t *testing.T) {
	job := UsageReportJob{}

	require.NoError(t, job.SetReceivers([]string{"ops@example.com", "cto@example.com"}))
	require.JSONEq(t, `["ops@example.com","cto@example.com"]`, job.Receivers)

	receivers, err := job.ParsedReceivers()
	require.NoError(t, err)
	require.Equal(t, []string{"ops@example.com", "cto@example.com"}, receivers)

	require.NoError(t, job.SetLastSnapshot(&UsageReportSnapshot{
		WindowStart:     100,
		WindowEnd:       200,
		DepartmentCount: 2,
		TopDepartments: []UsageReportTopDepartment{
			{DeptName: "Engineering", RequestCount: 9},
		},
	}))
	snapshot, err := job.ParsedLastSnapshot()
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, int64(100), snapshot.WindowStart)
	require.Len(t, snapshot.TopDepartments, 1)

	require.NoError(t, job.SetReceivers(nil))
	require.JSONEq(t, `[]`, job.Receivers)
	require.NoError(t, job.SetLastSnapshot(nil))
	require.JSONEq(t, `{}`, job.LastSnapshot)
	snapshot, err = job.ParsedLastSnapshot()
	require.NoError(t, err)
	require.Nil(t, snapshot)
}

func TestUsageReportJobTextFieldsDoNotDeclareDBDefaults(t *testing.T) {
	models := []any{
		UsageSnapshot{},
		UsageReportJob{},
	}

	for _, item := range models {
		modelType := reflect.TypeOf(item)
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			gormTag := field.Tag.Get("gorm")
			if strings.Contains(gormTag, "type:text") {
				require.NotContainsf(t, gormTag, "default:", "%s.%s text field must not declare a DB default", modelType.Name(), field.Name)
			}
		}
	}
}

func TestMigrateCreatesUsageReportJobTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasTable(&UsageReportJob{}))
	for _, column := range []string{
		"tenant_id",
		"scope_key",
		"department_id",
		"include_descendants",
		"receivers",
		"frequency",
		"range_type",
		"enabled",
		"status",
		"last_run_at",
		"next_run_at",
		"last_success_at",
		"error_reason",
		"last_snapshot",
	} {
		require.True(t, db.Migrator().HasColumn(&UsageReportJob{}, column), column)
	}
	require.True(t, db.Migrator().HasIndex(&UsageReportJob{}, "uq_usage_report_job_scope"))
	require.False(t, db.Migrator().HasIndex(&UsageReportJob{}, "uq_usage_report_job_tenant"))
}

func TestUsageReportJobSupportsIndependentDepartmentScopes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	tenantJob := UsageReportJob{TenantId: 7, Frequency: UsageReportFrequencyDaily, RangeType: UsageReportRangeLast7Days}
	require.NoError(t, tenantJob.SetReceivers([]string{"tenant@example.com"}))
	require.NoError(t, tenantJob.SetLastSnapshot(nil))
	require.NoError(t, db.Create(&tenantJob).Error)

	departmentId := 9
	departmentJob := UsageReportJob{
		TenantId:           7,
		DepartmentId:       &departmentId,
		IncludeDescendants: true,
		Frequency:          UsageReportFrequencyWeekly,
		RangeType:          UsageReportRangeLast30Days,
	}
	require.NoError(t, departmentJob.SetReceivers([]string{"dept@example.com"}))
	require.NoError(t, departmentJob.SetLastSnapshot(nil))
	require.NoError(t, db.Create(&departmentJob).Error)

	var jobs []UsageReportJob
	require.NoError(t, db.Where("tenant_id = ?", 7).Order("id ASC").Find(&jobs).Error)
	require.Len(t, jobs, 2)
	require.Equal(t, UsageReportScopeTenant, jobs[0].ScopeKey)
	require.Equal(t, UsageReportScopeDepartmentPrefix+"9", jobs[1].ScopeKey)
}

func TestMigrateUpgradesExistingUsageReportJobScopeIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyUsageReportJobForMigration{}))
	require.True(t, db.Migrator().HasIndex(&UsageReportJob{}, "uq_usage_report_job_tenant"))

	for _, tenantID := range []int{1, 2} {
		job := legacyUsageReportJobForMigration{
			TenantId:     tenantID,
			Receivers:    "[]",
			Frequency:    UsageReportFrequencyDaily,
			RangeType:    UsageReportRangeLast7Days,
			Enabled:      false,
			Status:       UsageReportStatusPending,
			ErrorReason:  "",
			LastSnapshot: "{}",
		}
		require.NoError(t, db.Create(&job).Error)
	}

	require.NoError(t, Migrate(db))
	require.True(t, db.Migrator().HasColumn(&UsageReportJob{}, "scope_key"))
	require.True(t, db.Migrator().HasIndex(&UsageReportJob{}, "uq_usage_report_job_scope"))
	require.False(t, db.Migrator().HasIndex(&UsageReportJob{}, "uq_usage_report_job_tenant"))

	departmentID := 9
	departmentJob := UsageReportJob{
		TenantId:           1,
		DepartmentId:       &departmentID,
		IncludeDescendants: true,
		Frequency:          UsageReportFrequencyDaily,
		RangeType:          UsageReportRangeLast7Days,
	}
	require.NoError(t, departmentJob.SetReceivers([]string{"dept@example.com"}))
	require.NoError(t, departmentJob.SetLastSnapshot(nil))
	require.NoError(t, db.Create(&departmentJob).Error)

	var tenantJobs []UsageReportJob
	require.NoError(t, db.Where("tenant_id = ?", 1).Find(&tenantJobs).Error)
	require.Len(t, tenantJobs, 2)
}
