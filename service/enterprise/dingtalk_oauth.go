package enterprise

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const (
	DingTalkIdentityStatusActive   = 1
	DingTalkIdentityStatusDisabled = 2

	DingTalkOAuthLoginStatusExisting = "existing"
	DingTalkOAuthLoginStatusCreated  = "created"
	DingTalkOAuthLoginStatusBound    = "bound"
)

type DingTalkOAuthClient interface {
	DingTalkDirectoryClient
	ExchangeOAuthCode(ctx context.Context, appKey string, appSecret string, code string) (DingTalkOAuthToken, error)
	GetOAuthUserInfo(ctx context.Context, userAccessToken string) (DingTalkOAuthUserInfo, error)
}

type DingTalkOAuthIdentity struct {
	UnionId          string
	OpenId           string
	ExternalUserId   string
	Name             string
	Email            string
	Mobile           string
	Active           *bool
	DepartmentIds    []int64
	ScopeVerified    bool
	oauthVerified    bool
	verifiedByServer bool
}

type DingTalkOAuthService struct {
	db     *gorm.DB
	client DingTalkOAuthClient
}

type DingTalkOAuthResult struct {
	User              *model.User
	Binding           entmodel.DingTalkIdentity
	Memberships       []UserDepartmentItem
	DepartmentCount   int
	LoginStatus       string
	UsedLocalSnapshot bool
	AutoSynced        bool
}

func NewDingTalkOAuthService(db *gorm.DB, client DingTalkOAuthClient) *DingTalkOAuthService {
	if client == nil {
		client = NewDingTalkClient()
	}
	return &DingTalkOAuthService{db: db, client: client}
}

func (s *DingTalkOAuthService) ResolveIdentity(ctx context.Context, tenantId int, code string) (DingTalkOAuthIdentity, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return DingTalkOAuthIdentity{}, ErrDingTalkOAuthCodeMissing
	}
	config, err := s.getEnabledConfig(tenantId)
	if err != nil {
		return DingTalkOAuthIdentity{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, dingTalkDirectoryVerificationTimeout)
	defer cancel()

	token, err := s.client.ExchangeOAuthCode(ctx, config.AppKey, config.AppSecret, code)
	if err != nil {
		logDingTalkOAuthProviderError("oauth_token", err)
		return DingTalkOAuthIdentity{}, ErrDingTalkOAuthProviderFailed
	}
	identity := DingTalkOAuthIdentity{
		UnionId: strings.TrimSpace(token.UnionId),
		OpenId:  strings.TrimSpace(token.OpenId),
	}
	oauthUser, err := s.client.GetOAuthUserInfo(ctx, token.AccessToken)
	if err != nil {
		logDingTalkOAuthProviderError("oauth_userinfo", err)
	} else {
		identity.UnionId = firstNonEmpty(oauthUser.UnionId, identity.UnionId)
		identity.OpenId = firstNonEmpty(oauthUser.OpenId, identity.OpenId)
	}
	if identity.UnionId == "" && identity.OpenId == "" {
		return DingTalkOAuthIdentity{}, ErrDingTalkOAuthIdentityMissing
	}
	if identity.UnionId == "" {
		return DingTalkOAuthIdentity{}, ErrDingTalkOAuthEmployeeNotFound
	}
	identity.oauthVerified = true
	return identity, nil
}

func logDingTalkOAuthProviderError(stage string, err error) {
	var apiErr *DingTalkAPIError
	if errors.As(err, &apiErr) {
		common.SysError(fmt.Sprintf("[DingTalk OAuth] %s failed: stage=%s summary=%s http_status=%d err_code=%d network=%t", stage, apiErr.Stage, apiErr.Summary, apiErr.HTTPStatus, apiErr.ErrCode, apiErr.Network))
		return
	}
	common.SysError(fmt.Sprintf("[DingTalk OAuth] %s failed", stage))
}

func (s *DingTalkOAuthService) LoginWithIdentity(ctx context.Context, tenantId int, identity DingTalkOAuthIdentity, affiliateCode string) (DingTalkOAuthResult, error) {
	config, err := s.getEnabledConfig(tenantId)
	if err != nil {
		return DingTalkOAuthResult{}, err
	}
	identity = normalizeDingTalkOAuthIdentity(identity)
	if !isOAuthVerifiedDingTalkIdentity(identity) {
		return DingTalkOAuthResult{}, ErrDingTalkOAuthEmployeeNotFound
	}

	result, foundBinding, err := s.loginWithLocalSnapshot(ctx, config, identity)
	if err == nil {
		return result, nil
	}
	if !errors.Is(err, ErrDingTalkOAuthOutOfScope) {
		return DingTalkOAuthResult{}, err
	}

	autoResult, err := NewDingTalkDirectoryService(s.db, s.client).EnsureEmployeeSynced(ctx, DingTalkAutoSyncInput{
		TenantId: tenantId,
		Identity: identity,
	})
	if err != nil {
		return DingTalkOAuthResult{}, err
	}
	if autoResult.User == nil {
		return DingTalkOAuthResult{}, ErrDingTalkOAuthOutOfScope
	}
	memberships, err := s.readLocalDingTalkMembershipSnapshot(config.TenantId, autoResult.User.Id, autoResult.Binding.ExternalUserId)
	if err != nil {
		return DingTalkOAuthResult{}, err
	}
	loginStatus := DingTalkOAuthLoginStatusBound
	if autoResult.CreatedUser {
		loginStatus = DingTalkOAuthLoginStatusCreated
	} else if foundBinding {
		loginStatus = DingTalkOAuthLoginStatusExisting
	}
	return DingTalkOAuthResult{
		User:            autoResult.User,
		Binding:         autoResult.Binding,
		Memberships:     memberships,
		DepartmentCount: len(memberships),
		LoginStatus:     loginStatus,
		AutoSynced:      true,
	}, nil
}

