package agentplatform

import (
	"encoding/json"
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newLifecycleServiceForTest(t *testing.T) (*LifecycleService, *gorm.DB, apmodel.Resource) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	resource := apmodel.Resource{ResourceType: apmodel.ResourceTypeSkill, DisplayName: "Skill", OwnerUserId: 100}
	require.NoError(t, db.Create(&resource).Error)

	versionService := NewResourceVersionService(db)
	_, err = versionService.Create(resource.ResourceId, CreateResourceVersionInput{
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Skill: &SkillDetailInput{
			InvokeSchema:   json.RawMessage(`{"type":"object"}`),
			OutputSchema:   json.RawMessage(`{"type":"object"}`),
			InvokeMode:     "sync",
			TimeoutSeconds: intPtr(30),
			BindingConfig:  json.RawMessage(`{"provider":"demo"}`),
		},
		CreatedBy: 100,
	})
	require.NoError(t, err)
	_, err = versionService.Create(resource.ResourceId, CreateResourceVersionInput{
		Version:         "1.1.0",
		ContractVersion: "2026-06",
		Schema:          json.RawMessage(`{"type":"object"}`),
		Skill: &SkillDetailInput{
			InvokeSchema:   json.RawMessage(`{"type":"object"}`),
			OutputSchema:   json.RawMessage(`{"type":"object"}`),
			InvokeMode:     "sync",
			TimeoutSeconds: intPtr(45),
			BindingConfig:  json.RawMessage(`{"provider":"demo-v2"}`),
		},
		CreatedBy: 100,
	})
	require.NoError(t, err)

	return NewLifecycleService(db), db, resource
}

func TestLifecycleServicePublishesAndRollsBackByExplicitAction(t *testing.T) {
	svc, db, resource := newLifecycleServiceForTest(t)

	published, err := svc.Publish(resource.ResourceId, "1.0.0", 100, "req-publish-1")
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceStatusPublished, published.CurrentStatus)
	require.Equal(t, "1.0.0", published.CurrentVersion)

	var persisted apmodel.Resource
	require.NoError(t, db.Where("resource_id = ?", resource.ResourceId).First(&persisted).Error)
	require.Equal(t, apmodel.ResourceStatusPublished, persisted.Status)
	require.Equal(t, "1.0.0", persisted.LatestVersion)

	rolledBack, err := svc.Rollback(resource.ResourceId, "1.1.0", 100, "req-rollback-1")
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceStatusPublished, rolledBack.CurrentStatus)
	require.Equal(t, "1.1.0", rolledBack.CurrentVersion)
	require.Equal(t, "1.0.0", rolledBack.PreviousVersion)

	require.NoError(t, db.Where("resource_id = ?", resource.ResourceId).First(&persisted).Error)
	require.Equal(t, "1.1.0", persisted.LatestVersion)
}

func TestLifecycleServiceExplicitStatusActionsAndAuditTrail(t *testing.T) {
	svc, db, resource := newLifecycleServiceForTest(t)

	_, err := svc.Publish(resource.ResourceId, "1.0.0", 100, "req-publish-2")
	require.NoError(t, err)

	disabled, err := svc.Disable(resource.ResourceId, 100, "req-disable-1")
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceStatusDisabled, disabled.CurrentStatus)

	revoked, err := svc.Revoke(resource.ResourceId, 100, "req-revoke-1")
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceStatusRevoked, revoked.CurrentStatus)

	offline, err := svc.Offline(resource.ResourceId, 100, "req-offline-1")
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceStatusOffline, offline.CurrentStatus)

	var audits []apmodel.AdminAction
	require.NoError(t, db.Order("id ASC").Find(&audits).Error)
	require.GreaterOrEqual(t, len(audits), 4)
	require.Equal(t, "agentplatform.resource.publish", audits[0].ActionType)
	require.Equal(t, "agentplatform.resource.disable", audits[1].ActionType)
	require.Equal(t, "agentplatform.resource.revoke", audits[2].ActionType)
	require.Equal(t, "agentplatform.resource.offline", audits[3].ActionType)
}

func TestLifecycleServiceRejectsMissingRollbackTarget(t *testing.T) {
	svc, _, resource := newLifecycleServiceForTest(t)

	_, err := svc.Rollback(resource.ResourceId, "", 100, "req-rollback-missing")
	require.ErrorIs(t, err, ErrInvalidLifecycleInput)

	_, err = svc.Rollback(resource.ResourceId, "9.9.9", 100, "req-rollback-missing-version")
	require.ErrorIs(t, err, ErrRollbackVersionMissing)
}
