package enterprise_test

import (
	"fmt"
	"strings"
	"testing"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type legacyQuotaRequestScopeMigration struct {
	Id              int `gorm:"primaryKey"`
	TenantId        int
	DepartmentId    int
	RequesterUserId int
	RequestedQuota  int64
	Status          string
}

func (legacyQuotaRequestScopeMigration) TableName() string {
	return entmodel.QuotaRequest{}.TableName()
}

func TestQuotaRequestMigrationCreatesLifecycleAndRouteColumns(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	require.True(t, db.Migrator().HasTable(&entmodel.QuotaRequest{}))
	for _, column := range []string{
		"tenant_id",
		"department_budget_id",
		"budget_scope_snapshot",
		"budget_name_snapshot",
		"requester_user_id",
		"idempotency_key",
		"owner_count_snapshot",
		"fallback",
		"submitted_at",
		"approved_at",
		"rejected_at",
		"fulfilled_at",
		"processed_at",
	} {
		require.Truef(t, db.Migrator().HasColumn(&entmodel.QuotaRequest{}, column), "missing column %s", column)
	}
	require.True(t, db.Migrator().HasIndex(&entmodel.QuotaRequest{}, "uq_ent_quota_req_idempotency"))
}

func TestQuotaRequestMigrationBackfillsLegacyBudgetScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyQuotaRequestScopeMigration{}))
	require.NoError(t, db.Create(&legacyQuotaRequestScopeMigration{
		Id:              1,
		TenantId:        2,
		DepartmentId:    10,
		RequesterUserId: 20,
		RequestedQuota:  100,
		Status:          entmodel.QuotaRequestStatusSubmitted,
	}).Error)

	require.NoError(t, entmodel.AutoMigrate(db))
	require.NoError(t, entmodel.AutoMigrate(db))

	var request entmodel.QuotaRequest
	require.NoError(t, db.First(&request, 1).Error)
	require.Equal(t, entmodel.DepartmentBudgetScopeDepartment, request.BudgetScopeSnapshot)
	require.Empty(t, request.BudgetNameSnapshot)
}