func (s *DingTalkOAuthService) loginWithLocalSnapshot(ctx context.Context, config entmodel.DingTalkConfig, identity DingTalkOAuthIdentity) (DingTalkOAuthResult, bool, error) {
	var binding entmodel.DingTalkIdentity
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND identity_key = ?", config.TenantId, dingTalkIdentityKey(identity)).First(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DingTalkOAuthResult{}, false, ErrDingTalkOAuthOutOfScope
		}
		return DingTalkOAuthResult{}, false, err
	}
	if binding.Status != DingTalkIdentityStatusActive {
		return DingTalkOAuthResult{}, true, ErrDingTalkOAuthUserDisabled
	}
	var user model.User
	if err := s.db.WithContext(ctx).Where("id = ?", binding.UserId).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DingTalkOAuthResult{}, true, ErrDingTalkOAuthOutOfScope
		}
		return DingTalkOAuthResult{}, true, err
	}
	if user.Status != common.UserStatusEnabled {
		return DingTalkOAuthResult{}, true, ErrDingTalkOAuthUserDisabled
	}
	memberships, err := s.readLocalDingTalkMembershipSnapshot(config.TenantId, user.Id, binding.ExternalUserId)
	if err != nil {
		return DingTalkOAuthResult{}, true, err
	}
	updates := map[string]any{
		"corp_id":       config.CorpId,
		"last_login_at": time.Now().Unix(),
	}
	if identity.OpenId != "" {
		updates["open_id"] = identity.OpenId
	}
	if err := s.db.WithContext(ctx).Model(&binding).Updates(updates).Error; err != nil {
		return DingTalkOAuthResult{}, true, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", binding.Id).First(&binding).Error; err != nil {
		return DingTalkOAuthResult{}, true, err
	}
	return DingTalkOAuthResult{
		User:              &user,
		Binding:           binding,
		Memberships:       memberships,
		DepartmentCount:   len(memberships),
		LoginStatus:       DingTalkOAuthLoginStatusExisting,
		UsedLocalSnapshot: true,
	}, true, nil
}

func (s *DingTalkOAuthService) BindIdentityToUser(ctx context.Context, tenantId int, userId int, identity DingTalkOAuthIdentity) (entmodel.DingTalkIdentity, error) {
	if userId <= 0 {
		return entmodel.DingTalkIdentity{}, ErrDingTalkOAuthIdentityMissing
	}
	config, err := s.getEnabledConfig(tenantId)
	if err != nil {
		return entmodel.DingTalkIdentity{}, err
	}
	identity = normalizeDingTalkOAuthIdentity(identity)
	if !isOAuthVerifiedDingTalkIdentity(identity) {
		return entmodel.DingTalkIdentity{}, ErrDingTalkOAuthEmployeeNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, dingTalkDirectoryVerificationTimeout)
	defer cancel()
	identity, _, _, err = NewDingTalkDirectoryService(s.db, s.client).loadVerifiedEmployee(ctx, config, identity)
	if err != nil {
		return entmodel.DingTalkIdentity{}, err
	}

	var binding entmodel.DingTalkIdentity
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := model.LockForUpdate(tx).Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}
		if user.Status != common.UserStatusEnabled {
			return ErrDingTalkOAuthUserDisabled
		}
		if _, err := (&DingTalkOAuthService{db: tx, client: s.client}).readLocalDingTalkMembershipSnapshot(config.TenantId, user.Id, identity.ExternalUserId); err != nil {
			return err
		}

		existing, found, err := (&DingTalkOAuthService{db: tx, client: s.client}).findIdentityBinding(config.TenantId, identity)
		if err != nil {
			return err
		}
		if found && existing.UserId != userId {
			return ErrDingTalkOAuthBindingConflict
		}
		if found {
			binding = existing
			return tx.Model(&binding).Updates(map[string]any{
				"corp_id":          config.CorpId,
				"identity_key":     dingTalkIdentityKey(identity),
				"union_id":         identity.UnionId,
				"open_id":          identity.OpenId,
				"external_user_id": identity.ExternalUserId,
				"mobile":           identity.Mobile,
				"status":           DingTalkIdentityStatusActive,
				"last_login_at":    time.Now().Unix(),
			}).Error
		}

		binding = entmodel.DingTalkIdentity{
			TenantId:       config.TenantId,
			CorpId:         config.CorpId,
			IdentityKey:    dingTalkIdentityKey(identity),
			UnionId:        identity.UnionId,
			OpenId:         identity.OpenId,
			ExternalUserId: identity.ExternalUserId,
			Mobile:         identity.Mobile,
			UserId:         userId,
			Status:         DingTalkIdentityStatusActive,
			LastLoginAt:    time.Now().Unix(),
		}
		return tx.Create(&binding).Error
	})
	if err != nil {
		return entmodel.DingTalkIdentity{}, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", binding.Id).First(&binding).Error; err != nil {
		return entmodel.DingTalkIdentity{}, err
	}
	return binding, nil
}

