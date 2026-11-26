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
	leader := true
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1:  {{DeptId: 10, Name: "Engineering"}},
			10: {{DeptId: 11, Name: "Platform"}},
			11: {},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-1", UnionId: "union-1", Name: "Alice", Email: "alice@example.com", LeaderInDept: &leader}},
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
	var syncedUser model.User
	require.NoError(t, db.First(&syncedUser).Error)
	require.Equal(t, "alice", syncedUser.Username)
	require.NotContains(t, syncedUser.Username, "dt_")
	var membershipCount int64
	require.NoError(t, db.Model(&entmodel.UserDepartment{}).Where("external_source = ?", constant.EnterpriseExternalSourceDingTalk).Count(&membershipCount).Error)
	require.Equal(t, int64(2), membershipCount)
	var ownerCount int64
	require.NoError(t, db.Model(&entmodel.DepartmentRole{}).Where("source = ? AND effect = ?", constant.EnterpriseDepartmentRoleSourceDingTalkOwner, constant.EnterpriseDepartmentRoleEffectAllow).Count(&ownerCount).Error)
	require.Equal(t, int64(1), ownerCount)
}

func TestDingTalkSyncInactivatesStaleOwnerFactWithoutTouchingManualOverrides(t *testing.T) {
	leader := true
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1: {{DeptId: 10, Name: "Engineering"}},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-owner", UnionId: "union-owner", Name: "Owner", LeaderInDept: &leader}},
		},
	})

	_, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})
	require.NoError(t, err)
	var owner model.User
	require.NoError(t, db.Where("username = ?", "owner").First(&owner).Error)
	var department entmodel.Department
	require.NoError(t, db.Where("external_id = ?", "10").First(&department).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       owner.Id,
		DepartmentId: department.Id,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualDenyOverride,
		Effect:       constant.EnterpriseDepartmentRoleEffectDeny,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	svc, _ = newDingTalkSyncTestServiceWithDB(t, db, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1: {{DeptId: 10, Name: "Engineering"}},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-owner", UnionId: "union-owner", Name: "Owner"}},
		},
	})
	_, err = svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})
	require.NoError(t, err)

	var dingTalkRole entmodel.DepartmentRole
	require.NoError(t, db.Where("user_id = ? AND department_id = ? AND source = ?", owner.Id, department.Id, constant.EnterpriseDepartmentRoleSourceDingTalkOwner).First(&dingTalkRole).Error)
	require.Equal(t, constant.EnterpriseDepartmentRoleStatusInactive, dingTalkRole.Status)
	var manualDeny entmodel.DepartmentRole
	require.NoError(t, db.Where("user_id = ? AND department_id = ? AND source = ?", owner.Id, department.Id, constant.EnterpriseDepartmentRoleSourceManualDenyOverride).First(&manualDeny).Error)
	require.Equal(t, constant.EnterpriseDepartmentRoleStatusActive, manualDeny.Status)
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
	require.Equal(t, constant.EnterpriseMembershipStatusLeft, staleMembership.Status)
}

func TestDingTalkSyncMarksEmailConflictWithoutBindingUnsafeUser(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1: {{DeptId: 10, Name: "Engineering"}},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-conflict", UnionId: "union-conflict", Name: "Conflict", Email: "taken@example.com"}},
		},
	})
	require.NoError(t, db.Create(&model.User{Id: 701, Username: "taken", Email: "taken@example.com", Status: common.UserStatusEnabled, Group: "default", AffCode: "take"}).Error)

	task, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})

	require.NoError(t, err)
	require.Equal(t, constant.DingTalkSyncTaskStatusSucceeded, task.Status)
	require.Equal(t, 0, task.UsersCreated)
	require.Equal(t, 1, task.SkippedCount)

	var conflict entmodel.DingTalkSyncConflict
	require.NoError(t, db.Where("external_user_id = ?", "staff-conflict").First(&conflict).Error)
	require.Equal(t, constant.DingTalkSyncConflictStatusPending, conflict.Status)
	require.Equal(t, "email", conflict.ConflictType)
	require.Equal(t, 701, conflict.CandidateUserId)

	var bindingCount int64
	require.NoError(t, db.Model(&entmodel.DingTalkIdentity{}).Where("identity_key = ?", "union:union-conflict").Count(&bindingCount).Error)
	require.Zero(t, bindingCount)
	var membershipCount int64
	require.NoError(t, db.Model(&entmodel.UserDepartment{}).Where("external_user_id = ?", "staff-conflict").Count(&membershipCount).Error)
	require.Zero(t, membershipCount)
}

