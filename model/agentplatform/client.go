package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrInvalidClientBody = errors.New("agent platform client body invalid")
)

type Client struct {
	Id                    int       `json:"id" gorm:"primaryKey"`
	ClientId              string    `json:"client_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_client_id;not null"`
	Slug                  string    `json:"slug" gorm:"type:varchar(128);uniqueIndex:idx_ap_client_slug;not null"`
	DisplayName           string    `json:"display_name" gorm:"type:varchar(255);not null"`
	ClientType            string    `json:"client_type" gorm:"type:varchar(32);not null"`
	Status                string    `json:"status" gorm:"type:varchar(32);not null;index:idx_ap_client_status"`
	AllowedGrantTypesJSON string    `json:"allowed_grant_types_json" gorm:"column:allowed_grant_types_json;type:text"`
	RedirectURIsJSON      string    `json:"redirect_uris_json" gorm:"column:redirect_uris_json;type:text"`
	AllowedScopesJSON     string    `json:"allowed_scopes_json" gorm:"column:allowed_scopes_json;type:text"`
	ContractVersion       string    `json:"contract_version" gorm:"type:varchar(64);not null"`
	CapabilitiesJSON      string    `json:"capabilities_json" gorm:"column:capabilities_json;type:text"`
	ExtensionsJSON        string    `json:"extensions_json" gorm:"column:extensions_json;type:text"`
	AllowClientCredentials bool     `json:"allow_client_credentials" gorm:"not null;default:false"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (Client) TableName() string {
	return "agent_platform_clients"
}

func (c *Client) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(c.ClientId) == "" {
		clientID, err := GenerateClientID()
		if err != nil {
			return err
		}
		c.ClientId = clientID
	}
	return c.applyDefaultsAndValidate()
}

func (c *Client) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	if c.Id == 0 && strings.TrimSpace(c.ClientId) == "" {
		clientID, err := GenerateClientID()
		if err != nil {
			return err
		}
		c.ClientId = clientID
	}
	return c.applyDefaultsAndValidate()
}

func (c *Client) applyDefaultsAndValidate() error {
	c.ClientId = strings.TrimSpace(c.ClientId)
	c.Slug = strings.TrimSpace(strings.ToLower(c.Slug))
	c.DisplayName = strings.TrimSpace(c.DisplayName)
	c.ClientType = strings.TrimSpace(strings.ToLower(c.ClientType))
	c.Status = strings.TrimSpace(strings.ToLower(c.Status))
	c.AllowedGrantTypesJSON = strings.TrimSpace(c.AllowedGrantTypesJSON)
	c.RedirectURIsJSON = strings.TrimSpace(c.RedirectURIsJSON)
	c.AllowedScopesJSON = strings.TrimSpace(c.AllowedScopesJSON)
	c.ContractVersion = strings.TrimSpace(c.ContractVersion)
	c.CapabilitiesJSON = strings.TrimSpace(c.CapabilitiesJSON)
	c.ExtensionsJSON = strings.TrimSpace(c.ExtensionsJSON)

	if c.Status == "" {
		c.Status = "invalid_integration"
	}
	if c.ClientId == "" || c.Slug == "" || c.DisplayName == "" || c.ClientType == "" {
		return ErrInvalidClientBody
	}
	return nil
}

func GenerateClientID() (string, error) {
	key, err := common.GenerateRandomCharsKey(29)
	if err != nil {
		return "", err
	}
	return "cli_" + key, nil
}