func (s *DingTalkOAuthService) getEnabledConfig(tenantId int) (entmodel.DingTalkConfig, error) {
	var config entmodel.DingTalkConfig
	if err := s.db.Where("tenant_id = ?", tenantId).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entmodel.DingTalkConfig{}, ErrDingTalkConfigNotFound
		}
		return entmodel.DingTalkConfig{}, err
	}
	if !config.LoginEnabled {
		return entmodel.DingTalkConfig{}, ErrDingTalkOAuthNotEnabled
	}
	if strings.TrimSpace(config.AppKey) == "" || strings.TrimSpace(config.AppSecret) == "" {
		return entmodel.DingTalkConfig{}, ErrDingTalkMissingCredentials
	}
	return config, nil
}

func (s *DingTalkOAuthService) findIdentityBinding(tenantId int, identity DingTalkOAuthIdentity) (entmodel.DingTalkIdentity, bool, error) {
	var binding entmodel.DingTalkIdentity
	if err := s.db.Where("tenant_id = ? AND identity_key = ?", tenantId, dingTalkIdentityKey(identity)).First(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entmodel.DingTalkIdentity{}, false, nil
		}
		return entmodel.DingTalkIdentity{}, false, err
	}
	return binding, true, nil
}

func (s *DingTalkOAuthService) availableDingTalkUsername(identity DingTalkOAuthIdentity) string {
	base := firstReadableEnterpriseUsername(
		identity.Name,
		identity.Email,
		identity.Mobile,
		identity.ExternalUserId,
		identity.UnionId,
		identity.OpenId,
	)
	return resolveAvailableEnterpriseUsername(
		s.db,
		base,
		identity.ExternalUserId,
		identity.UnionId,
		identity.OpenId,
		identity.Mobile,
	)
}

func (s *DingTalkOAuthService) AvailableReadableUsernameForTest(identity DingTalkOAuthIdentity) string {
	return s.availableDingTalkUsername(identity)
}

func (s *DingTalkOAuthService) readLocalDingTalkMembershipSnapshot(tenantId int, userId int, externalUserId string) ([]UserDepartmentItem, error) {
	externalUserId = strings.TrimSpace(externalUserId)
	if externalUserId == "" {
		return nil, ErrDingTalkOAuthOutOfScope
	}
	query := MembershipQuery{
		TenantId:       &tenantId,
		ExternalSource: constant.EnterpriseExternalSourceDingTalk,
		Status:         intPtr(constant.EnterpriseMembershipStatusActive),
	}
	result, err := NewDepartmentMembershipService(s.db).ListUserDepartments(userId, query)
	if err != nil {
		return nil, err
	}
	memberships := make([]UserDepartmentItem, 0, len(result.Items))
	for _, membership := range result.Items {
		if strings.TrimSpace(membership.ExternalUserId) == externalUserId {
			memberships = append(memberships, membership)
		}
	}
	if len(memberships) == 0 {
		return nil, ErrDingTalkOAuthOutOfScope
	}
	return memberships, nil
}

func normalizeDingTalkOAuthIdentity(identity DingTalkOAuthIdentity) DingTalkOAuthIdentity {
	identity.UnionId = strings.TrimSpace(identity.UnionId)
	identity.OpenId = strings.TrimSpace(identity.OpenId)
	identity.ExternalUserId = strings.TrimSpace(identity.ExternalUserId)
	identity.Name = strings.TrimSpace(identity.Name)
	identity.Email = strings.TrimSpace(identity.Email)
	identity.Mobile = strings.TrimSpace(identity.Mobile)
	identity.DepartmentIds = normalizeDingTalkDepartmentUser(DingTalkDepartmentUserInfo{DeptIdList: identity.DepartmentIds}).DeptIdList
	return identity
}

func isOAuthVerifiedDingTalkIdentity(identity DingTalkOAuthIdentity) bool {
	return identity.oauthVerified && identity.UnionId != ""
}

func dingTalkIdentityKey(identity DingTalkOAuthIdentity) string {
	if strings.TrimSpace(identity.UnionId) != "" {
		return "union:" + strings.TrimSpace(identity.UnionId)
	}
	return "open:" + strings.TrimSpace(identity.OpenId)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