func TestDingTalkSyncMarksMobileConflictWithoutBindingUnsafeUser(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1: {{DeptId: 10, Name: "Engineering"}},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-mobile-conflict", UnionId: "union-mobile-conflict", Name: "Mobile Conflict", Mobile: "13800000000"}},
		},
	})
	require.NoError(t, db.Create(&model.User{Id: 702, Username: "mobile", Status: common.UserStatusEnabled, Group: "default", AffCode: "mobi"}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkIdentity{TenantId: 0, IdentityKey: "union:existing-mobile", UnionId: "existing-mobile", ExternalUserId: "staff-existing", Mobile: "13800000000", UserId: 702, Status: entservice.DingTalkIdentityStatusActive}).Error)

	task, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})

	require.NoError(t, err)
	require.Equal(t, constant.DingTalkSyncTaskStatusSucceeded, task.Status)
	require.Equal(t, 0, task.UsersCreated)
	var conflict entmodel.DingTalkSyncConflict
	require.NoError(t, db.Where("external_user_id = ?", "staff-mobile-conflict").First(&conflict).Error)
	require.Equal(t, "mobile", conflict.ConflictType)
	require.Equal(t, 702, conflict.CandidateUserId)
}

func TestDingTalkSyncMarksEmailMobileConflictWithoutSingleCandidate(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1: {{DeptId: 10, Name: "Engineering"}},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			10: {{UserId: "staff-split", UnionId: "union-split", Name: "Split Candidate", Email: "split@example.com", Mobile: "13822222222"}},
		},
	})
	require.NoError(t, db.Create(&model.User{Id: 708, Username: "email-candidate", Email: "split@example.com", Status: common.UserStatusEnabled, Group: "default", AffCode: "emca"}).Error)
	require.NoError(t, db.Create(&model.User{Id: 709, Username: "mobile-owner", Status: common.UserStatusEnabled, Group: "default", AffCode: "mown"}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkIdentity{
		TenantId:       0,
		IdentityKey:    "union:mobile-owner",
		UnionId:        "mobile-owner",
		ExternalUserId: "staff-mobile-owner",
		Mobile:         "13822222222",
		UserId:         709,
		Status:         entservice.DingTalkIdentityStatusActive,
	}).Error)

	task, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})

	require.NoError(t, err)
	require.Equal(t, constant.DingTalkSyncTaskStatusSucceeded, task.Status)
	var conflict entmodel.DingTalkSyncConflict
	require.NoError(t, db.Where("external_user_id = ?", "staff-split").First(&conflict).Error)
	require.Equal(t, "email_mobile", conflict.ConflictType)
	require.Zero(t, conflict.CandidateUserId)
}

func TestDingTalkSyncResolveConflictByCandidateBindsIdentityAndMarksResolved(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{})
	require.NoError(t, db.Create(&model.User{Id: 704, Username: "candidate", Email: "candidate@example.com", Status: common.UserStatusEnabled, Group: "default", AffCode: "cand"}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkSyncConflict{
		TenantId:        0,
		TaskId:          12,
		LastTaskId:      12,
		ExternalUserId:  "staff-resolve",
		UnionId:         "union-resolve",
		Mobile:          "13900000000",
		Email:           "candidate@example.com",
		Name:            "Candidate",
		ConflictType:    "email",
		CandidateUserId: 704,
		Details:         "email_matches_existing_local_user",
		Status:          constant.DingTalkSyncConflictStatusPending,
	}).Error)

	resolved, err := svc.ResolveConflictByCandidate(context.Background(), entservice.DingTalkSyncConflictResolveInput{
		TenantId:   0,
		ConflictId: 1,
		ActorId:    999,
	})

	require.NoError(t, err)
	require.Equal(t, constant.DingTalkSyncConflictStatusResolved, resolved.Status)
	require.Equal(t, 999, resolved.ResolvedBy)
	require.NotZero(t, resolved.ResolvedAt)

	var binding entmodel.DingTalkIdentity
	require.NoError(t, db.Where("tenant_id = ? AND identity_key = ?", 0, "union:union-resolve").First(&binding).Error)
	require.Equal(t, 704, binding.UserId)
	require.Equal(t, "staff-resolve", binding.ExternalUserId)
	require.Equal(t, entservice.DingTalkIdentityStatusActive, binding.Status)
}

