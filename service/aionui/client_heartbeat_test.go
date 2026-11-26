package aionui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	aionuimodel "github.com/QuantumNous/new-api/model/aionui"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestClientHeartbeatReportUpsertsAuthoritativeInstallation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &entmodel.Department{}, &entmodel.UserDepartment{}))
	require.NoError(t, aionuimodel.Migrate(db))
	require.NoError(t, aionuimodel.Migrate(db))
	require.NoError(t, db.Create(&model.User{
		Id:          7,
		Username:    "alice",
		DisplayName: "Alice Authoritative",
		Email:       "Alice@Example.com",
		Status:      common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&entmodel.Department{
		Id:     3,
		Name:   "Engineering",
		Status: constant.DepartmentStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		UserId:       7,
		DepartmentId: 3,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)

	service := NewClientHeartbeatService(db)
	firstReportedAt := time.Date(2026, 9, 11, 1, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return firstReportedAt }
	response, err := service.Report(testClientHeartbeatInput("device-1", "192.168.1.10"))
	require.NoError(t, err)
	assert.Equal(t, firstReportedAt.Format(time.RFC3339), response.AcceptedAt)
	assert.Equal(t, 1800, response.NextHeartbeatSeconds)

	secondReportedAt := firstReportedAt.Add(30 * time.Minute)
	service.now = func() time.Time { return secondReportedAt }
	_, err = service.Report(testClientHeartbeatInput("device-1", "10.0.0.8"))
	require.NoError(t, err)
	_, err = service.Report(testClientHeartbeatInput("device-2", "172.16.1.8"))
	require.NoError(t, err)

	var installation aionuimodel.ClientInstallation
	require.NoError(t, db.Where("device_id = ?", "device-1").First(&installation).Error)
	assert.Equal(t, "alice@example.com", installation.Email)
	assert.Equal(t, "Alice Authoritative", installation.DisplayName)
	assert.Equal(t, "[\"Engineering\"]", installation.DepartmentNamesJSON)
	assert.Equal(t, "10.0.0.8", installation.LanIP)
	assert.Equal(t, firstReportedAt, installation.FirstHeartbeatAt)
	assert.Equal(t, secondReportedAt, installation.LastHeartbeatAt)

	listed, err := service.List(ClientInstallationQuery{})
	require.NoError(t, err)
	assert.Equal(t, 2, listed.Total)
	assert.Equal(t, 1, listed.Summary.TotalUsers)
	assert.Equal(t, 2, listed.Summary.TotalInstallations)
	assert.Equal(t, 1, listed.Summary.ActiveUsers)
	assert.Equal(t, 2, listed.Summary.ActiveInstallations)
}

func TestClientHeartbeatRejectsNonRFC1918IPv4(t *testing.T) {
	testCases := []string{
		"8.8.8.8",
		"127.0.0.1",
		"169.254.1.1",
		"192.168.1.2,10.0.0.1",
		"fd00::1",
	}
	for _, lanIP := range testCases {
		t.Run(lanIP, func(t *testing.T) {
			_, err := validateClientHeartbeatInput(testClientHeartbeatInput("device-1", lanIP))
			require.ErrorIs(t, err, ErrClientHeartbeatInvalidInput)
		})
	}
}

func TestClientHeartbeatDatabaseMigrationMatrix(t *testing.T) {
	testCases := []struct {
		name string
		dsn  string
		open func(string) gorm.Dialector
	}{
		{name: "mysql", dsn: os.Getenv("AIONUI_HEARTBEAT_TEST_MYSQL_DSN"), open: mysql.Open},
		{name: "postgres", dsn: os.Getenv("AIONUI_HEARTBEAT_TEST_POSTGRES_DSN"), open: postgres.Open},
	}
	runnable := false
	for _, testCase := range testCases {
		if strings.TrimSpace(testCase.dsn) == "" {
			continue
		}
		runnable = true
		t.Run(testCase.name, func(t *testing.T) {
			db, err := gorm.Open(testCase.open(testCase.dsn), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(&model.User{}, &entmodel.Department{}, &entmodel.UserDepartment{}))
			require.NoError(t, aionuimodel.Migrate(db))
			require.NoError(t, aionuimodel.Migrate(db))
			require.True(t, db.Migrator().HasTable(&aionuimodel.ClientInstallation{}))
			require.True(t, db.Migrator().HasIndex(&aionuimodel.ClientInstallation{}, "uq_aionui_client_installations_device"))

			require.NoError(t, db.Create(&model.User{
				Id:       7,
				Username: "alice",
				Email:    "alice@example.com",
				Status:   common.UserStatusEnabled,
			}).Error)
			service := NewClientHeartbeatService(db)
			_, err = service.Report(testClientHeartbeatInput("device-1", "192.168.1.20"))
			require.NoError(t, err)
			listed, err := service.List(ClientInstallationQuery{})
			require.NoError(t, err)
			assert.Equal(t, 1, listed.Summary.TotalInstallations)
		})
	}
	if !runnable {
		t.Skip("AIONUI_HEARTBEAT_TEST_MYSQL_DSN and AIONUI_HEARTBEAT_TEST_POSTGRES_DSN are not configured")
	}
}

func testClientHeartbeatInput(deviceID string, lanIP string) ClientHeartbeatInput {
	return ClientHeartbeatInput{
		UserId:        7,
		DeviceId:      deviceID,
		ClientVersion: "1.2.3",
		Platform:      "windows_x64",
		LanIP:         lanIP,
		User: dtoaionui.ClientHeartbeatUserInfo{
			Email:       "spoofed@example.com",
			Name:        "Spoofed User",
			Departments: []string{"Spoofed Department"},
		},
	}
}
