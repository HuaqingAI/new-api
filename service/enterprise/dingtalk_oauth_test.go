package enterprise_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeDingTalkOAuthClient struct {
	contact      entservice.DingTalkContactUserInfo
	canonical    entservice.DingTalkDepartmentUserInfo
	children     map[int64][]entservice.DingTalkDepartmentInfo
	err          error
	directoryErr error
}

func (f *fakeDingTalkOAuthClient) GetAccessToken(context.Context, string, string) (string, error) {
	if f.directoryErr != nil {
		return "", f.directoryErr
	}
	return "app-token", nil
}

func (f *fakeDingTalkOAuthClient) ExchangeOAuthCode(context.Context, string, string, string) (entservice.DingTalkOAuthToken, error) {
	if f.err != nil {
		return entservice.DingTalkOAuthToken{}, f.err
	}
	return entservice.DingTalkOAuthToken{AccessToken: "user-token", UnionId: "union-1", OpenId: "open-1"}, nil
}

func (f *fakeDingTalkOAuthClient) GetOAuthUserInfo(context.Context, string) (entservice.DingTalkOAuthUserInfo, error) {
	if f.err != nil {
		return entservice.DingTalkOAuthUserInfo{}, f.err
	}
	return entservice.DingTalkOAuthUserInfo{UnionId: "union-1", OpenId: "open-1"}, nil
}

func (f *fakeDingTalkOAuthClient) GetContactUserByUnionId(context.Context, string, string) (entservice.DingTalkContactUserInfo, error) {
	if f.directoryErr != nil {
		return entservice.DingTalkContactUserInfo{}, f.directoryErr
	}
	return f.contact, nil
}

func (f *fakeDingTalkOAuthClient) GetUserById(context.Context, string, string) (entservice.DingTalkDepartmentUserInfo, error) {
	if f.directoryErr != nil {
		return entservice.DingTalkDepartmentUserInfo{}, f.directoryErr
	}
	return f.canonical, nil
}

func (f *fakeDingTalkOAuthClient) ListSubDepartments(_ context.Context, _ string, departmentId int64) ([]entservice.DingTalkDepartmentInfo, error) {
	if f.directoryErr != nil {
		return nil, f.directoryErr
	}
	return f.children[departmentId], nil
}

func TestDingTalkOAuthLoginAutoSyncsVerifiedEmployee(t *testing.T) {
	svc, db, _ := newDingTalkOAuthTestService(t)
	identity := resolveDingTalkTestIdentity(t, svc)

	result, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.NoError(t, err)
	require.Equal(t, entservice.DingTalkOAuthLoginStatusCreated, result.LoginStatus)
	require.True(t, result.AutoSynced)
	require.Equal(t, 1, result.DepartmentCount)
	require.NotZero(t, result.User.Id)

	var binding entmodel.DingTalkIdentity
	require.NoError(t, db.Where("tenant_id = ? AND identity_key = ?", 0, "union:union-1").First(&binding).Error)
	require.Equal(t, "corp-id", binding.CorpId)
	require.Equal(t, result.User.Id, binding.UserId)

	var departmentCount int64
	require.NoError(t, db.Model(&entmodel.Department{}).Where("tenant_id = ?", 0).Count(&departmentCount).Error)
	require.Equal(t, int64(2), departmentCount)
	var membershipCount int64
	require.NoError(t, db.Model(&entmodel.UserDepartment{}).Where("user_id = ?", result.User.Id).Count(&membershipCount).Error)
	require.Equal(t, int64(1), membershipCount)
}

func TestDingTalkOAuthLoginAutoSyncIsIdempotent(t *testing.T) {
	svc, db, client := newDingTalkOAuthTestService(t)
	identity := resolveDingTalkTestIdentity(t, svc)

	first, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.NoError(t, err)
	client.directoryErr = errors.New("directory unavailable")
	identity = resolveDingTalkTestIdentity(t, svc)
	second, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.NoError(t, err)
	require.Equal(t, first.User.Id, second.User.Id)
	require.Equal(t, entservice.DingTalkOAuthLoginStatusExisting, second.LoginStatus)
	require.True(t, second.UsedLocalSnapshot)
	assert.False(t, second.AutoSynced)

	var identityCount int64
	require.NoError(t, db.Model(&entmodel.DingTalkIdentity{}).Count(&identityCount).Error)
	require.Equal(t, int64(1), identityCount)
	var membershipCount int64
	require.NoError(t, db.Model(&entmodel.UserDepartment{}).Count(&membershipCount).Error)
	require.Equal(t, int64(1), membershipCount)
}

func TestDingTalkOAuthLoginAutoSyncSerializesConcurrentFirstLogins(t *testing.T) {
	svc, db, _ := newDingTalkOAuthTestService(t)
	identity := resolveDingTalkTestIdentity(t, svc)

	errors := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
			errors <- err
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	requireDingTalkLocalRows(t, db, 1, 1, 1)
}