func TestDingTalkSyncResolveConflictUpdatesCandidateExistingBinding(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{})
	require.NoError(t, db.Create(&model.User{Id: 705, Username: "mobile-candidate", Status: common.UserStatusEnabled, Group: "default", AffCode: "mbca"}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkIdentity{
		TenantId:       0,
		IdentityKey:    "union:old-mobile",
		UnionId:        "old-mobile",
		ExternalUserId: "staff-old",
		Mobile:         "13811111111",
		UserId:         705,
		Status:         entservice.DingTalkIdentityStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkSyncConflict{
		TenantId:        0,
		TaskId:          13,
		LastTaskId:      13,
		ExternalUserId:  "staff-new",
		UnionId:         "union-new",
		Mobile:          "13811111111",
		Name:            "Mobile Candidate",
		ConflictType:    "mobile",
		CandidateUserId: 705,
		Status:          constant.DingTalkSyncConflictStatusPending,
	}).Error)

	_, err := svc.ResolveConflictByCandidate(context.Background(), entservice.DingTalkSyncConflictResolveInput{
		TenantId:   0,
		ConflictId: 1,
		ActorId:    999,
	})

	require.NoError(t, err)
	var bindings []entmodel.DingTalkIdentity
	require.NoError(t, db.Where("tenant_id = ? AND user_id = ?", 0, 705).Find(&bindings).Error)
	require.Len(t, bindings, 1)
	require.Equal(t, "union:union-new", bindings[0].IdentityKey)
	require.Equal(t, "staff-new", bindings[0].ExternalUserId)
}

func TestDingTalkSyncResolveConflictRejectsMissingCandidateAndIdentityConflict(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{})
	require.NoError(t, db.Create(&model.User{Id: 706, Username: "candidate-a", Email: "candidate-a@example.com", Status: common.UserStatusEnabled, Group: "default", AffCode: "cana"}).Error)
	require.NoError(t, db.Create(&model.User{Id: 707, Username: "candidate-b", Status: common.UserStatusEnabled, Group: "default", AffCode: "canb"}).Error)
	require.NoError(t, db.Create(&model.User{Id: 708, Username: "candidate-c", Status: common.UserStatusEnabled, Group: "default", AffCode: "canc"}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkIdentity{
		TenantId:       0,
		IdentityKey:    "union:already-bound",
		UnionId:        "already-bound",
		ExternalUserId: "staff-bound",
		UserId:         707,
		Status:         entservice.DingTalkIdentityStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkSyncConflict{
		TenantId:       0,
		TaskId:         14,
		LastTaskId:     14,
		ExternalUserId: "staff-no-candidate",
		UnionId:        "union-no-candidate",
		ConflictType:   "email",
		Status:         constant.DingTalkSyncConflictStatusPending,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkSyncConflict{
		TenantId:        0,
		TaskId:          15,
		LastTaskId:      15,
		ExternalUserId:  "staff-bound",
		UnionId:         "already-bound",
		Email:           "candidate-a@example.com",
		ConflictType:    "email",
		CandidateUserId: 706,
		Status:          constant.DingTalkSyncConflictStatusPending,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkIdentity{
		TenantId:       0,
		IdentityKey:    "union:split-mobile",
		UnionId:        "split-mobile",
		ExternalUserId: "staff-split-mobile",
		Mobile:         "13833333333",
		UserId:         708,
		Status:         entservice.DingTalkIdentityStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkSyncConflict{
		TenantId:        0,
		TaskId:          16,
		LastTaskId:      16,
		ExternalUserId:  "staff-split-bind",
		UnionId:         "union-split-bind",
		Mobile:          "13833333333",
		Email:           "candidate-a@example.com",
		ConflictType:    "email_mobile",
		CandidateUserId: 706,
		Status:          constant.DingTalkSyncConflictStatusPending,
	}).Error)

	_, err := svc.ResolveConflictByCandidate(context.Background(), entservice.DingTalkSyncConflictResolveInput{
		TenantId:   0,
		ConflictId: 1,
		ActorId:    999,
	})
	require.ErrorIs(t, err, entservice.ErrDingTalkSyncConflictNoCandidate)

	_, err = svc.ResolveConflictByCandidate(context.Background(), entservice.DingTalkSyncConflictResolveInput{
		TenantId:   0,
		ConflictId: 2,
		ActorId:    999,
	})
	require.ErrorIs(t, err, entservice.ErrDingTalkOAuthBindingConflict)

	_, err = svc.ResolveConflictByCandidate(context.Background(), entservice.DingTalkSyncConflictResolveInput{
		TenantId:   0,
		ConflictId: 3,
		ActorId:    999,
	})
	require.ErrorIs(t, err, entservice.ErrDingTalkSyncConflictNoCandidate)
}

func TestDingTalkSyncTracksDepartmentRenameAndMoveHistory(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1:  {{DeptId: 10, Name: "Engineering"}},
			10: {{DeptId: 11, Name: "Platform"}},
			11: {},
		},
	})
	parentId := 99
	require.NoError(t, db.Create(&entmodel.Department{Id: 99, TenantId: 0, Name: "Legacy Parent", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeDingTalk, ExternalId: "99"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 11, TenantId: 0, Name: "Old Platform", ParentId: &parentId, Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeDingTalk, ExternalId: "11"}).Error)

	task, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})

	require.NoError(t, err)
	require.Equal(t, 1, task.DepartmentsUpdated)
	var platform entmodel.Department
	require.NoError(t, db.Where("external_id = ?", "11").First(&platform).Error)
	require.Equal(t, "Platform", platform.Name)
	var engineering entmodel.Department
	require.NoError(t, db.Where("external_id = ?", "10").First(&engineering).Error)
	require.NotNil(t, platform.ParentId)
	require.Equal(t, engineering.Id, *platform.ParentId)
	history, err := platform.ParsedNameHistory()
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "Old Platform", history[0].Name)
	require.NotZero(t, history[0].ChangedAt)
}

