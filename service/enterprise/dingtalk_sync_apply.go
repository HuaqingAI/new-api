package enterprise

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

func (s *DingTalkSyncService) syncDepartmentTree(ctx context.Context, taskId int, tenantId int, corpId string, accessToken string, dingTalkDepartmentId int64, localParentId *int, snapshot *dingTalkSyncSnapshot) {
	departments, err := s.client.ListSubDepartments(ctx, accessToken, dingTalkDepartmentId)
	if err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectDepartment, strconv.FormatInt(dingTalkDepartmentId, 10), "department_list_failed")
		return
	}
	for _, department := range departments {
		localDepartment, ok := s.upsertDepartment(ctx, taskId, tenantId, department, localParentId, snapshot)
		if !ok {
			continue
		}
		s.syncDepartmentUsers(ctx, taskId, tenantId, corpId, accessToken, department.DeptId, localDepartment.Id, snapshot)
		childParentId := localDepartment.Id
		s.syncDepartmentTree(ctx, taskId, tenantId, corpId, accessToken, department.DeptId, &childParentId, snapshot)
	}
}

func (s *DingTalkSyncService) upsertDepartment(ctx context.Context, taskId int, tenantId int, department DingTalkDepartmentInfo, localParentId *int, snapshot *dingTalkSyncSnapshot) (entmodel.Department, bool) {
	externalId := strconv.FormatInt(department.DeptId, 10)
	if department.DeptId <= 0 || strings.TrimSpace(department.Name) == "" {
		s.incrementTaskCounter(ctx, taskId, "skipped_count", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectDepartment, ObjectExternalId: externalId, Action: constant.DingTalkSyncLogActionSkipped, Status: constant.DingTalkSyncLogStatusSkipped, Message: "department_missing_required_fields"})
		return entmodel.Department{}, false
	}
	snapshot.seenDepartmentExternalIds[externalId] = struct{}{}

	var existing entmodel.Department
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND source_type = ? AND external_id = ?", tenantId, constant.DepartmentSourceTypeDingTalk, externalId).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		created := entmodel.Department{
			TenantId:   tenantId,
			Name:       strings.TrimSpace(department.Name),
			ParentId:   localParentId,
			Status:     constant.DepartmentStatusEnabled,
			SourceType: constant.DepartmentSourceTypeDingTalk,
			ExternalId: externalId,
			SyncStatus: constant.DepartmentSyncStatusOK,
		}
		if err := s.db.WithContext(ctx).Create(&created).Error; err != nil {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectDepartment, externalId, "department_create_failed")
			return entmodel.Department{}, false
		}
		s.incrementTaskCounter(ctx, taskId, "departments_created", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectDepartment, ObjectExternalId: externalId, Action: constant.DingTalkSyncLogActionCreated, Status: constant.DingTalkSyncLogStatusSuccess, Message: "department_created"})
		return created, true
	}
	if err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectDepartment, externalId, "department_lookup_failed")
		return entmodel.Department{}, false
	}

	name := strings.TrimSpace(department.Name)
	if !sameOptionalInt(existing.ParentId, localParentId) || existing.Name != name || existing.Status != constant.DepartmentStatusEnabled || existing.SyncStatus != constant.DepartmentSyncStatusOK {
		nameHistory := existing.NameHistory
		if existing.Name != name {
			entries, err := existing.ParsedNameHistory()
			if err != nil {
				s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectDepartment, externalId, "department_name_history_parse_failed")
				return entmodel.Department{}, false
			}
			entries = appendDepartmentNameHistory(entries, existing.Name)
			existing.NameHistory = nameHistory
			if err := existing.SetNameHistory(entries); err != nil {
				s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectDepartment, externalId, "department_name_history_update_failed")
				return entmodel.Department{}, false
			}
			nameHistory = existing.NameHistory
		}
		if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
			"name":         name,
			"parent_id":    localParentId,
			"status":       constant.DepartmentStatusEnabled,
			"sync_status":  constant.DepartmentSyncStatusOK,
			"sync_error":   "",
			"name_history": nameHistory,
		}).Error; err != nil {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectDepartment, externalId, "department_update_failed")
			return entmodel.Department{}, false
		}
		s.incrementTaskCounter(ctx, taskId, "departments_updated", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectDepartment, ObjectExternalId: externalId, Action: constant.DingTalkSyncLogActionUpdated, Status: constant.DingTalkSyncLogStatusSuccess, Message: "department_updated"})
		if err := s.db.WithContext(ctx).Where("id = ?", existing.Id).First(&existing).Error; err != nil {
			return entmodel.Department{}, false
		}
	} else {
		s.incrementTaskCounter(ctx, taskId, "skipped_count", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectDepartment, ObjectExternalId: externalId, Action: constant.DingTalkSyncLogActionSkipped, Status: constant.DingTalkSyncLogStatusSkipped, Message: "department_unchanged"})
	}
	return existing, true
}

