package enterprise

import (
	"errors"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type PermissionService struct {
	db *gorm.DB
}

type DepartmentAdminRoleInput struct {
	TenantId     int
	UserId       int
	DepartmentId int
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

func (s *PermissionService) GrantDepartmentAdmin(input DepartmentAdminRoleInput) (entmodel.DepartmentRole, error) {
	if input.UserId <= 0 || input.DepartmentId <= 0 {
		return entmodel.DepartmentRole{}, ErrInvalidAdminActionInput
	}
	if err := ensureEnterpriseUserExists(s.db, input.UserId); err != nil {
		return entmodel.DepartmentRole{}, err
	}
	if err := ensureEnterpriseDepartmentExists(s.db, input.TenantId, input.DepartmentId); err != nil {
		return entmodel.DepartmentRole{}, err
	}

	var role entmodel.DepartmentRole
	err := s.db.Where(
		"tenant_id = ? AND user_id = ? AND department_id = ? AND role = ?",
		input.TenantId,
		input.UserId,
		input.DepartmentId,
		constant.EnterpriseDepartmentRoleDeptAdmin,
	).First(&role).Error
	if err == nil {
		err = s.db.Model(&role).Updates(map[string]any{
			"status":     constant.EnterpriseDepartmentRoleStatusActive,
			"updated_at": time.Now().Unix(),
		}).Error
		if err != nil {
			return entmodel.DepartmentRole{}, err
		}
		role.Status = constant.EnterpriseDepartmentRoleStatusActive
		return role, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return entmodel.DepartmentRole{}, err
	}

	role = entmodel.DepartmentRole{
		TenantId:     input.TenantId,
		UserId:       input.UserId,
		DepartmentId: input.DepartmentId,
		Role:         constant.EnterpriseDepartmentRoleDeptAdmin,
		Status:       constant.EnterpriseDepartmentRoleStatusActive,
	}
	if err := s.db.Create(&role).Error; err != nil {
		return entmodel.DepartmentRole{}, err
	}
	return role, nil
}

func (s *PermissionService) RevokeDepartmentAdmin(input DepartmentAdminRoleInput) error {
	if input.UserId <= 0 || input.DepartmentId <= 0 {
		return ErrInvalidAdminActionInput
	}
	tx := s.db.Model(&entmodel.DepartmentRole{}).Where(
		"tenant_id = ? AND user_id = ? AND department_id = ? AND role = ? AND status = ?",
		input.TenantId,
		input.UserId,
		input.DepartmentId,
		constant.EnterpriseDepartmentRoleDeptAdmin,
		constant.EnterpriseDepartmentRoleStatusActive,
	).Updates(map[string]any{
		"status":     constant.EnterpriseDepartmentRoleStatusInactive,
		"updated_at": time.Now().Unix(),
	})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrDepartmentRoleNotFound
	}
	return nil
}

func (s *PermissionService) CanManageDepartment(userId int, tenantId int, departmentId int) (bool, error) {
	if userId <= 0 || departmentId <= 0 {
		return false, nil
	}
	ids, err := s.ListManageableDepartmentIds(userId, tenantId)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == departmentId {
			return true, nil
		}
	}
	return false, nil
}

func (s *PermissionService) ListManageableDepartmentIds(userId int, tenantId int) ([]int, error) {
	if userId <= 0 {
		return []int{}, nil
	}

	var roles []entmodel.DepartmentRole
	if err := s.db.Where(
		"tenant_id = ? AND user_id = ? AND role = ? AND status = ?",
		tenantId,
		userId,
		constant.EnterpriseDepartmentRoleDeptAdmin,
		constant.EnterpriseDepartmentRoleStatusActive,
	).Find(&roles).Error; err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return []int{}, nil
	}

	var departments []entmodel.Department
	if err := s.db.Where("tenant_id = ?", tenantId).Find(&departments).Error; err != nil {
		return nil, err
	}

	existing := make(map[int]struct{}, len(departments))
	childrenByParent := make(map[int][]int, len(departments))
	for _, department := range departments {
		existing[department.Id] = struct{}{}
		if department.ParentId != nil {
			childrenByParent[*department.ParentId] = append(childrenByParent[*department.ParentId], department.Id)
		}
	}

	manageable := make(map[int]struct{})
	for _, role := range roles {
		if _, ok := existing[role.DepartmentId]; !ok {
			continue
		}
		addDepartmentAndDescendants(role.DepartmentId, childrenByParent, manageable, map[int]bool{})
	}

	ids := make([]int, 0, len(manageable))
	for id := range manageable {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

func addDepartmentAndDescendants(id int, childrenByParent map[int][]int, out map[int]struct{}, path map[int]bool) {
	if path[id] {
		return
	}
	path[id] = true
	out[id] = struct{}{}
	for _, childId := range childrenByParent[id] {
		addDepartmentAndDescendants(childId, childrenByParent, out, path)
	}
	delete(path, id)
}

func ensureEnterpriseUserExists(db *gorm.DB, userId int) error {
	var count int64
	if err := db.Model(&model.User{}).Where("id = ?", userId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrUserNotFound
	}
	return nil
}

func ensureEnterpriseDepartmentExists(db *gorm.DB, tenantId int, departmentId int) error {
	var count int64
	if err := db.Model(&entmodel.Department{}).Where("tenant_id = ? AND id = ?", tenantId, departmentId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrDepartmentNotFound
	}
	return nil
}