func TestDingTalkSyncMarksOldMembershipLeftOnTransfer(t *testing.T) {
	svc, db := newDingTalkSyncTestService(t, fakeDingTalkSyncClient{
		departmentsByParent: map[int64][]entservice.DingTalkDepartmentInfo{
			1:  {{DeptId: 10, Name: "Engineering"}, {DeptId: 20, Name: "Security"}},
			10: {},
			20: {},
		},
		usersByDepartment: map[int64][]entservice.DingTalkDepartmentUserInfo{
			20: {{UserId: "staff-transfer", UnionId: "union-transfer", Name: "Transfer"}},
		},
	})
	require.NoError(t, db.Create(&model.User{Id: 703, Username: "transfer", Status: common.UserStatusEnabled, Group: "default", AffCode: "tran"}).Error)
	require.NoError(t, db.Create(&entmodel.DingTalkIdentity{TenantId: 0, IdentityKey: "union:union-transfer", UnionId: "union-transfer", ExternalUserId: "staff-transfer", UserId: 703, Status: entservice.DingTalkIdentityStatusActive}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 10, TenantId: 0, Name: "Engineering", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeDingTalk, ExternalId: "10"}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{TenantId: 0, UserId: 703, DepartmentId: 10, ExternalUserId: "staff-transfer", ExternalSource: constant.EnterpriseExternalSourceDingTalk, Status: constant.EnterpriseMembershipStatusActive}).Error)

	task, err := svc.StartFullSync(context.Background(), entservice.DingTalkSyncStartInput{RunInline: true})

	require.NoError(t, err)
	require.Equal(t, 1, task.MembershipsCreated)
	require.Equal(t, 1, task.MembershipsDisabled)
	var oldMembership entmodel.UserDepartment
	require.NoError(t, db.Where("user_id = ? AND department_id = ?", 703, 10).First(&oldMembership).Error)
	require.Equal(t, constant.EnterpriseMembershipStatusLeft, oldMembership.Status)
	require.NotZero(t, oldMembership.LeftAt)
	var newDepartment entmodel.Department
	require.NoError(t, db.Where("external_id = ?", "20").First(&newDepartment).Error)
	var newMembership entmodel.UserDepartment
	require.NoError(t, db.Where("user_id = ? AND department_id = ?", 703, newDepartment.Id).First(&newMembership).Error)
	require.Equal(t, constant.EnterpriseMembershipStatusActive, newMembership.Status)
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
	return newDingTalkSyncTestServiceWithDB(t, db, client)
}

func newDingTalkSyncTestServiceWithDB(t *testing.T, db *gorm.DB, client fakeDingTalkSyncClient) (*entservice.DingTalkSyncService, *gorm.DB) {
	t.Helper()
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	var existingConfig int64
	require.NoError(t, db.Model(&entmodel.DingTalkConfig{}).Where("tenant_id = ?", 0).Count(&existingConfig).Error)
	if existingConfig == 0 {
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
	}
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
