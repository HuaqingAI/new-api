package enterprise

import (
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

type DepartmentBudgetService struct {
	db *gorm.DB
}

type CreateDepartmentBudgetInput struct {
	TenantId       int
	Type           string
	TotalQuota     *int64
	CycleQuota     *int64
	CycleType      string
	CycleStartedAt *int64
	CustomSeconds  *int64
	ExpiresAt      *int64
}

type ResizeDepartmentBudgetInput struct {
	TotalQuota *int64
	CycleQuota *int64
}

type DepartmentBudgetSortField string

const (
	DepartmentBudgetSortUsageRatio DepartmentBudgetSortField = "usage_ratio"
	DepartmentBudgetSortRemaining  DepartmentBudgetSortField = "remaining"
	DepartmentBudgetSortType       DepartmentBudgetSortField = "type"
	DepartmentBudgetSortStatus     DepartmentBudgetSortField = "status"
)

type DepartmentBudgetThresholdState string

const (
	DepartmentBudgetThresholdStateHealthy  DepartmentBudgetThresholdState = "healthy"
	DepartmentBudgetThresholdStateWarning  DepartmentBudgetThresholdState = "warning"
	DepartmentBudgetThresholdStateCritical DepartmentBudgetThresholdState = "critical"
)

type DepartmentBudgetListQuery struct {
	SortBy             string
	SortOrder          string
	IncludeDescendants bool
}

type DepartmentBudgetThresholds struct {
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}

type DepartmentBudgetItem struct {
	Id             int                            `json:"id"`
	TenantId       int                            `json:"tenant_id"`
	DepartmentId   int                            `json:"department_id"`
	DepartmentName string                         `json:"department_name"`
	Type           string                         `json:"type"`
	Status         string                         `json:"status"`
	TotalQuota     int64                          `json:"total_quota"`
	Remaining      int64                          `json:"remaining"`
	AllocatedTotal int64                          `json:"allocated_total"`
	CycleQuota     int64                          `json:"cycle_quota"`
	CycleType      string                         `json:"cycle_type"`
	CycleStartedAt int64                          `json:"cycle_started_at"`
	CustomSeconds  int64                          `json:"custom_seconds"`
	ExpiresAt      int64                          `json:"expires_at"`
	ParentStatus   string                         `json:"parent_status"`
	UsageRatio     float64                        `json:"usage_ratio"`
	ThresholdState DepartmentBudgetThresholdState `json:"threshold_state"`
	CreatedAt      int64                          `json:"created_at"`
	UpdatedAt      int64                          `json:"updated_at"`
}

type DepartmentBudgetListResult struct {
	Items               []DepartmentBudgetItem     `json:"items"`
	Thresholds          DepartmentBudgetThresholds `json:"thresholds"`
	ScopeDepartmentId   *int                       `json:"scope_department_id,omitempty"`
	ScopeDepartmentName string                     `json:"scope_department_name"`
	IncludeDescendants  bool                       `json:"include_descendants"`
	ScopeDepartmentIds  []int                      `json:"scope_department_ids"`
}

type DepartmentBudgetWalletDetail struct {
	AllocationId             int    `json:"allocation_id"`
	AllocationStatus         string `json:"allocation_status"`
	TargetUserId             int    `json:"target_user_id"`
	TargetUsername           string `json:"target_username"`
	TargetDisplayName        string `json:"target_display_name"`
	WalletId                 int    `json:"wallet_id"`
	WalletStatus             string `json:"wallet_status"`
	Quota                    int64  `json:"quota"`
	RemainQuota              int64  `json:"remain_quota"`
	CycleType                string `json:"cycle_type"`
	CycleStartedAt           int64  `json:"cycle_started_at"`
	NextResetTime            int64  `json:"next_reset_time"`
	ExpiresAt                int64  `json:"expires_at"`
	SourceAllocationId       int    `json:"source_allocation_id"`
	SourceParentBudgetId     int    `json:"source_parent_budget_id"`
	SourceParentBudgetType   string `json:"source_parent_budget_type"`
	SourceParentBudgetStatus string `json:"source_parent_budget_status"`
	CommittedQuota           int64  `json:"committed_quota"`
	ProcessedAt              int64  `json:"processed_at"`
	CreatedAt                int64  `json:"created_at"`
	UpdatedAt                int64  `json:"updated_at"`
	Reason                   string `json:"reason"`
}

type DepartmentBudgetDetailResult struct {
	Budget     DepartmentBudgetItem           `json:"budget"`
	Wallets    []DepartmentBudgetWalletDetail `json:"wallets"`
	Thresholds DepartmentBudgetThresholds     `json:"thresholds"`
}

type departmentBudgetWalletDetailRow struct {
	entmodel.QuotaAllocation
	TargetUsername    string
	TargetDisplayName string
	WalletAmountTotal int64
	WalletAmountUsed  int64
	WalletStatus      string
	WalletNextReset   int64
	WalletEndTime     int64
	WalletSourceAlloc int
}

func normalizeDepartmentBudgetForDisplay(
	budget entmodel.DepartmentBudget,
) entmodel.DepartmentBudget {
	switch budget.Type {
	case entmodel.DepartmentBudgetTypeSubscription:
		if budget.AllocatedTotal < 0 {
			budget.AllocatedTotal = 0
		}
		if budget.CycleQuota < 0 {
			budget.CycleQuota = 0
		}
		budget.Remaining = maxInt64(budget.CycleQuota-budget.AllocatedTotal, 0)
	default:
		if budget.TotalQuota < 0 {
			budget.TotalQuota = 0
		}
		if budget.Remaining < 0 {
			budget.Remaining = 0
		}
		if budget.TotalQuota > 0 && budget.Remaining > budget.TotalQuota {
			budget.Remaining = budget.TotalQuota
		}
	}
	return budget
}

func NewDepartmentBudgetService(db *gorm.DB) *DepartmentBudgetService {
	return &DepartmentBudgetService{db: db}
}

func (s *DepartmentBudgetService) Create(departmentId int, input CreateDepartmentBudgetInput) (DepartmentBudgetItem, error) {
	if departmentId <= 0 {
		return DepartmentBudgetItem{}, ErrInvalidDepartmentBudgetInput
	}
	if err := s.ensureDepartmentExists(input.TenantId, departmentId); err != nil {
		return DepartmentBudgetItem{}, err
	}

	budgetType := strings.TrimSpace(input.Type)

	switch budgetType {
	case entmodel.DepartmentBudgetTypeBalance:
		if input.TotalQuota == nil || *input.TotalQuota <= 0 {
			return DepartmentBudgetItem{}, ErrDepartmentBudgetInvalidQuota
		}
		budget := entmodel.DepartmentBudget{
			TenantId:     input.TenantId,
			DepartmentId: departmentId,
			Type:         budgetType,
			Status:       entmodel.DepartmentBudgetStatusActive,
			TotalQuota:   *input.TotalQuota,
			Remaining:    *input.TotalQuota,
			ExpiresAt:    int64OrZero(input.ExpiresAt),
		}
		if err := s.db.Create(&budget).Error; err != nil {
			return DepartmentBudgetItem{}, err
		}
		return s.mapDepartmentBudgetItem(budget, ""), nil
	case entmodel.DepartmentBudgetTypeSubscription:
		cycleQuota := int64OrZero(input.CycleQuota)
		if cycleQuota <= 0 {
			return DepartmentBudgetItem{}, ErrDepartmentBudgetInvalidCycleQuota
		}
		cycleType := model.NormalizeResetPeriod(input.CycleType)
		if cycleType == model.SubscriptionResetNever {
			return DepartmentBudgetItem{}, ErrDepartmentBudgetInvalidCycleType
		}
		customSeconds := int64OrZero(input.CustomSeconds)
		if cycleType == model.SubscriptionResetCustom && customSeconds <= 0 {
			return DepartmentBudgetItem{}, ErrDepartmentBudgetInvalidCustomSeconds
		}
		cycleStartedAt := int64OrZero(input.CycleStartedAt)
		if cycleStartedAt <= 0 {
			return DepartmentBudgetItem{}, ErrDepartmentBudgetInvalidCycleStartedAt
		}
		budget := entmodel.DepartmentBudget{
			TenantId:       input.TenantId,
			DepartmentId:   departmentId,
			Type:           budgetType,
			Status:         entmodel.DepartmentBudgetStatusActive,
			CycleQuota:     cycleQuota,
			Remaining:      cycleQuota,
			CycleType:      cycleType,
			CycleStartedAt: cycleStartedAt,
			CustomSeconds:  customSeconds,
			ExpiresAt:      int64OrZero(input.ExpiresAt),
		}
		if err := s.db.Create(&budget).Error; err != nil {
			return DepartmentBudgetItem{}, err
		}
		return s.mapDepartmentBudgetItem(budget, ""), nil
	default:
		return DepartmentBudgetItem{}, ErrDepartmentBudgetInvalidType
	}
}

func (s *DepartmentBudgetService) Pause(departmentId int, budgetId int, tenantId int) (DepartmentBudgetItem, error) {
	return s.updateStatus(departmentId, budgetId, tenantId, entmodel.DepartmentBudgetStatusActive, entmodel.DepartmentBudgetStatusPaused)
}

func (s *DepartmentBudgetService) Resume(departmentId int, budgetId int, tenantId int) (DepartmentBudgetItem, error) {
	return s.updateStatus(departmentId, budgetId, tenantId, entmodel.DepartmentBudgetStatusPaused, entmodel.DepartmentBudgetStatusActive)
}

func (s *DepartmentBudgetService) Resize(departmentId int, budgetId int, tenantId int, input ResizeDepartmentBudgetInput) (DepartmentBudgetItem, error) {
	if departmentId <= 0 || budgetId <= 0 {
		return DepartmentBudgetItem{}, ErrInvalidDepartmentBudgetInput
	}
	if err := s.ensureDepartmentExists(tenantId, departmentId); err != nil {
		return DepartmentBudgetItem{}, err
	}

	var result DepartmentBudgetItem
	err := withBudgetMutationRetry(func() error {
		return s.db.Transaction(func(tx *gorm.DB) error {
			budget, err := s.lockBudget(tx, tenantId, departmentId, budgetId)
			if err != nil {
				return err
			}
			if budget.Status != entmodel.DepartmentBudgetStatusActive {
				return ErrDepartmentBudgetStatusTransitionInvalid
			}
			switch budget.Type {
			case entmodel.DepartmentBudgetTypeBalance:
				if input.TotalQuota == nil || input.CycleQuota != nil {
					return ErrDepartmentBudgetTypeImmutable
				}
				if *input.TotalQuota <= 0 {
					return ErrDepartmentBudgetInvalidQuota
				}
				used := budget.TotalQuota - budget.Remaining
				if used < 0 {
					used = 0
				}
				if *input.TotalQuota < used {
					return ErrDepartmentBudgetResizeBelowCommitted
				}
				budget.TotalQuota = *input.TotalQuota
				budget.Remaining = *input.TotalQuota - used
			case entmodel.DepartmentBudgetTypeSubscription:
				if input.CycleQuota == nil || input.TotalQuota != nil {
					return ErrDepartmentBudgetTypeImmutable
				}
				if *input.CycleQuota <= 0 {
					return ErrDepartmentBudgetInvalidCycleQuota
				}
				if *input.CycleQuota < budget.AllocatedTotal {
					return ErrDepartmentBudgetResizeBelowCommitted
				}
				budget.CycleQuota = *input.CycleQuota
				budget.Remaining = maxInt64(*input.CycleQuota-budget.AllocatedTotal, 0)
			default:
				return ErrDepartmentBudgetInvalidType
			}
			if err := tx.Save(&budget).Error; err != nil {
				return err
			}
			result = s.mapDepartmentBudgetItem(budget, "")
			return nil
		})
	})
	return result, err
}

func (s *DepartmentBudgetService) GetByDepartment(departmentId int, tenantId int) (*DepartmentBudgetItem, error) {
	if departmentId <= 0 {
		return nil, ErrInvalidDepartmentBudgetInput
	}
	if err := s.ensureDepartmentExists(tenantId, departmentId); err != nil {
		return nil, err
	}
	budget, err := s.latestBudget(departmentId, tenantId)
	if err != nil {
		return nil, err
	}
	if budget == nil {
		return nil, nil
	}
	item := s.mapDepartmentBudgetItem(*budget, "")
	return &item, nil
}

func (s *DepartmentBudgetService) updateStatus(departmentId int, budgetId int, tenantId int, fromStatus string, toStatus string) (DepartmentBudgetItem, error) {
	if departmentId <= 0 || budgetId <= 0 {
		return DepartmentBudgetItem{}, ErrInvalidDepartmentBudgetInput
	}
	if err := s.ensureDepartmentExists(tenantId, departmentId); err != nil {
		return DepartmentBudgetItem{}, err
	}

	var result DepartmentBudgetItem
	err := withBudgetMutationRetry(func() error {
		return s.db.Transaction(func(tx *gorm.DB) error {
			budget, err := s.lockBudget(tx, tenantId, departmentId, budgetId)
			if err != nil {
				return err
			}
			if budget.Status != fromStatus {
				return ErrDepartmentBudgetStatusTransitionInvalid
			}
			budget.Status = toStatus
			if err := tx.Save(&budget).Error; err != nil {
				return err
			}
			if err := syncBudgetChildrenForStatusTx(tx, budget); err != nil {
				return err
			}
			result = s.mapDepartmentBudgetItem(budget, "")
			return nil
		})
	})
	return result, err
}

func (s *DepartmentBudgetService) lockBudget(tx *gorm.DB, tenantId int, departmentId int, budgetId int) (entmodel.DepartmentBudget, error) {
	var budget entmodel.DepartmentBudget
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("tenant_id = ? AND department_id = ? AND id = ?", tenantId, departmentId, budgetId).
		First(&budget).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return entmodel.DepartmentBudget{}, ErrQuotaAllocationBudgetNotFound
		}
		return entmodel.DepartmentBudget{}, err
	}
	return budget, nil
}

