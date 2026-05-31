package agentplatform

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

func OAuthAuthorize(c *gin.Context) {
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
		UserId:              c.GetInt("id"),
	})
	if err != nil {
		writeOAuthError(c, err)
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

func OAuthToken(c *gin.Context) {
	var req dtoagentplatform.OAuthTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request params")
		return
	}
	result, err := oauthTokenService().Exchange(apservice.TokenExchangeInput{
		ClientId:     req.ClientId,
		Code:         req.Code,
		CodeVerifier: req.CodeVerifier,
		RedirectURI:  req.RedirectURI,
	})
	if err != nil {
		writeOAuthError(c, err)
		return
	}
	common.ApiSuccess(c, dtoagentplatform.OAuthTokenResponse{
		AccessToken:     result.AccessToken,
		TokenType:       result.TokenType,
		ExpiresIn:       result.ExpiresIn,
		Scope:           result.Scope,
		ContractVersion: result.ContractVersion,
	})
}

func oauthAuthorizeService() *apservice.OAuthAuthorizeService {
	return apservice.NewOAuthAuthorizeService(model.DB)
}

func oauthTokenService() *apservice.OAuthTokenService {
	return apservice.NewOAuthTokenService(model.DB)
}

func writeOAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apservice.ErrInvalidAuthorizeInput), errors.Is(err, apservice.ErrInvalidTokenExchangeInput), errors.Is(err, apservice.ErrAuthorizationCodeInvalid):
		common.ApiErrorMsg(c, "invalid request params")
	case errors.Is(err, apservice.ErrUnauthorizedClient):
		common.ApiErrorMsg(c, "client not found")
	default:
		common.ApiError(c, err)
	}
}
