package agentplatform

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type oauthAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupOAuthControllerTest(t *testing.T) (*gin.Engine, *gorm.DB, apmodel.Client) {
	return setupOAuthControllerTestWithSession(t, true)
}

func setupOAuthControllerTestWithSession(t *testing.T, authenticated bool) (*gin.Engine, *gorm.DB, apmodel.Client) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.CryptoSecret = "agent-platform-test-secret"

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:                  "cherry-studio",
		DisplayName:           "Cherry Studio",
		ClientType:            "desktop",
		Status:                "active",
		AllowedGrantTypesJSON: `["authorization_code","refresh_token"]`,
		RedirectURIsJSON:      `["https://example.com/callback"]`,
		AllowedScopesJSON:     `["skills.read"]`,
		ContractVersion:       "2026-06",
		CapabilitiesJSON:      `{"discovery":true}`,
	}
	require.NoError(t, db.Create(&client).Error)

	store := cookie.NewStore([]byte("secret"))
	router := gin.New()
	router.Use(sessions.Sessions("test-session", store))
	router.Use(func(c *gin.Context) {
		if authenticated {
			session := sessions.Default(c)
			session.Set("id", 999)
			require.NoError(t, session.Save())
		}
		c.Next()
	})
	router.GET("/api/agent-platform/oauth/authorize", OAuthAuthorize)
	router.POST("/api/agent-platform/oauth/token", OAuthToken)
	router.POST("/api/agent-platform/oauth/revoke", OAuthRevoke)
	return router, db, client
}

func performOAuthRequest(t *testing.T, router *gin.Engine, method string, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = common.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeOAuthAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) oauthAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response oauthAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestOAuthControllerWorkflow(t *testing.T) {
	router, _, client := setupOAuthControllerTest(t)

	verifier := "verifier-1"
	challenge := oauthTestPKCEChallenge(verifier)
	authURL := "/api/agent-platform/oauth/authorize?client_id=" + client.ClientId + "&redirect_uri=https://example.com/callback&scope=skills.read&state=s1&code_challenge=" + challenge + "&code_challenge_method=S256"
	authorize := performOAuthRequest(t, router, http.MethodGet, authURL, nil)
	authorizeResp := decodeOAuthAPIResponse(t, authorize)
	require.True(t, authorizeResp.Success, authorizeResp.Message)

	var authorizeData struct {
		AuthorizationCode string `json:"authorization_code"`
	}
	require.NoError(t, common.Unmarshal(authorizeResp.Data, &authorizeData))

	token := performOAuthRequest(t, router, http.MethodPost, "/api/agent-platform/oauth/token", map[string]any{
		"client_id":     client.ClientId,
		"grant_type":    "authorization_code",
		"code":          authorizeData.AuthorizationCode,
		"code_verifier": verifier,
		"redirect_uri":  "https://example.com/callback",
	})
	tokenResp := decodeOAuthAPIResponse(t, token)
	require.True(t, tokenResp.Success, tokenResp.Message)

	var tokenData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		GrantId      string `json:"grant_id"`
	}
	require.NoError(t, common.Unmarshal(tokenResp.Data, &tokenData))
	require.NotEmpty(t, tokenData.AccessToken)
	require.NotEmpty(t, tokenData.RefreshToken)
	require.NotEmpty(t, tokenData.GrantId)
}

func TestOAuthControllerRedirectModeSendsBrowserToClientCallback(t *testing.T) {
	router, _, client := setupOAuthControllerTest(t)

	challenge := oauthTestPKCEChallenge("verifier-redirect")
	authURL := "/api/agent-platform/oauth/authorize?client_id=" + client.ClientId + "&redirect_uri=https://example.com/callback&scope=skills.read&state=s-redirect&code_challenge=" + challenge + "&code_challenge_method=S256&response_mode=redirect"
	recorder := performOAuthRequest(t, router, http.MethodGet, authURL, nil)

	require.Equal(t, http.StatusFound, recorder.Code)
	location := recorder.Header().Get("Location")
	require.NotEmpty(t, location)
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "https", parsed.Scheme)
	require.Equal(t, "example.com", parsed.Host)
	require.Equal(t, "/callback", parsed.Path)
	require.NotEmpty(t, parsed.Query().Get("code"))
	require.Equal(t, "s-redirect", parsed.Query().Get("state"))
}

func TestOAuthControllerRedirectModeSendsAnonymousBrowserToSignIn(t *testing.T) {
	router, _, client := setupOAuthControllerTestWithSession(t, false)

	challenge := oauthTestPKCEChallenge("verifier-login")
	authURL := "/api/agent-platform/oauth/authorize?client_id=" + client.ClientId + "&redirect_uri=https://example.com/callback&scope=skills.read&state=s-login&code_challenge=" + challenge + "&code_challenge_method=S256&response_mode=redirect"
	recorder := performOAuthRequest(t, router, http.MethodGet, authURL, nil)

	require.Equal(t, http.StatusFound, recorder.Code)
	location := recorder.Header().Get("Location")
	require.NotEmpty(t, location)
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "/sign-in", parsed.Path)
	require.Equal(t, authURL, parsed.Query().Get("redirect"))
}

