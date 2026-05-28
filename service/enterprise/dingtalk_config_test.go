package enterprise_test

import (
	"testing"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDingTalkConfigTestService(t *testing.T) (*entservice.DingTalkConfigService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, entmodel.AutoMigrate(db))
	return entservice.NewDingTalkConfigService(db), db
}

func TestDingTalkConfigSaveAndGetMaskSecret(t *testing.T) {
	svc, db := newDingTalkConfigTestService(t)
	secret := "plain-secret"

	saved, err := svc.Save(entservice.DingTalkConfigInput{
		CorpId:       "ding-corp",
		AppKey:       "ding-key",
		AppSecret:    &secret,
		CallbackUrl:  "https://example.com/api/oauth/dingtalk",
		SyncScope:    "1,2",
		LoginEnabled: true,
		SyncEnabled:  true,
	})
	require.NoError(t, err)
	require.True(t, saved.HasAppSecret)
	require.Equal(t, "ding-corp", saved.CorpId)

	fetched, err := svc.Get(0)
	require.NoError(t, err)
	require.True(t, fetched.HasAppSecret)
	require.Equal(t, "ding-key", fetched.AppKey)

	var config entmodel.DingTalkConfig
	require.NoError(t, db.Where("tenant_id = ?", 0).First(&config).Error)
	require.Equal(t, secret, config.AppSecret)
}

func TestDingTalkConfigKeepsExistingSecretWhenOmittedOrBlank(t *testing.T) {
	svc, db := newDingTalkConfigTestService(t)
	secret := "initial-secret"

	_, err := svc.Save(entservice.DingTalkConfigInput{
		CorpId:       "corp",
		AppKey:       "app-key",
		AppSecret:    &secret,
		CallbackUrl:  "https://example.com/callback",
		LoginEnabled: true,
	})
	require.NoError(t, err)

	blankSecret := "   "
	updated, err := svc.Save(entservice.DingTalkConfigInput{
		CorpId:      "corp-updated",
		AppKey:      "app-key-updated",
		AppSecret:   &blankSecret,
		CallbackUrl: "https://example.com/callback-updated",
		SyncEnabled: true,
	})
	require.NoError(t, err)
	require.True(t, updated.HasAppSecret)
	require.False(t, updated.LoginEnabled)
	require.True(t, updated.SyncEnabled)

	var config entmodel.DingTalkConfig
	require.NoError(t, db.Where("tenant_id = ?", 0).First(&config).Error)
	require.Equal(t, secret, config.AppSecret)
	require.Equal(t, "corp-updated", config.CorpId)
}

func TestDingTalkConfigRejectsEnableWithoutCredentials(t *testing.T) {
	svc, _ := newDingTalkConfigTestService(t)

	_, err := svc.Save(entservice.DingTalkConfigInput{
		CorpId:       "corp",
		CallbackUrl:  "https://example.com/callback",
		LoginEnabled: true,
	})
	require.ErrorIs(t, err, entservice.ErrDingTalkMissingCredentials)
}

func TestDingTalkConfigRejectsInvalidCallbackWhenEnabling(t *testing.T) {
	svc, _ := newDingTalkConfigTestService(t)
	secret := "secret"

	_, err := svc.Save(entservice.DingTalkConfigInput{
		CorpId:       "corp",
		AppKey:       "key",
		AppSecret:    &secret,
		CallbackUrl:  "http://example.com/callback",
		LoginEnabled: true,
	})
	require.ErrorIs(t, err, entservice.ErrDingTalkInvalidCallbackURL)
}

func TestDingTalkConfigAllowsDraftWithoutCredentials(t *testing.T) {
	svc, _ := newDingTalkConfigTestService(t)

	saved, err := svc.Save(entservice.DingTalkConfigInput{
		CorpId:    "corp",
		SyncScope: "1,2,3",
	})
	require.NoError(t, err)
	require.False(t, saved.HasAppSecret)
	require.False(t, saved.LoginEnabled)
	require.False(t, saved.SyncEnabled)
}
