package aionui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDesktopAuthServiceExchangesCodeOnceAndValidatesToken(t *testing.T) {
	db := newDesktopAuthTestDB(t)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.Create(&model.User{
		Id:          100,
		Username:    "alice",
		DisplayName: "Alice",
		Email:       "Alice@Example.COM",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     "alice",
	}).Error)
	require.NoError(t, db.Create(&entmodel.Department{Id: 7, Name: "研发部"}).Error)
	require.NoError(t, db.Create(&entmodel.UserDepartment{
		UserId:       100,
		DepartmentId: 7,
		Status:       constant.EnterpriseMembershipStatusActive,
	}).Error)

	now := time.Date(2026, 7, 28, 13, 0, 0, 0, time.UTC)
	service := NewDesktopAuthService()
	service.now = func() time.Time {
		return now
	}

	code, err := service.IssueCode(
		&model.User{Id: 100, Username: "alice", DisplayName: "Alice", Email: "Alice@Example.COM", Status: common.UserStatusEnabled},
		DesktopRedirectURI,
		"state-1234567890",
	)
	require.NoError(t, err)

	token, err := service.ExchangeCode(dtoaionui.DesktopTokenRequest{
		Code:       code,
		DeviceId:   "device-1",
		AppVersion: "2.1.41",
	})
	require.NoError(t, err)
	require.NotEmpty(t, token.AccessToken)
	require.Equal(t, "alice@example.com", token.User.Email)
	require.Equal(t, []string{"研发部"}, token.User.Departments)
	require.Equal(t, now.Add(desktopTokenTTL).Unix(), token.ExpiresAt)

	_, err = service.ExchangeCode(dtoaionui.DesktopTokenRequest{Code: code, DeviceId: "device-1"})
	require.ErrorIs(t, err, ErrInvalidCode)

	claims, err := service.ValidateAuthorization("Bearer " + token.AccessToken)
	require.NoError(t, err)
	require.Equal(t, 100, claims.UserId)
	require.Equal(t, "alice@example.com", claims.Email)
	require.Equal(t, "device-1", claims.DeviceId)
}

func TestDesktopAuthServiceValidatesPersistedTokenAfterRestart(t *testing.T) {
	db := newDesktopAuthTestDB(t)
	model.DB = db
	model.LOG_DB = db
	redisValues := map[string]string{}
	oldRedisAvailable := desktopRedisAvailable
	oldRedisSet := desktopRedisSet
	oldRedisGet := desktopRedisGet
	desktopRedisAvailable = func() bool {
		return true
	}
	desktopRedisSet = func(key string, value string, _ time.Duration) error {
		redisValues[key] = value
		return nil
	}
	desktopRedisGet = func(key string) (string, error) {
		value, ok := redisValues[key]
		if !ok {
			return "", errors.New("missing key")
		}
		return value, nil
	}
	t.Cleanup(func() {
		desktopRedisAvailable = oldRedisAvailable
		desktopRedisSet = oldRedisSet
		desktopRedisGet = oldRedisGet
	})
	require.NoError(t, db.Create(&model.User{
		Id:          101,
		Username:    "bob",
		DisplayName: "Bob",
		Email:       "bob@example.com",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     "bob",
	}).Error)

	now := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	service := NewDesktopAuthService()
	service.now = func() time.Time {
		return now
	}
	code, err := service.IssueCode(
		&model.User{Id: 101, Username: "bob", DisplayName: "Bob", Email: "bob@example.com", Status: common.UserStatusEnabled},
		DesktopRedirectURI,
		"state-1234567890",
	)
	require.NoError(t, err)
	token, err := service.ExchangeCode(dtoaionui.DesktopTokenRequest{
		Code:     code,
		DeviceId: "device-2",
	})
	require.NoError(t, err)

	restarted := NewDesktopAuthService()
	restarted.now = func() time.Time {
		return now.Add(time.Minute)
	}
	claims, err := restarted.ValidateAuthorization("Bearer " + token.AccessToken)

	require.NoError(t, err)
	require.Equal(t, 101, claims.UserId)
	require.Equal(t, "bob@example.com", claims.Email)
	require.Equal(t, "device-2", claims.DeviceId)
}

func TestDesktopAuthServiceRejectsUserWithoutEmail(t *testing.T) {
	service := NewDesktopAuthService()

	_, err := service.IssueCode(
		&model.User{Id: 100, Username: "alice", Status: common.UserStatusEnabled},
		DesktopRedirectURI,
		"state-1234567890",
	)

	require.ErrorIs(t, err, ErrEmailRequired)
}

func TestValidateDesktopRedirectURIAllowsLoopbackCallback(t *testing.T) {
	require.NoError(t, ValidateDesktopRedirectURI(DesktopRedirectURI))
	require.NoError(t, ValidateDesktopRedirectURI("http://127.0.0.1:49152/new-api/callback"))
	require.NoError(t, ValidateDesktopRedirectURI("http://localhost:49152/new-api/callback"))

	require.ErrorIs(t, ValidateDesktopRedirectURI("http://127.0.0.1/new-api/callback"), ErrInvalidRedirectURI)
	require.ErrorIs(t, ValidateDesktopRedirectURI("http://127.0.0.1:49152/other"), ErrInvalidRedirectURI)
	require.ErrorIs(t, ValidateDesktopRedirectURI("http://example.com:49152/new-api/callback"), ErrInvalidRedirectURI)
	require.ErrorIs(t, ValidateDesktopRedirectURI("https://127.0.0.1:49152/new-api/callback"), ErrInvalidRedirectURI)
}

func newDesktopAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &entmodel.Department{}, &entmodel.UserDepartment{}))

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.RedisEnabled = oldRedisEnabled
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
