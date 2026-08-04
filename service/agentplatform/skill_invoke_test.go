package agentplatform

import (
	"testing"
	"time"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSkillInvokeServiceForTest(t *testing.T) (*SkillInvokeService, *gorm.DB, string, apmodel.Resource) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:                  "invoke-client",
		DisplayName:           "Invoke Client",
		ClientType:            "desktop",
		Status:                "active",
		AllowedGrantTypesJSON: `["client_credentials"]`,
		AllowedScopesJSON:     `["ap.resources.read","ap.skills.invoke"]`,
		ContractVersion:       "2026-06",
		CapabilitiesJSON:      `{"discovery":true}`,
	}
	require.NoError(t, db.Create(&client).Error)

	resource := apmodel.Resource{
		ResourceType:  apmodel.ResourceTypeSkill,
		DisplayName:   "Invoke Skill",
		OwnerUserId:   100,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}
	require.NoError(t, db.Create(&resource).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      resource.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)
	now := time.Now().UTC()
	require.NoError(t, db.Create(&apmodel.Exposure{
		ResourceId:          resource.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           client.ClientId,
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-skill",
		PublishedAt:         &now,
	}).Error)

	return NewSkillInvokeService(db), db, client.ClientId, resource
}

func TestSkillInvokeServiceReturnsContractInvalidWithoutInvokeConfig(t *testing.T) {
	svc, _, clientID, resource := newSkillInvokeServiceForTest(t)

	_, err := svc.Invoke(SkillInvokeInput{
		ClientID:   clientID,
		ResourceID: resource.ResourceId,
		Payload:    []byte(`{"input":"demo"}`),
	})
	require.ErrorIs(t, err, ErrOpenCapabilityContractInvalid)
}
