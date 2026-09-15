package enterprise

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type BudgetDelegationService struct {
	db *gorm.DB
}

type CreateBudgetDelegationInput struct {
	TenantId           int
	SourceDepartmentId int
	SourceBudgetId     int
	TargetDepartmentId int
	TargetBudgetId     int
	ActorId            int
	CommittedQuota     int64
	Reason             string
}

type SupersedeBudgetDelegationInput struct {
	TenantId           int
	DelegationId       int
	SourceDepartmentId int
	ActorId            int
	NewCommittedQuota  int64
	Reason             string
}

type BudgetDelegationListQuery struct {
	TenantId     int
	DepartmentId int
}

type BudgetDelegationItem struct {
	Id                         int    `json:"id"`
	TenantId                   int    `json:"tenant_id"`
	SourceDepartmentId         int    `json:"source_department_id"`
	SourceDepartmentName       string `json:"source_department_name"`
	SourceBudgetId             int    `json:"source_budget_id"`
	TargetDepartmentId         int    `json:"target_department_id"`
	TargetDepartmentName       string `json:"target_department_name"`
	TargetBudgetId             int    `json:"target_budget_id"`
	ActorId                    int    `json:"actor_id"`
	CommittedQuota             int64  `json:"committed_quota"`
	BudgetTypeSnapshot         string `json:"budget_type_snapshot"`
	CycleTypeSnapshot          string `json:"cycle_type_snapshot"`
	BeforeSourceBudgetSnapshot string `json:"before_source_budget_snapshot"`
	AfterSourceBudgetSnapshot  string `json:"after_source_budget_snapshot"`
	BeforeTargetBudgetSnapshot string `json:"before_target_budget_snapshot"`
	AfterTargetBudgetSnapshot  string `json:"after_target_budget_snapshot"`
	Status                     string `json:"status"`
	SupersededById             int    `json:"superseded_by_id"`
	ProcessedAt                int64  `json:"processed_at"`
	Reason                     string `json:"reason"`
	CreatedAt                  int64  `json:"created_at"`
	UpdatedAt                  int64  `json:"updated_at"`
}

func NewBudgetDelegationService(db *gorm.DB) *BudgetDelegationService {
	return &BudgetDelegationService{db: db}
}

func (s *BudgetDelegationService) Create(input CreateBudgetDelegationInput) (BudgetDelegationItem, error) {
	if input.SourceBudgetId <= 0 || input.SourceDepartmentId <= 0 || input.TargetBudgetId <= 0 || input.TargetDepartmentId <= 0 || input.ActorId <= 0 {
		return BudgetDelegationItem{}, ErrBudgetDelegationInvalidInput
	}
	if input.CommittedQuota <= 0 {
		return BudgetDelegationItem{}, ErrBudgetDelegationQuotaInvalid
	}
	if input.SourceDepartmentId == input.TargetDepartmentId {
		return BudgetDelegationItem{}, ErrBudgetDelegationTargetNotDescendant
	}

	var result BudgetDelegationItem
	err := withBudgetMutationRetry(func() error {
		return s.db.Transaction(func(tx *gorm.DB) error {
			var err error
			result, err = s.createTx(tx, input)
			return err
		})
	})
	return result, err
}

