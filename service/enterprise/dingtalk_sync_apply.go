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
	"gorm.io/gorm"
)

func (s *DingTalkSyncService) syncDepartmentTree(ctx context.Context, taskId int, tenantId int, accessToken string, dingTalkDepartmentId int64, localParentId *int, snapshot *dingTalkSyncSnapshot) {
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
		s.syncDepartmentUsers(ctx, taskId, tenantId, accessToken, department.DeptId, localDepartment.Id, snapshot)
		childParentId := localDepartment.Id
		s.syncDepartmentTree(ctx, taskId, tenantId, accessToken, department.DeptId, &childParentId, snapshot)
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
		if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
			"name":        name,
			"parent_id":   localParentId,
			"status":      constant.DepartmentStatusEnabled,
			"sync_status": constant.DepartmentSyncStatusOK,
			"sync_error":  "",
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

func (s *DingTalkSyncService) syncDepartmentUsers(ctx context.Context, taskId int, tenantId int, accessToken string, dingTalkDepartmentId int64, localDepartmentId int, snapshot *dingTalkSyncSnapshot) {
	users, err := s.client.ListDepartmentUsers(ctx, accessToken, dingTalkDepartmentId)
	if err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectUser, strconv.FormatInt(dingTalkDepartmentId, 10), "department_users_failed")
		return
	}
	for _, dingTalkUser := range users {
		user, ok := s.upsertUser(ctx, taskId, tenantId, dingTalkUser)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%d:%d:%s", user.Id, localDepartmentId, constant.EnterpriseExternalSourceDingTalk)
		snapshot.seenMembershipKeys[key] = struct{}{}
		s.upsertMembership(ctx, taskId, tenantId, localDepartmentId, user.Id, dingTalkUser.UserId)
	}
}

func (s *DingTalkSyncService) upsertUser(ctx context.Context, taskId int, tenantId int, dingTalkUser DingTalkDepartmentUserInfo) (*model.User, bool) {
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
		s.updateDingTalkIdentity(ctx, tenantId, dingTalkUser, existing.Id)
		s.incrementTaskCounter(ctx, taskId, "users_updated", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectUser, ObjectExternalId: dingTalkUser.UserId, Action: constant.DingTalkSyncLogActionUpdated, Status: constant.DingTalkSyncLogStatusSuccess, Message: "user_identity_updated"})
		return &existing, true
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
	s.updateDingTalkIdentity(ctx, tenantId, dingTalkUser, user.Id)
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
