package enterprise

import (
	"fmt"
	"strings"
	"testing"
	"time"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNextDingTalkScheduleTimeUsesConfiguredTimezone(t *testing.T) {
	from := time.Date(2026, 9, 9, 15, 30, 0, 0, time.UTC)
	next, err := nextDingTalkScheduleTime("0 0 * * *", "Asia/Shanghai", from)
	require.NoError(t, err)
	require.Equal(t, int64(1788969600), next.Unix()) // 2026-09-10 00:00 Asia/Shanghai
}

func TestDingTalkSyncTaskIndexesAllowManualHistoryAndGuardActiveRuns(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entmodel.DingTalkSyncTask{}))
	require.True(t, db.Migrator().HasIndex(&entmodel.DingTalkSyncTask{}, "uq_enterprise_dingtalk_sync_tasks_scheduled_occurrence"))
	require.True(t, db.Migrator().HasIndex(&entmodel.DingTalkSyncTask{}, "uq_enterprise_dingtalk_sync_tasks_active"))
	first := entmodel.DingTalkSyncTask{TenantId: 1, Mode: "full", Status: "succeeded"}
	second := entmodel.DingTalkSyncTask{TenantId: 1, Mode: "full", Status: "succeeded"}
	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&second).Error)
	active := "full"
	require.NoError(t, db.Create(&entmodel.DingTalkSyncTask{TenantId: 1, Mode: "full", Status: "pending", ActiveKey: &active}).Error)
	require.Error(t, db.Create(&entmodel.DingTalkSyncTask{TenantId: 1, Mode: "full", Status: "pending", ActiveKey: &active}).Error)
}

func TestDingTalkConfigScheduleDefaultsAndValidation(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entmodel.DingTalkConfig{}))
	svc := NewDingTalkConfigService(db)
	secret := "secret"
	enabled := true
	saved, err := svc.Save(DingTalkConfigInput{CorpId: "corp", AppKey: "key", AppSecret: &secret, CallbackUrl: "https://example.com/callback", SyncEnabled: true, ScheduledFullSyncEnabled: &enabled})
	require.NoError(t, err)
	require.Equal(t, DefaultDingTalkScheduleCron, saved.ScheduledFullSyncCron)
	require.Equal(t, DefaultDingTalkScheduleTimezone, saved.ScheduledFullSyncTimezone)
	require.Greater(t, saved.ScheduledFullSyncNextRunAt, int64(0))
	bad := "0 0 0 * * *"
	_, err = svc.Save(DingTalkConfigInput{CorpId: "corp", AppKey: "key", CallbackUrl: "https://example.com/callback", SyncEnabled: true, ScheduledFullSyncEnabled: &enabled, ScheduledFullSyncCron: bad})
	require.ErrorIs(t, err, ErrDingTalkScheduleCronInvalid)
}

func TestNextDingTalkScheduleTimeRejectsUnsupportedExpressions(t *testing.T) {
	_, err := nextDingTalkScheduleTime("", "UTC", time.Now())
	require.NoError(t, err)
	for _, expression := range []string{"@daily", "0 0 0 * * *", "0 0 * * * "} {
		if expression == "0 0 * * * " {
			// Trailing whitespace is normalized and remains valid.
			_, err := nextDingTalkScheduleTime(expression, "UTC", time.Now())
			require.NoError(t, err)
			continue
		}
		_, err := nextDingTalkScheduleTime(expression, "UTC", time.Now())
		require.Error(t, err)
	}
	_, err = nextDingTalkScheduleTime("0 0 * * *", "Not/A/Timezone", time.Now())
	require.Error(t, err)
}
