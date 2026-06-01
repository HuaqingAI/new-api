package enterprise

import (
	"errors"
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

type quotaAllocationBudgetReservation struct {
	beforeSnapshot string
	afterSnapshot  string
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

const (
	QuotaAllocationProcessTriggerManual = "manual_revoke"
	QuotaAllocationProcessTriggerExpiry = "balance_expiry_task"
	QuotaAllocationProcessTriggerSync   = "wallet_state_sync_task"
)

type RevokeQuotaAllocationInput struct {
	TenantId      int
	DepartmentId  int
	AllocationId  int
	ActorId       int
	RevokeReason  string
	TriggeredBy   string
	TriggeredTime int64
}

type QuotaAllocationItem struct {
	Id                     int    `json:"id"`
	TenantId               int    `json:"tenant_id"`
	DepartmentBudgetId     int    `json:"department_budget_id"`
	DepartmentId           int    `json:"department_id"`
	TargetUserId           int    `json:"target_user_id"`
	TargetUsername         string `json:"target_username"`
	TargetDisplayName      string `json:"target_display_name"`
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
	ProcessedAt            int64  `json:"processed_at"`
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
	err := s.withAllocationRetry(func() error {
		return s.db.Transaction(func(tx *gorm.DB) error {
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
			reservation, err := s.reserveBudgetQuota(tx, budget, input.CommittedQuota)
			if err != nil {
				return err
			}

			wallet, err := model.CreateEnterpriseAllocationSubscriptionTx(tx, input.TargetUserId, 0, input.CommittedQuota, budget.CycleType, budget.CycleStartedAt, budget.CustomSeconds, budget.ExpiresAt)
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
				BeforeBudgetSnapshot:   reservation.beforeSnapshot,
				AfterBudgetSnapshot:    reservation.afterSnapshot,
				Status:                 entmodel.QuotaAllocationStatusActive,
			}
			if err := s.ensureNoAllocationWalletConflict(tx, wallet.Id); err != nil {
				return err
			}
			if err := tx.Create(&allocation).Error; err != nil {
				return err
			}
			backfillResult := tx.Model(&model.UserSubscription{}).
				Where("id = ? AND source_allocation_id = 0", wallet.Id).
				Update("source_allocation_id", allocation.Id)
			if backfillResult.Error != nil {
				return backfillResult.Error
			}
			if backfillResult.RowsAffected == 0 {
				return errors.New("enterprise allocation wallet backfill conflict")
			}
			if err := s.ensureAllocationBackfillInvariant(tx, wallet.Id, allocation.Id); err != nil {
				return err
			}
			wallet.SourceAllocationId = allocation.Id

			result = mapQuotaAllocationItem(allocation)
			return nil
		})
	})
	return result, err
}

func (s *QuotaAllocationService) Revoke(input RevokeQuotaAllocationInput) (QuotaAllocationItem, error) {
	if input.AllocationId <= 0 || input.DepartmentId <= 0 {
		return QuotaAllocationItem{}, ErrQuotaAllocationInvalidInput
	}
	if input.TriggeredTime <= 0 {
		input.TriggeredTime = common.GetTimestamp()
	}
	if strings.TrimSpace(input.TriggeredBy) == "" {
		input.TriggeredBy = QuotaAllocationProcessTriggerManual
	}
	if input.TriggeredBy == QuotaAllocationProcessTriggerManual && input.ActorId <= 0 {
		return QuotaAllocationItem{}, ErrQuotaAllocationInvalidInput
	}

	var result QuotaAllocationItem
	err := s.withAllocationRetry(func() error {
		return s.db.Transaction(func(tx *gorm.DB) error {
			var allocation entmodel.QuotaAllocation
			query := tx.Set("gorm:query_option", "FOR UPDATE").
				Where("id = ? AND department_id = ?", input.AllocationId, input.DepartmentId)
			if input.TenantId > 0 {
				query = query.Where("tenant_id = ?", input.TenantId)
			}
			if err := query.First(&allocation).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return ErrQuotaAllocationBudgetNotFound
				}
				return err
			}

			if allocation.ProcessedAt > 0 || allocation.Status == entmodel.QuotaAllocationStatusRevoked || allocation.Status == entmodel.QuotaAllocationStatusExpired {
				result = mapQuotaAllocationItem(allocation)
				return nil
			}

			var budget entmodel.DepartmentBudget
			if err := tx.Set("gorm:query_option", "FOR UPDATE").
				Where("id = ? AND department_id = ?", allocation.DepartmentBudgetId, allocation.DepartmentId).
				First(&budget).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return ErrQuotaAllocationBudgetNotFound
				}
				return err
			}
			var wallet model.UserSubscription
			if err := tx.Set("gorm:query_option", "FOR UPDATE").
				Where("id = ? AND source_allocation_id = ?", allocation.WalletId, allocation.Id).
				First(&wallet).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return ErrQuotaAllocationWalletNotFound
				}
				return err
			}

			nextStatus := entmodel.QuotaAllocationStatusRevoked
			walletStatus := "revoked"
			if input.TriggeredBy == QuotaAllocationProcessTriggerExpiry {
				nextStatus = entmodel.QuotaAllocationStatusExpired
				walletStatus = "expired"
			}

				beforeSnapshot, err := marshalDepartmentBudgetSnapshot(budget)
			if err != nil {
				return err
			}
			if err := s.applyAllocationRevokeBudgetUpdate(tx, &budget, allocation, wallet); err != nil {
				return err
			}
			if err := tx.Model(&model.UserSubscription{}).
				Where("id = ? AND status IN ?", wallet.Id, []string{"active", "paused"}).
				Updates(map[string]any{
					"status":     walletStatus,
					"updated_at": common.GetTimestamp(),
				}).Error; err != nil {
				return err
			}

			var refreshedBudget entmodel.DepartmentBudget
			if err := tx.Where("id = ?", budget.Id).First(&refreshedBudget).Error; err != nil {
				return err
			}
				afterSnapshot, err := marshalDepartmentBudgetSnapshot(refreshedBudget)
			if err != nil {
				return err
			}
			reason := strings.TrimSpace(allocation.Reason)
			if extraReason := strings.TrimSpace(input.RevokeReason); extraReason != "" {
				if reason != "" {
					reason += " | "
				}
				reason += extraReason
			}
			if trigger := strings.TrimSpace(input.TriggeredBy); trigger != "" {
				if reason != "" {
					reason += " | "
				}
				reason += trigger
			}
			if err := tx.Model(&entmodel.QuotaAllocation{}).
				Where("id = ? AND processed_at = 0", allocation.Id).
				Updates(map[string]any{
					"status":                 nextStatus,
					"processed_at":           input.TriggeredTime,
					"actor_id":               input.ActorId,
					"reason":                 reason,
					"before_budget_snapshot": beforeSnapshot,
					"after_budget_snapshot":  afterSnapshot,
					"updated_at":             common.GetTimestamp(),
				}).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", allocation.Id).First(&allocation).Error; err != nil {
				return err
			}
			result = mapQuotaAllocationItem(allocation)
			return nil
		})
	})
	return result, err
}

