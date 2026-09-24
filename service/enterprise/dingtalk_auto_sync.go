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
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

const dingTalkDirectoryVerificationTimeout = 30 * time.Second

var (
	errDingTalkIdentityConcurrentCreate = errors.New("dingtalk identity was created concurrently")
	dingTalkAutoSyncGroup               singleflight.Group
)

type DingTalkAutoSyncInput struct {
	TenantId int
	Identity DingTalkOAuthIdentity
}

type DingTalkAutoSyncResult struct {
	User        *model.User
	Binding     entmodel.DingTalkIdentity
	Memberships []UserDepartmentItem
	CreatedUser bool
	CreatedRows int
}

type DingTalkAutoSyncError struct {
	err error
}

func (e *DingTalkAutoSyncError) Error() string {
	return e.err.Error()
}

func (e *DingTalkAutoSyncError) Unwrap() error {
	return e.err
}

type DingTalkDirectoryService struct {
	db       *gorm.DB
	client   DingTalkDirectoryClient
	resolver *DingTalkScopeResolver
}

func NewDingTalkDirectoryService(db *gorm.DB, client DingTalkDirectoryClient) *DingTalkDirectoryService {
	if client == nil {
		client = NewDingTalkClient()
	}
	return &DingTalkDirectoryService{
		db:       db,
		client:   client,
		resolver: defaultDingTalkScopeResolver,
	}
}

func (s *DingTalkDirectoryService) EnsureEmployeeSynced(ctx context.Context, input DingTalkAutoSyncInput) (DingTalkAutoSyncResult, error) {
	if input.TenantId < 0 {
		input.TenantId = 0
	}
	config, err := s.getAutoSyncConfig(input.TenantId)
	if err != nil {
		return DingTalkAutoSyncResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, dingTalkDirectoryVerificationTimeout)
	defer cancel()

	identity, tree, matchedDepartmentIds, err := s.loadVerifiedEmployee(ctx, config, input.Identity)
	if err != nil {
		return DingTalkAutoSyncResult{}, &DingTalkAutoSyncError{err: err}
	}

	value, err, _ := dingTalkAutoSyncGroup.Do(fmt.Sprintf("%d:%s", config.TenantId, dingTalkIdentityKey(identity)), func() (any, error) {
		var result DingTalkAutoSyncResult
		var writeErr error
		for attempt := 0; attempt < 2; attempt++ {
			result, writeErr = s.writeVerifiedEmployee(ctx, config, identity, tree, matchedDepartmentIds)
			if !errors.Is(writeErr, errDingTalkIdentityConcurrentCreate) && !errors.Is(writeErr, model.ErrEmailAlreadyTaken) && !isDingTalkUniqueConstraintError(writeErr) {
				break
			}
		}
		if writeErr != nil {
			return nil, writeErr
		}
		if result.CreatedUser && result.User != nil {
			result.User.FinalizeOAuthUserCreation(0)
		}
		return result, nil
	})
	if err != nil {
		return DingTalkAutoSyncResult{}, &DingTalkAutoSyncError{err: err}
	}
	result, ok := value.(DingTalkAutoSyncResult)
	if !ok {
		return DingTalkAutoSyncResult{}, &DingTalkAutoSyncError{err: ErrDingTalkOAuthProviderFailed}
	}
	return result, nil
}

func (s *DingTalkDirectoryService) getAutoSyncConfig(tenantId int) (entmodel.DingTalkConfig, error) {
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
	if !config.SyncEnabled || !config.AutoSyncOnLogin {
		return entmodel.DingTalkConfig{}, ErrDingTalkAutoSyncDisabled
	}
	return config, nil
}

