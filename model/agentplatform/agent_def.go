package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var ErrInvalidAgentDefBody = errors.New("agent platform agent def body invalid")

const (
	AgentCliTypeOpenCode = "opencode"
	AgentCliTypeCodex    = "codex"

	AgentCategoryGeneral        = "general"
	AgentCategoryOperations     = "operations"
	AgentCategoryCustomerService = "customer_service"
	AgentCategoryLogistics      = "logistics"
	AgentCategoryMarketing      = "marketing"
	AgentCategoryFinance        = "finance"
	AgentCategoryHR             = "hr"
)

var allowedAgentCategories = []string{
	AgentCategoryGeneral,
	AgentCategoryOperations,
	AgentCategoryCustomerService,
	AgentCategoryLogistics,
	AgentCategoryMarketing,
	AgentCategoryFinance,
	AgentCategoryHR,
}

type AgentDef struct {
	Id              int       `json:"id" gorm:"primaryKey"`
	ResourceId      string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_agent_def_version;not null"`
	ResourceVersion string    `json:"resource_version" gorm:"type:varchar(64);uniqueIndex:idx_ap_agent_def_version;not null"`
	CliType         string    `json:"cli_type" gorm:"type:varchar(32);not null;default:'opencode'"`
	Name            string    `json:"name" gorm:"type:varchar(255)"`
	Description     string    `json:"description" gorm:"type:text"`
	CategoriesJSON  string    `json:"categories_json" gorm:"type:text"`
	Avatar          string    `json:"avatar" gorm:"type:text"`
	Instructions    string    `json:"instructions" gorm:"type:text"`
	ModelConfigJSON string    `json:"model_config_json" gorm:"type:text"`
	PackagePath     string    `json:"package_path" gorm:"type:text"`
	PackageSha256   string    `json:"package_sha256" gorm:"type:varchar(64)"`
	PackageSize     int64     `json:"package_size" gorm:"not null;default:0"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (AgentDef) TableName() string {
	return "agent_platform_agent_defs"
}

func (d *AgentDef) BeforeCreate(tx *gorm.DB) error {
	return d.applyDefaultsAndValidate()
}

func (d *AgentDef) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return d.applyDefaultsAndValidate()
}

func (d *AgentDef) applyDefaultsAndValidate() error {
	d.ResourceId = strings.TrimSpace(d.ResourceId)
	d.ResourceVersion = strings.TrimSpace(d.ResourceVersion)
	d.CliType = strings.TrimSpace(strings.ToLower(d.CliType))
	d.Name = strings.TrimSpace(d.Name)
	d.Description = strings.TrimSpace(d.Description)
	d.CategoriesJSON = strings.TrimSpace(d.CategoriesJSON)
	d.Avatar = strings.TrimSpace(d.Avatar)
	d.Instructions = strings.TrimSpace(d.Instructions)
	d.ModelConfigJSON = strings.TrimSpace(d.ModelConfigJSON)
	d.PackagePath = strings.TrimSpace(d.PackagePath)
	d.PackageSha256 = strings.TrimSpace(d.PackageSha256)
	if d.ResourceId == "" || d.ResourceVersion == "" || d.Name == "" || !ValidAgentCliType(d.CliType) {
		return ErrInvalidAgentDefBody
	}
	if _, err := parseStoredAgentCategoriesJSON(d.CategoriesJSON); err != nil {
		return ErrInvalidAgentDefBody
	}
	return nil
}

func ValidAgentCliType(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == AgentCliTypeOpenCode || value == AgentCliTypeCodex
}

func ValidAgentCategory(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	for _, candidate := range allowedAgentCategories {
		if value == candidate {
			return true
		}
	}
	return false
}

func NormalizeAgentCategories(values []string) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" {
			continue
		}
		if !ValidAgentCategory(value) {
			return nil, ErrInvalidAgentDefBody
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil, ErrInvalidAgentDefBody
	}
	return out, nil
}

func DefaultAgentCategories() []string {
	return []string{AgentCategoryGeneral}
}

func (d *AgentDef) Categories() []string {
	categories, err := parseStoredAgentCategoriesJSON(d.CategoriesJSON)
	if err != nil || len(categories) == 0 {
		return DefaultAgentCategories()
	}
	return categories
}

func (d *AgentDef) SetCategories(values []string) error {
	categories, err := NormalizeAgentCategories(values)
	if err != nil {
		return err
	}
	data, err := common.Marshal(categories)
	if err != nil {
		return err
	}
	d.CategoriesJSON = string(data)
	return nil
}

func parseStoredAgentCategoriesJSON(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ErrInvalidAgentDefBody
	}
	var categories []string
	if err := common.Unmarshal([]byte(value), &categories); err != nil {
		return nil, ErrInvalidAgentDefBody
	}
	return NormalizeAgentCategories(categories)
}
