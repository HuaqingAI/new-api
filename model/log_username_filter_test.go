package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetAllLogsUsernameFilterMatchesCurrentUsernameAndHistoricalSnapshot(t *testing.T) {
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	LOG_DB = db
	require.NoError(t, db.AutoMigrate(&User{}, &Log{}))

	require.NoError(t, db.Create(&User{Id: 1, Username: "alice_ops", Password: "password123", AffCode: "alice-ops"}).Error)
	require.NoError(t, db.Create(&Log{
		Id:        1,
		UserId:    1,
		Username:  "alice",
		Type:      LogTypeConsume,
		ModelName: "gpt-4o-mini",
		CreatedAt: 1717117200,
	}).Error)

	logs, total, err := GetAllLogs(LogTypeConsume, 0, 0, "", "alice_ops", "", 0, 20, 0, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	require.Equal(t, "alice", logs[0].Username)
}
