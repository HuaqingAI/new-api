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

func TestQuotaRequestMigrationCreatesLifecycleAndRouteColumns(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))

	require.True(t, db.Migrator().HasTable(&entmodel.QuotaRequest{}))
	for _, column := range []string{
		"tenant_id",
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
