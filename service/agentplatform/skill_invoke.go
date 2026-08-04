package agentplatform

import (
	"errors"
	"net/http"
	"strings"

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
	db *gorm.DB
}

func NewSkillInvokeService(db *gorm.DB) *SkillInvokeService {
	return &SkillInvokeService{db: db}
}

func (s *SkillInvokeService) WithHTTPClient(_ skillHTTPDoer) *SkillInvokeService {
	return s
}

func (s *SkillInvokeService) Invoke(input SkillInvokeInput) (SkillInvokeResult, error) {
	if s == nil || s.db == nil {
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
	return SkillInvokeResult{}, ErrOpenCapabilityContractInvalid
}
