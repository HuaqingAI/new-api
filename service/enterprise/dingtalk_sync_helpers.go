package enterprise

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

func (s *DingTalkSyncService) disableStaleRecords(ctx context.Context, taskId int, tenantId int, snapshot *dingTalkSyncSnapshot) {
	var departments []entmodel.Department
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND source_type = ?", tenantId, constant.DepartmentSourceTypeDingTalk).Find(&departments).Error; err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectDepartment, "", "stale_department_query_failed")
		return
	}
	for _, department := range departments {
		if _, ok := snapshot.seenDepartmentExternalIds[department.ExternalId]; ok {
			continue
		}
		if department.Status == constant.DepartmentStatusDisabled {
			continue
		}
		if err := s.db.WithContext(ctx).Model(&department).Updates(map[string]any{"status": constant.DepartmentStatusDisabled, "sync_status": constant.DepartmentSyncStatusWarning, "sync_error": "not_seen_in_latest_sync"}).Error; err != nil {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectDepartment, department.ExternalId, "department_disable_failed")
			continue
		}
		s.incrementTaskCounter(ctx, taskId, "departments_disabled", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectDepartment, ObjectExternalId: department.ExternalId, Action: constant.DingTalkSyncLogActionDisabled, Status: constant.DingTalkSyncLogStatusSuccess, Message: "department_disabled"})
	}

	var memberships []entmodel.UserDepartment
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND external_source = ? AND status = ?", tenantId, constant.EnterpriseExternalSourceDingTalk, constant.EnterpriseMembershipStatusActive).Find(&memberships).Error; err != nil {
		s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectMembership, "", "stale_membership_query_failed")
		return
	}
	for _, membership := range memberships {
		key := fmt.Sprintf("%d:%d:%s", membership.UserId, membership.DepartmentId, membership.ExternalSource)
		if _, ok := snapshot.seenMembershipKeys[key]; ok {
			continue
		}
		now := time.Now().Unix()
		if err := s.db.WithContext(ctx).Model(&membership).Updates(map[string]any{"status": constant.EnterpriseMembershipStatusInactive, "left_at": now}).Error; err != nil {
			s.logSyncFailure(ctx, taskId, tenantId, constant.DingTalkSyncObjectMembership, membership.ExternalUserId, "membership_disable_failed")
			continue
		}
		s.incrementTaskCounter(ctx, taskId, "memberships_disabled", 1)
		s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: constant.DingTalkSyncObjectMembership, ObjectExternalId: membership.ExternalUserId, Action: constant.DingTalkSyncLogActionDisabled, Status: constant.DingTalkSyncLogStatusSuccess, Message: "membership_disabled"})
	}
}

func (s *DingTalkSyncService) updateDingTalkIdentity(ctx context.Context, tenantId int, dingTalkUser DingTalkDepartmentUserInfo, userId int) {
	identityKey := dingtalkSyncIdentityKey(dingTalkUser)
	if identityKey == "" {
		return
	}
	var existing entmodel.DingTalkIdentity
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND identity_key = ?", tenantId, identityKey).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return
	}
	if err == gorm.ErrRecordNotFound {
		_ = s.db.WithContext(ctx).Create(&entmodel.DingTalkIdentity{
			TenantId:       tenantId,
			IdentityKey:    identityKey,
			UnionId:        strings.TrimSpace(dingTalkUser.UnionId),
			ExternalUserId: strings.TrimSpace(dingTalkUser.UserId),
			UserId:         userId,
			Status:         DingTalkIdentityStatusActive,
		}).Error
		return
	}
	_ = s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"identity_key":     identityKey,
		"union_id":         strings.TrimSpace(dingTalkUser.UnionId),
		"external_user_id": strings.TrimSpace(dingTalkUser.UserId),
		"user_id":          userId,
		"status":           DingTalkIdentityStatusActive,
	}).Error
}

