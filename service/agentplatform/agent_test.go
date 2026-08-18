package agentplatform

import (
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAgentServiceForTest(t *testing.T) (*AgentService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))
	return NewAgentService(db), db
}

func TestAgentServiceCreatesListsAndUpdatesOnlyAgents(t *testing.T) {
	svc, db := newAgentServiceForTest(t)

	created, err := svc.Create(AgentCreateInput{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "My Agent",
		Categories:  []string{apmodel.AgentCategoryGeneral},
		OwnerUserId: 101,
		TenantId:    7,
	})
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceTypeAgent, created.ResourceType)

	require.NoError(t, db.Create(&apmodel.Resource{
		ResourceType: apmodel.ResourceTypeKnowledge,
		DisplayName:  "Other Resource",
		OwnerUserId:  101,
	}).Error)

	list, err := svc.List(AgentQuery{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Len(t, list.Items, 1)
	require.Equal(t, created.ResourceId, list.Items[0].ResourceId)

	updated, err := svc.Update(created.ResourceId, AgentUpdateInput{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "My Agent V2",
		Categories:  []string{apmodel.AgentCategoryOperations, apmodel.AgentCategoryFinance},
	})
	require.NoError(t, err)
	require.Equal(t, "My Agent V2", updated.DisplayName)
	require.Equal(t, []string{apmodel.AgentCategoryOperations, apmodel.AgentCategoryFinance}, updated.Categories)

	fetched, err := svc.Get(created.ResourceId)
	require.NoError(t, err)
	require.Equal(t, []string{apmodel.AgentCategoryOperations, apmodel.AgentCategoryFinance}, fetched.Categories)
}

func TestAgentServiceGetRejectsNonAgentResources(t *testing.T) {
	svc, db := newAgentServiceForTest(t)

	resource := apmodel.Resource{
		ResourceType: apmodel.ResourceTypeSkill,
		DisplayName:  "Skill",
		OwnerUserId:  100,
	}
	require.NoError(t, db.Create(&resource).Error)

	_, err := svc.Get(resource.ResourceId)
	require.ErrorIs(t, err, ErrResourceNotFound)
}