func (s *DingTalkDirectoryService) loadVerifiedEmployee(ctx context.Context, config entmodel.DingTalkConfig, identity DingTalkOAuthIdentity) (DingTalkOAuthIdentity, dingTalkScopeTree, []int64, error) {
	identity = normalizeDingTalkOAuthIdentity(identity)
	if identity.UnionId == "" {
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthEmployeeNotFound
	}
	appToken, err := s.client.GetAccessToken(ctx, config.AppKey, config.AppSecret)
	if err != nil {
		logDingTalkOAuthProviderError("access_token", err)
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthEmployeeNotFound
	}
	contact, err := s.client.GetContactUserByUnionId(ctx, appToken, identity.UnionId)
	if err != nil || strings.TrimSpace(contact.UserId) == "" {
		if err != nil {
			logDingTalkOAuthProviderError("contact_user", err)
		}
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthEmployeeNotFound
	}
	contact.UnionId = strings.TrimSpace(contact.UnionId)
	if contact.UnionId != "" && contact.UnionId != identity.UnionId {
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthEmployeeNotFound
	}
	if contact.Active != nil && !*contact.Active {
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthUserDisabled
	}
	canonical, err := s.client.GetUserById(ctx, appToken, contact.UserId)
	if err != nil {
		logDingTalkOAuthProviderError("user_get", err)
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthEmployeeNotFound
	}
	canonical = normalizeDingTalkDepartmentUser(canonical)
	if canonical.UserId == "" || canonical.UserId != strings.TrimSpace(contact.UserId) || len(canonical.DeptIdList) == 0 || canonical.Active == nil {
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthEmployeeNotFound
	}
	if !*canonical.Active {
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthUserDisabled
	}
	if canonical.UnionId != "" && canonical.UnionId != identity.UnionId {
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthEmployeeNotFound
	}
	tree, _, err := s.resolver.Resolve(ctx, s.client, config, appToken)
	if err != nil {
		logDingTalkOAuthProviderError("scope_tree", err)
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthProviderFailed
	}
	matchedDepartmentIds := dingTalkScopeDepartmentIds(tree, canonical.DeptIdList)
	if len(matchedDepartmentIds) == 0 {
		return DingTalkOAuthIdentity{}, dingTalkScopeTree{}, nil, ErrDingTalkOAuthOutOfScope
	}
	identity.ExternalUserId = canonical.UserId
	identity.UnionId = firstNonEmpty(canonical.UnionId, identity.UnionId)
	identity.Name = canonical.Name
	identity.Email = canonical.Email
	identity.Mobile = canonical.Mobile
	identity.Active = canonical.Active
	identity.DepartmentIds = canonical.DeptIdList
	identity.ScopeVerified = true
	identity.verifiedByServer = true
	return identity, tree, matchedDepartmentIds, nil
}