func (s *DepartmentBudgetService) ListByDepartment(departmentId int, tenantId int, query DepartmentBudgetListQuery) (*DepartmentBudgetListResult, error) {
	if departmentId <= 0 {
		return nil, ErrInvalidDepartmentBudgetInput
	}
	scope, err := ResolveDepartmentScope(s.db, tenantId, &departmentId, query.IncludeDescendants)
	if err != nil {
		return nil, err
	}

	var budgets []entmodel.DepartmentBudget
	dbQuery := s.db.Where("tenant_id = ? AND department_id IN ?", tenantId, scope.DepartmentIds)
	if err := dbQuery.Find(&budgets).Error; err != nil {
		return nil, err
	}

	departmentNames, err := s.loadDepartmentNames(tenantId, scope.DepartmentIds)
	if err != nil {
		return nil, err
	}

	items := make([]DepartmentBudgetItem, 0, len(budgets))
	for _, budget := range budgets {
		items = append(items, s.mapDepartmentBudgetItem(normalizeDepartmentBudgetForDisplay(budget), departmentNames[budget.DepartmentId]))
	}
	sortDepartmentBudgetItems(items, query)

	return &DepartmentBudgetListResult{
		Items:               items,
		Thresholds:          currentDepartmentBudgetThresholds(),
		ScopeDepartmentId:   scope.DepartmentId,
		ScopeDepartmentName: scope.DepartmentName,
		IncludeDescendants:  query.IncludeDescendants,
		ScopeDepartmentIds:  append([]int{}, scope.ScopeDepartmentIds...),
	}, nil
}

