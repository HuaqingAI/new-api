package enterprise

import (
	"errors"
	"net/url"
	"strings"
	"time"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type DingTalkConfigService struct {
	db *gorm.DB
}

type DingTalkConfigInput struct {
	TenantId                  int
	CorpId                    string
	AppKey                    string
	AppSecret                 *string
	CallbackUrl               string
	SyncScope                 string
	LoginEnabled              bool
	SyncEnabled               bool
	AutoSyncOnLogin           *bool
	ScheduledFullSyncEnabled  *bool
	ScheduledFullSyncCron     string
	ScheduledFullSyncTimezone string
}

func NewDingTalkConfigService(db *gorm.DB) *DingTalkConfigService {
	return &DingTalkConfigService{db: db}
}

func (s *DingTalkConfigService) Get(tenantId int) (dtoenterprise.DingTalkConfigResponse, error) {
	var config entmodel.DingTalkConfig
	if err := s.db.Where("tenant_id = ?", tenantId).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dtoenterprise.DingTalkConfigResponse{
				TenantId:                  tenantId,
				ScheduledFullSyncCron:     DefaultDingTalkScheduleCron,
				ScheduledFullSyncTimezone: DefaultDingTalkScheduleTimezone,
				ScheduledFullSyncRevision: 1,
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
	autoSyncOnLogin := existing.AutoSyncOnLogin
	if input.AutoSyncOnLogin != nil {
		autoSyncOnLogin = *input.AutoSyncOnLogin
	}
	if errors.Is(err, gorm.ErrRecordNotFound) && input.AutoSyncOnLogin == nil {
		autoSyncOnLogin = input.SyncEnabled
	}
	scheduledEnabled := existing.ScheduledFullSyncEnabled
	if input.ScheduledFullSyncEnabled != nil {
		scheduledEnabled = *input.ScheduledFullSyncEnabled
	}
	scheduledCron := strings.TrimSpace(input.ScheduledFullSyncCron)
	if scheduledCron == "" {
		scheduledCron = strings.TrimSpace(existing.ScheduledFullSyncCron)
	}
	if scheduledCron == "" {
		scheduledCron = DefaultDingTalkScheduleCron
	}
	scheduledTimezone := strings.TrimSpace(input.ScheduledFullSyncTimezone)
	if scheduledTimezone == "" {
		scheduledTimezone = strings.TrimSpace(existing.ScheduledFullSyncTimezone)
	}
	if scheduledTimezone == "" {
		scheduledTimezone = DefaultDingTalkScheduleTimezone
	}
	if err := validateDingTalkConfig(input, appSecret, autoSyncOnLogin); err != nil {
		return dtoenterprise.DingTalkConfigResponse{}, err
	}
	if _, err := nextDingTalkScheduleTime(scheduledCron, scheduledTimezone, time.Now()); err != nil {
		return dtoenterprise.DingTalkConfigResponse{}, err
	}
	if scheduledEnabled {
		if !input.SyncEnabled {
			return dtoenterprise.DingTalkConfigResponse{}, ErrDingTalkSyncNotEnabled
		}
		if strings.TrimSpace(appSecret) == "" {
			return dtoenterprise.DingTalkConfigResponse{}, ErrDingTalkMissingCredentials
		}
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		config := entmodel.DingTalkConfig{
			TenantId:                   input.TenantId,
			CorpId:                     input.CorpId,
			AppKey:                     input.AppKey,
			AppSecret:                  appSecret,
			CallbackUrl:                input.CallbackUrl,
			SyncScope:                  input.SyncScope,
			LoginEnabled:               input.LoginEnabled,
			SyncEnabled:                input.SyncEnabled,
			AutoSyncOnLogin:            autoSyncOnLogin,
			ScheduledFullSyncEnabled:   scheduledEnabled,
			ScheduledFullSyncCron:      scheduledCron,
			ScheduledFullSyncTimezone:  scheduledTimezone,
			ScheduledFullSyncRevision:  1,
			ScheduledFullSyncUpdatedAt: time.Now().Unix(),
		}
		if scheduledEnabled {
			next, _ := nextDingTalkScheduleTime(scheduledCron, scheduledTimezone, time.Now())
			config.ScheduledFullSyncNextRunAt = next.Unix()
		}
		if err := s.db.Create(&config).Error; err != nil {
			return dtoenterprise.DingTalkConfigResponse{}, err
		}
		ClearDingTalkScopeCache(input.TenantId)
		return mapDingTalkConfig(config), nil
	}

	updates := map[string]any{
		"corp_id":                      input.CorpId,
		"app_key":                      input.AppKey,
		"callback_url":                 input.CallbackUrl,
		"sync_scope":                   input.SyncScope,
		"login_enabled":                input.LoginEnabled,
		"sync_enabled":                 input.SyncEnabled,
		"auto_sync_on_login":           autoSyncOnLogin,
		"scheduled_full_sync_enabled":  scheduledEnabled,
		"scheduled_full_sync_cron":     scheduledCron,
		"scheduled_full_sync_timezone": scheduledTimezone,
	}
	if existing.ScheduledFullSyncRevision <= 0 {
		existing.ScheduledFullSyncRevision = 1
	}
	if existing.ScheduledFullSyncEnabled != scheduledEnabled || existing.ScheduledFullSyncCron != scheduledCron || existing.ScheduledFullSyncTimezone != scheduledTimezone {
		existing.ScheduledFullSyncRevision++
		updates["scheduled_full_sync_revision"] = existing.ScheduledFullSyncRevision
		updates["scheduled_full_sync_updated_at"] = time.Now().Unix()
		if scheduledEnabled {
			next, _ := nextDingTalkScheduleTime(scheduledCron, scheduledTimezone, time.Now())
			updates["scheduled_full_sync_next_run_at"] = next.Unix()
		} else {
			updates["scheduled_full_sync_next_run_at"] = 0
		}
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
	ClearDingTalkScopeCache(input.TenantId)
	return mapDingTalkConfig(existing), nil
}

func normalizeDingTalkConfigInput(input DingTalkConfigInput) DingTalkConfigInput {
	input.CorpId = strings.TrimSpace(input.CorpId)
	input.AppKey = strings.TrimSpace(input.AppKey)
	input.CallbackUrl = strings.TrimSpace(input.CallbackUrl)
	input.SyncScope = strings.TrimSpace(input.SyncScope)
	input.ScheduledFullSyncCron = strings.TrimSpace(input.ScheduledFullSyncCron)
	input.ScheduledFullSyncTimezone = strings.TrimSpace(input.ScheduledFullSyncTimezone)
	return input
}

func validateDingTalkConfig(input DingTalkConfigInput, appSecret string, autoSyncOnLogin bool) error {
	if input.CallbackUrl != "" && !isValidDingTalkCallbackURL(input.CallbackUrl) {
		return ErrDingTalkInvalidCallbackURL
	}
	if !input.LoginEnabled && !input.SyncEnabled && !autoSyncOnLogin {
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
		Id:                          config.Id,
		TenantId:                    config.TenantId,
		CorpId:                      config.CorpId,
		AppKey:                      config.AppKey,
		CallbackUrl:                 config.CallbackUrl,
		SyncScope:                   config.SyncScope,
		LoginEnabled:                config.LoginEnabled,
		SyncEnabled:                 config.SyncEnabled,
		AutoSyncOnLogin:             config.AutoSyncOnLogin,
		ScheduledFullSyncEnabled:    config.ScheduledFullSyncEnabled,
		ScheduledFullSyncCron:       config.ScheduledFullSyncCron,
		ScheduledFullSyncTimezone:   config.ScheduledFullSyncTimezone,
		ScheduledFullSyncNextRunAt:  config.ScheduledFullSyncNextRunAt,
		ScheduledFullSyncLastRunAt:  config.ScheduledFullSyncLastRunAt,
		ScheduledFullSyncLastTaskId: config.ScheduledFullSyncLastTaskId,
		ScheduledFullSyncLastStatus: config.ScheduledFullSyncLastStatus,
		ScheduledFullSyncLastError:  config.ScheduledFullSyncLastError,
		ScheduledFullSyncRevision:   config.ScheduledFullSyncRevision,
		ScheduledFullSyncUpdatedAt:  config.ScheduledFullSyncUpdatedAt,
		HasAppSecret:                strings.TrimSpace(config.AppSecret) != "",
		CreatedAt:                   config.CreatedAt,
		UpdatedAt:                   config.UpdatedAt,
	}
}
