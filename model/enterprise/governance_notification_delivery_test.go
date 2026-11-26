package enterprise_test

import (
	"testing"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGovernanceNotificationDeliveryTracePayloadRoundTrip(t *testing.T) {
	delivery := entmodel.GovernanceNotificationDelivery{}
	require.NoError(t, delivery.SetTracePayload(&entmodel.GovernanceNotificationTracePayload{
		TraceId:      "quota_request:42",
		SourceType:   "quota_request",
		SourceId:     42,
		ActionType:   "enterprise.organization.quota_request.approve",
		DepartmentId: 7,
		QuotaDelta:   80,
		Status:       "fulfilled",
	}))

	parsed, err := delivery.ParsedTracePayload()
	require.NoError(t, err)
	require.Equal(t, "quota_request:42", parsed.TraceId)
	require.Equal(t, int64(80), parsed.QuotaDelta)
}

func TestGovernanceNotificationDeliveryMigrationAndDedupe(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))
	require.True(t, db.Migrator().HasTable(&entmodel.GovernanceNotificationDelivery{}))
	require.True(t, db.Migrator().HasColumn(&entmodel.GovernanceNotificationDelivery{}, "dedupe_key"))
	require.True(t, db.Migrator().HasColumn(&entmodel.GovernanceNotificationDelivery{}, "trace_payload"))

	first := entmodel.GovernanceNotificationDelivery{
		TenantId:        0,
		SourceType:      "quota_request",
		SourceId:        42,
		TraceId:         "quota_request:42",
		ActionType:      "enterprise.organization.quota_request.approve",
		RecipientUserId: 1001,
		RecipientKind:   "requester",
		DedupeKey:       "quota_request:42:requester",
	}
	require.NoError(t, first.SetTracePayload(nil))
	require.NoError(t, db.Create(&first).Error)

	second := first
	second.Id = 0
	require.Error(t, db.Create(&second).Error)
}
