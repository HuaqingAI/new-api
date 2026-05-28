package enterprise

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/gin-contrib/sessions"
	"gorm.io/gorm"
)

const (
	DingTalkIdentityStatusActive   = 1
	DingTalkIdentityStatusDisabled = 2

	DingTalkOAuthLoginStatusExisting = "existing"
	DingTalkOAuthLoginStatusCreated  = "created"
	DingTalkOAuthLoginStatusBound    = "bound"
)

type DingTalkOAuthIdentity struct {
	UnionId        string
	OpenId         string
	ExternalUserId string
	Name           string
	Email          string
	Mobile         string
	Active         *bool
}

type DingTalkOAuthService struct {
	db     *gorm.DB
	client *DingTalkClient
}

type DingTalkOAuthResult struct {
	User              *model.User
	Binding           entmodel.DingTalkIdentity
	Memberships       []UserDepartmentItem
	DepartmentCount   int
	LoginStatus       string
	UsedLocalSnapshot bool
}

func NewDingTalkOAuthService(db *gorm.DB, client *DingTalkClient) *DingTalkOAuthService {
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

	token, err := s.client.ExchangeOAuthCode(ctx, config.AppKey, config.AppSecret, code)
	if err != nil {
		return DingTalkOAuthIdentity{}, ErrDingTalkOAuthProviderFailed
	}
	oauthUser, err := s.client.GetOAuthUserInfo(ctx, token.AccessToken)
	if err != nil {
		return DingTalkOAuthIdentity{}, ErrDingTalkOAuthProviderFailed
	}

	identity := DingTalkOAuthIdentity{
		UnionId: strings.TrimSpace(oauthUser.UnionId),
		OpenId:  strings.TrimSpace(oauthUser.OpenId),
		Name:    strings.TrimSpace(oauthUser.Nick),
		Email:   strings.TrimSpace(oauthUser.Email),
		Mobile:  strings.TrimSpace(oauthUser.Mobile),
	}
	if identity.UnionId != "" {
		appToken, err := s.client.GetAccessToken(ctx, config.AppKey, config.AppSecret)
		if err != nil {
			return DingTalkOAuthIdentity{}, ErrDingTalkOAuthProviderFailed
		}
		contactUser, err := s.client.GetContactUserByUnionId(ctx, appToken, identity.UnionId)
		if err != nil {
			return DingTalkOAuthIdentity{}, ErrDingTalkOAuthProviderFailed
		}
		identity.ExternalUserId = strings.TrimSpace(contactUser.UserId)
		identity.Active = contactUser.Active
		if identity.Name == "" {
			identity.Name = strings.TrimSpace(contactUser.Name)
		}
		if identity.Email == "" {
			identity.Email = strings.TrimSpace(contactUser.Email)
		}
		if identity.Mobile == "" {
			identity.Mobile = strings.TrimSpace(contactUser.Mobile)
		}
	}
	if identity.UnionId == "" && identity.OpenId == "" {
		return DingTalkOAuthIdentity{}, ErrDingTalkOAuthIdentityMissing
	}
	return identity, nil
}

