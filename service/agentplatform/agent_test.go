package agentplatform

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	appmodel "github.com/QuantumNous/new-api/model"
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
	require.NoError(t, db.AutoMigrate(&appmodel.User{}, &appmodel.Token{}, &appmodel.Ability{}))
	return NewAgentService(db), db
}

func TestAgentServiceCreatesListsAndUpdatesOnlyAgents(t *testing.T) {
	svc, db := newAgentServiceForTest(t)

	created, err := svc.Create(AgentCreateInput{
		CliType:     apmodel.AgentCliTypeOpenCode,
		DisplayName: "My Agent",
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

	updated, err := svc.Update(created.ResourceId, AgentUpdateInput{CliType: apmodel.AgentCliTypeOpenCode, DisplayName: "My Agent V2"})
	require.NoError(t, err)
	require.Equal(t, "My Agent V2", updated.DisplayName)
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

func TestAgentServiceListsTokenModelLimits(t *testing.T) {
	svc, db := newAgentServiceForTest(t)
	token := createAgentServiceToken(t, db, 101, "gpt-5.5,gpt-4.1")

	result, err := svc.ListTokenModels(token.Id)
	require.NoError(t, err)
	require.Len(t, result.Models, 2)
	require.Equal(t, "gpt-4.1", result.Models[0].Model)
	require.Equal(t, "gpt-5.5", result.Models[1].Model)
	require.Equal(t, token.Key, result.Token.Key)
}

func TestAgentServiceListsAllUsersModelKeysWithSearch(t *testing.T) {
	svc, db := newAgentServiceForTest(t)
	require.NoError(t, db.Create(&appmodel.User{Id: 101, Username: "alice", DisplayName: "Alice", AffCode: "alice-agent"}).Error)
	require.NoError(t, db.Create(&appmodel.User{Id: 202, Username: "bob", DisplayName: "Bob", AffCode: "bob-agent"}).Error)
	createAgentServiceToken(t, db, 101, "gpt-5.5")
	bobToken := createAgentServiceToken(t, db, 202, "gpt-4.1,gpt-4o")

	items, err := svc.ListModelKeys("Bob")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, bobToken.Id, items[0].Id)
	require.Equal(t, 202, items[0].UserId)
	require.Equal(t, "Bob", items[0].UserName)
	require.Equal(t, 2, items[0].ModelCount)
	require.True(t, items[0].Available)

	allItems, err := svc.ListModelKeys("")
	require.NoError(t, err)
	require.Len(t, allItems, 2)
}

func TestAgentServiceDetailShowsModelKeyOwnedByAnotherUser(t *testing.T) {
	svc, db := newAgentServiceForTest(t)
	require.NoError(t, db.Create(&appmodel.User{Id: 202, Username: "bob", DisplayName: "Bob", AffCode: "bob-cross-agent"}).Error)
	token := createAgentServiceToken(t, db, 202, "gpt-5.5")

	created, err := svc.Create(AgentCreateInput{
		CliType:      apmodel.AgentCliTypeOpenCode,
		DisplayName:  "Cross user key Agent",
		OwnerUserId:  101,
		ModelTokenId: token.Id,
		DefaultModel: "gpt-5.5",
	})
	require.NoError(t, err)
	require.Equal(t, token.Id, created.ModelTokenId)
	require.Equal(t, 202, created.ModelTokenUserId)
	require.Equal(t, "Bob", created.ModelTokenUserName)
	require.Equal(t, "Agent service key", created.ModelTokenName)
	require.NotEmpty(t, created.ModelTokenMaskedKey)
}

func TestAgentServiceRejectsDefaultModelOutsideTokenLimits(t *testing.T) {
	svc, db := newAgentServiceForTest(t)
	token := createAgentServiceToken(t, db, 101, "gpt-5.5")

	_, err := svc.Create(AgentCreateInput{
		CliType:      apmodel.AgentCliTypeOpenCode,
		DisplayName:  "Model checked Agent",
		OwnerUserId:  101,
		ModelTokenId: token.Id,
		DefaultModel: "gpt-4.1",
	})
	require.ErrorIs(t, err, ErrInvalidResourceInput)
}

func createAgentServiceToken(t *testing.T, db *gorm.DB, userID int, models string) appmodel.Token {
	t.Helper()
	token := appmodel.Token{
		UserId:             userID,
		Key:                fmt.Sprintf("sk-agent-service-test-%d", userID),
		Status:             common.TokenStatusEnabled,
		Name:               "Agent service key",
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        -1,
		UnlimitedQuota:     true,
		ModelLimitsEnabled: true,
		ModelLimits:        models,
	}
	require.NoError(t, db.Create(&token).Error)
	return token
}
