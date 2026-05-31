package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ExposureVisibilityHidden  = "hidden"
	ExposureVisibilityVisible = "visible"
	ExposureVisibilityRevoked = "revoked"

	ExposureCallableDisabled = "disabled"
	ExposureCallableEnabled  = "enabled"
	ExposureCallableRevoked  = "revoked"

	defaultExposureTTLSeconds = 300
)

var (
	ErrInvalidExposureBody  = errors.New("agent platform exposure body invalid")
	ErrInvalidExposureState = errors.New("agent platform exposure state invalid")
)

var allowedExposureVisibilityStates = map[string]struct{}{
	ExposureVisibilityHidden:  {},
	ExposureVisibilityVisible: {},
	ExposureVisibilityRevoked: {},
}

var allowedExposureCallableStates = map[string]struct{}{
	ExposureCallableDisabled: {},
	ExposureCallableEnabled:  {},
	ExposureCallableRevoked:  {},
}

type Exposure struct {
	Id                  int        `json:"id" gorm:"primaryKey"`
	ExposureId          string     `json:"exposure_id" gorm:"type:varchar(40);uniqueIndex:idx_ap_exposure_id;not null"`
	ResourceId          string     `json:"resource_id" gorm:"type:varchar(40);index:idx_ap_exposure_resource;not null"`
	ResourceVersion     string     `json:"resource_version" gorm:"type:varchar(64);index:idx_ap_exposure_resource;not null"`
	ClientKey           string     `json:"client_key" gorm:"type:varchar(128);index:idx_ap_exposure_target;not null"`
	ClientScope         string     `json:"client_scope" gorm:"type:varchar(128);index:idx_ap_exposure_target;not null;default:''"`
	VisibilityState     string     `json:"visibility_state" gorm:"type:varchar(16);index:idx_ap_exposure_visibility;not null"`
	CallableState       string     `json:"callable_state" gorm:"type:varchar(16);index:idx_ap_exposure_callable;not null"`
	FreshnessTTLSeconds int        `json:"freshness_ttl_seconds" gorm:"not null"`
	ETag                string     `json:"etag" gorm:"type:varchar(128);not null"`
	ExtensionsJSON      string     `json:"extensions_json" gorm:"column:extensions_json;type:text"`
	PublishedAt         *time.Time `json:"published_at"`
	RevokedAt           *time.Time `json:"revoked_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (Exposure) TableName() string {
	return "agent_platform_resource_exposures"
}

func (e *Exposure) BeforeCreate(tx *gorm.DB) error {
	return e.applyDefaultsAndValidate()
}

func (e *Exposure) BeforeSave(tx *gorm.DB) error {
	if isPartialUpdate(tx) {
		return nil
	}
	return e.applyDefaultsAndValidate()
}

func (e *Exposure) applyDefaultsAndValidate() error {
	e.ExposureId = strings.TrimSpace(e.ExposureId)
	e.ResourceId = strings.TrimSpace(e.ResourceId)
	e.ResourceVersion = strings.TrimSpace(e.ResourceVersion)
	e.ClientKey = strings.TrimSpace(e.ClientKey)
	e.ClientScope = strings.TrimSpace(e.ClientScope)
	e.VisibilityState = strings.TrimSpace(strings.ToLower(e.VisibilityState))
	e.CallableState = strings.TrimSpace(strings.ToLower(e.CallableState))
	e.ETag = strings.TrimSpace(e.ETag)
	e.ExtensionsJSON = strings.TrimSpace(e.ExtensionsJSON)

	if e.ExposureId == "" {
		exposureID, err := GenerateExposureID()
		if err != nil {
			return err
		}
		e.ExposureId = exposureID
	}
	if e.VisibilityState == "" {
		e.VisibilityState = ExposureVisibilityHidden
	}
	if e.CallableState == "" {
		e.CallableState = ExposureCallableDisabled
	}
	if e.FreshnessTTLSeconds <= 0 {
		e.FreshnessTTLSeconds = defaultExposureTTLSeconds
	}
	if e.ETag == "" {
		etag, err := GenerateExposureETag()
		if err != nil {
			return err
		}
		e.ETag = etag
	}
	if e.PublishedAt == nil {
		now := time.Now().UTC()
		e.PublishedAt = &now
	}

	if e.ResourceId == "" || e.ResourceVersion == "" || e.ClientKey == "" {
		return ErrInvalidExposureBody
	}
	if _, ok := allowedExposureVisibilityStates[e.VisibilityState]; !ok {
		return ErrInvalidExposureState
	}
	if _, ok := allowedExposureCallableStates[e.CallableState]; !ok {
		return ErrInvalidExposureState
	}
	if e.VisibilityState != ExposureVisibilityVisible && e.CallableState == ExposureCallableEnabled {
		return ErrInvalidExposureState
	}
	if e.VisibilityState == ExposureVisibilityRevoked && e.RevokedAt == nil {
		now := time.Now().UTC()
		e.RevokedAt = &now
	}
	if e.CallableState == ExposureCallableRevoked && e.RevokedAt == nil {
		now := time.Now().UTC()
		e.RevokedAt = &now
	}
	return nil
}

func GenerateExposureID() (string, error) {
	key, err := common.GenerateRandomCharsKey(31)
	if err != nil {
		return "", err
	}
	return "exp_" + key, nil
}

func GenerateExposureETag() (string, error) {
	return common.GenerateRandomCharsKey(24)
}
