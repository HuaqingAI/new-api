package enterprise

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	AlertRiskTypeSensitiveWords = "sensitive_words"
	AlertRiskTypeUnknown        = "unknown"
	AlertActionBlocked          = "blocked"
	AlertActionUnknown          = "unknown"
	maxAlertSummaryLength       = 256
)

type AlertService struct {
	db  *gorm.DB
	now func() time.Time
}

type RecordRiskEventInput struct {
	TenantId        int
	ModelName       string
	RiskType        string
	ActionResult    string
	Summary         string
	SensitiveHits   []string
	EventOccurredAt time.Time
}

type alertEventRow struct {
	DepartmentId   int
	DepartmentName string
	ExternalSource string
	Status         int
}

func NewAlertService(db *gorm.DB) *AlertService {
	if db == nil {
		db = model.DB
	}
	return &AlertService{
		db:  db,
		now: time.Now,
	}
}

func RecordRiskEvent(c *gin.Context, input RecordRiskEventInput) {
	_ = NewAlertService(nil).RecordRiskEvent(c, input)
}

func (s *AlertService) RecordRiskEvent(c *gin.Context, input RecordRiskEventInput) error {
	if s == nil || s.db == nil {
		return nil
	}

	userId := common.GetContextKeyInt(c, constant.ContextKeyUserId)
	username := common.GetContextKeyString(c, constant.ContextKeyUserName)
	requestId := c.GetString(common.RequestIdKey)
	if input.ModelName == "" {
		input.ModelName = common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
	}
	if input.TenantId < 0 {
		input.TenantId = 0
	}
	if input.EventOccurredAt.IsZero() {
		input.EventOccurredAt = s.now()
	}

	snapshot, err := s.loadDepartmentSnapshot(input.TenantId, userId)
	if err != nil {
		logger.LogWarn(c, fmt.Sprintf("enterprise alert event persistence skipped: snapshot_lookup_failed: %s", err.Error()))
		return nil
	}

	event := entmodel.AlertEvent{
		TenantId:     input.TenantId,
		UserId:       userId,
		Username:     username,
		RequestId:    requestId,
		ModelName:    input.ModelName,
		RiskType:     normalizeAlertRiskType(input.RiskType),
		ActionResult: normalizeAlertActionResult(input.ActionResult),
		Summary:      sanitizeRiskSummary(input.RiskType, input.Summary, input.SensitiveHits),
		CreatedAt:    input.EventOccurredAt.Unix(),
		UpdatedAt:    input.EventOccurredAt.Unix(),
	}
	if err := event.SetDepartmentSnapshot(snapshot); err != nil {
		logger.LogWarn(c, fmt.Sprintf("enterprise alert event persistence skipped: snapshot_encode_failed: %s", err.Error()))
		return nil
	}
	if err := s.db.Create(&event).Error; err != nil {
		logger.LogWarn(c, fmt.Sprintf("enterprise alert event persistence skipped: create_failed: %s", err.Error()))
		return nil
	}
	return nil
}

func (s *AlertService) loadDepartmentSnapshot(tenantId int, userId int) ([]entmodel.AlertEventDepartmentSnapshot, error) {
	if userId <= 0 {
		return []entmodel.AlertEventDepartmentSnapshot{}, nil
	}

	var rows []alertEventRow
	err := s.db.Table("enterprise_user_departments AS ud").
		Select("ud.department_id, d.name AS department_name, ud.external_source, ud.status").
		Joins("LEFT JOIN enterprise_departments AS d ON d.id = ud.department_id AND d.tenant_id = ud.tenant_id").
		Where("ud.tenant_id = ? AND ud.user_id = ? AND ud.status = ?", tenantId, userId, constant.EnterpriseMembershipStatusActive).
		Order("ud.department_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	snapshot := make([]entmodel.AlertEventDepartmentSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshot = append(snapshot, entmodel.AlertEventDepartmentSnapshot{
			DepartmentId:   row.DepartmentId,
			DepartmentName: row.DepartmentName,
			ExternalSource: row.ExternalSource,
			Status:         row.Status,
		})
	}
	return snapshot, nil
}

func sanitizeRiskSummary(riskType string, summary string, hits []string) string {
	if normalizeAlertRiskType(riskType) == AlertRiskTypeSensitiveWords {
		if hitCount := countNonEmptyHits(hits); hitCount > 0 {
			return fmt.Sprintf("%d sensitive word hits", hitCount)
		}
		return "sensitive words detected"
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return ""
	}

	for _, hit := range hits {
		hit = strings.TrimSpace(hit)
		if hit == "" {
			continue
		}
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(hit))
		summary = re.ReplaceAllString(summary, "[REDACTED]")
	}
	summary = strings.Join(strings.Fields(summary), " ")
	if len(summary) > maxAlertSummaryLength {
		summary = summary[:maxAlertSummaryLength]
	}
	return summary
}

func normalizeAlertRiskType(riskType string) string {
	riskType = strings.TrimSpace(riskType)
	if riskType == "" {
		return AlertRiskTypeUnknown
	}
	return riskType
}

func normalizeAlertActionResult(actionResult string) string {
	actionResult = strings.TrimSpace(actionResult)
	if actionResult == "" {
		return AlertActionUnknown
	}
	return actionResult
}

func countNonEmptyHits(hits []string) int {
	count := 0
	for _, hit := range hits {
		if strings.TrimSpace(hit) != "" {
			count++
		}
	}
	return count
}