func (s *DingTalkSyncService) syncDepartmentUsers(ctx context.Context, taskId int, tenantId int, corpId string, accessToken string, dingTalkDepartmentId int64, localDepartmentId int, snapshot *dingTalkSyncSnapshot) {
	users, err := s.client.ListDepartmentUsers(ctx, accessToken, dingTalkDepartmentId)
	if err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectUser, strconv.FormatInt(dingTalkDepartmentId, 10), "department_users_failed")
		return
	}
	for _, dingTalkUser := range users {
		user, ok := s.upsertUser(ctx, taskId, tenantId, corpId, dingTalkUser)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%d:%d:%s", user.Id, localDepartmentId, constant.EnterpriseExternalSourceDingTalk)
		snapshot.seenMembershipKeys[key] = struct{}{}
		s.upsertMembership(ctx, taskId, tenantId, localDepartmentId, user.Id, dingTalkUser.UserId)
		if dingTalkUser.LeaderInDept != nil && *dingTalkUser.LeaderInDept {
			ownerKey := fmt.Sprintf("%d:%d:%s", user.Id, localDepartmentId, constant.EnterpriseDepartmentRoleSourceDingTalkOwner)
			snapshot.seenOwnerKeys[ownerKey] = struct{}{}
			s.upsertDingTalkDepartmentOwner(ctx, taskId, tenantId, localDepartmentId, user.Id, dingTalkUser.UserId)
		}
	}
}

func (s *DingTalkSyncService) upsertDingTalkDepartmentOwner(ctx context.Context, taskId int, tenantId int, departmentId int, userId int, externalUserId string) {
	role, err := NewPermissionService(s.db.WithContext(ctx)).UpsertDingTalkDepartmentOwner(DepartmentAdminRoleInput{
		TenantId:     tenantId,
		UserId:       userId,
		DepartmentId: departmentId,
	})
	if err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectOwner, externalUserId, "owner_fact_upsert_failed")
		return
	}
	action := constant.DingTalkSyncLogActionCreated
	message := "owner_fact_created"
	if role.UpdatedAt > role.CreatedAt {
		action = constant.DingTalkSyncLogActionUpdated
		message = "owner_fact_updated"
	}
	s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectOwner, ObjectExternalId: externalUserId, Action: action, Status: constant.DingTalkSyncLogStatusSuccess, Message: message})
}

