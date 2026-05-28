package enterprise

import (
	"sort"

	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type DepartmentService struct {
	db *gorm.DB
}

func NewDepartmentService(db *gorm.DB) *DepartmentService {
	return &DepartmentService{db: db}
}

func (s *DepartmentService) GetDepartmentTree() ([]dtoenterprise.DepartmentTreeNode, error) {
	var departments []modelenterprise.Department
	if err := s.db.Order("id ASC").Find(&departments).Error; err != nil {
		return nil, err
	}
	if len(departments) == 0 {
		return []dtoenterprise.DepartmentTreeNode{}, nil
	}
	return BuildDepartmentTree(departments)
}

func BuildDepartmentTree(departments []modelenterprise.Department) ([]dtoenterprise.DepartmentTreeNode, error) {
	nodes := make(map[int]*dtoenterprise.DepartmentTreeNode, len(departments))
	order := make([]int, 0, len(departments))
	childIdsByParentId := make(map[int][]int, len(departments))
	rootIds := make([]int, 0, len(departments))

	for _, department := range departments {
		history, err := department.ParsedNameHistory()
		if err != nil {
			return nil, ErrDepartmentNameHistoryInvalid
		}
		dtoHistory := make([]dtoenterprise.DepartmentNameHistoryEntry, 0, len(history))
		for _, item := range history {
			dtoHistory = append(dtoHistory, dtoenterprise.DepartmentNameHistoryEntry{
				Name:      item.Name,
				ChangedAt: item.ChangedAt,
			})
		}
		nodes[department.Id] = &dtoenterprise.DepartmentTreeNode{
			Id:          department.Id,
			TenantId:    department.TenantId,
			Name:        department.Name,
			ParentId:    department.ParentId,
			Status:      department.Status,
			SourceType:  department.SourceType,
			ExternalId:  department.ExternalId,
			SyncStatus:  department.SyncStatus,
			SyncError:   department.SyncError,
			NameHistory: dtoHistory,
			CreatedAt:   department.CreatedAt,
			UpdatedAt:   department.UpdatedAt,
			DeletedAt:   department.DeletedAt,
			Children:    []dtoenterprise.DepartmentTreeNode{},
		}
		order = append(order, department.Id)
	}

	for _, id := range order {
		node := nodes[id]
		if node.ParentId == nil {
			rootIds = append(rootIds, id)
			continue
		}
		if _, ok := nodes[*node.ParentId]; !ok {
			rootIds = append(rootIds, id)
			continue
		}
		childIdsByParentId[*node.ParentId] = append(childIdsByParentId[*node.ParentId], id)
	}

	sort.Ints(rootIds)
	for parentId := range childIdsByParentId {
		sort.Ints(childIdsByParentId[parentId])
	}

	visited := make(map[int]bool, len(departments))
	result := make([]dtoenterprise.DepartmentTreeNode, 0, len(rootIds))
	for _, rootId := range rootIds {
		result = append(result, cloneDepartmentSubtree(rootId, nodes, childIdsByParentId, map[int]bool{}, visited))
	}
	for _, id := range order {
		if !visited[id] {
			result = append(result, cloneDepartmentSubtree(id, nodes, childIdsByParentId, map[int]bool{}, visited))
		}
	}
	return result, nil
}

func cloneDepartmentSubtree(
	id int,
	nodes map[int]*dtoenterprise.DepartmentTreeNode,
	childIdsByParentId map[int][]int,
	path map[int]bool,
	visited map[int]bool,
) dtoenterprise.DepartmentTreeNode {
	node := *nodes[id]
	visited[id] = true
	path[id] = true
	node.Children = []dtoenterprise.DepartmentTreeNode{}
	for _, childId := range childIdsByParentId[id] {
		if path[childId] {
			continue
		}
		node.Children = append(node.Children, cloneDepartmentSubtree(childId, nodes, childIdsByParentId, path, visited))
	}
	delete(path, id)
	return node
}
