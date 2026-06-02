package enterprise_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

type dingTalkHTTPClientFunc func(req *http.Request) (*http.Response, error)

func (f dingTalkHTTPClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDingTalkOAuthUsesNewAPIBaseURLAndCamelCaseTokenResponse(t *testing.T) {
	httpClient := dingTalkHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "api.local", r.URL.Host)
		require.Equal(t, "/v1.0/oauth2/userAccessToken", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"accessToken":" user-token ","expireIn":7200,"refreshToken":"refresh-token","refreshExpireIn":604800,"openId":"open-1","unionId":"union-1"}`)),
			Header:     make(http.Header),
		}, nil
	})

	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL("https://open.local"),
		entservice.WithDingTalkAPIBaseURL("https://api.local"),
		entservice.WithDingTalkHTTPClient(httpClient),
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
	httpClient := dingTalkHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "api.local", r.URL.Host)
		require.Equal(t, "/v1.0/contact/users/me", r.URL.Path)
		require.Equal(t, "user-token", r.Header.Get("x-acs-dingtalk-access-token"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"nick":"Zhang San","openId":"open-1","unionId":"union-1","email":"zhangsan@example.com","mobile":"155****3240"}`)),
			Header:     make(http.Header),
		}, nil
	})

	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL("https://open.local"),
		entservice.WithDingTalkAPIBaseURL("https://api.local"),
		entservice.WithDingTalkHTTPClient(httpClient),
	)

	user, err := client.GetOAuthUserInfo(context.Background(), "user-token")

	require.NoError(t, err)
	require.Equal(t, "union-1", user.UnionId)
	require.Equal(t, "open-1", user.OpenId)
	require.Equal(t, "Zhang San", user.Nick)
	require.Equal(t, "zhangsan@example.com", user.Email)
	require.Equal(t, "155****3240", user.Mobile)
}
