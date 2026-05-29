package enterprise_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestDingTalkConnectivitySuccess(t *testing.T) {
	svc, _ := newDingTalkConnectivityTestService(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gettoken":
			require.Equal(t, "app-key", r.URL.Query().Get("appkey"))
			require.Equal(t, "plain-secret", r.URL.Query().Get("appsecret"))
			_, _ = w.Write([]byte(`{"errcode":0,"access_token":"token-secret"}`))
		case "/topapi/v2/department/listsub":
			require.Equal(t, "token-secret", r.URL.Query().Get("access_token"))
			_, _ = w.Write([]byte(`{"errcode":0,"result":[{"dept_id":1}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	result, err := svc.Test(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, dtoenterprise.DingTalkConnectivityAuthSuccess, result.Code)
	require.Equal(t, "address_book_probe", result.Stage)
	require.Equal(t, "address_book_probe_ok", result.Summary)
}

func TestDingTalkConnectivityMapsInvalidCredentials(t *testing.T) {
	svc, _ := newDingTalkConnectivityTestService(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/gettoken", r.URL.Path)
		_, _ = w.Write([]byte(`{"errcode":40014,"errmsg":"invalid app secret: plain-secret"}`))
	})

	result, err := svc.Test(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, dtoenterprise.DingTalkConnectivityAuthInvalidCredentials, result.Code)
	require.Equal(t, "access_token", result.Stage)
	require.Equal(t, "token_invalid_credentials", result.Summary)
}

func TestDingTalkConnectivityMapsPermissionInsufficient(t *testing.T) {
	svc, _ := newDingTalkConnectivityTestService(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gettoken":
			_, _ = w.Write([]byte(`{"errcode":0,"access_token":"token-secret"}`))
		case "/topapi/v2/department/listsub":
			_, _ = w.Write([]byte(`{"errcode":60020,"errmsg":"access denied token-secret"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	result, err := svc.Test(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, dtoenterprise.DingTalkConnectivityPermissionInsufficient, result.Code)
	require.Equal(t, "address_book_probe", result.Stage)
	require.Equal(t, "address_book_permission_denied", result.Summary)
}

func TestDingTalkConnectivityMapsNetworkFailure(t *testing.T) {
	svc, server := newDingTalkConnectivityTestService(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":0,"access_token":"token-secret"}`))
	})
	server.Close()

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

func newDingTalkConnectivityTestService(t *testing.T, handler http.HandlerFunc) (*entservice.DingTalkConnectivityService, *httptest.Server) {
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

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL(server.URL),
		entservice.WithDingTalkHTTPClient(&http.Client{Timeout: time.Second}),
	)
	return entservice.NewDingTalkConnectivityService(db, client), server
}
