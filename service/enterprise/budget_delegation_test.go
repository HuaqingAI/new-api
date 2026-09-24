package enterprise_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newBudgetDelegationTestService(t *testing.T) (*entservice.BudgetDelegationService, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)

	oldDB := model.DB
	model.DB = db
	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	t.Cleanup(func() {
		model.DB = oldDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		_ = sqlDB.Close()
	})

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.SubscriptionPlan{}, &model.UserSubscription{}))
	require.NoError(t, entmodel.AutoMigrate(db))
	require.NoError(t, db.Create(&model.User{Id: 1001, Username: "owner", Password: "pwd", Group: "default", AffCode: "owner-aff"}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 1, TenantId: 0, Name: "HQ", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	parentID := 1
	require.NoError(t, db.Create(&entmodel.Department{Id: 2, TenantId: 0, Name: "Engineering", ParentId: &parentID, Status: constant.EnterpriseDepartmentStatusActive}).Error)
	childParentID := 2
	require.NoError(t, db.Create(&entmodel.Department{Id: 3, TenantId: 0, Name: "Platform", ParentId: &childParentID, Status: constant.EnterpriseDepartmentStatusActive}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 4, TenantId: 0, Name: "Finance", Status: constant.EnterpriseDepartmentStatusActive}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       1001,
		DepartmentId: 1,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualGrant,
		Effect:       constant.EnterpriseDepartmentRoleEffectAllow,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           11,
		TenantId:     0,
		DepartmentId: 1,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   1000,
		Remaining:    1000,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           12,
		TenantId:     0,
		DepartmentId: 3,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   200,
		Remaining:    200,
	}).Error)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           13,
		TenantId:     0,
		DepartmentId: 4,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   150,
		Remaining:    150,
	}).Error)

	return entservice.NewBudgetDelegationService(db), db
}

func TestCreateBudgetDelegationSupportsAncestorToDescendant(t *testing.T) {
	svc, db := newBudgetDelegationTestService(t)

	item, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 3,
		TargetBudgetId:     12,
		ActorId:            1001,
		CommittedQuota:     300,
		Reason:             "delegate to platform",
	})
	require.NoError(t, err)
	require.Equal(t, int64(300), item.CommittedQuota)
	require.Equal(t, entmodel.BudgetDelegationStatusActive, item.Status)
	require.Contains(t, item.BeforeSourceBudgetSnapshot, `"remaining":1000`)
	require.Contains(t, item.AfterTargetBudgetSnapshot, `"remaining":500`)

	var source entmodel.DepartmentBudget
	require.NoError(t, db.First(&source, 11).Error)
	require.Equal(t, int64(700), source.Remaining)
	require.Equal(t, int64(1000), source.TotalQuota)

	var target entmodel.DepartmentBudget
	require.NoError(t, db.First(&target, 12).Error)
	require.Equal(t, int64(500), target.TotalQuota)
	require.Equal(t, int64(500), target.Remaining)
}

func TestCreateBudgetDelegationSupportsDirectChild(t *testing.T) {
	svc, db := newBudgetDelegationTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:           14,
		TenantId:     0,
		DepartmentId: 2,
		Type:         entmodel.DepartmentBudgetTypeBalance,
		Status:       entmodel.DepartmentBudgetStatusActive,
		TotalQuota:   120,
		Remaining:    120,
	}).Error)

	item, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 2,
		TargetBudgetId:     14,
		ActorId:            1001,
		CommittedQuota:     180,
		Reason:             "delegate to engineering",
	})
	require.NoError(t, err)
	require.Equal(t, 2, item.TargetDepartmentId)
	require.Equal(t, int64(180), item.CommittedQuota)

	var source entmodel.DepartmentBudget
	require.NoError(t, db.First(&source, 11).Error)
	require.Equal(t, int64(820), source.Remaining)

	var target entmodel.DepartmentBudget
	require.NoError(t, db.First(&target, 14).Error)
	require.Equal(t, int64(300), target.TotalQuota)
	require.Equal(t, int64(300), target.Remaining)
}