func (s *BudgetDelegationService) Supersede(input SupersedeBudgetDelegationInput) (BudgetDelegationItem, error) {
	if input.DelegationId <= 0 || input.SourceDepartmentId <= 0 || input.ActorId <= 0 {
		return BudgetDelegationItem{}, ErrBudgetDelegationInvalidInput
	}
	if input.NewCommittedQuota <= 0 {
		return BudgetDelegationItem{}, ErrBudgetDelegationQuotaInvalid
	}

	var result BudgetDelegationItem
	err := withBudgetMutationRetry(func() error {
		return s.db.Transaction(func(tx *gorm.DB) error {
			var existing entmodel.BudgetDelegation
			query := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ? AND source_department_id = ?", input.DelegationId, input.SourceDepartmentId)
			if input.TenantId > 0 {
				query = query.Where("tenant_id = ?", input.TenantId)
			}
			if err := query.First(&existing).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrBudgetDelegationNotFound
				}
				return err
			}
			if existing.Status != entmodel.BudgetDelegationStatusActive {
				return ErrBudgetDelegationNotFound
			}

			sourceBudget, targetBudget, err := s.lockDelegationBudgets(tx, existing.TenantId, existing.SourceDepartmentId, existing.SourceBudgetId, existing.TargetDepartmentId, existing.TargetBudgetId)
			if err != nil {
				return err
			}
			if err := s.validateDelegationGovernance(tx, input.ActorId, existing.TenantId, existing.SourceDepartmentId, existing.TargetDepartmentId); err != nil {
				return err
			}
			if err := s.validateBudgetDelegationType(sourceBudget, targetBudget); err != nil {
				return err
			}

			if err := applyBudgetReleaseUpdate(tx, sourceBudget, existing.CommittedQuota); err != nil {
				return err
			}
			reverseTarget, reverseErr := buildBudgetDelegationReverseTargetUpdate(tx, targetBudget, existing.CommittedQuota)
			if reverseTarget.Error != nil {
				return reverseTarget.Error
			}
			if reverseTarget.RowsAffected == 0 {
				return reverseErr
			}

			now := common.GetTimestamp()
			next, err := s.createTx(tx, CreateBudgetDelegationInput{
				TenantId:           existing.TenantId,
				SourceDepartmentId: existing.SourceDepartmentId,
				SourceBudgetId:     existing.SourceBudgetId,
				TargetDepartmentId: existing.TargetDepartmentId,
				TargetBudgetId:     existing.TargetBudgetId,
				ActorId:            input.ActorId,
				CommittedQuota:     input.NewCommittedQuota,
				Reason:             strings.TrimSpace(input.Reason),
			})
			if err != nil {
				return err
			}

			if err := tx.Model(&entmodel.BudgetDelegation{}).
				Where("id = ?", existing.Id).
				Updates(map[string]any{
					"status":           entmodel.BudgetDelegationStatusSuperseded,
					"superseded_by_id": next.Id,
					"processed_at":     now,
					"actor_id":         input.ActorId,
					"updated_at":       now,
				}).Error; err != nil {
				return err
			}
			result, err = s.getByIDTx(tx, existing.TenantId, next.Id)
			return err
		})
	})
	return result, err
}

func (s *BudgetDelegationService) createTx(tx *gorm.DB, input CreateBudgetDelegationInput) (BudgetDelegationItem, error) {
	sourceBudget, targetBudget, err := s.lockDelegationBudgets(tx, input.TenantId, input.SourceDepartmentId, input.SourceBudgetId, input.TargetDepartmentId, input.TargetBudgetId)
	if err != nil {
		return BudgetDelegationItem{}, err
	}
	if err := s.validateCreateDelegation(tx, input, sourceBudget, targetBudget); err != nil {
		return BudgetDelegationItem{}, err
	}

	sourceReservation, err := reserveDepartmentBudgetQuota(tx, sourceBudget, input.CommittedQuota)
	if err != nil {
		return BudgetDelegationItem{}, err
	}

	beforeTargetSnapshot, err := marshalDepartmentBudgetSnapshot(targetBudget)
	if err != nil {
		return BudgetDelegationItem{}, err
	}
	targetUpdate := buildBudgetCreditUpdate(tx, targetBudget, input.CommittedQuota)
	if targetUpdate.Error != nil {
		return BudgetDelegationItem{}, targetUpdate.Error
	}
	var refreshedTarget entmodel.DepartmentBudget
	if err := tx.Where("id = ?", targetBudget.Id).First(&refreshedTarget).Error; err != nil {
		return BudgetDelegationItem{}, err
	}
	afterTargetSnapshot, err := marshalDepartmentBudgetSnapshot(refreshedTarget)
	if err != nil {
		return BudgetDelegationItem{}, err
	}

	delegation := entmodel.BudgetDelegation{
		TenantId:                   input.TenantId,
		SourceDepartmentId:         input.SourceDepartmentId,
		SourceBudgetId:             input.SourceBudgetId,
		TargetDepartmentId:         input.TargetDepartmentId,
		TargetBudgetId:             input.TargetBudgetId,
		ActorId:                    input.ActorId,
		CommittedQuota:             input.CommittedQuota,
		BudgetTypeSnapshot:         sourceBudget.Type,
		CycleTypeSnapshot:          sourceBudget.CycleType,
		BeforeSourceBudgetSnapshot: sourceReservation.beforeSnapshot,
		AfterSourceBudgetSnapshot:  sourceReservation.afterSnapshot,
		BeforeTargetBudgetSnapshot: beforeTargetSnapshot,
		AfterTargetBudgetSnapshot:  afterTargetSnapshot,
		Status:                     entmodel.BudgetDelegationStatusActive,
		Reason:                     strings.TrimSpace(input.Reason),
	}
	if err := tx.Create(&delegation).Error; err != nil {
		return BudgetDelegationItem{}, err
	}

	return s.getByIDTx(tx, input.TenantId, delegation.Id)
}

