package enterprise

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type QuotaAllocationService struct {
	db *gorm.DB
}

type CreateQuotaAllocationInput struct {
	TenantId           int
	DepartmentBudgetId int
	DepartmentId       int
	TargetUserId       int
	ActorId            int
	CommittedQuota     int64
	Reason             string
}

type QuotaAllocationItem struct {
	Id                     int    `json:"id"`
	TenantId               int    `json:"tenant_id"`
	DepartmentBudgetId     int    `json:"department_budget_id"`
	DepartmentId           int    `json:"department_id"`
	TargetUserId           int    `json:"target_user_id"`
	WalletId               int    `json:"wallet_id"`
	ActorId                int    `json:"actor_id"`
	CommittedQuota         int64  `json:"committed_quota"`
	BudgetTypeSnapshot     string `json:"budget_type_snapshot"`
	CycleTypeSnapshot      string `json:"cycle_type_snapshot"`
	CycleStartedAtSnapshot int64  `json:"cycle_started_at_snapshot"`
	CustomSecondsSnapshot  int64  `json:"custom_seconds_snapshot"`
	ExpiresAtSnapshot      int64  `json:"expires_at_snapshot"`
	Reason                 string `json:"reason"`
	Status                 string `json:"status"`
	CreatedAt              int64  `json:"created_at"`
	UpdatedAt              int64  `json:"updated_at"`
}

func NewQuotaAllocationService(db *gorm.DB) *QuotaAllocationService {
	return &QuotaAllocationService{db: db}
}

func (s *QuotaAllocationService) Create(input CreateQuotaAllocationInput) (QuotaAllocationItem, error) {
	if input.DepartmentBudgetId <= 0 || input.DepartmentId <= 0 || input.TargetUserId <= 0 || input.ActorId <= 0 {
		return QuotaAllocationItem{}, ErrQuotaAllocationInvalidInput
	}
	if input.CommittedQuota <= 0 {
		return QuotaAllocationItem{}, ErrQuotaAllocationQuotaInvalid
	}
	if err := s.ensureTargetUserInDepartment(input.TenantId, input.DepartmentId, input.TargetUserId); err != nil {
		return QuotaAllocationItem{}, err
	}

	var result QuotaAllocationItem
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var budget entmodel.DepartmentBudget
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("id = ? AND tenant_id = ? AND department_id = ?", input.DepartmentBudgetId, input.TenantId, input.DepartmentId).
			First(&budget).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrQuotaAllocationBudgetNotFound
			}
			return err
		}
		if budget.Status != entmodel.DepartmentBudgetStatusActive {
			return ErrQuotaAllocationBudgetInactive
		}
		if budget.Remaining < input.CommittedQuota {
			return ErrQuotaAllocationQuotaExceeded
		}

		beforeSnapshot, err := common.Marshal(map[string]any{
			"id":                budget.Id,
			"type":              budget.Type,
			"remaining":         budget.Remaining,
			"total_quota":       budget.TotalQuota,
			"cycle_quota":       budget.CycleQuota,
			"cycle_type":        budget.CycleType,
			"cycle_started_at":  budget.CycleStartedAt,
			"custom_seconds":    budget.CustomSeconds,
			"expires_at":        budget.ExpiresAt,
			"department_id":     budget.DepartmentId,
			"department_budget": budget.Id,
		})
		if err != nil {
			return err
		}

		updateResult := tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ? AND remaining >= ?", budget.Id, input.CommittedQuota).
			Updates(map[string]any{
				"remaining":  gorm.Expr("remaining - ?", input.CommittedQuota),
				"updated_at": common.GetTimestamp(),
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return ErrQuotaAllocationQuotaExceeded
		}

		wallet, err := model.CreateEnterpriseAllocationSubscriptionTx(tx, input.TargetUserId, 0, input.CommittedQuota, budget.CycleType, budget.CycleStartedAt, budget.CustomSeconds, budget.ExpiresAt)
		if err != nil {
			return err
		}

		var refreshedBudget entmodel.DepartmentBudget
		if err := tx.Where("id = ?", budget.Id).First(&refreshedBudget).Error; err != nil {
			return err
		}
		afterSnapshot, err := common.Marshal(map[string]any{
			"id":               refreshedBudget.Id,
			"type":             refreshedBudget.Type,
			"remaining":        refreshedBudget.Remaining,
			"total_quota":      refreshedBudget.TotalQuota,
			"cycle_quota":      refreshedBudget.CycleQuota,
			"cycle_type":       refreshedBudget.CycleType,
			"cycle_started_at": refreshedBudget.CycleStartedAt,
			"custom_seconds":   refreshedBudget.CustomSeconds,
			"expires_at":       refreshedBudget.ExpiresAt,
		})
		if err != nil {
			return err
		}

		allocation := entmodel.QuotaAllocation{
			TenantId:               input.TenantId,
			DepartmentBudgetId:     budget.Id,
			DepartmentId:           input.DepartmentId,
			TargetUserId:           input.TargetUserId,
			WalletId:               wallet.Id,
			ActorId:                input.ActorId,
			CommittedQuota:         input.CommittedQuota,
			BudgetTypeSnapshot:     budget.Type,
			CycleTypeSnapshot:      budget.CycleType,
			CycleStartedAtSnapshot: budget.CycleStartedAt,
			CustomSecondsSnapshot:  budget.CustomSeconds,
			ExpiresAtSnapshot:      budget.ExpiresAt,
			Reason:                 strings.TrimSpace(input.Reason),
			BeforeBudgetSnapshot:   string(beforeSnapshot),
			AfterBudgetSnapshot:    string(afterSnapshot),
			Status:                 entmodel.QuotaAllocationStatusActive,
		}
		if err := tx.Create(&allocation).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.UserSubscription{}).
			Where("id = ?", wallet.Id).
			Update("source_allocation_id", allocation.Id).Error; err != nil {
			return err
		}
		wallet.SourceAllocationId = allocation.Id

		result = mapQuotaAllocationItem(allocation)
		return nil
	})
	return result, err
}