func TestCreateBudgetDelegationRejectsNonDescendant(t *testing.T) {
	svc, _ := newBudgetDelegationTestService(t)

	_, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 4,
		TargetBudgetId:     13,
		ActorId:            1001,
		CommittedQuota:     100,
	})
	require.ErrorIs(t, err, entservice.ErrBudgetDelegationTargetNotDescendant)
}

func TestCreateBudgetDelegationRejectsTypeMismatch(t *testing.T) {
	svc, db := newBudgetDelegationTestService(t)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 12).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"cycle_quota":      int64(200),
		"cycle_type":       "monthly",
		"cycle_started_at": time.Now().Unix(),
	}).Error)

	_, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 3,
		TargetBudgetId:     12,
		ActorId:            1001,
		CommittedQuota:     100,
	})
	require.ErrorIs(t, err, entservice.ErrBudgetDelegationBudgetTypeMismatch)
}

func TestCreateBudgetDelegationRejectsSelectedBudgetTypeMismatchDespiteOtherSameDepartmentPools(t *testing.T) {
	svc, db := newBudgetDelegationTestService(t)
	now := time.Now().Unix()
	require.NoError(t, db.Create(&entmodel.DepartmentBudget{
		Id:             15,
		TenantId:       0,
		DepartmentId:   1,
		Type:           entmodel.DepartmentBudgetTypeSubscription,
		Status:         entmodel.DepartmentBudgetStatusActive,
		Remaining:      500,
		AllocatedTotal: 0,
		CycleQuota:     500,
		CycleType:      "monthly",
		CycleStartedAt: now,
	}).Error)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 12).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(200),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(200),
		"cycle_type":       "monthly",
		"cycle_started_at": now,
	}).Error)

	_, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 3,
		TargetBudgetId:     12,
		ActorId:            1001,
		CommittedQuota:     100,
	})
	require.ErrorIs(t, err, entservice.ErrBudgetDelegationBudgetTypeMismatch)

	item, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     15,
		TargetDepartmentId: 3,
		TargetBudgetId:     12,
		ActorId:            1001,
		CommittedQuota:     100,
	})
	require.NoError(t, err)
	require.Equal(t, entmodel.DepartmentBudgetTypeSubscription, item.BudgetTypeSnapshot)
}

func TestCreateBudgetDelegationRejectsBudgetInsufficiency(t *testing.T) {
	svc, _ := newBudgetDelegationTestService(t)

	_, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 3,
		TargetBudgetId:     12,
		ActorId:            1001,
		CommittedQuota:     5000,
	})
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationBudgetInsufficient)
	require.ErrorIs(t, err, entservice.ErrQuotaAllocationBalanceRemainingInsufficient)
}

func TestCreateBudgetDelegationRejectsManualDenyOnTargetDepartment(t *testing.T) {
	svc, db := newBudgetDelegationTestService(t)
	require.NoError(t, db.Create(&entmodel.DepartmentRole{
		TenantId:     0,
		UserId:       1001,
		DepartmentId: 3,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:       constant.EnterpriseDepartmentRoleSourceManualDenyOverride,
		Effect:       constant.EnterpriseDepartmentRoleEffectDeny,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}).Error)

	_, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 3,
		TargetBudgetId:     12,
		ActorId:            1001,
		CommittedQuota:     100,
	})
	require.ErrorIs(t, err, entservice.ErrBudgetDelegationPermissionDenied)
}

func TestCreateBudgetDelegationSubscriptionTracksCycleQuota(t *testing.T) {
	svc, db := newBudgetDelegationTestService(t)
	now := time.Now().Unix()
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 11).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(600),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(600),
		"cycle_type":       "monthly",
		"cycle_started_at": now,
	}).Error)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 12).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(100),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(100),
		"cycle_type":       "monthly",
		"cycle_started_at": now,
	}).Error)

	_, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 3,
		TargetBudgetId:     12,
		ActorId:            1001,
		CommittedQuota:     250,
	})
	require.NoError(t, err)

	var source entmodel.DepartmentBudget
	require.NoError(t, db.First(&source, 11).Error)
	require.Equal(t, int64(250), source.AllocatedTotal)
	require.Equal(t, int64(350), source.Remaining)

	var target entmodel.DepartmentBudget
	require.NoError(t, db.First(&target, 12).Error)
	require.Equal(t, int64(350), target.CycleQuota)
	require.Equal(t, int64(350), target.Remaining)
}