func (s *DingTalkDirectoryService) writeVerifiedEmployee(ctx context.Context, config entmodel.DingTalkConfig, identity DingTalkOAuthIdentity, tree dingTalkScopeTree, matchedDepartmentIds []int64) (DingTalkAutoSyncResult, error) {
	var result DingTalkAutoSyncResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		identityKey := dingTalkIdentityKey(identity)
		var binding entmodel.DingTalkIdentity
		bindingErr := model.LockForUpdate(tx).
			Where("tenant_id = ? AND identity_key = ?", config.TenantId, identityKey).
			First(&binding).Error
		if bindingErr != nil && !errors.Is(bindingErr, gorm.ErrRecordNotFound) {
			return bindingErr
		}

		user := &model.User{}
		loginStatus := DingTalkOAuthLoginStatusExisting
		if bindingErr == nil {
			if binding.Status != DingTalkIdentityStatusActive {
				return ErrDingTalkOAuthUserDisabled
			}
			if err := model.LockForUpdate(tx).Where("id = ?", binding.UserId).First(user).Error; err != nil {
				return err
			}
			if user.Status != common.UserStatusEnabled {
				return ErrDingTalkOAuthUserDisabled
			}
		} else {
			var err error
			user, loginStatus, err = s.findOrCreateVerifiedUser(tx, config.TenantId, identity)
			if err != nil {
				return err
			}
			binding = entmodel.DingTalkIdentity{
				TenantId:       config.TenantId,
				CorpId:         config.CorpId,
				IdentityKey:    identityKey,
				UnionId:        identity.UnionId,
				OpenId:         identity.OpenId,
				ExternalUserId: identity.ExternalUserId,
				Mobile:         identity.Mobile,
				UserId:         user.Id,
				Status:         DingTalkIdentityStatusActive,
				LastLoginAt:    time.Now().Unix(),
			}
			if err := tx.Create(&binding).Error; err != nil {
				if isDingTalkUniqueConstraintError(err) {
					return errDingTalkIdentityConcurrentCreate
				}
				return err
			}
		}

		if err := tx.Model(&binding).Updates(map[string]any{
			"corp_id":          config.CorpId,
			"union_id":         identity.UnionId,
			"open_id":          identity.OpenId,
			"external_user_id": identity.ExternalUserId,
			"mobile":           identity.Mobile,
			"status":           DingTalkIdentityStatusActive,
			"last_login_at":    time.Now().Unix(),
		}).Error; err != nil {
			return err
		}

		includedDepartmentIds := dingTalkScopePathDepartmentIds(tree, matchedDepartmentIds)
		localDepartmentIds := make(map[int64]int, len(includedDepartmentIds))
		createdRows := 0
		for _, externalDepartmentId := range includedDepartmentIds {
			department, ok := tree.Departments[externalDepartmentId]
			if !ok {
				return ErrDingTalkOAuthProviderFailed
			}
			var localParentId *int
			if department.ParentId > 0 {
				parentId, found := localDepartmentIds[department.ParentId]
				if !found {
					return ErrDingTalkOAuthProviderFailed
				}
				localParentId = &parentId
			}
			local, created, err := upsertDingTalkAutoSyncDepartment(tx, config.TenantId, department, localParentId)
			if err != nil {
				return err
			}
			if created {
				createdRows++
			}
			localDepartmentIds[externalDepartmentId] = local.Id
		}
		for _, externalDepartmentId := range matchedDepartmentIds {
			departmentId, found := localDepartmentIds[externalDepartmentId]
			if !found {
				return ErrDingTalkOAuthProviderFailed
			}
			created, err := upsertDingTalkAutoSyncMembership(tx, config.TenantId, user.Id, departmentId, identity.ExternalUserId)
			if err != nil {
				return err
			}
			if created {
				createdRows++
			}
		}
		if err := tx.Where("id = ?", binding.Id).First(&binding).Error; err != nil {
			return err
		}
		result = DingTalkAutoSyncResult{
			User:        user,
			Binding:     binding,
			CreatedUser: loginStatus == DingTalkOAuthLoginStatusCreated,
			CreatedRows: createdRows,
		}
		return nil
	})
	if err != nil {
		return DingTalkAutoSyncResult{}, err
	}
	memberships, err := (&DingTalkOAuthService{db: s.db}).readLocalDingTalkMembershipSnapshot(config.TenantId, result.User.Id, identity.ExternalUserId)
	if err != nil {
		return DingTalkAutoSyncResult{}, err
	}
	result.Memberships = memberships
	return result, nil
}

