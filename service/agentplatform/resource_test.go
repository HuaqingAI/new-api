package agentplatform

import (
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newResourceServiceForTest(t *testing.T) (*ResourceService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))
	return NewResourceService(db), db
}

func TestResourceServiceCreatesAllResourceTypesInSharedRegistry(t *testing.T) {
	svc, db := newResourceServiceForTest(t)

	created := make([]ResourceItem, 0, 3)
	for _, resourceType := range []string{
		apmodel.ResourceTypeSkill,
		apmodel.ResourceTypeKnowledge,
		apmodel.ResourceTypeAgent,
	} {
		item, err := svc.Create(CreateResourceInput{
			ResourceType: resourceType,
			DisplayName:  resourceType + " name",
			OwnerUserId:  100,
		})
		require.NoError(t, err)
		created = append(created, item)
	}

	var count int64
	require.NoError(t, db.Model(&apmodel.Resource{}).Count(&count).Error)
	require.Equal(t, int64(3), count)
	require.Equal(t, apmodel.ResourceTypeSkill, created[0].ResourceType)
	require.Equal(t, apmodel.ResourceTypeKnowledge, created[1].ResourceType)
	require.Equal(t, apmodel.ResourceTypeAgent, created[2].ResourceType)
}

func TestResourceServiceListAndGetPreserveStableIdentity(t *testing.T) {
	svc, db := newResourceServiceForTest(t)

	item, err := svc.Create(CreateResourceInput{
		ResourceType: apmodel.ResourceTypeSkill,
		DisplayName:  "Stable Skill",
		OwnerUserId:  101,
		TenantId:     9,
	})
	require.NoError(t, err)

	fetched, err := svc.GetByResourceID(item.ResourceId)
	require.NoError(t, err)
	require.Equal(t, item.ResourceId, fetched.ResourceId)
	require.Equal(t, item.ResourceType, fetched.ResourceType)

	require.NoError(t, db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", item.ResourceId).
		Updates(map[string]any{"display_name": "Stable Skill v2", "latest_version": "ver_001"}).Error)

	updated, err := svc.GetByResourceID(item.ResourceId)
	require.NoError(t, err)
	require.Equal(t, item.ResourceId, updated.ResourceId)
	require.Equal(t, "ver_001", updated.LatestVersion)
	require.Equal(t, "Stable Skill v2", updated.DisplayName)

	result, err := svc.List(ListResourcesQuery{
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, item.ResourceId, result.Items[0].ResourceId)
}

func TestResourceServiceListExcludesRevokedResources(t *testing.T) {
	svc, db := newResourceServiceForTest(t)

	visible, err := svc.Create(CreateResourceInput{
		ResourceType: apmodel.ResourceTypeMCP,
		DisplayName:  "Visible MCP",
		OwnerUserId:  101,
		Status:       apmodel.ResourceStatusPublished,
	})
	require.NoError(t, err)
	revoked, err := svc.Create(CreateResourceInput{
		ResourceType: apmodel.ResourceTypeMCP,
		DisplayName:  "Deleted MCP",
		OwnerUserId:  101,
		Status:       apmodel.ResourceStatusPublished,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&apmodel.Resource{}).
		Where("resource_id = ?", revoked.ResourceId).
		Update("status", apmodel.ResourceStatusRevoked).Error)

	result, err := svc.List(ListResourcesQuery{
		ResourceType: apmodel.ResourceTypeMCP,
		Page:         1,
		PageSize:     20,
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, visible.ResourceId, result.Items[0].ResourceId)
}

func TestResourceServiceRejectsInvalidListFilter(t *testing.T) {
	svc, _ := newResourceServiceForTest(t)

	_, err := svc.List(ListResourcesQuery{
		ResourceType: "workflow",
		Page:         1,
		PageSize:     20,
	})
	require.ErrorIs(t, err, ErrInvalidResourceInput)
}