func TestDingTalkOAuthRejectsDirectoryFailuresWithoutCreatingAnAccount(t *testing.T) {
	svc, db, client := newDingTalkOAuthTestService(t)
	identity := resolveDingTalkTestIdentity(t, svc)
	client.directoryErr = errors.New("directory unavailable")

	_, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.ErrorIs(t, err, entservice.ErrDingTalkOAuthEmployeeNotFound)

	requireDingTalkLocalRows(t, db, 0, 0, 0)
}

func TestDingTalkOAuthRejectsDisabledAndOutOfScopeEmployees(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		svc, db, client := newDingTalkOAuthTestService(t)
		identity := resolveDingTalkTestIdentity(t, svc)
		disabled := false
		client.canonical.Active = &disabled

		_, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
		require.ErrorIs(t, err, entservice.ErrDingTalkOAuthUserDisabled)
		requireDingTalkLocalRows(t, db, 0, 0, 0)
	})
	t.Run("out of scope", func(t *testing.T) {
		svc, db, client := newDingTalkOAuthTestService(t)
		identity := resolveDingTalkTestIdentity(t, svc)
		client.canonical.DeptIdList = []int64{99}

		_, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
		require.ErrorIs(t, err, entservice.ErrDingTalkOAuthOutOfScope)
		requireDingTalkLocalRows(t, db, 0, 0, 0)
	})
}

func TestDingTalkOAuthRejectsMismatchedDirectoryIdentity(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(*fakeDingTalkOAuthClient)
	}{
		{
			name: "contact union id",
			mutate: func(client *fakeDingTalkOAuthClient) {
				client.contact.UnionId = "union-other"
			},
		},
		{
			name: "canonical union id",
			mutate: func(client *fakeDingTalkOAuthClient) {
				client.canonical.UnionId = "union-other"
			},
		},
		{
			name: "canonical user id",
			mutate: func(client *fakeDingTalkOAuthClient) {
				client.canonical.UserId = "staff-other"
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			svc, db, client := newDingTalkOAuthTestService(t)
			identity := resolveDingTalkTestIdentity(t, svc)
			testCase.mutate(client)

			_, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
			require.ErrorIs(t, err, entservice.ErrDingTalkOAuthEmployeeNotFound)
			requireDingTalkLocalRows(t, db, 0, 0, 0)
		})
	}
}

func TestDingTalkOAuthRequiresOAuthVerifiedIdentity(t *testing.T) {
	svc, db, _ := newDingTalkOAuthTestService(t)
	enabled := true

	_, err := svc.LoginWithIdentity(context.Background(), 0, entservice.DingTalkOAuthIdentity{
		UnionId:        "union-1",
		ExternalUserId: "staff-1",
		Active:         &enabled,
		DepartmentIds:  []int64{2},
		ScopeVerified:  true,
	}, "")
	require.ErrorIs(t, err, entservice.ErrDingTalkOAuthEmployeeNotFound)
	requireDingTalkLocalRows(t, db, 0, 0, 0)
}

func TestDingTalkOAuthBindingStillRequiresDirectoryVerification(t *testing.T) {
	svc, db, client := newDingTalkOAuthTestService(t)
	target := &model.User{Username: "binding-target", Status: common.UserStatusEnabled, Group: "default", AffCode: "binding-target"}
	require.NoError(t, db.Create(target).Error)
	identity := resolveDingTalkTestIdentity(t, svc)
	client.directoryErr = errors.New("directory unavailable")

	_, err := svc.BindIdentityToUser(context.Background(), 0, target.Id, identity)

	require.ErrorIs(t, err, entservice.ErrDingTalkOAuthEmployeeNotFound)
	var bindingCount int64
	require.NoError(t, db.Model(&entmodel.DingTalkIdentity{}).Count(&bindingCount).Error)
	assert.Zero(t, bindingCount)
}

func TestDingTalkOAuthAllowsExistingMembershipWhenAutoSyncIsDisabled(t *testing.T) {
	svc, db, _ := newDingTalkOAuthTestService(t)
	identity := resolveDingTalkTestIdentity(t, svc)
	created, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.NoError(t, err)
	require.NotZero(t, created.User.Id)
	require.NoError(t, db.Model(&entmodel.DingTalkConfig{}).Where("tenant_id = ?", 0).Update("auto_sync_on_login", false).Error)

	result, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.NoError(t, err)
	require.Equal(t, entservice.DingTalkOAuthLoginStatusExisting, result.LoginStatus)
	require.True(t, result.UsedLocalSnapshot)

	require.NoError(t, db.Where("id = ?", created.User.Id).Delete(&model.User{}).Error)
	_, err = svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.ErrorIs(t, err, entservice.ErrDingTalkAutoSyncDisabled)
}

