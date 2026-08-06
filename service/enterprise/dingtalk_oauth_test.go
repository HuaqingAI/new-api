package enterprise_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDingTalkOAuthLoginUsesExistingBindingAndLocalSnapshot(t *testing.T) {
	svc, db := newDingTalkOAuthTestService(t)
	require.NoError(t, db.Create(&model.User{Id: 200, Username: "dingtalk-user", DisplayName: "Ding User", Status: common.UserStatusEnabled, Group: "vip", AffCode: "dt01"}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkIdentity{
		TenantId:       0,
		CorpId:         "corp-id",
		IdentityKey:    "union:union-1",
		UnionId:        "union-1",
		OpenId:         "open-1",
		ExternalUserId: "staff-1",
		UserId:         200,
		Status:         entservice.DingTalkIdentityStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 10, TenantId: 0, Name: "Engineering", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 11, TenantId: 0, Name: "Security", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	require.NoError(t, db.Create(&[]entmodel.UserDepartment{
		{
			TenantId:       0,
			UserId:         200,
			DepartmentId:   10,
			ExternalUserId: "staff-1",
			ExternalSource: constant.EnterpriseExternalSourceDingTalk,
			Status:         constant.EnterpriseMembershipStatusActive,
		},
		{
			TenantId:       0,
			UserId:         200,
			DepartmentId:   11,
			ExternalUserId: "staff-1",
			ExternalSource: constant.EnterpriseExternalSourceDingTalk,
			Status:         constant.EnterpriseMembershipStatusActive,
		},
	}).Error)

	result, err := svc.LoginWithIdentity(context.Background(), 0, entservice.DingTalkOAuthIdentity{
		UnionId:        "union-1",
		OpenId:         "open-updated",
		ExternalUserId: "staff-1",
	}, "")

	require.NoError(t, err)
	require.Equal(t, 200, result.User.Id)
	require.Equal(t, 2, result.DepartmentCount)
	require.True(t, result.UsedLocalSnapshot)

	var count int64
	require.NoError(t, db.Model(&entmodel.UserDepartment{}).Where("user_id = ?", 200).Count(&count).Error)
	require.Equal(t, int64(2), count)
}

func TestDingTalkOAuthLoginBindsUniqueExistingEmail(t *testing.T) {
	svc, db := newDingTalkOAuthTestService(t)
	require.NoError(t, db.Create(&model.User{Id: 201, Username: "alice", DisplayName: "Alice", Email: "alice@example.com", Status: common.UserStatusEnabled, Group: "default", AffCode: "ali1"}).Error)
	requireDingTalkMembershipSnapshot(t, db, 201, 13, "staff-email")

	result, err := svc.LoginWithIdentity(context.Background(), 0, entservice.DingTalkOAuthIdentity{
		UnionId:        "union-email",
		OpenId:         "open-email",
		ExternalUserId: "staff-email",
		Name:           "Alice Ding",
		Email:          "alice@example.com",
	}, "")

	require.NoError(t, err)
	require.Equal(t, 201, result.User.Id)
	require.Equal(t, entservice.DingTalkOAuthLoginStatusBound, result.LoginStatus)

	var binding entmodel.DingTalkIdentity
	require.NoError(t, db.Where("identity_key = ?", "union:union-email").First(&binding).Error)
	require.Equal(t, 201, binding.UserId)
	require.Equal(t, "staff-email", binding.ExternalUserId)
}

func TestDingTalkOAuthLoginRejectsEmailConflict(t *testing.T) {
	svc, db := newDingTalkOAuthTestService(t)
	require.NoError(t, db.Create(&model.User{Id: 202, Username: "dup1", Email: "dup@example.com", Status: common.UserStatusEnabled, Group: "default", AffCode: "dup1"}).Error)
	require.NoError(t, db.Create(&model.User{Id: 203, Username: "dup2", Email: "dup@example.com", Status: common.UserStatusEnabled, Group: "default", AffCode: "dup2"}).Error)

	_, err := svc.LoginWithIdentity(context.Background(), 0, entservice.DingTalkOAuthIdentity{
		UnionId: "union-conflict",
		OpenId:  "open-conflict",
		Email:   "dup@example.com",
	}, "")

	require.ErrorIs(t, err, entservice.ErrDingTalkOAuthBindingConflict)
}

func TestDingTalkOAuthRejectsDisabledEmployeeAndDisabledBinding(t *testing.T) {
	svc, db := newDingTalkOAuthTestService(t)
	disabled := false

	_, err := svc.LoginWithIdentity(context.Background(), 0, entservice.DingTalkOAuthIdentity{
		UnionId: "union-disabled",
		OpenId:  "open-disabled",
		Active:  &disabled,
	}, "")
	require.ErrorIs(t, err, entservice.ErrDingTalkOAuthUserDisabled)

	require.NoError(t, db.Create(&model.User{Id: 204, Username: "disabled-bind", Status: common.UserStatusEnabled, Group: "default", AffCode: "db01"}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkIdentity{
		TenantId:    0,
		CorpId:      "corp-id",
		IdentityKey: "union:union-disabled-binding",
		UnionId:     "union-disabled-binding",
		UserId:      204,
		Status:      entservice.DingTalkIdentityStatusDisabled,
	}).Error)

	_, err = svc.LoginWithIdentity(context.Background(), 0, entservice.DingTalkOAuthIdentity{
		UnionId: "union-disabled-binding",
	}, "")
	require.ErrorIs(t, err, entservice.ErrDingTalkOAuthUserDisabled)
}

func TestDingTalkOAuthRejectsOutOfScopeWhenContactUserHasNoSnapshot(t *testing.T) {
	svc, db := newDingTalkOAuthTestService(t)
	require.NoError(t, db.Create(&model.User{Id: 205, Username: "outscope", Email: "out@example.com", Status: common.UserStatusEnabled, Group: "default", AffCode: "out1"}).Error)

	_, err := svc.LoginWithIdentity(context.Background(), 0, entservice.DingTalkOAuthIdentity{
		UnionId:        "union-out",
		OpenId:         "open-out",
		ExternalUserId: "staff-out",
		Email:          "out@example.com",
	}, "")

	require.ErrorIs(t, err, entservice.ErrDingTalkOAuthOutOfScope)
}

func TestDingTalkOAuthCreatesUserWhenRegistrationEnabledAndNoContactSnapshot(t *testing.T) {
	svc, db := newDingTalkOAuthTestService(t)
	previousRegisterEnabled := common.RegisterEnabled
	common.RegisterEnabled = true
	t.Cleanup(func() {
		common.RegisterEnabled = previousRegisterEnabled
	})

	result, err := svc.LoginWithIdentity(context.Background(), 0, entservice.DingTalkOAuthIdentity{
		UnionId: "union-new",
		OpenId:  "open-new",
		Name:    "New Employee",
		Email:   "new.employee@example.com",
	}, "")

	require.NoError(t, err)
	require.Equal(t, entservice.DingTalkOAuthLoginStatusCreated, result.LoginStatus)
	require.NotZero(t, result.User.Id)
	require.Equal(t, "new_employee", result.User.Username)
	require.NotContains(t, result.User.Username, "dt_")

	var binding entmodel.DingTalkIdentity
	require.NoError(t, db.Where("identity_key = ?", "union:union-new").First(&binding).Error)
	require.Equal(t, result.User.Id, binding.UserId)
}

func TestDingTalkOAuthBindIdentityToCurrentUser(t *testing.T) {
	svc, db := newDingTalkOAuthTestService(t)
	require.NoError(t, db.Create(&model.User{Id: 206, Username: "current", Status: common.UserStatusEnabled, Group: "default", AffCode: "cur1"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 12, TenantId: 0, Name: "Product", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         206,
		DepartmentId:   12,
		ExternalUserId: "staff-bind",
		ExternalSource: constant.EnterpriseExternalSourceDingTalk,
		Status:         constant.EnterpriseMembershipStatusActive,
	}).Error)

	binding, err := svc.BindIdentityToUser(context.Background(), 0, 206, entservice.DingTalkOAuthIdentity{
		UnionId:        "union-bind",
		OpenId:         "open-bind",
		ExternalUserId: "staff-bind",
	})

	require.NoError(t, err)
	require.Equal(t, 206, binding.UserId)
	require.Equal(t, "union:union-bind", binding.IdentityKey)
}

func TestDingTalkOAuthResolveIdentityDoesNotRequireAddressBookLookup(t *testing.T) {
	_, db := newDingTalkOAuthTestService(t)
	httpClient := dingTalkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/gettoken":
			return dingTalkJSONResponse(`{"errcode":0,"access_token":"app-token"}`), nil
		case "/topapi/user/getbyunionid":
			return dingTalkJSONResponse(`{"errcode":60020,"errmsg":"access denied"}`), nil
		case "/v1.0/oauth2/userAccessToken":
			return dingTalkJSONResponse(`{"accessToken":"user-token","openId":"open-1","unionId":"union-1"}`), nil
		case "/v1.0/contact/users/me":
			return dingTalkJSONResponse(`{"openId":"open-1","unionId":"union-1","nick":"Ding User"}`), nil
		default:
			return dingTalkStatusResponse(http.StatusNotFound, `{}`), nil
		}
	})
	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL("https://openapi.example.test"),
		entservice.WithDingTalkAPIBaseURL("https://api.example.test"),
		entservice.WithDingTalkHTTPClient(&http.Client{
			Timeout:   time.Second,
			Transport: httpClient,
		}),
	)
	svc := entservice.NewDingTalkOAuthService(db, client)

	identity, err := svc.ResolveIdentity(context.Background(), 0, "code-1")

	require.NoError(t, err)
	require.Equal(t, "union-1", identity.UnionId)
	require.Equal(t, "open-1", identity.OpenId)
	require.Equal(t, "", identity.ExternalUserId)
}

