package agentplatform

import (
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newResourceVersionServiceForTest(t *testing.T) (*ResourceVersionService, *gorm.DB, apmodel.Resource) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	resource := apmodel.Resource{ResourceType: apmodel.ResourceTypeSkill, DisplayName: "Skill", OwnerUserId: 100}
	require.NoError(t, db.Create(&resource).Error)

	return NewResourceVersionService(db), db, resource
}

func TestResourceVersionServiceCreatesMetadataOnlyVersion(t *testing.T) {
	svc, db, resource := newResourceVersionServiceForTest(t)

	item, err := svc.Create(resource.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Summary:         "first release",
		CreatedBy:       100,
	})
	require.NoError(t, err)
	require.Equal(t, resource.ResourceId, item.ResourceId)
	require.Equal(t, apmodel.ResourceTypeSkill, item.ResourceType)
	require.Equal(t, "1.0.0", item.Version)
	require.Equal(t, "first release", item.Summary)
	require.Equal(t, apmodel.ResourceStatusDraft, item.Status)

	var persisted apmodel.Resource
	require.NoError(t, db.Where("resource_id = ?", resource.ResourceId).First(&persisted).Error)
	require.Equal(t, "1.0.0", persisted.LatestVersion)

	fetched, err := svc.Get(resource.ResourceId, "1.0.0")
	require.NoError(t, err)
	require.Equal(t, item.Version, fetched.Version)
}

func TestResourceVersionServiceRejectsInvalidMetadata(t *testing.T) {
	svc, _, resource := newResourceVersionServiceForTest(t)

	_, err := svc.Create(resource.ResourceId, CreateResourceVersionInput{
		ContractVersion: "2026-06",
		CreatedBy:       100,
	})
	require.ErrorIs(t, err, ErrInvalidResourceVersionInput)

	_, err = svc.Create(resource.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
	})
	require.ErrorIs(t, err, ErrInvalidResourceVersionInput)
}