func (s *DingTalkSyncService) upsertUser(ctx context.Context, taskId int, tenantId int, corpId string, dingTalkUser DingTalkDepartmentUserInfo) (*model.User, bool) {
	if strings.TrimSpace(dingTalkUser.UserId) == "" {
		s.incrementTaskCounter(ctx, taskId, "skipped_count", 1)
		return nil, false
	}
	var binding entmodel.DingTalkIdentity
	identityKey := dingtalkSyncIdentityKey(dingTalkUser)
	if identityKey != "" {
		err := s.db.WithContext(ctx).Where("tenant_id = ? AND identity_key = ?", tenantId, identityKey).First(&binding).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectUser, dingTalkUser.UserId, "identity_lookup_failed")
			return nil, false
		}
	}
	if binding.UserId > 0 {
		var existing model.User
		if err := s.db.WithContext(ctx).Where("id = ?", binding.UserId).First(&existing).Error; err != nil {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectUser, dingTalkUser.UserId, "bound_user_lookup_failed")
			return nil, false
		}
		s.updateDingTalkIdentity(ctx, tenantId, corpId, dingTalkUser, existing.Id)
		s.incrementTaskCounter(ctx, taskId, "users_updated", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectUser, ObjectExternalId: dingTalkUser.UserId, Action: constant.DingTalkSyncLogActionUpdated, Status: constant.DingTalkSyncLogStatusSuccess, Message: "user_identity_updated"})
		return &existing, true
	}

	conflict, found, err := s.detectUserConflict(ctx, tenantId, dingTalkUser)
	if err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectUser, dingTalkUser.UserId, "user_conflict_lookup_failed")
		return nil, false
	}
	if found {
		if err := s.recordSyncConflict(ctx, taskId, tenantId, dingTalkUser, conflict); err != nil {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectUser, dingTalkUser.UserId, "user_conflict_record_failed")
			return nil, false
		}
		s.incrementTaskCounter(ctx, taskId, "skipped_count", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{
			TaskId:           taskId,
			TenantId:         tenantId,
			ObjectType:       constant.DingTalkSyncObjectConflict,
			ObjectExternalId: strings.TrimSpace(dingTalkUser.UserId),
			Action:           constant.DingTalkSyncLogActionConflictPending,
			Status:           constant.DingTalkSyncLogStatusWarning,
			Message:          "user_conflict_pending",
		})
		return nil, false
	}

	user := model.User{
		Username:    s.availableSyncUsername(dingTalkUser),
		DisplayName: firstNonEmpty(dingTalkUser.Name, dingTalkUser.Email, dingTalkUser.UserId),
		Email:       strings.TrimSpace(dingTalkUser.Email),
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	if err := user.InsertWithTx(s.db.WithContext(ctx), 0); err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectUser, dingTalkUser.UserId, "user_create_failed")
		return nil, false
	}
	s.updateDingTalkIdentity(ctx, tenantId, corpId, dingTalkUser, user.Id)
	s.incrementTaskCounter(ctx, taskId, "users_created", 1)
	s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectUser, ObjectExternalId: dingTalkUser.UserId, Action: constant.DingTalkSyncLogActionCreated, Status: constant.DingTalkSyncLogStatusSuccess, Message: "user_created"})
	return &user, true
}

func (s *DingTalkSyncService) upsertMembership(ctx context.Context, taskId int, tenantId int, departmentId int, userId int, externalUserId string) {
	var existing entmodel.UserDepartment
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ? AND department_id = ? AND external_source = ?", tenantId, userId, departmentId, constant.EnterpriseExternalSourceDingTalk).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		membership := entmodel.UserDepartment{
			TenantId:       tenantId,
			UserId:         userId,
			DepartmentId:   departmentId,
			ExternalUserId: externalUserId,
			ExternalSource: constant.EnterpriseExternalSourceDingTalk,
			Status:         constant.EnterpriseMembershipStatusActive,
			JoinedAt:       time.Now().Unix(),
		}
		if err := s.db.WithContext(ctx).Create(&membership).Error; err != nil {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectMembership, externalUserId, "membership_create_failed")
			return
		}
		s.incrementTaskCounter(ctx, taskId, "memberships_created", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectMembership, ObjectExternalId: externalUserId, Action: constant.DingTalkSyncLogActionCreated, Status: constant.DingTalkSyncLogStatusSuccess, Message: "membership_created"})
		return
	}
	if err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectMembership, externalUserId, "membership_lookup_failed")
		return
	}
	if existing.Status != constant.EnterpriseMembershipStatusActive || existing.ExternalUserId != externalUserId {
		if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{"external_user_id": externalUserId, "status": constant.EnterpriseMembershipStatusActive, "left_at": int64(0)}).Error; err != nil {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectMembership, externalUserId, "membership_update_failed")
			return
		}
		s.incrementTaskCounter(ctx, taskId, "memberships_updated", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectMembership, ObjectExternalId: externalUserId, Action: constant.DingTalkSyncLogActionUpdated, Status: constant.DingTalkSyncLogStatusSuccess, Message: "membership_updated"})
		return
	}
	s.incrementTaskCounter(ctx, taskId, "skipped_count", 1)
	s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectMembership, ObjectExternalId: externalUserId, Action: constant.DingTalkSyncLogActionSkipped, Status: constant.DingTalkSyncLogStatusSkipped, Message: "membership_unchanged"})
}

type dingTalkUserConflictCandidate struct {
	ConflictType    string
	CandidateUserId int
	Details         string
}

