package enterprise

import (
	"sort"

	"github.com/QuantumNous/new-api/constant"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type PermissionService struct {
	db *gorm.DB
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
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