func (s *DingTalkDirectoryService) findOrCreateVerifiedUser(tx *gorm.DB, tenantId int, identity DingTalkOAuthIdentity) (*model.User, string, error) {
	identityKey := dingTalkIdentityKey(identity)
	if identity.Mobile != "" {
		mobileConflict, err := findDingTalkMobileConflictCandidate(tx, tenantId, identity.Mobile, identityKey)
		if err != nil {
			return nil, "", err
		}
		if mobileConflict.Matched {
			if err := recordDingTalkLoginConflict(tx, tenantId, identity, "mobile", mobileConflict.CandidateUserId, "mobile_matches_existing_dingtalk_identity"); err != nil {
				return nil, "", err
			}
			return nil, "", ErrDingTalkOAuthBindingConflict
		}
	}
	if identity.Email != "" {
		users, err := findDingTalkUsersByEmail(tx, identity.Email)
		if err != nil {
			return nil, "", err
		}
		if len(users) > 1 || (len(users) == 1 && users[0].DeletedAt.Valid) {
			if err := recordDingTalkLoginConflict(tx, tenantId, identity, "email", 0, "email_matches_multiple_or_deleted_local_users"); err != nil {
				return nil, "", err
			}
			return nil, "", ErrDingTalkOAuthBindingConflict
		}
		if len(users) == 1 {
			var existing entmodel.DingTalkIdentity
			err := model.LockForUpdate(tx).Where("tenant_id = ? AND user_id = ?", tenantId, users[0].Id).First(&existing).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, "", err
			}
			if err == nil && existing.IdentityKey != identityKey {
				if err := recordDingTalkLoginConflict(tx, tenantId, identity, "email", users[0].Id, "local_user_is_bound_to_another_dingtalk_identity"); err != nil {
					return nil, "", err
				}
				return nil, "", ErrDingTalkOAuthBindingConflict
			}
			if users[0].Status != common.UserStatusEnabled {
				return nil, "", ErrDingTalkOAuthUserDisabled
			}
			return &users[0], DingTalkOAuthLoginStatusBound, nil
		}
	}
	user := &model.User{
		Username:    (&DingTalkOAuthService{db: tx}).availableDingTalkUsername(identity),
		DisplayName: firstNonEmpty(identity.Name, identity.Email, identity.ExternalUserId),
		Email:       identity.Email,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	if err := user.InsertWithTx(tx, 0); err != nil {
		return nil, "", err
	}
	return user, DingTalkOAuthLoginStatusCreated, nil
}

func findDingTalkUsersByEmail(db *gorm.DB, email string) ([]model.User, error) {
	email = model.NormalizeEmail(email)
	if email == "" {
		return []model.User{}, nil
	}
	var users []model.User
	if err := db.Unscoped().Where("email = ?", email).Limit(2).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func recordDingTalkLoginConflict(tx *gorm.DB, tenantId int, identity DingTalkOAuthIdentity, conflictType string, candidateUserId int, details string) error {
	var existing entmodel.DingTalkSyncConflict
	err := tx.Where("tenant_id = ? AND external_user_id = ? AND conflict_type = ?", tenantId, identity.ExternalUserId, conflictType).First(&existing).Error
	updates := map[string]any{
		"task_id":           0,
		"trigger_source":    "login",
		"last_task_id":      0,
		"union_id":          identity.UnionId,
		"mobile":            identity.Mobile,
		"email":             identity.Email,
		"name":              identity.Name,
		"candidate_user_id": candidateUserId,
		"details":           truncateSyncText(details, 1024),
		"status":            constant.DingTalkSyncConflictStatusPending,
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&entmodel.DingTalkSyncConflict{
			TenantId:        tenantId,
			TaskId:          0,
			TriggerSource:   "login",
			ExternalUserId:  identity.ExternalUserId,
			UnionId:         identity.UnionId,
			Mobile:          identity.Mobile,
			Email:           identity.Email,
			Name:            identity.Name,
			ConflictType:    conflictType,
			CandidateUserId: candidateUserId,
			Details:         truncateSyncText(details, 1024),
			Status:          constant.DingTalkSyncConflictStatusPending,
		}).Error
	}
	if err != nil {
		return err
	}
	return tx.Model(&existing).Updates(updates).Error
}

func dingTalkScopePathDepartmentIds(tree dingTalkScopeTree, matchedDepartmentIds []int64) []int64 {
	included := make(map[int64]struct{}, len(matchedDepartmentIds))
	for _, departmentId := range matchedDepartmentIds {
		for departmentId > 0 {
			if _, exists := included[departmentId]; exists {
				break
			}
			department, found := tree.Departments[departmentId]
			if !found {
				break
			}
			included[departmentId] = struct{}{}
			departmentId = department.ParentId
		}
	}
	ordered := make([]int64, 0, len(included))
	for _, departmentId := range tree.Order {
		if _, exists := included[departmentId]; exists {
			ordered = append(ordered, departmentId)
		}
	}
	return ordered
}