func (s *DepartmentBudgetService) GetDetail(departmentId int, budgetId int, tenantId int) (*DepartmentBudgetDetailResult, error) {
	if departmentId <= 0 || budgetId <= 0 {
		return nil, ErrInvalidDepartmentBudgetInput
	}
	if err := s.ensureDepartmentExists(tenantId, departmentId); err != nil {
		return nil, err
	}

	var budget entmodel.DepartmentBudget
	if err := s.db.Where("tenant_id = ? AND department_id = ? AND id = ?", tenantId, departmentId, budgetId).
		First(&budget).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrQuotaAllocationBudgetNotFound
		}
		return nil, err
	}

	var rows []departmentBudgetWalletDetailRow
	query := s.db.Table(entmodel.QuotaAllocation{}.TableName()+" AS allocations").
		Select(strings.Join([]string{
			"allocations.*",
			"users.username AS target_username",
			"users.display_name AS target_display_name",
			"wallets.amount_total AS wallet_amount_total",
			"wallets.amount_used AS wallet_amount_used",
			"wallets.status AS wallet_status",
			"wallets.next_reset_time AS wallet_next_reset",
			"wallets.end_time AS wallet_end_time",
			"wallets.source_allocation_id AS wallet_source_alloc",
		}, ", ")).
		Joins("LEFT JOIN users ON users.id = allocations.target_user_id").
		Joins("LEFT JOIN user_subscriptions AS wallets ON wallets.id = allocations.wallet_id AND wallets.source_allocation_id = allocations.id").
		Where("allocations.tenant_id = ? AND allocations.department_id = ? AND allocations.department_budget_id = ?", tenantId, departmentId, budgetId).
		Order("allocations.id DESC")
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	details := make([]DepartmentBudgetWalletDetail, 0, len(rows))
	for _, row := range rows {
		details = append(details, DepartmentBudgetWalletDetail{
			AllocationId:             row.Id,
			AllocationStatus:         row.Status,
			TargetUserId:             row.TargetUserId,
			TargetUsername:           row.TargetUsername,
			TargetDisplayName:        row.TargetDisplayName,
			WalletId:                 row.WalletId,
			WalletStatus:             row.WalletStatus,
			Quota:                    row.WalletAmountTotal,
			RemainQuota:              maxInt64(row.WalletAmountTotal-row.WalletAmountUsed, 0),
			CycleType:                row.CycleTypeSnapshot,
			CycleStartedAt:           row.CycleStartedAtSnapshot,
			NextResetTime:            row.WalletNextReset,
			ExpiresAt:                firstNonZero(row.WalletEndTime, row.ExpiresAtSnapshot),
			SourceAllocationId:       row.WalletSourceAlloc,
			SourceParentBudgetId:     row.DepartmentBudgetId,
			SourceParentBudgetType:   row.BudgetTypeSnapshot,
			SourceParentBudgetStatus: budget.Status,
			CommittedQuota:           row.CommittedQuota,
			ProcessedAt:              row.ProcessedAt,
			CreatedAt:                row.CreatedAt,
			UpdatedAt:                row.UpdatedAt,
			Reason:                   row.Reason,
		})
	}

	return &DepartmentBudgetDetailResult{
		Budget:     s.mapDepartmentBudgetItem(normalizeDepartmentBudgetForDisplay(budget), ""),
		Wallets:    details,
		Thresholds: currentDepartmentBudgetThresholds(),
	}, nil
}