type dingTalkConflictCandidateLookup struct {
	Matched         bool
	CandidateUserId int
}

func (s *DingTalkSyncService) detectUserConflict(ctx context.Context, tenantId int, dingTalkUser DingTalkDepartmentUserInfo) (dingTalkUserConflictCandidate, bool, error) {
	var candidates []dingTalkUserConflictCandidate
	email := strings.TrimSpace(dingTalkUser.Email)
	if email != "" {
		lookup, err := findDingTalkEmailConflictCandidate(s.db.WithContext(ctx), email)
		if err != nil {
			return dingTalkUserConflictCandidate{}, false, err
		}
		if lookup.Matched {
			candidates = append(candidates, dingTalkUserConflictCandidate{
				ConflictType:    "email",
				CandidateUserId: lookup.CandidateUserId,
				Details:         "email_matches_existing_local_user",
			})
		}
	}
	mobile := strings.TrimSpace(dingTalkUser.Mobile)
	if mobile != "" {
		identityKey := dingtalkSyncIdentityKey(dingTalkUser)
		lookup, err := findDingTalkMobileConflictCandidate(s.db.WithContext(ctx), tenantId, mobile, identityKey)
		if err != nil {
			return dingTalkUserConflictCandidate{}, false, err
		}
		if lookup.Matched {
			candidates = append(candidates, dingTalkUserConflictCandidate{
				ConflictType:    "mobile",
				CandidateUserId: lookup.CandidateUserId,
				Details:         "mobile_matches_existing_dingtalk_identity",
			})
		}
	}
	if len(candidates) == 0 {
		return dingTalkUserConflictCandidate{}, false, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].ConflictType < candidates[j].ConflictType
	})
	if len(candidates) > 1 {
		candidates[0].ConflictType = "email_mobile"
		candidates[0].CandidateUserId = sharedDingTalkSyncConflictCandidate(candidates)
		candidates[0].Details = "email_and_mobile_match_existing_accounts"
	}
	return candidates[0], true, nil
}

func validateDingTalkSyncConflictCandidate(db *gorm.DB, tenantId int, conflict entmodel.DingTalkSyncConflict) error {
	expectedUserId := conflict.CandidateUserId
	if expectedUserId <= 0 {
		return ErrDingTalkSyncConflictNoCandidate
	}
	identityKey := dingtalkSyncConflictIdentityKey(conflict)
	switch strings.TrimSpace(conflict.ConflictType) {
	case "email":
		lookup, err := findDingTalkEmailConflictCandidate(db, conflict.Email)
		if err != nil {
			return err
		}
		return requireDingTalkSyncConflictCandidate(lookup, expectedUserId)
	case "mobile":
		lookup, err := findDingTalkMobileConflictCandidate(db, tenantId, conflict.Mobile, identityKey)
		if err != nil {
			return err
		}
		return requireDingTalkSyncConflictCandidate(lookup, expectedUserId)
	case "email_mobile":
		emailLookup, err := findDingTalkEmailConflictCandidate(db, conflict.Email)
		if err != nil {
			return err
		}
		mobileLookup, err := findDingTalkMobileConflictCandidate(db, tenantId, conflict.Mobile, identityKey)
		if err != nil {
			return err
		}
		if err := requireDingTalkSyncConflictCandidate(emailLookup, expectedUserId); err != nil {
			return err
		}
		return requireDingTalkSyncConflictCandidate(mobileLookup, expectedUserId)
	default:
		return ErrDingTalkSyncConflictNoCandidate
	}
}

func requireDingTalkSyncConflictCandidate(lookup dingTalkConflictCandidateLookup, expectedUserId int) error {
	if !lookup.Matched || lookup.CandidateUserId != expectedUserId {
		return ErrDingTalkSyncConflictNoCandidate
	}
	return nil
}

func sharedDingTalkSyncConflictCandidate(candidates []dingTalkUserConflictCandidate) int {
	if len(candidates) == 0 || candidates[0].CandidateUserId <= 0 {
		return 0
	}
	candidateUserId := candidates[0].CandidateUserId
	for _, candidate := range candidates[1:] {
		if candidate.CandidateUserId != candidateUserId {
			return 0
		}
	}
	return candidateUserId
}

