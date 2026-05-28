package enterprise_test

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDepartmentMembershipChangesDoNotMutateGroupBillingSemantics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Channel{}))
	require.NoError(t, entmodel.AutoMigrate(db))

	require.NoError(t, db.Create(&model.User{
		Id:       100,
		Username: "alice",
		Password: "password123",
		Group:    "vip",
		AffCode:  "alice-boundary",
	}).Error)
	require.NoError(t, db.Create(&model.Token{
		Id:     200,
		UserId: 100,
		Key:    "story-1-3-token",
		Group:  "premium",
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Id:     300,
		Type:   constant.ChannelTypeOpenAI,
		Key:    "story-1-3-channel-key",
		Name:   "story-1-3-channel",
		Group:  "vip,premium",
		Models: "gpt-4",
	}).Error)
	require.NoError(t, db.Create(&[]entmodel.Department{
		{Id: 1, TenantId: 0, Name: "Engineering"},
		{Id: 2, TenantId: 0, Name: "Security"},
	}).Error)

	var beforeUser model.User
	var beforeToken model.Token
	var beforeChannel model.Channel
	require.NoError(t, db.First(&beforeUser, 100).Error)
	require.NoError(t, db.First(&beforeToken, 200).Error)
	require.NoError(t, db.First(&beforeChannel, 300).Error)
	beforeGroupRatio := ratio_setting.GetGroupRatio(beforeUser.Group)
	beforeModelRatio, beforeModelRatioFound, beforeModelRatioName := ratio_setting.GetModelRatio("gpt-4")

	result, err := entservice.NewDepartmentMembershipService(db).ReplaceUserDepartments(100, entservice.ReplaceUserDepartmentsInput{
		DepartmentIds:   []int{1, 2},
		ExternalSource:  constant.EnterpriseExternalSourceManual,
		DeactivateStale: true,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 2)

	var afterUser model.User
	var afterToken model.Token
	var afterChannel model.Channel
	require.NoError(t, db.First(&afterUser, 100).Error)
	require.NoError(t, db.First(&afterToken, 200).Error)
	require.NoError(t, db.First(&afterChannel, 300).Error)
	afterModelRatio, afterModelRatioFound, afterModelRatioName := ratio_setting.GetModelRatio("gpt-4")

	require.Equal(t, beforeUser.Group, afterUser.Group)
	require.Equal(t, beforeToken.Group, afterToken.Group)
	require.Equal(t, beforeChannel.Group, afterChannel.Group)
	require.Equal(t, beforeGroupRatio, ratio_setting.GetGroupRatio(afterUser.Group))
	require.Equal(t, beforeModelRatio, afterModelRatio)
	require.Equal(t, beforeModelRatioFound, afterModelRatioFound)
	require.Equal(t, beforeModelRatioName, afterModelRatioName)
}