func (s *DepartmentBudgetService) ensureDepartmentExists(tenantId int, departmentId int) error {
	var department entmodel.Department
	err := s.db.Where("tenant_id = ? AND id = ?", tenantId, departmentId).First(&department).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrDepartmentNotFound
		}
		return err
	}
	return nil
}

func (s *DepartmentBudgetService) mapDepartmentBudgetItem(budget entmodel.DepartmentBudget, departmentName string) DepartmentBudgetItem {
	budget = normalizeDepartmentBudgetForDisplay(budget)
	usageRatio := calculateDepartmentBudgetUsageRatio(budget)
	return DepartmentBudgetItem{
		Id:             budget.Id,
		TenantId:       budget.TenantId,
		DepartmentId:   budget.DepartmentId,
		DepartmentName: departmentName,
		Type:           budget.Type,
		Status:         budget.Status,
		TotalQuota:     budget.TotalQuota,
		Remaining:      budget.Remaining,
		AllocatedTotal: budget.AllocatedTotal,
		CycleQuota:     budget.CycleQuota,
		CycleType:      budget.CycleType,
		CycleStartedAt: budget.CycleStartedAt,
		CustomSeconds:  budget.CustomSeconds,
		ExpiresAt:      budget.ExpiresAt,
		ParentStatus:   budget.ParentStatus,
		UsageRatio:     usageRatio,
		ThresholdState: calculateDepartmentBudgetThresholdState(usageRatio),
		CreatedAt:      budget.CreatedAt,
		UpdatedAt:      budget.UpdatedAt,
	}
}

