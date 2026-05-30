package enterprise

import (
	"errors"
	"fmt"
	"net/mail"
	neturl "net/url"
	"regexp"
	"sort"
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

type AlertEventQuery struct {
	TenantId     int
	DepartmentId *int
	UserId       *int
	Username     string
	ModelName    string
	RiskType     string
	From         *int64
	To           *int64
	Page         int
	PageSize     int
}

type AlertEventItem struct {
	Id                 int
	TenantId           int
	UserId             int
	Username           string
	RequestId          string
	ModelName          string
	RiskType           string
	ActionResult       string
	CreatedAt          int64
	DepartmentSnapshot []entmodel.AlertEventDepartmentSnapshot
	Summary            string
}

type AlertEventListResult struct {
	Items    []AlertEventItem
	Total    int
	Page     int
	PageSize int
}

type AlertRuleChannelInput struct {
	Type          string
	Enabled       *bool
	Receivers     []string
	WebhookURL    string
	WebhookSecret *string
	RobotWebhook  string
	RobotSecret   *string
}

type AlertRuleInput struct {
	Id                  int
	TenantId            int
	Name                string
	Enabled             *bool
	RiskTypes           []string
	DepartmentIds       []int
	ChannelConfigs      []AlertRuleChannelInput
	DedupeWindowSeconds *int
	ActorId             int
}

type AlertRuleChannelItem struct {
	Type                     string
	Enabled                  bool
	Receivers                []string
	WebhookURL               string
	WebhookSecretConfigured  bool
	WebhookSecretMasked      string
	DingTalkRobotURL         string
	DingTalkSecretConfigured bool
	DingTalkSecretMasked     string
}

type AlertRuleItem struct {
	Id                  int
	TenantId            int
	Name                string
	Enabled             bool
	RiskTypes           []string
	DepartmentIds       []int
	ChannelConfigs      []AlertRuleChannelItem
	DedupeWindowSeconds int
	CreatedBy           int
	UpdatedBy           int
	CreatedAt           int64
	UpdatedAt           int64
}

type AlertRuleListResult struct {
	Items []AlertRuleItem
	Total int
}

type AlertRuleMutationResult struct {
	Item         AlertRuleItem
	PreviousItem *AlertRuleItem
	AuditSummary string
	AuditPayload map[string]any
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

func (s *AlertService) ListAlertEvents(query AlertEventQuery) (AlertEventListResult, error) {
	if s == nil || s.db == nil {
		return AlertEventListResult{Items: []AlertEventItem{}}, nil
	}
	if query.TenantId < 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.DepartmentId != nil && *query.DepartmentId <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.UserId != nil && *query.UserId <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.From != nil && *query.From <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.To != nil && *query.To <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.From != nil && query.To != nil && *query.From > *query.To {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}

	page, pageSize := normalizeAlertEventPage(query.Page, query.PageSize)
	db := s.db.Model(&entmodel.AlertEvent{}).Where("tenant_id = ?", query.TenantId)
	db = applyAlertEventFilters(db, query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return AlertEventListResult{Items: []AlertEventItem{}}, err
	}

	var events []entmodel.AlertEvent
	err := db.Order("created_at DESC").
		Order("id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&events).Error
	if err != nil {
		return AlertEventListResult{Items: []AlertEventItem{}}, err
	}

	items := make([]AlertEventItem, 0, len(events))
	for _, event := range events {
		snapshot, err := event.ParsedDepartmentSnapshot()
		if err != nil {
			return AlertEventListResult{}, err
		}
		items = append(items, AlertEventItem{
			Id:                 event.Id,
			TenantId:           event.TenantId,
			UserId:             event.UserId,
			Username:           event.Username,
			RequestId:          event.RequestId,
			ModelName:          event.ModelName,
			RiskType:           event.RiskType,
			ActionResult:       event.ActionResult,
			CreatedAt:          event.CreatedAt,
			DepartmentSnapshot: snapshot,
			Summary:            event.Summary,
		})
	}

	return AlertEventListResult{
		Items:    items,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AlertService) ListAlertRules(tenantId int) (AlertRuleListResult, error) {
	if s == nil || s.db == nil {
		return AlertRuleListResult{Items: []AlertRuleItem{}}, nil
	}
	if tenantId < 0 {
		return AlertRuleListResult{}, ErrAlertRuleInvalidInput
	}

	var rules []entmodel.AlertRule
	if err := s.db.Where("tenant_id = ?", tenantId).
		Order("updated_at DESC, id DESC").
		Find(&rules).Error; err != nil {
		return AlertRuleListResult{}, err
	}

	items := make([]AlertRuleItem, 0, len(rules))
	for _, rule := range rules {
		item, err := mapAlertRuleItem(rule)
		if err != nil {
			return AlertRuleListResult{}, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []AlertRuleItem{}
	}
	return AlertRuleListResult{Items: items, Total: len(items)}, nil
}

func (s *AlertService) GetAlertRule(tenantId int, ruleId int) (AlertRuleItem, error) {
	if s == nil || s.db == nil {
		return AlertRuleItem{}, ErrAlertRuleNotFound
	}
	if tenantId < 0 || ruleId <= 0 {
		return AlertRuleItem{}, ErrAlertRuleInvalidInput
	}

	rule, err := s.getAlertRuleModel(tenantId, ruleId)
	if err != nil {
		return AlertRuleItem{}, err
	}
	return mapAlertRuleItem(rule)
}

func (s *AlertService) SaveAlertRule(input AlertRuleInput) (AlertRuleMutationResult, error) {
	if s == nil || s.db == nil {
		return AlertRuleMutationResult{}, ErrAlertRuleInvalidInput
	}
	if input.TenantId < 0 || input.ActorId <= 0 {
		return AlertRuleMutationResult{}, ErrAlertRuleInvalidInput
	}

	var existing *entmodel.AlertRule
	var previousItem *AlertRuleItem
	if input.Id > 0 {
		rule, err := s.getAlertRuleModel(input.TenantId, input.Id)
		if err != nil {
			return AlertRuleMutationResult{}, err
		}
		existing = &rule
		item, err := mapAlertRuleItem(rule)
		if err != nil {
			return AlertRuleMutationResult{}, err
		}
		previousItem = &item
	}

	normalized, err := normalizeAlertRuleInput(input, existing)
	if err != nil {
		return AlertRuleMutationResult{}, err
	}

	var rule entmodel.AlertRule
	if existing == nil {
		rule = entmodel.AlertRule{
			TenantId:            input.TenantId,
			Name:                normalized.Name,
			Enabled:             normalized.Enabled,
			DedupeWindowSeconds: normalized.DedupeWindowSeconds,
			CreatedBy:           input.ActorId,
			UpdatedBy:           input.ActorId,
		}
		if err := rule.SetRiskTypes(normalized.RiskTypes); err != nil {
			return AlertRuleMutationResult{}, err
		}
		if err := rule.SetDepartmentIds(normalized.DepartmentIds); err != nil {
			return AlertRuleMutationResult{}, err
		}
		if err := rule.SetChannelConfigs(normalized.StoredChannels); err != nil {
			return AlertRuleMutationResult{}, err
		}
		if err := s.db.Create(&rule).Error; err != nil {
			return AlertRuleMutationResult{}, err
		}
	} else {
		rule = *existing
		rule.Name = normalized.Name
		rule.Enabled = normalized.Enabled
		rule.DedupeWindowSeconds = normalized.DedupeWindowSeconds
		rule.UpdatedBy = input.ActorId
		if err := rule.SetRiskTypes(normalized.RiskTypes); err != nil {
			return AlertRuleMutationResult{}, err
		}
		if err := rule.SetDepartmentIds(normalized.DepartmentIds); err != nil {
			return AlertRuleMutationResult{}, err
		}
		if err := rule.SetChannelConfigs(normalized.StoredChannels); err != nil {
			return AlertRuleMutationResult{}, err
		}
		if err := s.db.Model(existing).Updates(map[string]any{
			"name":                  rule.Name,
			"enabled":               rule.Enabled,
			"risk_types":            rule.RiskTypes,
			"department_ids":        rule.DepartmentIds,
			"channel_configs":       rule.ChannelConfigs,
			"dedupe_window_seconds": rule.DedupeWindowSeconds,
			"updated_by":            rule.UpdatedBy,
			"updated_at":            s.now().Unix(),
		}).Error; err != nil {
			return AlertRuleMutationResult{}, err
		}
		if err := s.db.Where("id = ? AND tenant_id = ?", existing.Id, input.TenantId).First(&rule).Error; err != nil {
			return AlertRuleMutationResult{}, err
		}
	}

	item, err := mapAlertRuleItem(rule)
	if err != nil {
		return AlertRuleMutationResult{}, err
	}
	return AlertRuleMutationResult{
		Item:         item,
		PreviousItem: previousItem,
		AuditSummary: buildAlertRuleDiffSummary(previousItem, item, false),
		AuditPayload: buildAlertRuleAuditPayload(item),
	}, nil
}

func (s *AlertService) DeleteAlertRule(tenantId int, ruleId int, actorId int) (AlertRuleMutationResult, error) {
	if s == nil || s.db == nil {
		return AlertRuleMutationResult{}, ErrAlertRuleInvalidInput
	}
	if tenantId < 0 || ruleId <= 0 || actorId <= 0 {
		return AlertRuleMutationResult{}, ErrAlertRuleInvalidInput
	}

	rule, err := s.getAlertRuleModel(tenantId, ruleId)
	if err != nil {
		return AlertRuleMutationResult{}, err
	}
	item, err := mapAlertRuleItem(rule)
	if err != nil {
		return AlertRuleMutationResult{}, err
	}
	if err := s.db.Delete(&rule).Error; err != nil {
		return AlertRuleMutationResult{}, err
	}
	return AlertRuleMutationResult{
		Item:         item,
		PreviousItem: &item,
		AuditSummary: buildAlertRuleDiffSummary(&item, item, true),
		AuditPayload: buildAlertRuleAuditPayload(item),
	}, nil
}

func (s *AlertService) getAlertRuleModel(tenantId int, ruleId int) (entmodel.AlertRule, error) {
	var rule entmodel.AlertRule
	if err := s.db.Where("tenant_id = ? AND id = ?", tenantId, ruleId).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entmodel.AlertRule{}, ErrAlertRuleNotFound
		}
		return entmodel.AlertRule{}, err
	}
	return rule, nil
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

func applyAlertEventFilters(db *gorm.DB, query AlertEventQuery) *gorm.DB {
	if query.DepartmentId != nil {
		tokenPattern := fmt.Sprintf("%%|%d|%%", *query.DepartmentId)
		snapshotPattern := fmt.Sprintf("%%\"department_id\":%d%%", *query.DepartmentId)
		legacySnapshotPattern := fmt.Sprintf("%%\"department_id\": %d%%", *query.DepartmentId)
		db = db.Where(
			"(department_tokens LIKE ?) OR ((department_tokens = '' OR department_tokens IS NULL) AND (department_snapshot LIKE ? OR department_snapshot LIKE ?))",
			tokenPattern,
			snapshotPattern,
			legacySnapshotPattern,
		)
	}
	if query.UserId != nil {
		db = db.Where("user_id = ?", *query.UserId)
	}
	if username := strings.TrimSpace(query.Username); username != "" {
		db = db.Where("username = ?", username)
	}
	if modelName := strings.TrimSpace(query.ModelName); modelName != "" {
		db = db.Where("model_name = ?", modelName)
	}
	if riskType := strings.TrimSpace(query.RiskType); riskType != "" {
		db = db.Where("risk_type = ?", riskType)
	}
	if query.From != nil {
		db = db.Where("created_at >= ?", *query.From)
	}
	if query.To != nil {
		db = db.Where("created_at <= ?", *query.To)
	}
	return db
}

func normalizeAlertEventPage(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
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

type normalizedAlertRuleInput struct {
	Name                string
	Enabled             bool
	RiskTypes           []string
	DepartmentIds       []int
	DedupeWindowSeconds int
	StoredChannels      []entmodel.AlertRuleChannelConfig
}

func normalizeAlertRuleInput(input AlertRuleInput, existing *entmodel.AlertRule) (normalizedAlertRuleInput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return normalizedAlertRuleInput{}, ErrAlertRuleInvalidInput
	}

	riskTypes := uniqueNonEmptyStrings(input.RiskTypes)
	if len(riskTypes) == 0 {
		return normalizedAlertRuleInput{}, ErrAlertRuleInvalidInput
	}

	departmentIds, err := uniquePositiveInts(input.DepartmentIds)
	if err != nil {
		return normalizedAlertRuleInput{}, ErrAlertRuleInvalidInput
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	dedupeWindowSeconds := 0
	if input.DedupeWindowSeconds != nil {
		dedupeWindowSeconds = *input.DedupeWindowSeconds
		if dedupeWindowSeconds < 0 {
			return normalizedAlertRuleInput{}, ErrAlertRuleInvalidInput
		}
	}

	existingConfigs := make(map[string]entmodel.AlertRuleChannelConfig)
	if existing != nil {
		parsed, err := existing.ParsedChannelConfigs()
		if err != nil {
			return normalizedAlertRuleInput{}, err
		}
		for _, item := range parsed {
			existingConfigs[item.Type] = item
		}
	}

	storedChannels, err := normalizeAlertRuleChannels(input.ChannelConfigs, existingConfigs)
	if err != nil {
		return normalizedAlertRuleInput{}, err
	}

	return normalizedAlertRuleInput{
		Name:                name,
		Enabled:             enabled,
		RiskTypes:           riskTypes,
		DepartmentIds:       departmentIds,
		DedupeWindowSeconds: dedupeWindowSeconds,
		StoredChannels:      storedChannels,
	}, nil
}

func normalizeAlertRuleChannels(inputs []AlertRuleChannelInput, existing map[string]entmodel.AlertRuleChannelConfig) ([]entmodel.AlertRuleChannelConfig, error) {
	byType := make(map[string]AlertRuleChannelInput, len(inputs))
	for _, item := range inputs {
		channelType := normalizeAlertRuleChannelType(item.Type)
		if channelType == "" {
			return nil, ErrAlertRuleInvalidInput
		}
		item.Type = channelType
		byType[channelType] = item
	}

	if len(byType) == 0 {
		if existingEmail, ok := existing[entmodel.AlertRuleChannelEmail]; ok {
			byType[entmodel.AlertRuleChannelEmail] = AlertRuleChannelInput{
				Type:      entmodel.AlertRuleChannelEmail,
				Enabled:   boolPtr(existingEmail.Enabled),
				Receivers: append([]string{}, existingEmail.Receivers...),
			}
		}
	}

	emailInput, ok := byType[entmodel.AlertRuleChannelEmail]
	if !ok {
		return nil, ErrAlertRuleChannelRequired
	}
	emailEnabled := emailInput.Enabled == nil || *emailInput.Enabled
	receivers, err := normalizeAlertRuleReceivers(emailInput.Receivers)
	if err != nil {
		return nil, err
	}
	if !emailEnabled || len(receivers) == 0 {
		return nil, ErrAlertRuleChannelRequired
	}

	channels := []entmodel.AlertRuleChannelConfig{
		{
			Type:      entmodel.AlertRuleChannelEmail,
			Enabled:   true,
			Receivers: receivers,
		},
	}

	if webhookInput, ok := byType[entmodel.AlertRuleChannelWebhook]; ok {
		config, err := mergeWebhookChannelConfig(webhookInput, existing[entmodel.AlertRuleChannelWebhook], false)
		if err != nil {
			return nil, err
		}
		channels = append(channels, config)
	} else if existingConfig, ok := existing[entmodel.AlertRuleChannelWebhook]; ok {
		channels = append(channels, existingConfig)
	}

	if robotInput, ok := byType[entmodel.AlertRuleChannelDingTalkRobot]; ok {
		config, err := mergeWebhookChannelConfig(robotInput, existing[entmodel.AlertRuleChannelDingTalkRobot], true)
		if err != nil {
			return nil, err
		}
		channels = append(channels, config)
	} else if existingConfig, ok := existing[entmodel.AlertRuleChannelDingTalkRobot]; ok {
		channels = append(channels, existingConfig)
	}

	return channels, nil
}

func boolPtr(value bool) *bool {
	return &value
}

func mergeWebhookChannelConfig(input AlertRuleChannelInput, existing entmodel.AlertRuleChannelConfig, dingTalk bool) (entmodel.AlertRuleChannelConfig, error) {
	enabled := input.Enabled != nil && *input.Enabled
	if input.Enabled == nil {
		enabled = existing.Enabled
	}

	if dingTalk {
		webhookURL := strings.TrimSpace(input.RobotWebhook)
		if webhookURL == "" {
			webhookURL = existing.RobotWebhook
		}
		if enabled && webhookURL == "" {
			return entmodel.AlertRuleChannelConfig{}, ErrAlertRuleInvalidWebhookURL
		}
		if webhookURL != "" && !isSupportedAlertWebhookURL(webhookURL) {
			return entmodel.AlertRuleChannelConfig{}, ErrAlertRuleInvalidWebhookURL
		}
		secret := existing.RobotSecret
		if input.RobotSecret != nil {
			secret = strings.TrimSpace(*input.RobotSecret)
		}
		return entmodel.AlertRuleChannelConfig{
			Type:         entmodel.AlertRuleChannelDingTalkRobot,
			Enabled:      enabled,
			RobotWebhook: webhookURL,
			RobotSecret:  secret,
		}, nil
	}

	webhookURL := strings.TrimSpace(input.WebhookURL)
	if webhookURL == "" {
		webhookURL = existing.WebhookURL
	}
	if enabled && webhookURL == "" {
		return entmodel.AlertRuleChannelConfig{}, ErrAlertRuleInvalidWebhookURL
	}
	if webhookURL != "" && !isSupportedAlertWebhookURL(webhookURL) {
		return entmodel.AlertRuleChannelConfig{}, ErrAlertRuleInvalidWebhookURL
	}
	secret := existing.WebhookSecret
	if input.WebhookSecret != nil {
		secret = strings.TrimSpace(*input.WebhookSecret)
	}
	return entmodel.AlertRuleChannelConfig{
		Type:          entmodel.AlertRuleChannelWebhook,
		Enabled:       enabled,
		WebhookURL:    webhookURL,
		WebhookSecret: secret,
	}, nil
}

func normalizeAlertRuleReceivers(receivers []string) ([]string, error) {
	if receivers == nil {
		receivers = []string{}
	}
	normalized := make([]string, 0, len(receivers))
	seen := make(map[string]struct{}, len(receivers))
	for _, item := range receivers {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		addr, err := mail.ParseAddress(value)
		if err != nil || addr.Address == "" {
			return nil, ErrAlertRuleInvalidEmail
		}
		key := strings.ToLower(addr.Address)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, addr.Address)
	}
	return normalized, nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func uniquePositiveInts(values []int) ([]int, error) {
	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			return nil, ErrAlertRuleInvalidInput
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Ints(out)
	return out, nil
}

func normalizeAlertRuleChannelType(value string) string {
	switch strings.TrimSpace(value) {
	case entmodel.AlertRuleChannelEmail:
		return entmodel.AlertRuleChannelEmail
	case entmodel.AlertRuleChannelWebhook:
		return entmodel.AlertRuleChannelWebhook
	case entmodel.AlertRuleChannelDingTalkRobot:
		return entmodel.AlertRuleChannelDingTalkRobot
	default:
		return ""
	}
}

func isSupportedAlertWebhookURL(raw string) bool {
	parsed, err := neturl.Parse(raw)
	if err != nil || parsed == nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

func mapAlertRuleItem(rule entmodel.AlertRule) (AlertRuleItem, error) {
	riskTypes, err := rule.ParsedRiskTypes()
	if err != nil {
		return AlertRuleItem{}, err
	}
	departmentIds, err := rule.ParsedDepartmentIds()
	if err != nil {
		return AlertRuleItem{}, err
	}
	configs, err := rule.ParsedChannelConfigs()
	if err != nil {
		return AlertRuleItem{}, err
	}
	items := make([]AlertRuleChannelItem, 0, len(configs))
	for _, config := range configs {
		items = append(items, mapAlertRuleChannelItem(config))
	}
	if items == nil {
		items = []AlertRuleChannelItem{}
	}
	return AlertRuleItem{
		Id:                  rule.Id,
		TenantId:            rule.TenantId,
		Name:                rule.Name,
		Enabled:             rule.Enabled,
		RiskTypes:           riskTypes,
		DepartmentIds:       departmentIds,
		ChannelConfigs:      items,
		DedupeWindowSeconds: rule.DedupeWindowSeconds,
		CreatedBy:           rule.CreatedBy,
		UpdatedBy:           rule.UpdatedBy,
		CreatedAt:           rule.CreatedAt,
		UpdatedAt:           rule.UpdatedAt,
	}, nil
}

func mapAlertRuleChannelItem(config entmodel.AlertRuleChannelConfig) AlertRuleChannelItem {
	item := AlertRuleChannelItem{
		Type:      config.Type,
		Enabled:   config.Enabled,
		Receivers: append([]string{}, config.Receivers...),
	}
	switch config.Type {
	case entmodel.AlertRuleChannelWebhook:
		item.WebhookURL = redactAlertWebhookURL(config.WebhookURL)
		item.WebhookSecretConfigured = strings.TrimSpace(config.WebhookSecret) != ""
		item.WebhookSecretMasked = maskAlertSecret(config.WebhookSecret)
	case entmodel.AlertRuleChannelDingTalkRobot:
		item.DingTalkRobotURL = redactAlertWebhookURL(config.RobotWebhook)
		item.DingTalkSecretConfigured = strings.TrimSpace(config.RobotSecret) != ""
		item.DingTalkSecretMasked = maskAlertSecret(config.RobotSecret)
	}
	if item.Receivers == nil {
		item.Receivers = []string{}
	}
	return item
}

func buildAlertRuleAuditPayload(item AlertRuleItem) map[string]any {
	channels := make([]map[string]any, 0, len(item.ChannelConfigs))
	for _, channel := range item.ChannelConfigs {
		entry := map[string]any{
			"type":    channel.Type,
			"enabled": channel.Enabled,
		}
		if channel.Type == entmodel.AlertRuleChannelEmail {
			entry["receivers"] = append([]string{}, channel.Receivers...)
		}
		if channel.Type == entmodel.AlertRuleChannelWebhook {
			entry["webhook_url"] = channel.WebhookURL
			entry["webhook_secret_configured"] = channel.WebhookSecretConfigured
		}
		if channel.Type == entmodel.AlertRuleChannelDingTalkRobot {
			entry["robot_webhook"] = channel.DingTalkRobotURL
			entry["robot_secret_configured"] = channel.DingTalkSecretConfigured
		}
		channels = append(channels, entry)
	}
	return map[string]any{
		"rule_id":               item.Id,
		"tenant_id":             item.TenantId,
		"name":                  item.Name,
		"enabled":               item.Enabled,
		"risk_types":            append([]string{}, item.RiskTypes...),
		"department_ids":        append([]int{}, item.DepartmentIds...),
		"channel_configs":       channels,
		"dedupe_window_seconds": item.DedupeWindowSeconds,
		"created_by":            item.CreatedBy,
		"updated_by":            item.UpdatedBy,
	}
}

func buildAlertRuleDiffSummary(previous *AlertRuleItem, current AlertRuleItem, deleted bool) string {
	if deleted {
		return fmt.Sprintf("Deleted alert rule #%d (%s)", current.Id, current.Name)
	}
	if previous == nil {
		return fmt.Sprintf("Created alert rule #%d (%s)", current.Id, current.Name)
	}
	return fmt.Sprintf(
		"Updated alert rule #%d (%s): enabled=%t risk_types=%d departments=%d channels=%d",
		current.Id,
		current.Name,
		current.Enabled,
		len(current.RiskTypes),
		len(current.DepartmentIds),
		len(current.ChannelConfigs),
	)
}

func redactAlertWebhookURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := neturl.Parse(raw)
	if err != nil || parsed == nil {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func maskAlertSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(secret)-4) + secret[len(secret)-4:]
}
