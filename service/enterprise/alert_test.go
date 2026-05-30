package enterprise

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAlertServiceTest(t *testing.T) (*AlertService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, entmodel.AutoMigrate(db))
	return NewAlertService(db), db
}

func newAlertTestContext(t *testing.T, requestID string, userID int, username string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyRequestStartTime, time.Unix(1717117200, 0))
	ctx.Set(common.RequestIdKey, requestID)
	common.SetContextKey(ctx, constant.ContextKeyUserId, userID)
	common.SetContextKey(ctx, constant.ContextKeyUserName, username)
	return ctx
}

func TestAlertServiceRecordRiskEventCapturesDepartmentSnapshot(t *testing.T) {
	service, db := setupAlertServiceTest(t)
	require.NoError(t, db.Create(&model.User{Id: 1001, Username: "alice", Password: "password123", Group: "default", AffCode: "alice-aff"}).Error)
	require.NoError(t, db.Create(&[]entmodel.Department{
		{Id: 11, TenantId: 0, Name: "Engineering", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"},
		{Id: 22, TenantId: 0, Name: "Security", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"},
	}).Error)
	require.NoError(t, db.Create(&[]entmodel.UserDepartment{
		{TenantId: 0, UserId: 1001, DepartmentId: 11, ExternalSource: constant.EnterpriseExternalSourceManual, Status: constant.EnterpriseMembershipStatusActive},
		{TenantId: 0, UserId: 1001, DepartmentId: 22, ExternalSource: constant.EnterpriseExternalSourceManual, Status: constant.EnterpriseMembershipStatusActive},
	}).Error)

	ctx := newAlertTestContext(t, "req-sensitive-1", 1001, "alice")
	err := service.RecordRiskEvent(ctx, RecordRiskEventInput{
		TenantId:        0,
		ModelName:       "gpt-4o-mini",
		RiskType:        AlertRiskTypeSensitiveWords,
		ActionResult:    AlertActionBlocked,
		Summary:         "2 sensitive word hits",
		SensitiveHits:   []string{"secret", "password"},
		EventOccurredAt: time.Unix(1717117201, 0),
	})
	require.NoError(t, err)

	var events []entmodel.AlertEvent
	require.NoError(t, db.Order("id ASC").Find(&events).Error)
	require.Len(t, events, 1)
	require.Equal(t, "alice", events[0].Username)
	require.Equal(t, "req-sensitive-1", events[0].RequestId)
	require.Equal(t, "gpt-4o-mini", events[0].ModelName)
	require.Equal(t, AlertRiskTypeSensitiveWords, events[0].RiskType)
	require.Equal(t, AlertActionBlocked, events[0].ActionResult)
	require.Equal(t, "2 sensitive word hits", events[0].Summary)
	require.NotContains(t, events[0].Summary, "secret")
	require.NotContains(t, events[0].Summary, "password")

	snapshot, err := events[0].ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot, 2)
	require.Equal(t, 11, snapshot[0].DepartmentId)
	require.Equal(t, "Engineering", snapshot[0].DepartmentName)
	require.Equal(t, 22, snapshot[1].DepartmentId)
	require.Equal(t, "Security", snapshot[1].DepartmentName)
}

func TestAlertServiceRecordRiskEventHandlesUnassignedUsersAndSanitizesSummary(t *testing.T) {
	service, db := setupAlertServiceTest(t)
	require.NoError(t, db.Create(&model.User{Id: 2002, Username: "bob", Password: "password123", Group: "default", AffCode: "bob-aff"}).Error)

	ctx := newAlertTestContext(t, "req-sensitive-2", 2002, "bob")
	err := service.RecordRiskEvent(ctx, RecordRiskEventInput{
		TenantId:      0,
		ModelName:     "gpt-4o",
		RiskType:      AlertRiskTypeSensitiveWords,
		ActionResult:  AlertActionBlocked,
		Summary:       "blocked prompt contains password and secret",
		SensitiveHits: []string{"password", "secret"},
	})
	require.NoError(t, err)

	var event entmodel.AlertEvent
	require.NoError(t, db.First(&event).Error)
	snapshot, err := event.ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.Empty(t, snapshot)
	require.Equal(t, "2 sensitive word hits", event.Summary)
	require.NotContains(t, event.Summary, "password")
	require.NotContains(t, event.Summary, "secret")
}

func TestAlertServiceRecordRiskEventReturnsNilOnDBFailure(t *testing.T) {
	service, db := setupAlertServiceTest(t)
	require.NoError(t, db.Create(&model.User{Id: 3003, Username: "charlie", Password: "password123", Group: "default", AffCode: "charlie-aff"}).Error)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	ctx := newAlertTestContext(t, "req-sensitive-3", 3003, "charlie")
	err = service.RecordRiskEvent(ctx, RecordRiskEventInput{
		TenantId:      0,
		ModelName:     "gpt-4.1",
		RiskType:      AlertRiskTypeSensitiveWords,
		ActionResult:  AlertActionBlocked,
		Summary:       "1 sensitive word hit",
		SensitiveHits: []string{"secret"},
	})
	require.NoError(t, err)
}

func TestAlertServiceRecordRiskEventUsesTenantScopedMembershipSnapshot(t *testing.T) {
	service, db := setupAlertServiceTest(t)
	require.NoError(t, db.Create(&model.User{Id: 4004, Username: "dora", Password: "password123", Group: "default", AffCode: "dora-aff"}).Error)
	require.NoError(t, db.Create(&[]entmodel.Department{
		{Id: 11, TenantId: 0, Name: "Default Tenant", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"},
		{Id: 77, TenantId: 7, Name: "Tenant Seven", Status: constant.DepartmentStatusEnabled, SourceType: constant.DepartmentSourceTypeManual, NameHistory: "[]"},
	}).Error)
	require.NoError(t, db.Create(&[]entmodel.UserDepartment{
		{TenantId: 0, UserId: 4004, DepartmentId: 11, ExternalSource: constant.EnterpriseExternalSourceManual, Status: constant.EnterpriseMembershipStatusActive},
		{TenantId: 7, UserId: 4004, DepartmentId: 77, ExternalSource: constant.EnterpriseExternalSourceManual, Status: constant.EnterpriseMembershipStatusActive},
	}).Error)

	ctx := newAlertTestContext(t, "req-sensitive-tenant", 4004, "dora")
	err := service.RecordRiskEvent(ctx, RecordRiskEventInput{
		TenantId:      7,
		ModelName:     "gpt-4o-mini",
		RiskType:      AlertRiskTypeSensitiveWords,
		ActionResult:  AlertActionBlocked,
		Summary:       "prompt contains secret",
		SensitiveHits: []string{"secret"},
	})
	require.NoError(t, err)

	var event entmodel.AlertEvent
	require.NoError(t, db.First(&event).Error)
	require.Equal(t, 7, event.TenantId)
	snapshot, err := event.ParsedDepartmentSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot, 1)
	require.Equal(t, "Tenant Seven", snapshot[0].DepartmentName)
	require.Equal(t, "1 sensitive word hits", event.Summary)
	require.NotContains(t, event.Summary, "prompt contains secret")
}
