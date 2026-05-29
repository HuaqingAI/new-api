package enterprise

import (
	"errors"
	"net/url"
	"strings"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type DingTalkConfigService struct {
	db *gorm.DB
}

type DingTalkConfigInput struct {
	TenantId     int
	CorpId       string
	AppKey       string
	AppSecret    *string
	CallbackUrl  string
	SyncScope    string
	LoginEnabled bool
	SyncEnabled  bool
}

func NewDingTalkConfigService(db *gorm.DB) *DingTalkConfigService {
	return &DingTalkConfigService{db: db}
}

func (s *DingTalkConfigService) Get(tenantId int) (dtoenterprise.DingTalkConfigResponse, error) {
	var config entmodel.DingTalkConfig
	if err := s.db.Where("tenant_id = ?", tenantId).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dtoenterprise.DingTalkConfigResponse{
				TenantId: tenantId,
			}, nil
		}
		return dtoenterprise.DingTalkConfigResponse{}, err
	}
	return mapDingTalkConfig(config), nil
}

func (s *DingTalkConfigService) Save(input DingTalkConfigInput) (dtoenterprise.DingTalkConfigResponse, error) {
	input = normalizeDingTalkConfigInput(input)

	var existing entmodel.DingTalkConfig
	err := s.db.Where("tenant_id = ?", input.TenantId).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return dtoenterprise.DingTalkConfigResponse{}, err
	}

	appSecret := existing.AppSecret
	hasSecretUpdate := input.AppSecret != nil && strings.TrimSpace(*input.AppSecret) != ""
	if hasSecretUpdate {
		appSecret = strings.TrimSpace(*input.AppSecret)
	}
	if err := validateDingTalkConfig(input, appSecret); err != nil {
		return dtoenterprise.DingTalkConfigResponse{}, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		config := entmodel.DingTalkConfig{
			TenantId:     input.TenantId,
			CorpId:       input.CorpId,
			AppKey:       input.AppKey,
			AppSecret:    appSecret,
			CallbackUrl:  input.CallbackUrl,
			SyncScope:    input.SyncScope,
			LoginEnabled: input.LoginEnabled,
			SyncEnabled:  input.SyncEnabled,
		}
		if err := s.db.Create(&config).Error; err != nil {
			return dtoenterprise.DingTalkConfigResponse{}, err
		}
		return mapDingTalkConfig(config), nil
	}

	updates := map[string]any{
		"corp_id":       input.CorpId,
		"app_key":       input.AppKey,
		"callback_url":  input.CallbackUrl,
		"sync_scope":    input.SyncScope,
		"login_enabled": input.LoginEnabled,
		"sync_enabled":  input.SyncEnabled,
	}
	if hasSecretUpdate {
		updates["app_secret"] = appSecret
	}
	if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
		return dtoenterprise.DingTalkConfigResponse{}, err
	}
	if err := s.db.Where("tenant_id = ?", input.TenantId).First(&existing).Error; err != nil {
		return dtoenterprise.DingTalkConfigResponse{}, err
	}
	return mapDingTalkConfig(existing), nil
}

func normalizeDingTalkConfigInput(input DingTalkConfigInput) DingTalkConfigInput {
	input.CorpId = strings.TrimSpace(input.CorpId)
	input.AppKey = strings.TrimSpace(input.AppKey)
	input.CallbackUrl = strings.TrimSpace(input.CallbackUrl)
	input.SyncScope = strings.TrimSpace(input.SyncScope)
	return input
}

func validateDingTalkConfig(input DingTalkConfigInput, appSecret string) error {
	if input.CallbackUrl != "" && !isValidDingTalkCallbackURL(input.CallbackUrl) {
		return ErrDingTalkInvalidCallbackURL
	}
	if !input.LoginEnabled && !input.SyncEnabled {
		return nil
	}
	if input.CorpId == "" || input.AppKey == "" || strings.TrimSpace(appSecret) == "" {
		return ErrDingTalkMissingCredentials
	}
	if input.CallbackUrl == "" || !isValidDingTalkCallbackURL(input.CallbackUrl) {
		return ErrDingTalkInvalidCallbackURL
	}
	return nil
}

func isValidDingTalkCallbackURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed == nil {
		return false
	}
	return parsed.Scheme == "https" && parsed.Host != ""
}

func mapDingTalkConfig(config entmodel.DingTalkConfig) dtoenterprise.DingTalkConfigResponse {
	return dtoenterprise.DingTalkConfigResponse{
		Id:           config.Id,
		TenantId:     config.TenantId,
		CorpId:       config.CorpId,
		AppKey:       config.AppKey,
		CallbackUrl:  config.CallbackUrl,
		SyncScope:    config.SyncScope,
		LoginEnabled: config.LoginEnabled,
		SyncEnabled:  config.SyncEnabled,
		HasAppSecret: strings.TrimSpace(config.AppSecret) != "",
		CreatedAt:    config.CreatedAt,
		UpdatedAt:    config.UpdatedAt,
	}
}