func (s *QuotaAllocationService) withAllocationRetry(run func() error) error {
	return withBudgetMutationRetry(run)
}

func shouldRetryQuotaAllocationTx(err error) bool {
	return shouldRetryBudgetMutationTx(err)
}

func (s *QuotaAllocationService) reserveBudgetQuota(tx *gorm.DB, budget entmodel.DepartmentBudget, committedQuota int64) (*quotaAllocationBudgetReservation, error) {
	return reserveDepartmentBudgetQuota(tx, budget, committedQuota)
}

func (s *QuotaAllocationService) ensureNoAllocationWalletConflict(tx *gorm.DB, walletId int) error {
	var count int64
	if err := tx.Model(&entmodel.QuotaAllocation{}).
		Where("wallet_id = ? AND status NOT IN ?", walletId, []string{entmodel.QuotaAllocationStatusRevoked, entmodel.QuotaAllocationStatusExpired, entmodel.QuotaAllocationStatusCanceled}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("enterprise allocation wallet already linked")
	}
	return nil
}

func (s *QuotaAllocationService) applyAllocationRevokeBudgetUpdate(tx *gorm.DB, budget *entmodel.DepartmentBudget, allocation entmodel.QuotaAllocation, wallet model.UserSubscription) error {
	if tx == nil || budget == nil {
		return ErrQuotaAllocationInvalidInput
	}
	unspent := wallet.AmountTotal - wallet.AmountUsed
	if unspent < 0 {
		unspent = 0
	}
	consumed := wallet.AmountUsed
	if consumed < 0 {
		consumed = 0
	}
	refundable := allocation.CommittedQuota - consumed
	if refundable < 0 {
		refundable = 0
	}
	if unspent < refundable {
		refundable = unspent
	}

	switch allocation.BudgetTypeSnapshot {
	case entmodel.DepartmentBudgetTypeSubscription:
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ?", budget.Id).
			Updates(map[string]any{
				"allocated_total": gorm.Expr("CASE WHEN allocated_total >= ? THEN allocated_total - ? ELSE 0 END", allocation.CommittedQuota, allocation.CommittedQuota),
				"remaining":       gorm.Expr("CASE WHEN cycle_quota >= (CASE WHEN allocated_total >= ? THEN allocated_total - ? ELSE 0 END) THEN cycle_quota - (CASE WHEN allocated_total >= ? THEN allocated_total - ? ELSE 0 END) ELSE 0 END", allocation.CommittedQuota, allocation.CommittedQuota, allocation.CommittedQuota, allocation.CommittedQuota),
				"updated_at":      common.GetTimestamp(),
			}).Error
	default:
		if refundable <= 0 {
			return nil
		}
		return tx.Model(&entmodel.DepartmentBudget{}).
			Where("id = ?", budget.Id).
			Updates(map[string]any{
				"remaining":  gorm.Expr("remaining + ?", refundable),
				"updated_at": common.GetTimestamp(),
			}).Error
	}
}