func (s *BudgetDelegationService) List(query BudgetDelegationListQuery) ([]BudgetDelegationItem, error) {
	if query.DepartmentId <= 0 {
		return []BudgetDelegationItem{}, ErrBudgetDelegationInvalidInput
	}
	type row struct {
		entmodel.BudgetDelegation
		SourceDepartmentName string
		TargetDepartmentName string
	}
	rows := make([]row, 0)
	dbQuery := s.db.Table(entmodel.BudgetDelegation{}.TableName()+" AS delegations").
		Select(strings.Join([]string{
			"delegations.*",
			"src.name AS source_department_name",
			"dst.name AS target_department_name",
		}, ", ")).
		Joins("LEFT JOIN enterprise_departments AS src ON src.id = delegations.source_department_id").
		Joins("LEFT JOIN enterprise_departments AS dst ON dst.id = delegations.target_department_id").
		Where("(delegations.source_department_id = ? OR delegations.target_department_id = ?)", query.DepartmentId, query.DepartmentId).
		Order("delegations.id DESC")
	if query.TenantId > 0 {
		dbQuery = dbQuery.Where("delegations.tenant_id = ?", query.TenantId)
	}
	if err := dbQuery.Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]BudgetDelegationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapBudgetDelegationItem(row.BudgetDelegation, row.SourceDepartmentName, row.TargetDepartmentName))
	}
	return items, nil
}

func (s *BudgetDelegationService) Get(tenantId int, delegationId int, departmentId int) (BudgetDelegationItem, error) {
	if delegationId <= 0 || departmentId <= 0 {
		return BudgetDelegationItem{}, ErrBudgetDelegationInvalidInput
	}
	item, err := s.getByIDTx(s.db, tenantId, delegationId)
	if err != nil {
		return BudgetDelegationItem{}, err
	}
	if item.SourceDepartmentId != departmentId && item.TargetDepartmentId != departmentId {
		return BudgetDelegationItem{}, ErrBudgetDelegationNotFound
	}
	return item, nil
}

func (s *BudgetDelegationService) getByIDTx(tx *gorm.DB, tenantId int, delegationId int) (BudgetDelegationItem, error) {
	type row struct {
		entmodel.BudgetDelegation
		SourceDepartmentName string
		TargetDepartmentName string
	}
	var result row
	query := tx.Table(entmodel.BudgetDelegation{}.TableName()+" AS delegations").
		Select(strings.Join([]string{
			"delegations.*",
			"src.name AS source_department_name",
			"dst.name AS target_department_name",
		}, ", ")).
		Joins("LEFT JOIN enterprise_departments AS src ON src.id = delegations.source_department_id").
		Joins("LEFT JOIN enterprise_departments AS dst ON dst.id = delegations.target_department_id").
		Where("delegations.id = ?", delegationId)
	if tenantId > 0 {
		query = query.Where("delegations.tenant_id = ?", tenantId)
	}
	if err := query.First(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return BudgetDelegationItem{}, ErrBudgetDelegationNotFound
		}
		return BudgetDelegationItem{}, err
	}
	return mapBudgetDelegationItem(result.BudgetDelegation, result.SourceDepartmentName, result.TargetDepartmentName), nil
}

func (s *BudgetDelegationService) lockDelegationBudgets(tx *gorm.DB, tenantId int, sourceDepartmentId int, sourceBudgetId int, targetDepartmentId int, targetBudgetId int) (entmodel.DepartmentBudget, entmodel.DepartmentBudget, error) {
	var sourceBudget entmodel.DepartmentBudget
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("id = ? AND tenant_id = ? AND department_id = ?", sourceBudgetId, tenantId, sourceDepartmentId).
		First(&sourceBudget).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entmodel.DepartmentBudget{}, entmodel.DepartmentBudget{}, ErrQuotaAllocationBudgetNotFound
		}
		return entmodel.DepartmentBudget{}, entmodel.DepartmentBudget{}, err
	}
	var targetBudget entmodel.DepartmentBudget
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("id = ? AND tenant_id = ? AND department_id = ?", targetBudgetId, tenantId, targetDepartmentId).
		First(&targetBudget).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entmodel.DepartmentBudget{}, entmodel.DepartmentBudget{}, ErrQuotaAllocationBudgetNotFound
		}
		return entmodel.DepartmentBudget{}, entmodel.DepartmentBudget{}, err
	}
	return sourceBudget, targetBudget, nil
}

