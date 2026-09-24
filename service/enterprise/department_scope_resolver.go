package enterprise

import (
	"sort"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type DepartmentScope struct {
	DepartmentId        *int
	DepartmentName      string
	IncludeDescendants  bool
	DepartmentIds       []int
	ScopeLabel          string
	IsAllDepartments    bool
	ScopeDepartmentIds  []int
}

func ResolveDepartmentScope(db *gorm.DB, tenantId int, departmentId *int, includeDescendants bool) (DepartmentScope, error) {
	if db == nil {
		return DepartmentScope{}, gorm.ErrInvalidDB
	}

	var departments []entmodel.Department
	if err := db.Where("tenant_id = ?", tenantId).Order("id ASC").Find(&departments).Error; err != nil {
		return DepartmentScope{}, err
	}

	if departmentId == nil {
		allIds := make([]int, 0, len(departments))
		for _, department := range departments {
			allIds = append(allIds, department.Id)
		}
		return DepartmentScope{
			DepartmentId:       nil,
			DepartmentName:     "",
			IncludeDescendants: false,
			DepartmentIds:      append([]int{}, allIds...),
			ScopeLabel:         "all_departments",
			IsAllDepartments:   true,
			ScopeDepartmentIds: append([]int{}, allIds...),
		}, nil
	}

	departmentByID := make(map[int]entmodel.Department, len(departments))
	childrenByParent := make(map[int][]int, len(departments))
	for _, department := range departments {
		departmentByID[department.Id] = department
		if department.ParentId != nil {
			childrenByParent[*department.ParentId] = append(childrenByParent[*department.ParentId], department.Id)
		}
	}

	root, ok := departmentByID[*departmentId]
	if !ok {
		return DepartmentScope{}, ErrDepartmentNotFound
	}

	ids := []int{root.Id}
	if includeDescendants {
		scopeSet := make(map[int]struct{}, len(departments))
		addDepartmentAndDescendants(root.Id, childrenByParent, scopeSet, map[int]bool{})
		ids = make([]int, 0, len(scopeSet))
		for id := range scopeSet {
			ids = append(ids, id)
		}
		sort.Ints(ids)
	}

	return DepartmentScope{
		DepartmentId:       departmentId,
		DepartmentName:     root.Name,
		IncludeDescendants: includeDescendants,
		DepartmentIds:      append([]int{}, ids...),
		ScopeLabel:         root.Name,
		IsAllDepartments:   false,
		ScopeDepartmentIds: append([]int{}, ids...),
	}, nil
}

func IsDepartmentAncestor(db *gorm.DB, tenantId int, ancestorDepartmentId int, targetDepartmentId int) (bool, error) {
	if db == nil {
		return false, gorm.ErrInvalidDB
	}
	if ancestorDepartmentId <= 0 || targetDepartmentId <= 0 || ancestorDepartmentId == targetDepartmentId {
		return false, nil
	}
	scope, err := ResolveDepartmentScope(db, tenantId, &ancestorDepartmentId, true)
	if err != nil {
		return false, err
	}
	for _, departmentId := range scope.DepartmentIds {
		if departmentId == targetDepartmentId {
			return true, nil
		}
	}
	return false, nil
}