func (s *DingTalkOAuthService) LoginWithIdentity(ctx context.Context, tenantId int, identity DingTalkOAuthIdentity, session sessions.Session) (DingTalkOAuthResult, error) {
	config, err := s.getEnabledConfig(tenantId)
	if err != nil {
		return DingTalkOAuthResult{}, err
	}
	identity = normalizeDingTalkOAuthIdentity(identity)
	if identity.UnionId == "" && identity.OpenId == "" {
		return DingTalkOAuthResult{}, ErrDingTalkOAuthIdentityMissing
	}
	if identity.Active != nil && !*identity.Active {
		return DingTalkOAuthResult{}, ErrDingTalkOAuthUserDisabled
	}

	var result DingTalkOAuthResult
	now := time.Now().Unix()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txSvc := &DingTalkOAuthService{db: tx, client: s.client}
		binding, found, err := txSvc.findIdentityBinding(config.TenantId, identity)
		if err != nil {
			return err
		}

		user := &model.User{}
		loginStatus := DingTalkOAuthLoginStatusExisting
		if found {
			if binding.Status != DingTalkIdentityStatusActive {
				return ErrDingTalkOAuthUserDisabled
			}
			if err := tx.Where("id = ?", binding.UserId).First(user).Error; err != nil {
				return err
			}
		} else {
			user, loginStatus, err = txSvc.findOrCreateUserForIdentity(identity, session)
			if err != nil {
				return err
			}
			binding = entmodel.DingTalkIdentity{
				TenantId:       config.TenantId,
				CorpId:         config.CorpId,
				IdentityKey:    dingTalkIdentityKey(identity),
				UnionId:        identity.UnionId,
				OpenId:         identity.OpenId,
				ExternalUserId: identity.ExternalUserId,
				Mobile:         identity.Mobile,
				UserId:         user.Id,
				Status:         DingTalkIdentityStatusActive,
			}
			if err := tx.Create(&binding).Error; err != nil {
				return err
			}
		}

		if user.Status != common.UserStatusEnabled {
			return ErrDingTalkOAuthUserDisabled
		}

		bindingUpdate := map[string]any{
			"corp_id":       config.CorpId,
			"identity_key":  dingTalkIdentityKey(identity),
			"open_id":       identity.OpenId,
			"last_login_at": now,
		}
		if identity.UnionId != "" {
			bindingUpdate["union_id"] = identity.UnionId
		}
		if identity.ExternalUserId != "" {
			bindingUpdate["external_user_id"] = identity.ExternalUserId
		}
		if identity.Mobile != "" {
			bindingUpdate["mobile"] = identity.Mobile
		}
		if err := tx.Model(&binding).Updates(bindingUpdate).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", binding.Id).First(&binding).Error; err != nil {
			return err
		}

		memberships, err := txSvc.readLocalDingTalkMembershipSnapshot(config.TenantId, user.Id, identity.ExternalUserId)
		if err != nil {
			return err
		}
		result = DingTalkOAuthResult{
			User:              user,
			Binding:           binding,
			Memberships:       memberships,
			DepartmentCount:   len(memberships),
			LoginStatus:       loginStatus,
			UsedLocalSnapshot: true,
		}
		return nil
	})
	if err != nil {
		return DingTalkOAuthResult{}, err
	}

	if result.LoginStatus == DingTalkOAuthLoginStatusCreated && result.User != nil {
		result.User.FinalizeOAuthUserCreation(inviterIdFromSession(session))
	}
	return result, nil
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
	if identity.UnionId == "" && identity.OpenId == "" {
		return entmodel.DingTalkIdentity{}, ErrDingTalkOAuthIdentityMissing
	}
	if identity.Active != nil && !*identity.Active {
		return entmodel.DingTalkIdentity{}, ErrDingTalkOAuthUserDisabled
	}

	var binding entmodel.DingTalkIdentity
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Where("id = ?", userId).First(&user).Error; err != nil {
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

func (s *DingTalkOAuthService) findOrCreateUserForIdentity(identity DingTalkOAuthIdentity, session sessions.Session) (*model.User, string, error) {
	if identity.Email != "" {
		user, found, err := s.findUniqueUserByEmail(identity.Email)
		if err != nil {
			return nil, "", err
		}
		if found {
			return user, DingTalkOAuthLoginStatusBound, nil
		}
	}
	if !common.RegisterEnabled {
		return nil, "", ErrDingTalkOAuthRegistrationDisabled
	}

	user := &model.User{
		Username:    s.availableDingTalkUsername(identity),
		DisplayName: firstNonEmpty(identity.Name, identity.Email, "DingTalk User"),
		Email:       identity.Email,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	inviterId := inviterIdFromSession(session)
	if err := user.InsertWithTx(s.db, inviterId); err != nil {
		return nil, "", err
	}
	return user, DingTalkOAuthLoginStatusCreated, nil
}

func (s *DingTalkOAuthService) findUniqueUserByEmail(email string) (*model.User, bool, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, false, nil
	}
	var users []model.User
	if err := s.db.Unscoped().Where("email = ?", email).Limit(2).Find(&users).Error; err != nil {
		return nil, false, err
	}
	if len(users) == 0 {
		return nil, false, nil
	}
	if len(users) > 1 || users[0].DeletedAt.Valid {
		return nil, false, ErrDingTalkOAuthBindingConflict
	}
	return &users[0], true, nil
}

func (s *DingTalkOAuthService) availableDingTalkUsername(identity DingTalkOAuthIdentity) string {
	candidate := strings.TrimSpace(identity.ExternalUserId)
	if candidate == "" {
		candidate = strings.TrimSpace(identity.UnionId)
	}
	if candidate == "" {
		candidate = strings.TrimSpace(identity.OpenId)
	}
	candidate = normalizeDingTalkUsername(candidate)
	if candidate == "" {
		candidate = "dingtalk"
	}

	prefix := "dt_" + candidate
	if len(prefix) > model.UserNameMaxLength {
		prefix = prefix[:model.UserNameMaxLength]
	}
	if exists, err := model.CheckUserExistOrDeleted(prefix, ""); err == nil && !exists {
		return prefix
	}
	for i := 0; i < 20; i++ {
		suffix := strconv.Itoa(model.GetMaxUserId() + 1 + i)
		baseLen := model.UserNameMaxLength - len(suffix) - 1
		if baseLen < 2 {
			baseLen = 2
		}
		base := prefix
		if len(base) > baseLen {
			base = base[:baseLen]
		}
		username := base + "_" + suffix
		if exists, err := model.CheckUserExistOrDeleted(username, ""); err == nil && !exists {
			return username
		}
	}
	return fmt.Sprintf("dt_%d", time.Now().UnixNano()%100000000)
}

func (s *DingTalkOAuthService) readLocalDingTalkMembershipSnapshot(tenantId int, userId int, externalUserId string) ([]UserDepartmentItem, error) {
	query := MembershipQuery{
		TenantId:       &tenantId,
		ExternalSource: constant.EnterpriseExternalSourceDingTalk,
		Status:         intPtr(constant.EnterpriseMembershipStatusActive),
	}
	result, err := NewDepartmentMembershipService(s.db).ListUserDepartments(userId, query)
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 && strings.TrimSpace(externalUserId) != "" {
		return nil, ErrDingTalkOAuthOutOfScope
	}
	return result.Items, nil
}

func normalizeDingTalkOAuthIdentity(identity DingTalkOAuthIdentity) DingTalkOAuthIdentity {
	identity.UnionId = strings.TrimSpace(identity.UnionId)
	identity.OpenId = strings.TrimSpace(identity.OpenId)
	identity.ExternalUserId = strings.TrimSpace(identity.ExternalUserId)
	identity.Name = strings.TrimSpace(identity.Name)
	identity.Email = strings.TrimSpace(identity.Email)
	identity.Mobile = strings.TrimSpace(identity.Mobile)
	return identity
}

func dingTalkIdentityKey(identity DingTalkOAuthIdentity) string {
	if strings.TrimSpace(identity.UnionId) != "" {
		return "union:" + strings.TrimSpace(identity.UnionId)
	}
	return "open:" + strings.TrimSpace(identity.OpenId)
}

func normalizeDingTalkUsername(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_' || r == '-':
			builder.WriteRune('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func inviterIdFromSession(session sessions.Session) int {
	if session == nil {
		return 0
	}
	affCode := session.Get("aff")
	if affCode == nil {
		return 0
	}
	code, ok := affCode.(string)
	if !ok || strings.TrimSpace(code) == "" {
		return 0
	}
	inviterId, _ := model.GetUserIdByAffCode(code)
	return inviterId
}
