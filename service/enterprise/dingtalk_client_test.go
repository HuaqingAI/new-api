package enterprise_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestDingTalkOAuthUsesNewAPIBaseURLAndCamelCaseTokenResponse(t *testing.T) {
	openAPIServer := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(openAPIServer.Close)
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1.0/oauth2/userAccessToken", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"accessToken":" user-token ","expireIn":7200,"refreshToken":"refresh-token","refreshExpireIn":604800,"openId":"open-1","unionId":"union-1"}`))
	}))
	t.Cleanup(apiServer.Close)

	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL(openAPIServer.URL),
		entservice.WithDingTalkAPIBaseURL(apiServer.URL),
		entservice.WithDingTalkHTTPClient(&http.Client{Timeout: time.Second}),
	)

	token, err := client.ExchangeOAuthCode(context.Background(), "app-key", "plain-secret", "code-1")

	require.NoError(t, err)
	require.Equal(t, "user-token", token.AccessToken)
	require.Equal(t, int64(7200), token.ExpiresIn)
	require.Equal(t, "refresh-token", token.RefreshToken)
	require.Equal(t, int64(604800), token.RefreshTokenExpiresIn)
	require.Equal(t, "open-1", token.OpenId)
	require.Equal(t, "union-1", token.UnionId)
}

func TestDingTalkOAuthUserInfoUsesNewAPIBaseURLAndCamelCaseIdentity(t *testing.T) {
	openAPIServer := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(openAPIServer.Close)
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1.0/contact/users/me", r.URL.Path)
		require.Equal(t, "user-token", r.Header.Get("x-acs-dingtalk-access-token"))
		_, _ = w.Write([]byte(`{"nick":"Zhang San","openId":"open-1","unionId":"union-1","email":"zhangsan@example.com","mobile":"155****3240"}`))
	}))
	t.Cleanup(apiServer.Close)

	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL(openAPIServer.URL),
		entservice.WithDingTalkAPIBaseURL(apiServer.URL),
		entservice.WithDingTalkHTTPClient(&http.Client{Timeout: time.Second}),
	)

	user, err := client.GetOAuthUserInfo(context.Background(), "user-token")

	require.NoError(t, err)
	require.Equal(t, "union-1", user.UnionId)
	require.Equal(t, "open-1", user.OpenId)
	require.Equal(t, "Zhang San", user.Nick)
	require.Equal(t, "zhangsan@example.com", user.Email)
	require.Equal(t, "155****3240", user.Mobile)
}