func (s *QuotaAllocationService) ListByBudget(tenantId int, departmentBudgetId int) ([]QuotaAllocationItem, error) {
	if departmentBudgetId <= 0 {
		return []QuotaAllocationItem{}, ErrQuotaAllocationInvalidInput
	}
	var rows []entmodel.QuotaAllocation
	query := s.db.Where("department_budget_id = ?", departmentBudgetId)
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if err := query.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]QuotaAllocationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapQuotaAllocationItem(row))
	}
	return items, nil
}

func (s *QuotaAllocationService) GetBudgetDepartment(tenantId int, departmentBudgetId int) (int, error) {
	if departmentBudgetId <= 0 {
		return 0, ErrQuotaAllocationInvalidInput
	}
	var budget entmodel.DepartmentBudget
	query := s.db.Where("id = ?", departmentBudgetId)
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if err := query.First(&budget).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, ErrQuotaAllocationBudgetNotFound
		}
		return 0, err
	}
	return budget.DepartmentId, nil
}

func (s *QuotaAllocationService) ensureTargetUserInDepartment(tenantId int, departmentId int, userId int) error {
	var user model.User
	if err := s.db.Where("id = ?", userId).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrUserNotFound
		}
		return err
	}
	var membership entmodel.UserDepartment
	if err := s.db.Where("tenant_id = ? AND department_id = ? AND user_id = ? AND status = ?", tenantId, departmentId, userId, constant.EnterpriseMembershipStatusActive).
		First(&membership).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrQuotaAllocationUserOutOfDepartment
		}
		return err
	}
	return nil
}

func mapQuotaAllocationItem(allocation entmodel.QuotaAllocation) QuotaAllocationItem {
	return QuotaAllocationItem{
		Id:                     allocation.Id,
		TenantId:               allocation.TenantId,
		DepartmentBudgetId:     allocation.DepartmentBudgetId,
		DepartmentId:           allocation.DepartmentId,
		TargetUserId:           allocation.TargetUserId,
		WalletId:               allocation.WalletId,
		ActorId:                allocation.ActorId,
		CommittedQuota:         allocation.CommittedQuota,
		BudgetTypeSnapshot:     allocation.BudgetTypeSnapshot,
		CycleTypeSnapshot:      allocation.CycleTypeSnapshot,
		CycleStartedAtSnapshot: allocation.CycleStartedAtSnapshot,
		CustomSecondsSnapshot:  allocation.CustomSecondsSnapshot,
		ExpiresAtSnapshot:      allocation.ExpiresAtSnapshot,
		Reason:                 allocation.Reason,
		Status:                 allocation.Status,
		CreatedAt:              allocation.CreatedAt,
		UpdatedAt:              allocation.UpdatedAt,
	}
}
