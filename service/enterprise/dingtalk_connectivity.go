package enterprise

import (
	"context"
	"errors"
	"strings"
	"time"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type DingTalkConnectivityService struct {
	db     *gorm.DB
	client *DingTalkClient
}

func NewDingTalkConnectivityService(db *gorm.DB, client *DingTalkClient) *DingTalkConnectivityService {
	if client == nil {
		client = NewDingTalkClient()
	}
	return &DingTalkConnectivityService{db: db, client: client}
}

func (s *DingTalkConnectivityService) Test(ctx context.Context, tenantId int) (dtoenterprise.DingTalkConnectivityResponse, error) {
	checkedAt := time.Now().Unix()
	var config entmodel.DingTalkConfig
	if err := s.db.Where("tenant_id = ?", tenantId).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dingTalkConnectivityResult(tenantId, dtoenterprise.DingTalkConnectivityAuthInvalidCredentials, "configuration", "missing_configuration", 0, checkedAt), nil
		}
		return dtoenterprise.DingTalkConnectivityResponse{}, err
	}

	if strings.TrimSpace(config.CallbackUrl) == "" || !isValidDingTalkCallbackURL(config.CallbackUrl) {
		return dingTalkConnectivityResult(tenantId, dtoenterprise.DingTalkConnectivityCallbackMisconfigured, "callback", "callback_url_invalid", 0, checkedAt), nil
	}
	if strings.TrimSpace(config.CorpId) == "" || strings.TrimSpace(config.AppKey) == "" || strings.TrimSpace(config.AppSecret) == "" {
		return dingTalkConnectivityResult(tenantId, dtoenterprise.DingTalkConnectivityAuthInvalidCredentials, "access_token", "missing_credentials", 0, checkedAt), nil
	}

	accessToken, err := s.client.GetAccessToken(ctx, config.AppKey, config.AppSecret)
	if err != nil {
		if apiErr, ok := dingTalkAPIError(err); ok {
			code := dtoenterprise.DingTalkConnectivityAuthInvalidCredentials
			if apiErr.Network {
				code = dtoenterprise.DingTalkConnectivityNetworkUnreachable
			}
			return dingTalkConnectivityResult(tenantId, code, apiErr.Stage, stableDingTalkSummary(apiErr, "token_invalid_credentials"), apiErr.HTTPStatus, checkedAt), nil
		}
		return dtoenterprise.DingTalkConnectivityResponse{}, err
	}

	if err := s.client.ProbeAddressBookPermission(ctx, accessToken); err != nil {
		if apiErr, ok := dingTalkAPIError(err); ok {
			code := dtoenterprise.DingTalkConnectivityPermissionInsufficient
			if apiErr.Network {
				code = dtoenterprise.DingTalkConnectivityNetworkUnreachable
			}
			return dingTalkConnectivityResult(tenantId, code, apiErr.Stage, stableDingTalkSummary(apiErr, "address_book_permission_denied"), apiErr.HTTPStatus, checkedAt), nil
		}
		return dtoenterprise.DingTalkConnectivityResponse{}, err
	}

	return dingTalkConnectivityResult(tenantId, dtoenterprise.DingTalkConnectivityAuthSuccess, "address_book_probe", "address_book_probe_ok", 0, checkedAt), nil
}

func dingTalkConnectivityResult(tenantId int, code string, stage string, summary string, httpStatus int, checkedAt int64) dtoenterprise.DingTalkConnectivityResponse {
	return dtoenterprise.DingTalkConnectivityResponse{
		TenantId:   tenantId,
		Code:       code,
		Stage:      stage,
		Summary:    summary,
		HTTPStatus: httpStatus,
		CheckedAt:  checkedAt,
	}
}

func stableDingTalkSummary(apiErr *DingTalkAPIError, fallback string) string {
	if apiErr == nil {
		return fallback
	}
	if strings.TrimSpace(apiErr.Summary) != "" {
		return strings.TrimSpace(apiErr.Summary)
	}
	return fallback
}