func TestOAuthControllerRefreshAndRevokeWorkflow(t *testing.T) {
	router, _, client := setupOAuthControllerTest(t)

	verifier := "verifier-2"
	challenge := oauthTestPKCEChallenge(verifier)
	authURL := "/api/agent-platform/oauth/authorize?client_id=" + client.ClientId + "&redirect_uri=https://example.com/callback&scope=skills.read&state=s2&code_challenge=" + challenge + "&code_challenge_method=S256"
	authorize := performOAuthRequest(t, router, http.MethodGet, authURL, nil)
	authorizeResp := decodeOAuthAPIResponse(t, authorize)
	require.True(t, authorizeResp.Success, authorizeResp.Message)

	var authorizeData struct {
		AuthorizationCode string `json:"authorization_code"`
	}
	require.NoError(t, common.Unmarshal(authorizeResp.Data, &authorizeData))

	token := performOAuthRequest(t, router, http.MethodPost, "/api/agent-platform/oauth/token", map[string]any{
		"client_id":     client.ClientId,
		"grant_type":    "authorization_code",
		"code":          authorizeData.AuthorizationCode,
		"code_verifier": verifier,
		"redirect_uri":  "https://example.com/callback",
	})
	tokenResp := decodeOAuthAPIResponse(t, token)
	require.True(t, tokenResp.Success, tokenResp.Message)

	var tokenData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, common.Unmarshal(tokenResp.Data, &tokenData))

	refresh := performOAuthRequest(t, router, http.MethodPost, "/api/agent-platform/oauth/token", map[string]any{
		"client_id":     client.ClientId,
		"grant_type":    "refresh_token",
		"refresh_token": tokenData.RefreshToken,
	})
	refreshResp := decodeOAuthAPIResponse(t, refresh)
	require.True(t, refreshResp.Success, refreshResp.Message)

	var refreshData struct {
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, common.Unmarshal(refreshResp.Data, &refreshData))
	require.NotEmpty(t, refreshData.RefreshToken)
	require.NotEqual(t, tokenData.RefreshToken, refreshData.RefreshToken)

	revoke := performOAuthRequest(t, router, http.MethodPost, "/api/agent-platform/oauth/revoke", map[string]any{
		"client_id":       client.ClientId,
		"token":           refreshData.RefreshToken,
		"token_type_hint": "refresh_token",
	})
	revokeResp := decodeOAuthAPIResponse(t, revoke)
	require.True(t, revokeResp.Success, revokeResp.Message)

	reuse := performOAuthRequest(t, router, http.MethodPost, "/api/agent-platform/oauth/token", map[string]any{
		"client_id":     client.ClientId,
		"grant_type":    "refresh_token",
		"refresh_token": refreshData.RefreshToken,
	})
	reuseResp := decodeOAuthAPIResponse(t, reuse)
	require.False(t, reuseResp.Success)
	require.Equal(t, "invalid request params", reuseResp.Message)
}

func TestOAuthControllerRejectsBadVerifier(t *testing.T) {
	router, _, client := setupOAuthControllerTest(t)

	verifier := "verifier-3"
	challenge := oauthTestPKCEChallenge(verifier)
	authURL := "/api/agent-platform/oauth/authorize?client_id=" + client.ClientId + "&redirect_uri=https://example.com/callback&scope=skills.read&state=s3&code_challenge=" + challenge + "&code_challenge_method=S256"
	authorize := performOAuthRequest(t, router, http.MethodGet, authURL, nil)
	authorizeResp := decodeOAuthAPIResponse(t, authorize)
	require.True(t, authorizeResp.Success, authorizeResp.Message)

	var authorizeData struct {
		AuthorizationCode string `json:"authorization_code"`
	}
	require.NoError(t, common.Unmarshal(authorizeResp.Data, &authorizeData))

	token := performOAuthRequest(t, router, http.MethodPost, "/api/agent-platform/oauth/token", map[string]any{
		"client_id":     client.ClientId,
		"grant_type":    "authorization_code",
		"code":          authorizeData.AuthorizationCode,
		"code_verifier": "wrong-verifier",
		"redirect_uri":  "https://example.com/callback",
	})
	tokenResp := decodeOAuthAPIResponse(t, token)
	require.False(t, tokenResp.Success)
	require.Equal(t, "invalid request params", tokenResp.Message)
}

func oauthTestPKCEChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