func TestDingTalkOAuthResolveIdentityUsesTokenIdentityWhenUserInfoFails(t *testing.T) {
	_, db := newDingTalkOAuthTestService(t)
	httpClient := dingTalkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/v1.0/oauth2/userAccessToken":
			return dingTalkJSONResponse(`{"accessToken":"user-token","openId":"open-token","unionId":"union-token"}`), nil
		case "/v1.0/contact/users/me":
			return dingTalkStatusResponse(http.StatusForbidden, `{"code":"Forbidden","message":"access denied"}`), nil
		default:
			return dingTalkStatusResponse(http.StatusNotFound, `{}`), nil
		}
	})
	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkAPIBaseURL("https://api.example.test"),
		entservice.WithDingTalkHTTPClient(&http.Client{
			Timeout:   time.Second,
			Transport: httpClient,
		}),
	)
	svc := entservice.NewDingTalkOAuthService(db, client)

	identity, err := svc.ResolveIdentity(context.Background(), 0, "code-1")

	require.NoError(t, err)
	require.Equal(t, "union-token", identity.UnionId)
	require.Equal(t, "open-token", identity.OpenId)
}

func newDingTalkOAuthTestService(t *testing.T) (*entservice.DingTalkOAuthService, *gorm.DB) {
	t.Helper()

	_, db := newDingTalkConfigTestService(t)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	common.RedisEnabled = false
	require.NoError(t, db.Create(&entmodel.DingTalkConfig{
		TenantId:     0,
		CorpId:       "corp-id",
		AppKey:       "app-key",
		AppSecret:    "plain-secret",
		CallbackUrl:  "https://example.com/api/oauth/dingtalk",
		LoginEnabled: true,
		SyncEnabled:  true,
	}).Error)
	common.RegisterEnabled = true

	return entservice.NewDingTalkOAuthService(db, nil), db
}