func TestDingTalkOAuthExistingUserKeepsLocalMembershipSnapshot(t *testing.T) {
	svc, db, client := newDingTalkOAuthTestService(t)
	identity := resolveDingTalkTestIdentity(t, svc)
	first, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.NoError(t, err)

	client.contact.UserId = "staff-2"
	client.canonical.UserId = "staff-2"
	identity = resolveDingTalkTestIdentity(t, svc)
	result, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.NoError(t, err)
	require.False(t, result.AutoSynced)
	require.True(t, result.UsedLocalSnapshot)
	require.Equal(t, first.User.Id, result.User.Id)

	var membership entmodel.UserDepartment
	require.NoError(t, db.Where("user_id = ?", result.User.Id).First(&membership).Error)
	require.Equal(t, "staff-1", membership.ExternalUserId)
}

func TestDingTalkOAuthAutoSyncCreatesEmployeeWhenPublicRegistrationIsDisabled(t *testing.T) {
	svc, _, _ := newDingTalkOAuthTestService(t)
	previousRegisterEnabled := common.RegisterEnabled
	common.RegisterEnabled = false
	t.Cleanup(func() { common.RegisterEnabled = previousRegisterEnabled })

	identity := resolveDingTalkTestIdentity(t, svc)
	result, err := svc.LoginWithIdentity(context.Background(), 0, identity, "")
	require.NoError(t, err)
	require.Equal(t, entservice.DingTalkOAuthLoginStatusCreated, result.LoginStatus)
}

func TestDingTalkOAuthUsernameGenerationPrefersReadableIdentity(t *testing.T) {
	svc, db, _ := newDingTalkOAuthTestService(t)
	require.NoError(t, db.Create(&model.User{Id: 250, Username: "alice", DisplayName: "Alice", Status: common.UserStatusEnabled, Group: "default", AffCode: "a1"}).Error)

	username := svc.AvailableReadableUsernameForTest(entservice.DingTalkOAuthIdentity{
		Name:           "Alice Chen",
		Email:          "alice.chen@example.com",
		ExternalUserId: "staff-100",
	})

	require.Equal(t, "alice_chen", username)
}

func newDingTalkOAuthTestService(t *testing.T) (*entservice.DingTalkOAuthService, *gorm.DB, *fakeDingTalkOAuthClient) {
	t.Helper()

	_, db := newDingTalkConfigTestService(t)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	common.RedisEnabled = false
	active := true
	client := &fakeDingTalkOAuthClient{
		contact: entservice.DingTalkContactUserInfo{UserId: "staff-1", UnionId: "union-1", Active: &active},
		canonical: entservice.DingTalkDepartmentUserInfo{
			UserId: "staff-1", UnionId: "union-1", Name: "Ding Employee", Email: "ding.employee@example.com", Mobile: "13800000000", Active: &active, DeptIdList: []int64{2},
		},
		children: map[int64][]entservice.DingTalkDepartmentInfo{
			1: {{DeptId: 2, Name: "Engineering", ParentId: 1}},
			2: {},
		},
	}
	require.NoError(t, db.Create(&entmodel.DingTalkConfig{
		TenantId:        0,
		CorpId:          "corp-id",
		AppKey:          "app-key",
		AppSecret:       "plain-secret",
		CallbackUrl:     "https://example.com/api/oauth/dingtalk",
		LoginEnabled:    true,
		SyncEnabled:     true,
		AutoSyncOnLogin: true,
	}).Error)
	common.RegisterEnabled = true
	entservice.ClearDingTalkScopeCache(0)

	return entservice.NewDingTalkOAuthService(db, client), db, client
}

func resolveDingTalkTestIdentity(t *testing.T, svc *entservice.DingTalkOAuthService) entservice.DingTalkOAuthIdentity {
	t.Helper()
	identity, err := svc.ResolveIdentity(context.Background(), 0, "code")
	require.NoError(t, err)
	return identity
}

func requireDingTalkLocalRows(t *testing.T, db *gorm.DB, users int64, identities int64, memberships int64) {
	t.Helper()
	var actualUsers int64
	require.NoError(t, db.Model(&model.User{}).Count(&actualUsers).Error)
	require.Equal(t, users, actualUsers)
	var actualIdentities int64
	require.NoError(t, db.Model(&entmodel.DingTalkIdentity{}).Count(&actualIdentities).Error)
	require.Equal(t, identities, actualIdentities)
	var actualMemberships int64
	require.NoError(t, db.Model(&entmodel.UserDepartment{}).Count(&actualMemberships).Error)
	require.Equal(t, memberships, actualMemberships)
}
