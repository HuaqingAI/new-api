package enterprise_test

import (
	"context"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeDingTalkSyncClient struct {
	departmentsByParent map[int64][]entservice.DingTalkDepartmentInfo
	usersByDepartment   map[int64][]entservice.DingTalkDepartmentUserInfo
	userListErrors      map[int64]error
}

func (f fakeDingTalkSyncClient) GetAccessToken(context.Context, string, string) (string, error) {
	return "app-token", nil
}

func (f fakeDingTalkSyncClient) ListSubDepartments(_ context.Context, _ string, departmentId int64) ([]entservice.DingTalkDepartmentInfo, error) {
	return f.departmentsByParent[departmentId], nil
}

func (f fakeDingTalkSyncClient) ListDepartmentUsers(_ context.Context, _ string, departmentId int64) ([]entservice.DingTalkDepartmentUserInfo, error) {
	if err := f.userListErrors[departmentId]; err != nil {
		return nil, err
	}
	return f.usersByDepartment[departmentId], nil
}

func TestDingTalkSyncFullSyncCreatesTreeUsersMembershipsAndIsIdempotent(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1:  {{DeptId: 10, Name: "Engineering"}},
			10: {{DeptId: 11, Name: "Platform"}},
			11: {},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-1", UnionId: "union-1", Name: "Alice", Email: "alice@example.com"}},
			11: {{UserId: "staff-1", UnionId: "union-1", Name: "Alice", Email: "alice@example.com"}},
		},
	})

	first, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{ActorId: 900, RunInline: true})
	require.NoError(t, err)
	require.Equal(t, constant.DingTalkSyncTaskStatusSucceeded, first.Status)
	require.Equal(t, 2, first.DepartmentsCreated)
	require.Equal(t, 1, first.UsersCreated)
	require.Equal(t, 2, first.MembershipsCreated)

	second, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{ActorId: 900, RunInline: true})
	require.NoError(t, err)
	require.Equal(t, constant.DingTalkSyncTaskStatusSucceeded, second.Status)
	require.Equal(t, 0, second.DepartmentsCreated)
	require.Equal(t, 0, second.UsersCreated)
	require.Equal(t, 0, second.MembershipsCreated)
	require.Greater(t, second.SkippedCount, 0)

	var departmentCount int64
	require.NoError(t, db.Model(&entmodel.Department{}).Where("source_type = ?", constant.DepartmentSourceTypeDingTalk).Count(&departmentCount).Error)
	require.Equal(t, int64(2), departmentCount)
	var userCount int64
	require.NoError(t, db.Model(&model.User{}).Count(&userCount).Error)
	require.Equal(t, int64(1), userCount)
	var membershipCount int64
	require.NoError(t, db.Model(&entmodel.UserDepartment{}).Where("external_source = ?", constant.EnterpriseExternalSourceDingTalk).Count(&membershipCount).Error)
	require.Equal(t, int64(2), membershipCount)
}

func TestDingTalkSyncDisablesStaleRecords(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1: {{DeptId: 10, Name: "Engineering"}},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-active", UnionId: "union-active", Name: "Active"}},
		},
	})
	require.NoError(t, db.Create(&model.User{Id: 700, Username: "stale", Status: common.UserStatusEnabled, Group: "default", AffCode: "sta1"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 99, TenantId: 0, Name: "Old", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeDingTalk, ExternalId: "99"}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{TenantId: 0, UserId: 700, DepartmentId: 99, ExternalUserId: "staff-stale", ExternalSource: constant.EnterpriseExternalSourceDingTalk, Status: constant.EnterpriseMembershipStatusActive}).Error)

	task, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})

	require.NoError(t, err)
	require.Equal(t, 1, task.DepartmentsDisabled)
	require.Equal(t, 1, task.MembershipsDisabled)
	var staleDepartment entmodel.Department
	require.NoError(t, db.Where("external_id = ?", "99").First(&staleDepartment).Error)
	require.Equal(t, constant.DepartmentStatusDisabled, staleDepartment.Status)
	var staleMembership entmodel.UserDepartment
	require.NoError(t, db.Where("external_user_id = ?", "staff-stale").First(&staleMembership).Error)
	require.Equal(t, constant.EnterpriseMembershipStatusInactive, staleMembership.Status)
}

func TestDingTalkSyncLogsFailuresWithoutRollback(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1: {{DeptId: 10, Name: "Engineering"}, {DeptId: 20, Name: "Broken"}},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-1", UnionId: "union-1", Name: "Alice"}},
		},
		userListErrors: map[int64]error{20: errors.New("permission denied")},
	})

	task, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})

	require.NoError(t, err)
	require.Equal(t, constant.DingTalkSyncTaskStatusFailed, task.Status)
	require.Equal(t, 1, task.FailedCount)
	require.Equal(t, 2, task.DepartmentsCreated)
	require.Equal(t, 1, task.UsersCreated)

	logs, err := svc.ListLogs(context.Background(), entservice.DingTalkSyncLogQuery{Status: constant.DingTalkSyncLogStatusFailed})
	require.NoError(t, err)
	require.Equal(t, 1, logs.Total)
	require.Equal(t, "department_users_failed", logs.Items[0].Message)

	var userCount int64
	require.NoError(t, db.Model(&model.User{}).Count(&userCount).Error)
	require.Equal(t, int64(1), userCount)
}

func newDingTalkSyncTestService(t *testing.T, client fakeDingTalkSyncClient) (*entservice.DingTalkSyncService, *gorm.DB) {
	t.Helper()
	_, db := newDingTalkConfigTestService(t)
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))
	require.NoError(t, db.Create(&entmodel.DingTalkConfig{
		TenantId:     0,
		CorpId:       "corp-id",
		AppKey:       "app-key",
		AppSecret:    "plain-secret",
		CallbackUrl:  "https://example.com/api/oauth/dingtalk",
		SyncScope:    "1",
		LoginEnabled: true,
		SyncEnabled:  true,
	}).Error)
	if client.departmentsByParent == nil {
		client.departmentsByParent = map[int64][]entservice.DingTalkDepartmentInfo{}
	}
	if client.usersByDepartment == nil {
		client.usersByDepartment = map[int64][]entservice.DingTalkDepartmentUserInfo{}
	}
	if client.userListErrors == nil {
		client.userListErrors = map[int64]error{}
	}
	return entservice.NewDingTalkSyncService(db, client), db
}