func requireDingTalkMembershipSnapshot(t *testing.T, db *gorm.DB, userId int, departmentId int, externalUserId string) {
	t.Helper()

	require.NoError(t, db.Create(&entmodel.Department{
		Id:       departmentId,
		TenantId: 0,
		Name:     "DingTalk Scope",
		Status:   constant.EnterpriseDepartmentStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		TenantId:       0,
		UserId:         userId,
		DepartmentId:   departmentId,
		ExternalUserId: externalUserId,
		ExternalSource: constant.EnterpriseExternalSourceDingTalk,
		Status:         constant.EnterpriseMembershipStatusActive,
	}).Error)
}

func TestDingTalkOAuthUsernameGenerationPrefersReadableIdentity(t *testing.T) {
	svc, db := newDingTalkOAuthTestService(t)
	require.NoError(t, db.Create(&model.User{Id: 250, Username: "alice", DisplayName: "Alice", Status: common.UserStatusEnabled, Group: "default", AffCode: "a1"}).Error)

	username := svc.AvailableReadableUsernameForTest(entservice.DingTalkOAuthIdentity{
		Name:           "Alice Chen",
		Email:          "alice.chen@example.com",
		ExternalUserId: "staff-100",
	})

	require.Equal(t, "alice_chen", username)
}
