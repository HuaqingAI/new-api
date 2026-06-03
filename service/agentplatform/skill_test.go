package agentplatform

import (
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSkillServiceForTest(t *testing.T) (*SkillService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))
	return NewSkillService(db), db
}

func TestSkillServiceCreatesListsAndUpdatesOnlySkills(t *testing.T) {
	svc, db := newSkillServiceForTest(t)

	created, err := svc.Create(SkillCreateInput{
		DisplayName: "My Skill",
		OwnerUserId: 101,
		TenantId:    7,
	})
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceTypeSkill, created.ResourceType)

	require.NoError(t, db.Create(&apmodel.Resource{
		ResourceType: apmodel.ResourceTypeKnowledge,
		DisplayName:  "Other Resource",
		OwnerUserId:  101,
	}).Error)

	list, err := svc.List(SkillQuery{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Len(t, list.Items, 1)
	require.Equal(t, created.ResourceId, list.Items[0].ResourceId)

	updated, err := svc.Update(created.ResourceId, SkillUpdateInput{DisplayName: "My Skill V2"})
	require.NoError(t, err)
	require.Equal(t, "My Skill V2", updated.DisplayName)
}

func TestSkillServiceGetRejectsNonSkillResources(t *testing.T) {
	svc, db := newSkillServiceForTest(t)

	resource := apmodel.Resource{
		ResourceType: apmodel.ResourceTypeAgent,
		DisplayName:  "Agent",
		OwnerUserId:  100,
	}
	require.NoError(t, db.Create(&resource).Error)

	_, err := svc.Get(resource.ResourceId)
	require.ErrorIs(t, err, ErrResourceNotFound)
}
