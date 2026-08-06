package agentplatform

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func OAuthAuthorize(c *gin.Context) {
	redirectMode := wantsOAuthAuthorizeRedirect(c)
	userID, ok := sessionUserID(c)
	if !ok || userID <= 0 {
		if redirectMode {
			redirectToSignIn(c)
			return
		}
		common.ApiErrorMsg(c, "not logged in")
		return
	}
	var req dtoagentplatform.OAuthAuthorizeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := oauthAuthorizeService().Authorize(apservice.AuthorizeInput{
		ClientId:            req.ClientId,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		State:               req.State,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		UserId:              userID,
	})
	if err != nil {
		writeOAuthError(c, err)
		return
	}
	if redirectMode {
		redirectOAuthAuthorizeSuccess(c, result)
		return
	}
	common.ApiSuccess(c, dtoagentplatform.OAuthAuthorizeResponse{
		ClientId:          result.ClientId,
		ContractVersion:   result.ContractVersion,
		Scope:             result.Scope,
		State:             result.State,
		AuthorizationCode: result.AuthorizationCode,
		RedirectURI:       result.RedirectURI,
		ConsentRecorded:   result.ConsentRecorded,
	})
}

func wantsOAuthAuthorizeRedirect(c *gin.Context) bool {
	if strings.EqualFold(strings.TrimSpace(c.Query("response_mode")), "redirect") {
		return true
	}
	return strings.Contains(strings.ToLower(c.GetHeader("Accept")), "text/html")
}

func redirectToSignIn(c *gin.Context) {
	target := c.Request.URL.RequestURI()
	c.Redirect(http.StatusFound, "/sign-in?redirect="+url.QueryEscape(target))
}

func redirectOAuthAuthorizeSuccess(c *gin.Context, result apservice.AuthorizeResult) {
	redirectURL, err := url.Parse(result.RedirectURI)
	if err != nil {
		writeOAuthError(c, apservice.ErrInvalidAuthorizeInput)
		return
	}
	query := redirectURL.Query()
	query.Set("code", result.AuthorizationCode)
	if strings.TrimSpace(result.State) != "" {
		query.Set("state", result.State)
	}
	redirectURL.RawQuery = query.Encode()
	c.Redirect(http.StatusFound, redirectURL.String())
}

func OAuthToken(c *gin.Context) {
	var req dtoagentplatform.OAuthTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := oauthTokenService().Exchange(apservice.TokenExchangeInput{
		ClientId:     req.ClientId,
		GrantType:    req.GrantType,
		Code:         req.Code,
		CodeVerifier: req.CodeVerifier,
		RedirectURI:  req.RedirectURI,
		RefreshToken: req.RefreshToken,
		Scope:        req.Scope,
	})
	if err != nil {
		writeOAuthError(c, err)
		return
	}
	common.ApiSuccess(c, dtoagentplatform.OAuthTokenResponse{
		AccessToken:      result.AccessToken,
		TokenType:        result.TokenType,
		ExpiresIn:        result.ExpiresIn,
		RefreshToken:     result.RefreshToken,
		RefreshExpiresIn: result.RefreshExpiresIn,
		Scope:            result.Scope,
		ContractVersion:  result.ContractVersion,
		GrantId:          result.GrantId,
	})
}

func OAuthRevoke(c *gin.Context) {
	var req dtoagentplatform.OAuthRevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	if err := oauthTokenService().Revoke(apservice.RevokeTokenInput{
		ClientId:      req.ClientId,
		Token:         req.Token,
		TokenTypeHint: req.TokenTypeHint,
		ActorUserID:   currentActorID(c),
		RequestID:     c.GetHeader("X-Request-ID"),
	}); err != nil {
		writeOAuthError(c, err)
		return
	}
	common.ApiSuccess(c, dtoagentplatform.OAuthRevokeResponse{Revoked: true})
}

func oauthAuthorizeService() *apservice.OAuthAuthorizeService {
	return apservice.NewOAuthAuthorizeService(model.DB)
}

func oauthTokenService() *apservice.OAuthTokenService {
	return apservice.NewOAuthTokenService(model.DB)
}

func writeOAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apservice.ErrInvalidAuthorizeInput),
		errors.Is(err, apservice.ErrInvalidTokenExchangeInput),
		errors.Is(err, apservice.ErrAuthorizationCodeInvalid),
		errors.Is(err, apservice.ErrRefreshTokenInvalid),
		errors.Is(err, apservice.ErrGrantRevoked):
		common.ApiErrorMsg(c, "invalid request params")
	case errors.Is(err, apservice.ErrUnauthorizedClient):
		common.ApiErrorMsg(c, "client not found")
	default:
		common.ApiError(c, err)
	}
}

func sessionUserID(c *gin.Context) (int, bool) {
	id := c.GetInt("id")
	return id, id > 0
}

func currentActorID(c *gin.Context) int {
	if id, ok := sessionUserID(c); ok && id > 0 {
		return id
	}
	if header := strings.TrimSpace(c.GetHeader("New-Api-User")); header != "" {
		return common.String2Int(header)
	}
	return 0
}
