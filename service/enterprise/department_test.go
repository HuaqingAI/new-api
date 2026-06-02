package enterprise

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	modelenterprise "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetDepartmentTreeEmptyTableReturnsEmptySlice(t *testing.T) {
	db := newDepartmentTestDB(t)

	nodes, err := NewDepartmentService(db).GetDepartmentTree()
	require.NoError(t, err)
	require.NotNil(t, nodes)
	require.Empty(t, nodes)
}

func TestBuildDepartmentTreeSupportsThreeLevels(t *testing.T) {
	parentId := 1
	childId := 2
	nodes, err := BuildDepartmentTree([]modelenterprise.Department{
		department(1, nil, "Company", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK),
		department(2, &parentId, "Engineering", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK),
		department(3, &childId, "Platform", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK),
	})

	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "Company", nodes[0].Name)
	require.Len(t, nodes[0].Children, 1)
	require.Equal(t, "Engineering", nodes[0].Children[0].Name)
	require.Len(t, nodes[0].Children[0].Children, 1)
	require.Equal(t, "Platform", nodes[0].Children[0].Children[0].Name)
	requireEmptyChildren(t, nodes[0].Children[0].Children[0])
}

func TestBuildDepartmentTreeKeepsDisabledDeletedAndFailedNodes(t *testing.T) {
	parentId := 1
	nodes, err := BuildDepartmentTree([]modelenterprise.Department{
		department(1, nil, "Company", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK),
		department(2, &parentId, "Disabled", constant.DepartmentStatusDisabled, constant.DepartmentSyncStatusWarning),
		department(3, &parentId, "Deleted upstream", constant.DepartmentStatusDeleted, constant.DepartmentSyncStatusFailed),
	})

	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Len(t, nodes[0].Children, 2)
	require.Equal(t, constant.DepartmentStatusDisabled, nodes[0].Children[0].Status)
	require.Equal(t, constant.DepartmentSyncStatusWarning, nodes[0].Children[0].SyncStatus)
	require.Equal(t, constant.DepartmentStatusDeleted, nodes[0].Children[1].Status)
	require.Equal(t, constant.DepartmentSyncStatusFailed, nodes[0].Children[1].SyncStatus)
}

func TestBuildDepartmentTreeParsesNameHistory(t *testing.T) {
	dept := department(1, nil, "Current", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK)
	require.NoError(t, dept.SetNameHistory([]modelenterprise.DepartmentNameHistoryEntry{
		{Name: "Previous", ChangedAt: 1700000000},
	}))

	nodes, err := BuildDepartmentTree([]modelenterprise.Department{dept})

	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, []dtoenterprise.DepartmentNameHistoryEntry{
		{Name: "Previous", ChangedAt: 1700000000},
	}, nodes[0].NameHistory)
}

func TestBuildDepartmentTreeReturnsOrphansAsRoots(t *testing.T) {
	missingParentId := 99
	nodes, err := BuildDepartmentTree([]modelenterprise.Department{
		department(2, &missingParentId, "Orphan", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusFailed),
	})

	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "Orphan", nodes[0].Name)
	require.Equal(t, &missingParentId, nodes[0].ParentId)
	require.Equal(t, constant.DepartmentSyncStatusFailed, nodes[0].SyncStatus)
	requireEmptyChildren(t, nodes[0])
}

func TestBuildDepartmentTreeHandlesCyclicParentsWithoutRecursingForever(t *testing.T) {
	firstParentId := 2
	secondParentId := 1
	nodes, err := BuildDepartmentTree([]modelenterprise.Department{
		department(1, &firstParentId, "Loop A", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusFailed),
		department(2, &secondParentId, "Loop B", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusFailed),
	})

	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "Loop A", nodes[0].Name)
	require.Len(t, nodes[0].Children, 1)
	require.Equal(t, "Loop B", nodes[0].Children[0].Name)
	requireEmptyChildren(t, nodes[0].Children[0])
}

func TestBuildDepartmentTreeHandlesSelfParentWithoutRecursingForever(t *testing.T) {
	parentId := 1
	nodes, err := BuildDepartmentTree([]modelenterprise.Department{
		department(1, &parentId, "Self loop", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusFailed),
	})

	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "Self loop", nodes[0].Name)
	requireEmptyChildren(t, nodes[0])
}

func TestBuildDepartmentTreeRejectsInvalidNameHistory(t *testing.T) {
	dept := department(1, nil, "Broken", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK)
	dept.NameHistory = "{"

	_, err := BuildDepartmentTree([]modelenterprise.Department{dept})

	require.ErrorIs(t, err, ErrDepartmentNameHistoryInvalid)
}

func TestResolveDepartmentScopeSupportsCurrentDescendantsOrphansAndTenantIsolation(t *testing.T) {
	db := newDepartmentTestDB(t)
	parentID := 1
	childID := 2
	require.NoError(t, db.Create(&[]modelenterprise.Department{
		department(1, nil, "Root", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK),
		department(2, &parentID, "Child", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK),
		department(3, &childID, "Grandchild", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK),
		department(4, nil, "Sibling", constant.DepartmentStatusEnabled, constant.DepartmentSyncStatusOK),
		{
			Id:          10,
			TenantId:    9,
			Name:        "Other Tenant Root",
			Status:      constant.DepartmentStatusEnabled,
			SourceType:  constant.DepartmentSourceTypeManual,
			SyncStatus:  constant.DepartmentSyncStatusOK,
			NameHistory: "[]",
		},
	}).Error)

	scope, err := ResolveDepartmentScope(db, 0, nil, false)
	require.NoError(t, err)
	require.True(t, scope.IsAllDepartments)
	require.Equal(t, []int{1, 2, 3, 4}, scope.DepartmentIds)

	scope, err = ResolveDepartmentScope(db, 0, &parentID, false)
	require.NoError(t, err)
	require.Equal(t, []int{1}, scope.DepartmentIds)
	require.Equal(t, "Root", scope.DepartmentName)

	scope, err = ResolveDepartmentScope(db, 0, &parentID, true)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3}, scope.DepartmentIds)

	missing := 999
	_, err = ResolveDepartmentScope(db, 0, &missing, true)
	require.ErrorIs(t, err, ErrDepartmentNotFound)
}

func newDepartmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, modelenterprise.Migrate(db))
	return db
}

func department(id int, parentId *int, name string, status int, syncStatus int) modelenterprise.Department {
	dept := modelenterprise.Department{
		Id:          id,
		TenantId:    0,
		Name:        name,
		ParentId:    parentId,
		Status:      status,
		SourceType:  constant.DepartmentSourceTypeDingTalk,
		ExternalId:  name,
		SyncStatus:  syncStatus,
		SyncError:   "",
		NameHistory: "[]",
		CreatedAt:   1700000000,
		UpdatedAt:   1700000000,
	}
	if syncStatus == constant.DepartmentSyncStatusFailed {
		dept.SyncError = "parent missing"
	}
	return dept
}

func requireEmptyChildren(t *testing.T, node dtoenterprise.DepartmentTreeNode) {
	t.Helper()
	require.NotNil(t, node.Children)
	require.Empty(t, node.Children)
}