func findDingTalkEmailConflictCandidate(db *gorm.DB, email string) (dingTalkConflictCandidateLookup, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return dingTalkConflictCandidateLookup{}, nil
	}
	var users []model.User
	if err := db.Unscoped().Where("email = ?", email).Limit(2).Find(&users).Error; err != nil {
		return dingTalkConflictCandidateLookup{}, err
	}
	if len(users) == 0 {
		return dingTalkConflictCandidateLookup{}, nil
	}
	lookup := dingTalkConflictCandidateLookup{Matched: true}
	if len(users) == 1 && !users[0].DeletedAt.Valid {
		lookup.CandidateUserId = users[0].Id
	}
	return lookup, nil
}

func findDingTalkMobileConflictCandidate(db *gorm.DB, tenantId int, mobile string, identityKey string) (dingTalkConflictCandidateLookup, error) {
	mobile = strings.TrimSpace(mobile)
	if mobile == "" {
		return dingTalkConflictCandidateLookup{}, nil
	}
	query := db.Where("tenant_id = ? AND mobile = ?", tenantId, mobile)
	if strings.TrimSpace(identityKey) != "" {
		query = query.Where("identity_key <> ?", strings.TrimSpace(identityKey))
	}
	var identities []entmodel.DingTalkIdentity
	if err := query.Limit(2).Find(&identities).Error; err != nil {
		return dingTalkConflictCandidateLookup{}, err
	}
	if len(identities) == 0 {
		return dingTalkConflictCandidateLookup{}, nil
	}
	lookup := dingTalkConflictCandidateLookup{Matched: true}
	if len(identities) == 1 {
		lookup.CandidateUserId = identities[0].UserId
	}
	return lookup, nil
}

func (s *DingTalkSyncService) recordSyncConflict(ctx context.Context, taskId int, tenantId int, dingTalkUser DingTalkDepartmentUserInfo, conflict dingTalkUserConflictCandidate) error {
	externalUserId := strings.TrimSpace(dingTalkUser.UserId)
	conflictType := strings.TrimSpace(conflict.ConflictType)
	if conflictType == "" {
		conflictType = "unknown"
	}

	var existing entmodel.DingTalkSyncConflict
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND external_user_id = ? AND conflict_type = ?", tenantId, externalUserId, conflictType).First(&existing).Error
	update := map[string]any{
		"task_id":           taskId,
		"trigger_source":    "full_sync",
		"last_task_id":      taskId,
		"union_id":          strings.TrimSpace(dingTalkUser.UnionId),
		"mobile":            strings.TrimSpace(dingTalkUser.Mobile),
		"email":             strings.TrimSpace(dingTalkUser.Email),
		"name":              strings.TrimSpace(dingTalkUser.Name),
		"candidate_user_id": conflict.CandidateUserId,
		"details":           truncateSyncText(conflict.Details, 1024),
		"status":            constant.DingTalkSyncConflictStatusPending,
		"resolved_by":       0,
		"resolved_at":       int64(0),
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record := entmodel.DingTalkSyncConflict{
			TenantId:        tenantId,
			TaskId:          taskId,
			TriggerSource:   "full_sync",
			ExternalUserId:  externalUserId,
			UnionId:         strings.TrimSpace(dingTalkUser.UnionId),
			Mobile:          strings.TrimSpace(dingTalkUser.Mobile),
			Email:           strings.TrimSpace(dingTalkUser.Email),
			Name:            strings.TrimSpace(dingTalkUser.Name),
			ConflictType:    conflictType,
			CandidateUserId: conflict.CandidateUserId,
			Details:         truncateSyncText(conflict.Details, 1024),
			Status:          constant.DingTalkSyncConflictStatusPending,
			LastTaskId:      taskId,
		}
		return s.db.WithContext(ctx).Create(&record).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&existing).Updates(update).Error
}

func appendDepartmentNameHistory(entries []entmodel.DepartmentNameHistoryEntry, name string) []entmodel.DepartmentNameHistoryEntry {
	name = strings.TrimSpace(name)
	if name == "" {
		return entries
	}
	if len(entries) > 0 && entries[len(entries)-1].Name == name {
		return entries
	}
	return append(entries, entmodel.DepartmentNameHistoryEntry{Name: name, ChangedAt: time.Now().Unix()})
}
