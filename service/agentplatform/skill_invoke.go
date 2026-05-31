package agentplatform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrSkillInvokeTimeout        = errors.New("agent platform skill invoke timeout")
	ErrSkillInvokeUpstreamFailed = errors.New("agent platform skill invoke upstream failed")
)

type SkillInvokeInput struct {
	ClientID   string
	ResourceID string
	Payload    []byte
}

type SkillInvokeResult struct {
	ResourceID      string
	ResourceVersion string
	ContractVersion string
	Output          []byte
}

type skillHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type SkillInvokeService struct {
	db         *gorm.DB
	httpClient skillHTTPDoer
}

func NewSkillInvokeService(db *gorm.DB) *SkillInvokeService {
	return &SkillInvokeService{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *SkillInvokeService) WithHTTPClient(client skillHTTPDoer) *SkillInvokeService {
	if client != nil {
		s.httpClient = client
	}
	return s
}

func (s *SkillInvokeService) Invoke(input SkillInvokeInput) (SkillInvokeResult, error) {
	if s == nil || s.db == nil || s.httpClient == nil {
		return SkillInvokeResult{}, ErrOpenCapabilityContractInvalid
	}
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	if input.ClientID == "" || input.ResourceID == "" || len(input.Payload) == 0 {
		return SkillInvokeResult{}, ErrOpenCapabilityContractInvalid
	}

	discovery := NewDiscoveryService(s.db)
	detail, err := discovery.Detail(input.ClientID, input.ResourceID)
	if err != nil {
		return SkillInvokeResult{}, err
	}
	if detail.ResourceType != apmodel.ResourceTypeSkill || detail.CallableState != apmodel.ExposureCallableEnabled {
		return SkillInvokeResult{}, ErrOpenCapabilityPermissionDenied
	}

	var skillDef apmodel.SkillDef
	if err := s.db.Where("resource_id = ? AND resource_version = ?", input.ResourceID, detail.ResourceVersion).First(&skillDef).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SkillInvokeResult{}, ErrOpenCapabilityContractInvalid
		}
		return SkillInvokeResult{}, err
	}

	binding, err := parseSkillBindingConfig(skillDef.BindingConfigJSON)
	if err != nil {
		return SkillInvokeResult{}, ErrOpenCapabilityContractInvalid
	}
	timeout := time.Duration(skillDef.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		return SkillInvokeResult{}, ErrOpenCapabilityContractInvalid
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, binding.Method, binding.URL, bytes.NewReader(input.Payload))
	if err != nil {
		return SkillInvokeResult{}, ErrOpenCapabilityContractInvalid
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range binding.Headers {
		req.Header.Set(key, value)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return SkillInvokeResult{}, ErrSkillInvokeTimeout
		}
		return SkillInvokeResult{}, fmt.Errorf("%w: %v", ErrSkillInvokeUpstreamFailed, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return SkillInvokeResult{}, fmt.Errorf("%w: %v", ErrSkillInvokeUpstreamFailed, readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return SkillInvokeResult{}, fmt.Errorf("%w: status=%d body=%s", ErrSkillInvokeUpstreamFailed, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return SkillInvokeResult{
		ResourceID:      detail.ResourceID,
		ResourceVersion: detail.ResourceVersion,
		ContractVersion: detail.ContractVersion,
		Output:          body,
	}, nil
}

type skillBindingConfig struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

func parseSkillBindingConfig(raw string) (skillBindingConfig, error) {
	var config skillBindingConfig
	if strings.TrimSpace(raw) == "" {
		return config, ErrOpenCapabilityContractInvalid
	}
	if err := common.UnmarshalJsonStr(raw, &config); err != nil {
		return config, err
	}
	config.Method = strings.TrimSpace(strings.ToUpper(config.Method))
	config.URL = strings.TrimSpace(config.URL)
	if config.Method == "" {
		config.Method = http.MethodPost
	}
	if config.URL == "" {
		return config, ErrOpenCapabilityContractInvalid
	}
	if config.Method != http.MethodPost && config.Method != http.MethodPut {
		return config, ErrOpenCapabilityContractInvalid
	}
	return config, nil
}
