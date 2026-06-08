package enterprise_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestDingTalkConnectivitySuccess(t *testing.T) {
	svc := newDingTalkConnectivityTestService(t, dingTalkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/gettoken":
			require.Equal(t, "app-key", req.URL.Query().Get("appkey"))
			require.Equal(t, "plain-secret", req.URL.Query().Get("appsecret"))
			return dingTalkJSONResponse(`{"errcode":0,"access_token":"token-secret"}`), nil
		case "/topapi/v2/department/listsub":
			require.Equal(t, "token-secret", req.URL.Query().Get("access_token"))
			return dingTalkJSONResponse(`{"errcode":0,"result":[{"dept_id":1}]}`), nil
		default:
			return dingTalkStatusResponse(http.StatusNotFound, `{}`), nil
		}
	}))

	result, err := svc.Test(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, dtoenterprise.DingTalkConnectivityAuthSuccess, result.Code)
	require.Equal(t, "address_book_probe", result.Stage)
	require.Equal(t, "address_book_probe_ok", result.Summary)
}

func TestDingTalkConnectivityMapsInvalidCredentials(t *testing.T) {
	svc := newDingTalkConnectivityTestService(t, dingTalkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/gettoken", req.URL.Path)
		return dingTalkJSONResponse(`{"errcode":40014,"errmsg":"invalid app secret: plain-secret"}`), nil
	}))

	result, err := svc.Test(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, dtoenterprise.DingTalkConnectivityAuthInvalidCredentials, result.Code)
	require.Equal(t, "access_token", result.Stage)
	require.Equal(t, "token_invalid_credentials", result.Summary)
}

func TestDingTalkConnectivityMapsPermissionInsufficient(t *testing.T) {
	svc := newDingTalkConnectivityTestService(t, dingTalkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/gettoken":
			return dingTalkJSONResponse(`{"errcode":0,"access_token":"token-secret"}`), nil
		case "/topapi/v2/department/listsub":
			return dingTalkJSONResponse(`{"errcode":60020,"errmsg":"access denied token-secret"}`), nil
		default:
			return dingTalkStatusResponse(http.StatusNotFound, `{}`), nil
		}
	}))

	result, err := svc.Test(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, dtoenterprise.DingTalkConnectivityPermissionInsufficient, result.Code)
	require.Equal(t, "address_book_probe", result.Stage)
	require.Equal(t, "address_book_permission_denied", result.Summary)
}

func TestDingTalkConnectivityMapsNetworkFailure(t *testing.T) {
	svc := newDingTalkConnectivityTestService(t, dingTalkNetworkFailureRoundTripper())

	result, err := svc.Test(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, dtoenterprise.DingTalkConnectivityNetworkUnreachable, result.Code)
	require.Equal(t, "access_token", result.Stage)
	require.Equal(t, "token_request_failed", result.Summary)
}

func TestDingTalkConnectivityMapsCallbackMisconfigured(t *testing.T) {
	_, db := newDingTalkConfigTestService(t)
	require.NoError(t, db.Create(&entmodel.DingTalkConfig{
		TenantId:    0,
		CorpId:      "corp-id",
		AppKey:      "app-key",
		AppSecret:   "plain-secret",
		CallbackUrl: "http://example.com/callback",
	}).Error)

	result, err := entservice.NewDingTalkConnectivityService(db, nil).Test(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, dtoenterprise.DingTalkConnectivityCallbackMisconfigured, result.Code)
	require.Equal(t, "callback", result.Stage)
	require.Equal(t, "callback_url_invalid", result.Summary)
}

func newDingTalkConnectivityTestService(t *testing.T, transport http.RoundTripper) *entservice.DingTalkConnectivityService {
	t.Helper()

	_, db := newDingTalkConfigTestService(t)
	secret := "plain-secret"
	require.NoError(t, db.Create(&entmodel.DingTalkConfig{
		TenantId:     0,
		CorpId:       "corp-id",
		AppKey:       "app-key",
		AppSecret:    secret,
		CallbackUrl:  "https://example.com/api/oauth/dingtalk",
		LoginEnabled: true,
		SyncEnabled:  true,
	}).Error)

	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL("https://openapi.example.test"),
		entservice.WithDingTalkHTTPClient(&http.Client{
			Timeout:   time.Second,
			Transport: transport,
		}),
	)
	return entservice.NewDingTalkConnectivityService(db, client)
}
