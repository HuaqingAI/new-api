package agentplatform

type OAuthAuthorizeRequest struct {
	ClientId            string `form:"client_id" json:"client_id"`
	RedirectURI         string `form:"redirect_uri" json:"redirect_uri"`
	Scope               string `form:"scope" json:"scope"`
	State               string `form:"state" json:"state"`
	CodeChallenge       string `form:"code_challenge" json:"code_challenge"`
	CodeChallengeMethod string `form:"code_challenge_method" json:"code_challenge_method"`
}

type OAuthAuthorizeResponse struct {
	ClientId          string `json:"client_id"`
	ContractVersion   string `json:"contract_version"`
	Scope             string `json:"scope"`
	State             string `json:"state"`
	AuthorizationCode string `json:"authorization_code"`
	RedirectURI       string `json:"redirect_uri"`
	ConsentRecorded   bool   `json:"consent_recorded"`
}

type OAuthTokenRequest struct {
	ClientId     string `json:"client_id"`
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

type OAuthTokenResponse struct {
	AccessToken     string `json:"access_token"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	Scope           string `json:"scope"`
	ContractVersion string `json:"contract_version"`
}
