package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAdminActionTestService(t *testing.T) (*entservice.AdminActionService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))
	return entservice.NewAdminActionService(db), db
}

func TestAdminActionWriteSanitizesSensitivePayload(t *testing.T) {
	svc, _ := newAdminActionTestService(t)

	require.NoError(t, svc.Write(entservice.AdminActionInput{
		ActorId:     100,
		ActionType:  entservice.AdminActionMembershipAdd,
		ObjectType:  entservice.AdminObjectDepartmentMember,
		ObjectId:    "1:200",
		DiffSummary: "added member with app_secret value",
		Payload: map[string]any{
			"user_id":      200,
			"app_secret":   "plain-secret",
			"access_token": "plain-token",
			"nested": map[string]any{
				"webhook_url": "https://example.invalid/hook",
			},
		},
	}))

	item, err := svc.Get(1)
	require.NoError(t, err)
	require.NotContains(t, item.DiffSummary, "app_secret")
	require.NotContains(t, item.Payload, "plain-secret")
	require.NotContains(t, item.Payload, "plain-token")
	require.NotContains(t, item.Payload, "example.invalid")
	require.Contains(t, item.Payload, "[REDACTED]")

	var payload map[string]any
	require.NoError(t, common.Unmarshal([]byte(item.Payload), &payload))
	require.Equal(t, float64(200), payload["user_id"])
	require.Equal(t, "[REDACTED]", payload["app_secret"])
	require.Equal(t, "[REDACTED]", payload["access_token"])
}

func TestAdminActionListFiltersAndPagination(t *testing.T) {
	svc, _ := newAdminActionTestService(t)
	require.NoError(t, svc.Write(entservice.AdminActionInput{
		ActorId:     100,
		ActionType:  entservice.AdminActionMembershipAdd,
		ObjectType:  entservice.AdminObjectDepartmentMember,
		ObjectId:    "1:200",
		DiffSummary: "first",
	}))
	require.NoError(t, svc.Write(entservice.AdminActionInput{
		ActorId:     101,
		ActionType:  entservice.AdminActionDeptAdminGrant,
		ObjectType:  entservice.AdminObjectDepartmentRole,
		ObjectId:    "2",
		DiffSummary: "second",
	}))

	actionType := entservice.AdminActionDeptAdminGrant
	result, err := svc.List(entservice.AdminActionQuery{
		ActionType: actionType,
		Page:       1,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, actionType, result.Items[0].ActionType)
	require.Equal(t, "2", result.Items[0].ObjectId)
}
