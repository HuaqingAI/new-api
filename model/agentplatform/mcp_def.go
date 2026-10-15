package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var ErrInvalidMcpDefBody = errors.New("agent platform mcp def body invalid")

type McpDef struct {
	Id         int       `json:"id" gorm:"primaryKey"`
	ResourceId string    `json:"resource_id" gorm:"type:varchar(40);uniqueIndex;not null"`
	ConfigJSON string    `json:"config_json" gorm:"column:config_json;type:text;not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (McpDef) TableName() string {
	return "agent_platform_mcp_defs"
}

func (d *McpDef) BeforeCreate(tx *gorm.DB) error {
	return d.applyDefaultsAndValidate()
}

func (d *McpDef) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return d.applyDefaultsAndValidate()
}

func (d *McpDef) applyDefaultsAndValidate() error {
	d.ResourceId = strings.TrimSpace(d.ResourceId)
	d.ConfigJSON = strings.TrimSpace(d.ConfigJSON)
	if d.ResourceId == "" || !validMcpConfigJSON(d.ConfigJSON) {
		return ErrInvalidMcpDefBody
	}
	return nil
}

func validMcpConfigJSON(raw string) bool {
	var payload struct {
		McpServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return false
	}
	if len(payload.McpServers) == 0 {
		return false
	}
	for name, server := range payload.McpServers {
		if strings.TrimSpace(name) == "" || len(server) == 0 {
			return false
		}
		serverType := strings.TrimSpace(strings.ToLower(common.Interface2String(server["type"])))
		switch serverType {
		case "", "stdio", "local":
			if strings.TrimSpace(common.Interface2String(server["command"])) == "" {
				return false
			}
		case "sse", "streamablehttp", "streamable_http", "http", "remote":
			if strings.TrimSpace(common.Interface2String(server["url"])) == "" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func McpDefConfigValidForService(raw string) bool {
	return validMcpConfigJSON(raw)
}
