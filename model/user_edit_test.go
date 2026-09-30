package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestUserEditPersistsRoleChanges(t *testing.T) {
	truncateTables(t)

	original := &User{
		Id:       701,
		Username: "role-edit-user",
		Password: "password123",
		AffCode:  "role-edit-aff",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, DB.Create(original).Error)

	updated := &User{
		Id:          701,
		Username:    original.Username,
		DisplayName: "Role Edit User",
		Role:        common.RoleAdminUser,
		Group:       original.Group,
		Remark:      "promoted",
	}
	require.NoError(t, updated.Edit(false))

	var persisted User
	require.NoError(t, DB.Where("id = ?", 701).First(&persisted).Error)
	require.Equal(t, common.RoleAdminUser, persisted.Role)
	require.Equal(t, "promoted", persisted.Remark)
}
