package enterprise

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
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

type DepartmentBudgetItem struct {
	Id             int    `json:"id"`
	TenantId       int    `json:"tenant_id"`
	DepartmentId   int    `json:"department_id"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	TotalQuota     int64  `json:"total_quota"`
	Remaining      int64  `json:"remaining"`
	CycleQuota     int64  `json:"cycle_quota"`
	CycleType      string `json:"cycle_type"`
	CycleStartedAt int64  `json:"cycle_started_at"`
	CustomSeconds  int64  `json:"custom_seconds"`
	ExpiresAt      int64  `json:"expires_at"`
	ParentStatus   string `json:"parent_status"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
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
	existing, err := s.latestBudget(departmentId, input.TenantId)
	if err != nil {
		return DepartmentBudgetItem{}, err
	}
	if existing != nil && existing.Type != "" && existing.Type != budgetType {
		return DepartmentBudgetItem{}, ErrDepartmentBudgetTypeImmutable
	}

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
		return mapDepartmentBudgetItem(budget), nil
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
		return mapDepartmentBudgetItem(budget), nil
	default:
		return DepartmentBudgetItem{}, ErrDepartmentBudgetInvalidType
	}
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
	item := mapDepartmentBudgetItem(*budget)
	return &item, nil
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

func mapDepartmentBudgetItem(budget entmodel.DepartmentBudget) DepartmentBudgetItem {
	return DepartmentBudgetItem{
		Id:             budget.Id,
		TenantId:       budget.TenantId,
		DepartmentId:   budget.DepartmentId,
		Type:           budget.Type,
		Status:         budget.Status,
		TotalQuota:     budget.TotalQuota,
		Remaining:      budget.Remaining,
		CycleQuota:     budget.CycleQuota,
		CycleType:      budget.CycleType,
		CycleStartedAt: budget.CycleStartedAt,
		CustomSeconds:  budget.CustomSeconds,
		ExpiresAt:      budget.ExpiresAt,
		ParentStatus:   budget.ParentStatus,
		CreatedAt:      budget.CreatedAt,
		UpdatedAt:      budget.UpdatedAt,
	}
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

func int64OrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
