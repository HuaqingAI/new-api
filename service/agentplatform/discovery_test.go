package agentplatform

import (
	"encoding/json"
	"testing"
	"time"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDiscoveryServiceForTest(t *testing.T) (*DiscoveryService, *gorm.DB, apmodel.Client, apmodel.Resource, apmodel.Resource, string) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:                  "cherry-studio",
		DisplayName:           "Cherry Studio",
		ClientType:            "desktop",
		Status:                "active",
		AllowedGrantTypesJSON: `["authorization_code"]`,
		AllowedScopesJSON:     `["ap.resources.read","ap.skills.invoke","ap.agents.read"]`,
		ContractVersion:       "2026-06",
		CapabilitiesJSON:      `{"discovery":true}`,
	}
	require.NoError(t, db.Create(&client).Error)

	skill := apmodel.Resource{
		ResourceType:  apmodel.ResourceTypeSkill,
		DisplayName:   "Published Skill",
		OwnerUserId:   100,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}
	hidden := apmodel.Resource{
		ResourceType:  apmodel.ResourceTypeKnowledge,
		DisplayName:   "Hidden Knowledge",
		OwnerUserId:   100,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}
	require.NoError(t, db.Create(&skill).Error)
	require.NoError(t, db.Create(&hidden).Error)

	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      skill.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		SchemaJSON:      `{"type":"object"}`,
		DetailJSON:      `{"skill":{"invoke_mode":"sync"}}`,
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      hidden.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		SchemaJSON:      `{"type":"object"}`,
		DetailJSON:      `{"knowledge":{"mode":"retrieval"}}`,
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)

	now := time.Now().UTC()
	require.NoError(t, db.Create(&apmodel.Exposure{
		ResourceId:          skill.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           client.ClientId,
		ClientScope:         "",
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-skill",
		ExtensionsJSON:      `{"cherry_studio":{"ui_variant":"enterprise"}}`,
		PublishedAt:         &now,
	}).Error)
	require.NoError(t, db.Create(&apmodel.Exposure{
		ResourceId:          hidden.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           "other-client",
		ClientScope:         "",
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-hidden",
		PublishedAt:         &now,
	}).Error)

	return NewDiscoveryService(db), db, client, skill, hidden, client.ClientId
}

func TestDiscoveryServiceReturnsPublishedProjectionsOnlyForClient(t *testing.T) {
	svc, _, _, skill, _, clientID := newDiscoveryServiceForTest(t)

	result, err := svc.Discovery(DiscoveryQuery{ClientID: clientID})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, skill.ResourceId, result.Items[0].ResourceID)
	require.Equal(t, OpenCapabilityFreshnessFresh, result.Items[0].Freshness)
}

func TestDiscoveryServiceReturnsDetailAndRefresh(t *testing.T) {
	svc, _, _, skill, _, clientID := newDiscoveryServiceForTest(t)

	detail, err := svc.Detail(clientID, skill.ResourceId)
	require.NoError(t, err)
	require.Equal(t, skill.ResourceId, detail.ResourceID)
	require.Equal(t, apmodel.ResourceTypeSkill, detail.ResourceType)
	require.Contains(t, detail.SupportedExtensions, "cherry_studio")
	require.True(t, detail.ContractCompatible)

	refresh, err := svc.Refresh(RefreshInput{ClientID: clientID, ResourceID: skill.ResourceId})
	require.NoError(t, err)
	require.Equal(t, detail.ETag, refresh.ETag)
	require.Equal(t, OpenCapabilityFreshnessFresh, refresh.Freshness)
	require.True(t, refresh.ContractCompatible)
}

func TestDiscoveryServiceMapsRevokedAndOfflineFreshness(t *testing.T) {
	svc, db, _, skill, _, clientID := newDiscoveryServiceForTest(t)

	var exposure apmodel.Exposure
	require.NoError(t, db.Where("resource_id = ? AND client_key = ?", skill.ResourceId, clientID).First(&exposure).Error)
	now := time.Now().UTC()
	require.NoError(t, db.Model(&exposure).Updates(map[string]any{
		"visibility_state": apmodel.ExposureVisibilityRevoked,
		"callable_state":   apmodel.ExposureCallableRevoked,
		"revoked_at":       now,
	}).Error)

	_, err := svc.Detail(clientID, skill.ResourceId)
	require.ErrorIs(t, err, ErrOpenCapabilityResourceRevoked)

	require.NoError(t, db.Model(&apmodel.Exposure{}).Where("id = ?", exposure.Id).Updates(map[string]any{
		"visibility_state": apmodel.ExposureVisibilityVisible,
		"callable_state":   apmodel.ExposureCallableEnabled,
		"revoked_at":       nil,
	}).Error)
	require.NoError(t, db.Model(&apmodel.Resource{}).Where("resource_id = ?", skill.ResourceId).Update("status", apmodel.ResourceStatusOffline).Error)

	_, err = svc.Detail(clientID, skill.ResourceId)
	require.ErrorIs(t, err, ErrOpenCapabilityResourceOffline)
}

func TestMapOpenCapabilityErrorIncludesStableFields(t *testing.T) {
	response := MapOpenCapabilityError(ErrOpenCapabilityContractInvalid, OpenCapabilityContext{
		RequestID:       "req-1",
		ResourceID:      "res-1",
		ResourceVersion: "1.0.0",
	})
	require.False(t, response.Success)
	require.Equal(t, OpenCapabilityCodeContractInvalid, response.Error.Code)
	require.Equal(t, "req-1", response.Error.RequestID)
	require.Equal(t, "res-1", response.Error.ResourceID)
	require.Equal(t, "1.0.0", response.Error.ResourceVersion)

	payload, err := json.Marshal(response)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"code":"contractInvalid"`)
}

func TestDiscoveryServiceDetectsContractVersionMismatch(t *testing.T) {
	svc, db, _, skill, _, clientID := newDiscoveryServiceForTest(t)
	require.NoError(t, db.Model(&apmodel.Client{}).Where("client_id = ?", clientID).Update("contract_version", "2026-07").Error)

	_, err := svc.Detail(clientID, skill.ResourceId)
	require.ErrorIs(t, err, ErrOpenCapabilityContractInvalid)
}

func TestDiscoveryServiceMarksRefreshAsStaleAndNonCompliant(t *testing.T) {
	svc, db, _, skill, _, clientID := newDiscoveryServiceForTest(t)

	oldPublishedAt := time.Now().UTC().Add(-10 * time.Minute)
	require.NoError(t, db.Model(&apmodel.Exposure{}).
		Where("resource_id = ? AND client_key = ?", skill.ResourceId, clientID).
		Updates(map[string]any{"published_at": oldPublishedAt}).Error)

	observedAt := time.Now().UTC().Unix()
	refresh, err := svc.Refresh(RefreshInput{
		ClientID:                clientID,
		ResourceID:              skill.ResourceId,
		ObservedETag:            "stale-etag",
		ObservedResourceVersion: "0.9.0",
		ObservedAtUnix:          &observedAt,
	})
	require.NoError(t, err)
	require.Equal(t, OpenCapabilityFreshnessStale, refresh.Freshness)
	require.Equal(t, "client_non_compliant_stale", refresh.Diagnostics.Reason)
	require.True(t, refresh.Diagnostics.ClientNonCompliant)
}
