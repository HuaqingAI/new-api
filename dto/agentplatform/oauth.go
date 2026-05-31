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
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type OAuthTokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	RefreshExpiresIn int    `json:"refresh_expires_in,omitempty"`
	Scope            string `json:"scope"`
	ContractVersion  string `json:"contract_version"`
	GrantId          string `json:"grant_id,omitempty"`
}

type OAuthRevokeRequest struct {
	ClientId      string `json:"client_id"`
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint"`
}