func upsertDingTalkAutoSyncDepartment(tx *gorm.DB, tenantId int, department DingTalkDepartmentInfo, parentId *int) (entmodel.Department, bool, error) {
	externalId := strings.TrimSpace(formatDingTalkDepartmentId(department.DeptId))
	var existing entmodel.Department
	err := tx.Where("tenant_id = ? AND source_type = ? AND external_id = ?", tenantId, constant.DepartmentSourceTypeDingTalk, externalId).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		created := entmodel.Department{
			TenantId:   tenantId,
			Name:       strings.TrimSpace(department.Name),
			ParentId:   parentId,
			Status:     constant.DepartmentStatusEnabled,
			SourceType: constant.DepartmentSourceTypeDingTalk,
			ExternalId: externalId,
			SyncStatus: constant.DepartmentSyncStatusOK,
		}
		if err := tx.Create(&created).Error; err != nil {
			return entmodel.Department{}, false, err
		}
		return created, true, nil
	}
	if err != nil {
		return entmodel.Department{}, false, err
	}
	name := strings.TrimSpace(department.Name)
	updates := map[string]any{
		"name":        name,
		"parent_id":   parentId,
		"status":      constant.DepartmentStatusEnabled,
		"sync_status": constant.DepartmentSyncStatusOK,
		"sync_error":  "",
	}
	if existing.Name != name {
		entries, err := existing.ParsedNameHistory()
		if err != nil {
			return entmodel.Department{}, false, err
		}
		entries = appendDepartmentNameHistory(entries, existing.Name)
		if err := existing.SetNameHistory(entries); err != nil {
			return entmodel.Department{}, false, err
		}
		updates["name_history"] = existing.NameHistory
	}
	if err := tx.Model(&existing).Updates(updates).Error; err != nil {
		return entmodel.Department{}, false, err
	}
	if err := tx.Where("id = ?", existing.Id).First(&existing).Error; err != nil {
		return entmodel.Department{}, false, err
	}
	return existing, false, nil
}

func upsertDingTalkAutoSyncMembership(tx *gorm.DB, tenantId int, userId int, departmentId int, externalUserId string) (bool, error) {
	var existing entmodel.UserDepartment
	err := tx.Where("tenant_id = ? AND user_id = ? AND department_id = ? AND external_source = ?", tenantId, userId, departmentId, constant.EnterpriseExternalSourceDingTalk).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, tx.Create(&entmodel.UserDepartment{
			TenantId:       tenantId,
			UserId:         userId,
			DepartmentId:   departmentId,
			ExternalUserId: externalUserId,
			ExternalSource: constant.EnterpriseExternalSourceDingTalk,
			Status:         constant.EnterpriseMembershipStatusActive,
			JoinedAt:       time.Now().Unix(),
		}).Error
	}
	if err != nil {
		return false, err
	}
	if existing.Status == constant.EnterpriseMembershipStatusActive && existing.ExternalUserId == externalUserId {
		return false, nil
	}
	return false, tx.Model(&existing).Updates(map[string]any{
		"external_user_id": externalUserId,
		"status":           constant.EnterpriseMembershipStatusActive,
		"left_at":          int64(0),
	}).Error
}

func normalizeDingTalkDepartmentUser(user DingTalkDepartmentUserInfo) DingTalkDepartmentUserInfo {
	user.UserId = strings.TrimSpace(user.UserId)
	user.UnionId = strings.TrimSpace(user.UnionId)
	user.Name = strings.TrimSpace(user.Name)
	user.Email = strings.TrimSpace(user.Email)
	user.Mobile = strings.TrimSpace(user.Mobile)
	departmentIds := make([]int64, 0, len(user.DeptIdList))
	seen := make(map[int64]struct{}, len(user.DeptIdList))
	for _, departmentId := range user.DeptIdList {
		if departmentId <= 0 {
			continue
		}
		if _, exists := seen[departmentId]; exists {
			continue
		}
		seen[departmentId] = struct{}{}
		departmentIds = append(departmentIds, departmentId)
	}
	user.DeptIdList = departmentIds
	return user
}

func formatDingTalkDepartmentId(id int64) string {
	return strconv.FormatInt(id, 10)
}

func isDingTalkUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}
