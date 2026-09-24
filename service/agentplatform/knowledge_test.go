package agentplatform

import (
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newKnowledgeServiceForTest(t *testing.T) (*KnowledgeService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))
	return NewKnowledgeService(db), db
}

func TestKnowledgeServiceCreatesListsAndUpdatesOnlyKnowledge(t *testing.T) {
	svc, db := newKnowledgeServiceForTest(t)

	created, err := svc.Create(KnowledgeCreateInput{
		DisplayName: "My Knowledge",
		OwnerUserId: 101,
		TenantId:    7,
	})
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceTypeKnowledge, created.ResourceType)

	require.NoError(t, db.Create(&apmodel.Resource{
		ResourceType: apmodel.ResourceTypeSkill,
		DisplayName:  "Other Resource",
		OwnerUserId:  101,
	}).Error)

	list, err := svc.List(KnowledgeQuery{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Len(t, list.Items, 1)
	require.Equal(t, created.ResourceId, list.Items[0].ResourceId)

	updated, err := svc.Update(created.ResourceId, KnowledgeUpdateInput{DisplayName: "My Knowledge V2"})
	require.NoError(t, err)
	require.Equal(t, "My Knowledge V2", updated.DisplayName)
}

func TestKnowledgeServiceGetRejectsNonKnowledgeResources(t *testing.T) {
	svc, db := newKnowledgeServiceForTest(t)

	resource := apmodel.Resource{
		ResourceType: apmodel.ResourceTypeAgent,
		DisplayName:  "Agent",
		OwnerUserId:  100,
	}
	require.NoError(t, db.Create(&resource).Error)

	_, err := svc.Get(resource.ResourceId)
	require.ErrorIs(t, err, ErrResourceNotFound)
}