func TestSupersedeBudgetDelegationClosesOldAndCreatesNew(t *testing.T) {
	svc, db := newBudgetDelegationTestService(t)

	first, err := svc.Create(entservice.CreateBudgetDelegationInput{
		TenantId:           0,
		SourceDepartmentId: 1,
		SourceBudgetId:     11,
		TargetDepartmentId: 3,
		TargetBudgetId:     12,
		ActorId:            1001,
		CommittedQuota:     200,
	})
	require.NoError(t, err)

	next, err := svc.Supersede(entservice.SupersedeBudgetDelegationInput{
		TenantId:           0,
		DelegationId:       first.Id,
		SourceDepartmentId: 1,
		ActorId:            1001,
		NewCommittedQuota:  350,
		Reason:             "increase",
	})
	require.NoError(t, err)
	require.NotEqual(t, first.Id, next.Id)

	var oldLedger entmodel.BudgetDelegation
	require.NoError(t, db.First(&oldLedger, first.Id).Error)
	require.Equal(t, entmodel.BudgetDelegationStatusSuperseded, oldLedger.Status)
	require.Equal(t, next.Id, oldLedger.SupersededById)
	require.NotZero(t, oldLedger.ProcessedAt)

	var source entmodel.DepartmentBudget
	require.NoError(t, db.First(&source, 11).Error)
	require.Equal(t, int64(650), source.Remaining)

	var target entmodel.DepartmentBudget
	require.NoError(t, db.First(&target, 12).Error)
	require.Equal(t, int64(550), target.TotalQuota)
	require.Equal(t, int64(550), target.Remaining)
}

func TestCreateBudgetDelegationConcurrentSubscriptionRespectsCycleQuotaInvariant(t *testing.T) {
	svc, db := newBudgetDelegationTestService(t)
	now := time.Now().Unix()
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 11).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(600),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(600),
		"cycle_type":       "monthly",
		"cycle_started_at": now,
	}).Error)
	require.NoError(t, db.Model(&entmodel.DepartmentBudget{}).Where("id = ?", 12).Updates(map[string]any{
		"type":             entmodel.DepartmentBudgetTypeSubscription,
		"remaining":        int64(100),
		"allocated_total":  int64(0),
		"cycle_quota":      int64(100),
		"cycle_type":       "monthly",
		"cycle_started_at": now,
	}).Error)

	const workers = 10
	const quotaPerRequest int64 = 80
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Create(entservice.CreateBudgetDelegationInput{
				TenantId:           0,
				SourceDepartmentId: 1,
				SourceBudgetId:     11,
				TargetDepartmentId: 3,
				TargetBudgetId:     12,
				ActorId:            1001,
				CommittedQuota:     quotaPerRequest,
			})
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successCount := 0
	for err := range results {
		if err == nil {
			successCount++
			continue
		}
		require.ErrorIs(t, err, entservice.ErrQuotaAllocationBudgetInsufficient)
		require.ErrorIs(t, err, entservice.ErrQuotaAllocationSubscriptionCycleAllocatedExceeded)
	}
	require.Equal(t, 7, successCount)

	var source entmodel.DepartmentBudget
	require.NoError(t, db.First(&source, 11).Error)
	require.Equal(t, int64(successCount)*quotaPerRequest, source.AllocatedTotal)
	require.Equal(t, int64(600)-int64(successCount)*quotaPerRequest, source.Remaining)

	var target entmodel.DepartmentBudget
	require.NoError(t, db.First(&target, 12).Error)
	require.Equal(t, int64(100)+int64(successCount)*quotaPerRequest, target.CycleQuota)
	require.Equal(t, target.CycleQuota, target.Remaining)

	var committedTotal int64
	require.NoError(t, db.Model(&entmodel.BudgetDelegation{}).
		Where("status = ?", entmodel.BudgetDelegationStatusActive).
		Select("COALESCE(SUM(committed_quota), 0)").
		Scan(&committedTotal).Error)
	require.Equal(t, source.AllocatedTotal, committedTotal)
	require.LessOrEqual(t, source.AllocatedTotal, source.CycleQuota)
}
