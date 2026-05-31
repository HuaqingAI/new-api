package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrInvalidAuthorizeInput = errors.New("agent platform authorize input invalid")
	ErrUnauthorizedClient    = errors.New("agent platform client unauthorized")
)

type AuthorizeInput struct {
	ClientId            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	UserId              int
}

type AuthorizeResult struct {
	ClientId          string
	ContractVersion   string
	Scope             string
	State             string
	AuthorizationCode string
	RedirectURI       string
	ConsentRecorded   bool
}

type OAuthAuthorizeService struct {
	db *gorm.DB
}

func NewOAuthAuthorizeService(db *gorm.DB) *OAuthAuthorizeService {
	return &OAuthAuthorizeService{db: db}
}

func (s *OAuthAuthorizeService) Authorize(input AuthorizeInput) (AuthorizeResult, error) {
	if s == nil || s.db == nil {
		return AuthorizeResult{}, ErrInvalidAuthorizeInput
	}
	input.ClientId = strings.TrimSpace(input.ClientId)
	input.RedirectURI = strings.TrimSpace(input.RedirectURI)
	input.Scope = strings.TrimSpace(input.Scope)
	input.State = strings.TrimSpace(input.State)
	input.CodeChallenge = strings.TrimSpace(input.CodeChallenge)
	input.CodeChallengeMethod = strings.TrimSpace(strings.ToUpper(input.CodeChallengeMethod))
	if input.ClientId == "" || input.RedirectURI == "" || input.Scope == "" || input.CodeChallenge == "" || input.CodeChallengeMethod != "S256" || input.UserId <= 0 {
		return AuthorizeResult{}, ErrInvalidAuthorizeInput
	}

	var client apmodel.Client
	if err := s.db.Where("client_id = ?", input.ClientId).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthorizeResult{}, ErrUnauthorizedClient
		}
		return AuthorizeResult{}, err
	}
	if client.Status != "active" {
		return AuthorizeResult{}, ErrUnauthorizedClient
	}
	if client.ContractVersion == "" || client.CapabilitiesJSON == "" {
		return AuthorizeResult{}, ErrUnauthorizedClient
	}
	if !scopeAllowed(client.AllowedScopesJSON, input.Scope) || !redirectAllowed(client.RedirectURIsJSON, input.RedirectURI) || !grantAllowed(client.AllowedGrantTypesJSON, "authorization_code") {
		return AuthorizeResult{}, ErrUnauthorizedClient
	}

	code, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return AuthorizeResult{}, err
	}
	now := time.Now().UTC()
	userID := input.UserId
	grant := apmodel.AuthorizationGrant{
		ClientId:          client.ClientId,
		UserId:            &userID,
		GrantType:         "authorization_code",
		ScopeText:         input.Scope,
		Status:            "authorized",
		ContractVersion:   client.ContractVersion,
		CodeChallenge:     input.CodeChallenge,
		CodeMethod:        input.CodeChallengeMethod,
		AuthorizationCode: code,
		CodeExpiresAt:     now.Add(5 * time.Minute),
		ConsentedAt:       &now,
	}
	if err := s.db.Create(&grant).Error; err != nil {
		if errors.Is(err, apmodel.ErrInvalidAuthorizationGrantBody) {
			return AuthorizeResult{}, ErrInvalidAuthorizeInput
		}
		return AuthorizeResult{}, err
	}

	return AuthorizeResult{
		ClientId:          client.ClientId,
		ContractVersion:   client.ContractVersion,
		Scope:             input.Scope,
		State:             input.State,
		AuthorizationCode: code,
		RedirectURI:       input.RedirectURI,
		ConsentRecorded:   true,
	}, nil
}

func grantAllowed(raw string, required string) bool {
	values, err := parseStringArray(raw)
	if err != nil {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), required) {
			return true
		}
	}
	return false
}

func redirectAllowed(raw string, redirectURI string) bool {
	values, err := parseStringArray(raw)
	if err != nil {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == redirectURI {
			return true
		}
	}
	return false
}

func scopeAllowed(raw string, scopeText string) bool {
	values, err := parseStringArray(raw)
	if err != nil {
		return false
	}
	allowed := map[string]struct{}{}
	for _, value := range values {
		allowed[strings.TrimSpace(value)] = struct{}{}
	}
	for _, requested := range strings.Fields(scopeText) {
		if _, ok := allowed[requested]; !ok {
			return false
		}
	}
	return true
}

func parseStringArray(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	var values []string
	if err := common.UnmarshalJsonStr(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}
