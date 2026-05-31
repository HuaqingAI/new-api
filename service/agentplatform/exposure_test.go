package agentplatform

import (
	"encoding/json"
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newExposureServiceForTest(t *testing.T) (*ExposureService, *gorm.DB, apmodel.Resource) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	resource := apmodel.Resource{ResourceType: apmodel.ResourceTypeSkill, DisplayName: "Skill", OwnerUserId: 1}
	require.NoError(t, db.Create(&resource).Error)
	version := apmodel.ResourceVersion{
		ResourceId:      resource.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       1,
	}
	require.NoError(t, db.Create(&version).Error)

	return NewExposureService(db), db, resource
}

func TestExposureServiceCreatesAndUpdatesStubTargetProjection(t *testing.T) {
	svc, _, resource := newExposureServiceForTest(t)

	item, err := svc.Create(resource.ResourceId, CreateExposureInput{
		ResourceVersion:     "1.0.0",
		ClientKey:           "client-a",
		ClientScope:         "placeholder",
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-a",
		Extensions:          json.RawMessage(`{"mode":"placeholder"}`),
	})
	require.NoError(t, err)
	require.Equal(t, apmodel.ExposureVisibilityVisible, item.VisibilityState)
	require.Equal(t, apmodel.ExposureCallableEnabled, item.CallableState)

	updated, err := svc.Update(resource.ResourceId, "client-a", UpdateExposureInput{
		VisibilityState: apmodel.ExposureVisibilityHidden,
		CallableState:   apmodel.ExposureCallableDisabled,
	})
	require.NoError(t, err)
	require.Equal(t, apmodel.ExposureVisibilityHidden, updated.VisibilityState)
	require.Equal(t, apmodel.ExposureCallableDisabled, updated.CallableState)
	require.NotNil(t, updated.RevokedAt)
}

func TestExposureServiceRejectsInvalidStates(t *testing.T) {
	svc, _, resource := newExposureServiceForTest(t)

	_, err := svc.Create(resource.ResourceId, CreateExposureInput{
		ResourceVersion: "1.0.0",
		ClientKey:       "client-a",
		VisibilityState: "ghost",
		CallableState:   apmodel.ExposureCallableEnabled,
	})
	require.ErrorIs(t, err, ErrInvalidExposureInput)
}