func (s *QuotaAllocationService) ensureAllocationBackfillInvariant(tx *gorm.DB, walletId int, allocationId int) error {
	var wallet model.UserSubscription
	if err := tx.Where("id = ?", walletId).First(&wallet).Error; err != nil {
		return err
	}
	if wallet.SourceAllocationId != allocationId {
		return errors.New("enterprise allocation wallet backfill mismatch")
	}
	return nil
}

func (s *QuotaAllocationService) ListByBudget(tenantId int, departmentBudgetId int) ([]QuotaAllocationItem, error) {
	if departmentBudgetId <= 0 {
		return []QuotaAllocationItem{}, ErrQuotaAllocationInvalidInput
	}
	type quotaAllocationListRow struct {
		entmodel.QuotaAllocation
		TargetUsername    string
		TargetDisplayName string
	}
	var rows []quotaAllocationListRow
	query := s.db.Model(&entmodel.QuotaAllocation{}).
		Select(
			"enterprise_quota_allocations.*",
			"users.username AS target_username",
			"users.display_name AS target_display_name",
		).
		Joins("LEFT JOIN users ON users.id = enterprise_quota_allocations.target_user_id").
		Where("department_budget_id = ?", departmentBudgetId)
	if tenantId > 0 {
		query = query.Where("tenant_id = ?", tenantId)
	}
	if err := query.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]QuotaAllocationItem, 0, len(rows))
	for _, row := range rows {
		item := mapQuotaAllocationItem(row.QuotaAllocation)
		item.TargetUsername = row.TargetUsername
		item.TargetDisplayName = row.TargetDisplayName
		items = append(items, item)
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
		ProcessedAt:            allocation.ProcessedAt,
		CreatedAt:              allocation.CreatedAt,
		UpdatedAt:              allocation.UpdatedAt,
	}
}