func (s *BudgetDelegationService) validateCreateDelegation(tx *gorm.DB, input CreateBudgetDelegationInput, sourceBudget entmodel.DepartmentBudget, targetBudget entmodel.DepartmentBudget) error {
	if sourceBudget.Status != entmodel.DepartmentBudgetStatusActive || targetBudget.Status != entmodel.DepartmentBudgetStatusActive {
		return ErrBudgetDelegationBudgetInactive
	}
	if err := s.validateBudgetDelegationType(sourceBudget, targetBudget); err != nil {
		return err
	}
	if err := s.validateDepartmentAncestry(tx, input.TenantId, input.SourceDepartmentId, input.TargetDepartmentId); err != nil {
		return err
	}
	return s.validateDelegationGovernance(tx, input.ActorId, input.TenantId, input.SourceDepartmentId, input.TargetDepartmentId)
}

func (s *BudgetDelegationService) validateBudgetDelegationType(sourceBudget entmodel.DepartmentBudget, targetBudget entmodel.DepartmentBudget) error {
	if sourceBudget.Type != targetBudget.Type {
		return ErrBudgetDelegationBudgetTypeMismatch
	}
	if sourceBudget.Type != entmodel.DepartmentBudgetTypeBalance && sourceBudget.Type != entmodel.DepartmentBudgetTypeSubscription {
		return ErrBudgetDelegationBudgetTypeMismatch
	}
	return nil
}

func (s *BudgetDelegationService) validateDepartmentAncestry(tx *gorm.DB, tenantId int, sourceDepartmentId int, targetDepartmentId int) error {
	isAncestor, err := IsDepartmentAncestor(tx, tenantId, sourceDepartmentId, targetDepartmentId)
	if err != nil {
		return err
	}
	if !isAncestor {
		return ErrBudgetDelegationTargetNotDescendant
	}
	return nil
}

func (s *BudgetDelegationService) validateDelegationGovernance(tx *gorm.DB, actorId int, tenantId int, sourceDepartmentId int, targetDepartmentId int) error {
	permissionService := NewPermissionService(tx)
	allowed, err := permissionService.CanGovernDepartment(actorId, tenantId, sourceDepartmentId)
	if err != nil {
		if errors.Is(err, ErrDepartmentOwnerDeniedByLocalRule) {
			return ErrBudgetDelegationPermissionDenied
		}
		return err
	}
	if !allowed {
		return ErrBudgetDelegationPermissionDenied
	}
	allowed, err = permissionService.CanGovernDepartment(actorId, tenantId, targetDepartmentId)
	if err != nil {
		if errors.Is(err, ErrDepartmentOwnerDeniedByLocalRule) {
			return ErrBudgetDelegationPermissionDenied
		}
		return err
	}
	if !allowed {
		return ErrBudgetDelegationPermissionDenied
	}
	return nil
}

func mapBudgetDelegationItem(delegation entmodel.BudgetDelegation, sourceDepartmentName string, targetDepartmentName string) BudgetDelegationItem {
	return BudgetDelegationItem{
		Id:                         delegation.Id,
		TenantId:                   delegation.TenantId,
		SourceDepartmentId:         delegation.SourceDepartmentId,
		SourceDepartmentName:       sourceDepartmentName,
		SourceBudgetId:             delegation.SourceBudgetId,
		TargetDepartmentId:         delegation.TargetDepartmentId,
		TargetDepartmentName:       targetDepartmentName,
		TargetBudgetId:             delegation.TargetBudgetId,
		ActorId:                    delegation.ActorId,
		CommittedQuota:             delegation.CommittedQuota,
		BudgetTypeSnapshot:         delegation.BudgetTypeSnapshot,
		CycleTypeSnapshot:          delegation.CycleTypeSnapshot,
		BeforeSourceBudgetSnapshot: delegation.BeforeSourceBudgetSnapshot,
		AfterSourceBudgetSnapshot:  delegation.AfterSourceBudgetSnapshot,
		BeforeTargetBudgetSnapshot: delegation.BeforeTargetBudgetSnapshot,
		AfterTargetBudgetSnapshot:  delegation.AfterTargetBudgetSnapshot,
		Status:                     delegation.Status,
		SupersededById:             delegation.SupersededById,
		ProcessedAt:                delegation.ProcessedAt,
		Reason:                     delegation.Reason,
		CreatedAt:                  delegation.CreatedAt,
		UpdatedAt:                  delegation.UpdatedAt,
	}
}