func (s *DepartmentBudgetService) loadDepartmentNames(tenantId int, departmentIds []int) (map[int]string, error) {
	if len(departmentIds) == 0 {
		return map[int]string{}, nil
	}
	var departments []entmodel.Department
	if err := s.db.Select("id", "name").Where("tenant_id = ? AND id IN ?", tenantId, departmentIds).Find(&departments).Error; err != nil {
		return nil, err
	}
	result := make(map[int]string, len(departments))
	for _, department := range departments {
		result[department.Id] = department.Name
	}
	return result, nil
}

func (s *DepartmentBudgetService) latestBudget(departmentId int, tenantId int) (*entmodel.DepartmentBudget, error) {
	var budget entmodel.DepartmentBudget
	err := s.db.Where("tenant_id = ? AND department_id = ?", tenantId, departmentId).Order("id DESC").First(&budget).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &budget, nil
}

func currentDepartmentBudgetThresholds() DepartmentBudgetThresholds {
	quotaSetting := operation_setting.GetQuotaSetting()
	return DepartmentBudgetThresholds{
		Warning:  quotaSetting.EnterpriseBudgetWarningThreshold,
		Critical: quotaSetting.EnterpriseBudgetCriticalThreshold,
	}
}

func calculateDepartmentBudgetUsageRatio(budget entmodel.DepartmentBudget) float64 {
	switch budget.Type {
	case entmodel.DepartmentBudgetTypeSubscription:
		if budget.CycleQuota <= 0 {
			return 0
		}
		return clampPercent(float64(budget.AllocatedTotal) * 100 / float64(budget.CycleQuota))
	default:
		if budget.TotalQuota <= 0 {
			return 0
		}
		used := budget.TotalQuota - budget.Remaining
		return clampPercent(float64(used) * 100 / float64(budget.TotalQuota))
	}
}

