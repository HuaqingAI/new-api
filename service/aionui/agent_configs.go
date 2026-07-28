package aionui

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
)

const (
	CliTypeOpenCode        = "opencode"
	AgentConfigUrlTypeFile = "file"
	defaultAgentVersion    = "1.0.0"
	defaultAgentAvatar     = "🤖"
)

var ErrUnsupportedCliType = errors.New("unsupported cli_type")

type AgentConfigService struct {
	agentsDir string
	now       func() time.Time
}

func NewAgentConfigService(agentsDir string) *AgentConfigService {
	return &AgentConfigService{
		agentsDir: agentsDir,
		now:       time.Now,
	}
}

func NewDefaultAgentConfigService() *AgentConfigService {
	return NewAgentConfigService(DefaultAgentConfigDir())
}

func DefaultAgentConfigDir() string {
	if configured := strings.TrimSpace(os.Getenv("AIONUI_AGENT_CONFIG_DIR")); configured != "" {
		return configured
	}
	return filepath.Join(".", "agents")
}

func ValidateAgentConfigRequest(cliType string) error {
	if strings.TrimSpace(cliType) == "" || cliType == CliTypeOpenCode {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedCliType, cliType)
}

func (s *AgentConfigService) List(email string, cliType string) (dtoaionui.AgentConfigsResponse, error) {
	if strings.TrimSpace(cliType) == "" {
		cliType = CliTypeOpenCode
	}
	if cliType != CliTypeOpenCode {
		return dtoaionui.AgentConfigsResponse{}, ErrUnsupportedCliType
	}

	agents, err := s.scanZipAgents()
	if err != nil {
		return dtoaionui.AgentConfigsResponse{}, err
	}

	return dtoaionui.AgentConfigsResponse{
		UserEmail: strings.TrimSpace(strings.ToLower(email)),
		CliType:   CliTypeOpenCode,
		Revision:  s.now().UTC().Format(time.RFC3339),
		Agents:    agents,
	}, nil
}

func (s *AgentConfigService) scanZipAgents() ([]dtoaionui.AgentConfigItem, error) {
	entries, err := os.ReadDir(s.agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []dtoaionui.AgentConfigItem{}, nil
		}
		return nil, err
	}

	manifest, err := s.loadManifest()
	if err != nil {
		return nil, err
	}
	items := make([]dtoaionui.AgentConfigItem, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~") || !strings.EqualFold(filepath.Ext(name), ".zip") {
			continue
		}
		fullPath, err := filepath.Abs(filepath.Join(s.agentsDir, name))
		if err != nil {
			return nil, err
		}
		sum, err := fileSHA256(fullPath)
		if err != nil {
			return nil, err
		}
		defaultId := normalizeAgentConfigID(strings.TrimSuffix(name, filepath.Ext(name)))
		meta := manifest[name]
		id := strings.TrimSpace(meta.Id)
		if id == "" {
			id = defaultId
		}
		agentName := strings.TrimSpace(meta.Name)
		if agentName == "" {
			agentName = id
		}
		version := strings.TrimSpace(meta.Version)
		if version == "" {
			version = defaultAgentVersion
		}
		avatar := strings.TrimSpace(meta.Avatar)
		if avatar == "" {
			avatar = defaultAgentAvatar
		}
		description := strings.TrimSpace(meta.Description)
		if description == "" {
			description = "Synced from new-api"
		}
		items = append(items, dtoaionui.AgentConfigItem{
			Id:          id,
			Url:         fileURL(fullPath),
			UrlType:     AgentConfigUrlTypeFile,
			Version:     version,
			Name:        agentName,
			Description: description,
			Avatar:      avatar,
			Sha256:      sum,
		})
	}

	sort.Slice(items, func(i int, j int) bool {
		return items[i].Id < items[j].Id
	})
	return items, nil
}

type agentConfigManifest struct {
	Agents []agentConfigManifestItem `json:"agents"`
}

type agentConfigManifestItem struct {
	File        string `json:"file"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
	Version     string `json:"version"`
}

func (s *AgentConfigService) loadManifest() (map[string]agentConfigManifestItem, error) {
	path := filepath.Join(s.agentsDir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]agentConfigManifestItem{}, nil
		}
		return nil, err
	}

	var manifest agentConfigManifest
	if err := common.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	items := make(map[string]agentConfigManifestItem, len(manifest.Agents))
	for _, agent := range manifest.Agents {
		file := strings.TrimSpace(agent.File)
		if file == "" {
			continue
		}
		items[file] = agent
	}
	return items, nil
}

func normalizeAgentConfigID(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	builder := strings.Builder{}
	lastDash := false
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	id := strings.Trim(builder.String(), "-")
	if id == "" {
		return "agent"
	}
	return id
}

func fileURL(path string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	return u.String()
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