func (s *DingTalkSyncService) availableSyncUsername(dingTalkUser DingTalkDepartmentUserInfo) string {
	base := normalizeDingTalkUsername(firstNonEmpty(dingTalkUser.UserId, dingTalkUser.UnionId, dingTalkUser.Mobile))
	if base == "" {
		base = "dingtalk"
	}
	prefix := "dt_" + base
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
		candidate := prefix
		if len(candidate) > baseLen {
			candidate = candidate[:baseLen]
		}
		username := candidate + "_" + suffix
		if exists, err := model.CheckUserExistOrDeleted(username, ""); err == nil && !exists {
			return username
		}
	}
	return fmt.Sprintf("dt_%d", time.Now().UnixNano()%100000000)
}

func (s *DingTalkSyncService) finishTask(ctx context.Context, taskId int) error {
	var task entmodel.DingTalkSyncTask
	if err := s.db.WithContext(ctx).Where("id = ?", taskId).First(&task).Error; err != nil {
		return err
	}
	status := constant.DingTalkSyncTaskStatusSucceeded
	if task.FailedCount > 0 {
		status = constant.DingTalkSyncTaskStatusFailed
	}
	return s.db.WithContext(ctx).Model(&task).Updates(map[string]any{"status": status, "progress": 100, "finished_at": time.Now().Unix()}).Error
}

func (s *DingTalkSyncService) finishTaskFailed(ctx context.Context, taskId int, summary string) {
	_ = s.db.WithContext(ctx).Model(&entmodel.DingTalkSyncTask{}).Where("id = ?", taskId).Updates(map[string]any{"status": constant.DingTalkSyncTaskStatusFailed, "progress": 100, "failed_count": gorm.Expr("failed_count + ?", 1), "error_summary": summary, "finished_at": time.Now().Unix()}).Error
}

func (s *DingTalkSyncService) logSyncFailure(ctx context.Context, taskId int, tenantId int, objectType string, externalId string, message string) {
	s.incrementTaskCounter(ctx, taskId, "failed_count", 1)
	_ = s.db.WithContext(ctx).Model(&entmodel.DingTalkSyncTask{}).Where("id = ? AND error_summary = ?", taskId, "").Update("error_summary", message).Error
	s.writeLog(ctx, entmodel.DingTalkSyncLog{TaskId: taskId, TenantId: tenantId, ObjectType: objectType, ObjectExternalId: externalId, Action: constant.DingTalkSyncLogActionFailed, Status: constant.DingTalkSyncLogStatusFailed, Message: message})
}

func (s *DingTalkSyncService) incrementTaskCounter(ctx context.Context, taskId int, column string, delta int) {
	_ = s.db.WithContext(ctx).Model(&entmodel.DingTalkSyncTask{}).Where("id = ?", taskId).Updates(map[string]any{column: gorm.Expr(column+" + ?", delta), "progress": 80}).Error
}

func (s *DingTalkSyncService) writeLog(ctx context.Context, log entmodel.DingTalkSyncLog) {
	log.Message = truncateSyncText(log.Message, 1024)
	log.ObjectExternalId = truncateSyncText(log.ObjectExternalId, 128)
	_ = s.db.WithContext(ctx).Create(&log).Error
}

func parseDingTalkSyncScope(scope string) []int64 {
	parts := strings.FieldsFunc(scope, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ';' || r == ' '
	})
	ids := make([]int64, 0, len(parts))
	seen := map[int64]struct{}{}
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func dingtalkSyncIdentityKey(user DingTalkDepartmentUserInfo) string {
	if strings.TrimSpace(user.UnionId) != "" {
		return "union:" + strings.TrimSpace(user.UnionId)
	}
	if strings.TrimSpace(user.UserId) != "" {
		return "user:" + strings.TrimSpace(user.UserId)
	}
	return ""
}

func sameOptionalInt(a *int, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func normalizeDingTalkSyncLogPage(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func truncateSyncText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func mapDingTalkSyncTask(task entmodel.DingTalkSyncTask) DingTalkSyncTaskItem {
	return DingTalkSyncTaskItem(task)
}

func mapDingTalkSyncLog(log entmodel.DingTalkSyncLog) DingTalkSyncLogItem {
	return DingTalkSyncLogItem(log)
}