func calculateDepartmentBudgetThresholdState(usageRatio float64) DepartmentBudgetThresholdState {
	thresholds := currentDepartmentBudgetThresholds()
	if usageRatio >= float64(thresholds.Critical) {
		return DepartmentBudgetThresholdStateCritical
	}
	if usageRatio >= float64(thresholds.Warning) {
		return DepartmentBudgetThresholdStateWarning
	}
	return DepartmentBudgetThresholdStateHealthy
}

func sortDepartmentBudgetItems(items []DepartmentBudgetItem, query DepartmentBudgetListQuery) {
	field := normalizeDepartmentBudgetSortField(query.SortBy)
	desc := strings.EqualFold(strings.TrimSpace(query.SortOrder), "desc")

	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		var less bool
		switch field {
		case DepartmentBudgetSortRemaining:
			less = left.Remaining < right.Remaining
		case DepartmentBudgetSortType:
			less = left.Type < right.Type
		case DepartmentBudgetSortStatus:
			less = left.Status < right.Status
		default:
			less = left.UsageRatio < right.UsageRatio
		}
		if equalsForSort(field, left, right) {
			less = left.Id < right.Id
		}
		if desc {
			return !less
		}
		return less
	})
}

func equalsForSort(field DepartmentBudgetSortField, left, right DepartmentBudgetItem) bool {
	switch field {
	case DepartmentBudgetSortRemaining:
		return left.Remaining == right.Remaining
	case DepartmentBudgetSortType:
		return left.Type == right.Type
	case DepartmentBudgetSortStatus:
		return left.Status == right.Status
	default:
		return left.UsageRatio == right.UsageRatio
	}
}

func normalizeDepartmentBudgetSortField(value string) DepartmentBudgetSortField {
	switch DepartmentBudgetSortField(strings.TrimSpace(value)) {
	case DepartmentBudgetSortRemaining, DepartmentBudgetSortType, DepartmentBudgetSortStatus:
		return DepartmentBudgetSortField(strings.TrimSpace(value))
	default:
		return DepartmentBudgetSortUsageRatio
	}
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func maxInt64(value int64, fallback int64) int64 {
	if value < fallback {
		return fallback
	}
	return value
}

func int64OrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func formatDepartmentBudgetSortField(value string) string {
	return fmt.Sprintf("%s", normalizeDepartmentBudgetSortField(value))
}
