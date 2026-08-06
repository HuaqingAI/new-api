package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrInvalidLifecycleInput  = errors.New("agent platform lifecycle input invalid")
	ErrRollbackVersionMissing = errors.New("agent platform rollback target version missing")
)

type LifecycleActionResult struct {
	ResourceId      string
	Action          string
	PreviousStatus  string
	CurrentStatus   string
	PreviousVersion string
	CurrentVersion  string
	TargetVersion   string
	RequestId       string
	AuditActionId   int
}

type LifecycleService struct {
	db *gorm.DB
}

func NewLifecycleService(db *gorm.DB) *LifecycleService {
	return &LifecycleService{db: db}
}

func (s *LifecycleService) Publish(resourceID string, version string, actorUserID int, requestID string) (LifecycleActionResult, error) {
	return s.transition(resourceID, "publish", version, apmodel.ResourceStatusPublished, actorUserID, requestID)
}

func (s *LifecycleService) Disable(resourceID string, actorUserID int, requestID string) (LifecycleActionResult, error) {
	return s.transition(resourceID, "disable", "", apmodel.ResourceStatusDisabled, actorUserID, requestID)
}

func (s *LifecycleService) Revoke(resourceID string, actorUserID int, requestID string) (LifecycleActionResult, error) {
	return s.transition(resourceID, "revoke", "", apmodel.ResourceStatusRevoked, actorUserID, requestID)
}

func (s *LifecycleService) Offline(resourceID string, actorUserID int, requestID string) (LifecycleActionResult, error) {
	return s.transition(resourceID, "offline", "", apmodel.ResourceStatusOffline, actorUserID, requestID)
}

func (s *LifecycleService) Rollback(resourceID string, version string, actorUserID int, requestID string) (LifecycleActionResult, error) {
	return s.transition(resourceID, "rollback", version, apmodel.ResourceStatusPublished, actorUserID, requestID)
}

