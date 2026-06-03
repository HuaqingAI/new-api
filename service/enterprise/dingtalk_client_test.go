package enterprise_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
)

func TestDingTalkOAuthUsesNewAPIBaseURLAndCamelCaseTokenResponse(t *testing.T) {
	httpClient := dingTalkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "https://api.example.test/v1.0/oauth2/userAccessToken", req.URL.String())
		require.Equal(t, "api.example.test", req.Host)
		require.Equal(t, "/v1.0/oauth2/userAccessToken", req.URL.Path)
		require.Equal(t, http.MethodPost, req.Method)
		return dingTalkJSONResponse(`{"accessToken":" user-token ","expireIn":7200,"refreshToken":"refresh-token","refreshExpireIn":604800,"openId":"open-1","unionId":"union-1"}`), nil
	})

	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL("https://openapi.example.test"),
		entservice.WithDingTalkAPIBaseURL("https://api.example.test"),
		entservice.WithDingTalkHTTPClient(&http.Client{
			Timeout:   time.Second,
			Transport: httpClient,
		}),
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
	httpClient := dingTalkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "https://api.example.test/v1.0/contact/users/me", req.URL.String())
		require.Equal(t, "/v1.0/contact/users/me", req.URL.Path)
		require.Equal(t, "user-token", req.Header.Get("x-acs-dingtalk-access-token"))
		return dingTalkJSONResponse(`{"nick":"Zhang San","openId":"open-1","unionId":"union-1","email":"zhangsan@example.com","mobile":"155****3240"}`), nil
	})

	client := entservice.NewDingTalkClient(
		entservice.WithDingTalkOpenAPIBaseURL("https://openapi.example.test"),
		entservice.WithDingTalkAPIBaseURL("https://api.example.test"),
		entservice.WithDingTalkHTTPClient(&http.Client{
			Timeout:   time.Second,
			Transport: httpClient,
		}),
	)

	user, err := client.GetOAuthUserInfo(context.Background(), "user-token")

	require.NoError(t, err)
	require.Equal(t, "union-1", user.UnionId)
	require.Equal(t, "open-1", user.OpenId)
	require.Equal(t, "Zhang San", user.Nick)
	require.Equal(t, "zhangsan@example.com", user.Email)
	require.Equal(t, "155****3240", user.Mobile)
}