func (s *LifecycleService) transition(resourceID string, action string, targetVersion string, targetStatus string, actorUserID int, requestID string) (LifecycleActionResult, error) {
	if s == nil || s.db == nil {
		return LifecycleActionResult{}, ErrInvalidLifecycleInput
	}
	resourceID = strings.TrimSpace(resourceID)
	targetVersion = strings.TrimSpace(targetVersion)
	requestID = strings.TrimSpace(requestID)
	if resourceID == "" || actorUserID <= 0 {
		return LifecycleActionResult{}, ErrInvalidLifecycleInput
	}
	if action == "publish" || action == "rollback" {
		if targetVersion == "" {
			return LifecycleActionResult{}, ErrInvalidLifecycleInput
		}
	}

	var resource apmodel.Resource
	if err := s.db.Where("resource_id = ?", resourceID).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = s.recordFailureAudit(resourceID, action, targetVersion, requestID, actorUserID, "resource not found")
			return LifecycleActionResult{}, ErrResourceNotFound
		}
		return LifecycleActionResult{}, err
	}

	previousStatus := resource.Status
	previousVersion := resource.LatestVersion
	result := LifecycleActionResult{
		ResourceId:      resource.ResourceId,
		Action:          action,
		PreviousStatus:  previousStatus,
		PreviousVersion: previousVersion,
		TargetVersion:   targetVersion,
		RequestId:       requestID,
	}

	var currentVersion apmodel.ResourceVersion
	var targetResourceVersion apmodel.ResourceVersion
	var hasCurrentVersion bool

	if previousVersion != "" {
		if err := s.db.Where("resource_id = ? AND version = ?", resource.ResourceId, previousVersion).First(&currentVersion).Error; err == nil {
			hasCurrentVersion = true
		}
	}
	if targetVersion != "" {
		if err := s.db.Where("resource_id = ? AND version = ?", resource.ResourceId, targetVersion).First(&targetResourceVersion).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = s.recordFailureAudit(resource.ResourceId, action, targetVersion, requestID, actorUserID, "target version not found")
				return LifecycleActionResult{}, ErrRollbackVersionMissing
			}
			return LifecycleActionResult{}, err
		}
	}

	now := time.Now().UTC()
	err := s.db.Transaction(func(tx *gorm.DB) error {
		switch action {
		case "publish":
			if hasCurrentVersion && currentVersion.Version != targetVersion && currentVersion.Status == apmodel.ResourceStatusPublished {
				if err := tx.Model(&apmodel.ResourceVersion{}).
					Where("resource_id = ? AND version = ?", resource.ResourceId, currentVersion.Version).
					Updates(map[string]any{"status": apmodel.ResourceStatusDeprecated}).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&apmodel.ResourceVersion{}).
				Where("resource_id = ? AND version = ?", resource.ResourceId, targetVersion).
				Updates(map[string]any{"status": apmodel.ResourceStatusPublished, "published_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&apmodel.Resource{}).
				Where("resource_id = ?", resource.ResourceId).
				Updates(map[string]any{"status": apmodel.ResourceStatusPublished, "latest_version": targetVersion}).Error; err != nil {
				return err
			}
			result.CurrentVersion = targetVersion
			result.CurrentStatus = apmodel.ResourceStatusPublished
		case "rollback":
			if hasCurrentVersion && currentVersion.Version != targetVersion && currentVersion.Status == apmodel.ResourceStatusPublished {
				if err := tx.Model(&apmodel.ResourceVersion{}).
					Where("resource_id = ? AND version = ?", resource.ResourceId, currentVersion.Version).
					Updates(map[string]any{"status": apmodel.ResourceStatusDeprecated}).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&apmodel.ResourceVersion{}).
				Where("resource_id = ? AND version = ?", resource.ResourceId, targetVersion).
				Updates(map[string]any{"status": apmodel.ResourceStatusPublished, "published_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&apmodel.Resource{}).
				Where("resource_id = ?", resource.ResourceId).
				Updates(map[string]any{"status": apmodel.ResourceStatusPublished, "latest_version": targetVersion}).Error; err != nil {
				return err
			}
			result.CurrentVersion = targetVersion
			result.CurrentStatus = apmodel.ResourceStatusPublished
		default:
			if hasCurrentVersion {
				if err := tx.Model(&apmodel.ResourceVersion{}).
					Where("resource_id = ? AND version = ?", resource.ResourceId, currentVersion.Version).
					Updates(map[string]any{"status": targetStatus}).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&apmodel.Resource{}).
				Where("resource_id = ?", resource.ResourceId).
				Update("status", targetStatus).Error; err != nil {
				return err
			}
			result.CurrentVersion = previousVersion
			result.CurrentStatus = targetStatus
		}

		auditID, err := s.createAdminAction(tx, createAdminActionInput{
			ActorUserID:     actorUserID,
			ActionType:      "agentplatform.resource." + action,
			ObjectType:      "resource",
			ObjectID:        resource.ResourceId,
			BeforeStatus:    previousStatus,
			AfterStatus:     result.CurrentStatus,
			TargetVersion:   targetVersion,
			RequestID:       requestID,
			Result:          "success",
			ErrorSummary:    "",
			PreviousVersion: previousVersion,
			CurrentVersion:  result.CurrentVersion,
		})
		if err != nil {
			return err
		}
		result.AuditActionId = auditID
		return nil
	})
	if err != nil {
		return LifecycleActionResult{}, err
	}
	return result, nil
}

type createAdminActionInput struct {
	ActorUserID     int
	ActionType      string
	ObjectType      string
	ObjectID        string
	BeforeStatus    string
	AfterStatus     string
	TargetVersion   string
	RequestID       string
	Result          string
	ErrorSummary    string
	PreviousVersion string
	CurrentVersion  string
}

func (s *LifecycleService) createAdminAction(tx *gorm.DB, input createAdminActionInput) (int, error) {
	payload, err := common.Marshal(map[string]any{
		"previous_version": input.PreviousVersion,
		"current_version":  input.CurrentVersion,
		"target_version":   input.TargetVersion,
	})
	if err != nil {
		return 0, err
	}
	action := apmodel.AdminAction{
		ActorUserId:   input.ActorUserID,
		ActionType:    input.ActionType,
		ObjectType:    input.ObjectType,
		ObjectId:      input.ObjectID,
		BeforeStatus:  input.BeforeStatus,
		AfterStatus:   input.AfterStatus,
		TargetVersion: input.TargetVersion,
		RequestId:     input.RequestID,
		Result:        input.Result,
		ErrorSummary:  input.ErrorSummary,
		Payload:       string(payload),
	}
	if err := tx.Create(&action).Error; err != nil {
		return 0, err
	}
	return action.Id, nil
}

func (s *LifecycleService) recordFailureAudit(resourceID string, action string, targetVersion string, requestID string, actorUserID int, errorSummary string) error {
	if s == nil || s.db == nil || actorUserID <= 0 {
		return nil
	}
	_, err := s.createAdminAction(s.db, createAdminActionInput{
		ActorUserID:   actorUserID,
		ActionType:    "agentplatform.resource." + action,
		ObjectType:    "resource",
		ObjectID:      resourceID,
		BeforeStatus:  "",
		AfterStatus:   "",
		TargetVersion: targetVersion,
		RequestID:     requestID,
		Result:        "failed",
		ErrorSummary:  errorSummary,
	})
	return err
}
