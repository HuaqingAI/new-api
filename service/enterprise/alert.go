package enterprise

import (
	"errors"
	"fmt"
	"net/mail"
	neturl "net/url"
	"regexp"
	"sort"
	"strconv"
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
	TenantId       int
	EventId        *int
	DepartmentId   *int
	UnassignedOnly *bool
	UserId         *int
	Username       string
	ModelName      string
	RiskType       string
	From           *int64
	To             *int64
	Page           int
	PageSize       int
}

type AlertEventItem struct {
	Id                 int
	TenantId           int
	UserId             int
	Username           string
	DisplayName        string
	UsernameSnapshot   string
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

type DepartmentRiskSummaryQuery struct {
	TenantId int
	From     int64
	To       int64
	Sort     UsageSummarySort
}

type DepartmentRiskEventEntry struct {
	DetailRoute    string
	DetailAPIPath  string
	DepartmentId   *int
	DepartmentName string
	From           int64
	To             int64
	UnassignedOnly bool
}

type DepartmentRiskSummaryItem struct {
	DeptId            *int
	DeptName          string
	IsUnassigned      bool
	WindowStart       int64
	WindowEnd         int64
	RiskEventCount    int64
	TotalRequestCount int64
	RiskRate          float64
	EventEntry        DepartmentRiskEventEntry
}

type DepartmentRiskTrendPoint struct {
	WindowStart                 int64
	WindowEnd                   int64
	RiskEventCount              int64
	TotalRequestCount           int64
	RiskRate                    float64
	UnassignedRiskEventCount    int64
	UnassignedTotalRequestCount int64
}

type DepartmentRiskFormula struct {
	Expression       string
	NumeratorLabel   string
	DenominatorLabel string
}

type DepartmentRiskSummaryResult struct {
	Items          []DepartmentRiskSummaryItem
	TopDepartments []DepartmentRiskSummaryItem
	Trend          []DepartmentRiskTrendPoint
	Unassigned     DepartmentRiskSummaryItem
	Formula        DepartmentRiskFormula
	DisclaimerKey  string
}

type parsedAlertEvent struct {
	event       entmodel.AlertEvent
	departments []entmodel.AlertEventDepartmentSnapshot
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

type AlertDeliveryQuery struct {
	TenantId       int
	RuleId         *int
	EventId        *int
	ManualParentId *int
	ChannelType    string
	Status         string
	TriggerSource  string
	Page           int
	PageSize       int
}

type AlertDeliveryTraceItem struct {
	EventId            int
	RequestId          string
	TenantId           int
	UserId             int
	Username           string
	DisplayName        string
	ModelName          string
	RiskType           string
	ActionResult       string
	EventCreatedAt     int64
	DepartmentSnapshot []entmodel.AlertEventDepartmentSnapshot
	DepartmentSummary  string
	EventSummary       string
	RuleId             int
	RuleName           string
	DetailRoute        string
	DetailAPIPath      string
}

type AlertDeliveryItem struct {
	Id             int
	TenantId       int
	EventId        int
	RuleId         int
	ChannelType    string
	Status         string
	AttemptCount   int
	MaxAttempts    int
	NextRetryAt    int64
	LastAttemptAt  int64
	SentAt         int64
	FinalFailedAt  int64
	ErrorReason    string
	DedupeKey      string
	TriggerSource  string
	ManualParentId *int
	TraceSummary   string
	CreatedAt      int64
	UpdatedAt      int64
	Trace          *AlertDeliveryTraceItem
}

type AlertDeliveryListResult struct {
	Items    []AlertDeliveryItem
	Total    int
	Page     int
	PageSize int
}

type AlertDeliveryResendResult struct {
	Item         AlertDeliveryItem
	Created      bool
	AuditSummary string
	AuditPayload map[string]any
}

type alertMatchedRule struct {
	Rule     entmodel.AlertRule
	Item     AlertRuleItem
	Channels []entmodel.AlertRuleChannelConfig
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
	if query.EventId != nil && *query.EventId <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.DepartmentId != nil && *query.DepartmentId <= 0 {
		return AlertEventListResult{}, ErrInvalidAlertEventQuery
	}
	if query.UnassignedOnly != nil && *query.UnassignedOnly && query.DepartmentId != nil {
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

	userIDs := make([]int, 0, len(events))
	seenUserIDs := make(map[int]struct{}, len(events))
	for _, event := range events {
		if event.UserId <= 0 {
			continue
		}
		if _, ok := seenUserIDs[event.UserId]; ok {
			continue
		}
		seenUserIDs[event.UserId] = struct{}{}
		userIDs = append(userIDs, event.UserId)
	}
	currentUsers, err := loadCurrentUserIdentities(s.db, userIDs)
	if err != nil {
		return AlertEventListResult{}, err
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
			Username:           firstNonEmpty(currentUsers[event.UserId].Username, event.Username),
			DisplayName:        currentUsers[event.UserId].DisplayName,
			UsernameSnapshot:   event.Username,
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

func (s *AlertService) GetDepartmentRiskSummary(query DepartmentRiskSummaryQuery) (DepartmentRiskSummaryResult, error) {
	if s == nil || s.db == nil {
		return DepartmentRiskSummaryResult{
			Items:          []DepartmentRiskSummaryItem{},
			TopDepartments: []DepartmentRiskSummaryItem{},
			Trend:          []DepartmentRiskTrendPoint{},
			Unassigned:     buildDepartmentRiskSummaryItem(nil, "", query.From, query.To, 0, 0, query.TenantId),
			Formula:        defaultDepartmentRiskFormula(),
			DisclaimerKey:  "enterprise.usage.multi_dept_disclaimer",
		}, nil
	}
	if query.TenantId < 0 || query.From <= 0 || query.To <= 0 || query.From >= query.To {
		return DepartmentRiskSummaryResult{}, ErrInvalidDepartmentRiskSummaryQuery
	}

	usageSummary, err := NewUsageAggregationService(s.db).GetDepartmentSummary(UsageSummaryQuery{
		TenantId: query.TenantId,
		From:     query.From,
		To:       query.To,
		Sort:     query.Sort,
	})
	if err != nil {
		return DepartmentRiskSummaryResult{}, err
	}

	type eventAggregate struct {
		deptId         *int
		deptName       string
		riskEventCount int64
	}

	var events []entmodel.AlertEvent
	if err := s.db.
		Where("tenant_id = ? AND created_at >= ? AND created_at < ?", query.TenantId, query.From, query.To).
		Order("created_at ASC, id ASC").
		Find(&events).Error; err != nil {
		return DepartmentRiskSummaryResult{}, err
	}

	parsedEvents := make([]parsedAlertEvent, 0, len(events))
	riskCounts := make(map[string]*eventAggregate)
	for _, event := range events {
		snapshot, err := event.ParsedDepartmentSnapshot()
		if err != nil {
			return DepartmentRiskSummaryResult{}, err
		}
		parsedEvents = append(parsedEvents, parsedAlertEvent{
			event:       event,
			departments: snapshot,
		})
		if len(snapshot) == 0 {
			key := usageBucketKey(nil)
			aggregate, ok := riskCounts[key]
			if !ok {
				aggregate = &eventAggregate{deptId: nil, deptName: entmodel.UsageSnapshotUnassignedDeptName}
				riskCounts[key] = aggregate
			}
			aggregate.riskEventCount++
			continue
		}
		seen := make(map[int]struct{}, len(snapshot))
		for _, dept := range snapshot {
			if dept.DepartmentId <= 0 {
				continue
			}
			if _, ok := seen[dept.DepartmentId]; ok {
				continue
			}
			seen[dept.DepartmentId] = struct{}{}
			deptId := dept.DepartmentId
			key := usageBucketKey(&deptId)
			aggregate, ok := riskCounts[key]
			if !ok {
				aggregate = &eventAggregate{deptId: &deptId, deptName: normalizeUsageDeptName(&deptId, dept.DepartmentName)}
				riskCounts[key] = aggregate
			}
			aggregate.riskEventCount++
		}
	}

	usageByKey := make(map[string]UsageDepartmentSummaryItem, len(usageSummary.Items))
	for _, item := range usageSummary.Items {
		usageByKey[usageBucketKey(item.DeptId)] = item
	}
	if _, ok := usageByKey[usageBucketKey(nil)]; !ok {
		usageByKey[usageBucketKey(nil)] = UsageDepartmentSummaryItem{
			DeptId:      nil,
			DeptName:    entmodel.UsageSnapshotUnassignedDeptName,
			WindowStart: query.From,
			WindowEnd:   query.To,
		}
	}
	for key, aggregate := range riskCounts {
		if _, ok := usageByKey[key]; ok {
			continue
		}
		usageByKey[key] = UsageDepartmentSummaryItem{
			DeptId:      aggregate.deptId,
			DeptName:    aggregate.deptName,
			WindowStart: query.From,
			WindowEnd:   query.To,
		}
	}

	var snapshots []entmodel.UsageSnapshot
	if err := s.db.
		Where("tenant_id = ? AND window_start >= ? AND window_end <= ?", query.TenantId, query.From, query.To).
		Order("window_start ASC, id ASC").
		Find(&snapshots).Error; err != nil {
		return DepartmentRiskSummaryResult{}, err
	}
	items := make([]DepartmentRiskSummaryItem, 0, len(usageByKey))
	for key, usageItem := range usageByKey {
		riskEventCount := int64(0)
		if aggregate, ok := riskCounts[key]; ok {
			riskEventCount = aggregate.riskEventCount
		}
		item := buildDepartmentRiskSummaryItem(
			usageItem.DeptId,
			normalizeUsageDeptName(usageItem.DeptId, usageItem.DeptName),
			query.From,
			query.To,
			riskEventCount,
			usageItem.RequestCount,
			query.TenantId,
		)
		items = append(items, item)
	}
	items = sortDepartmentRiskSummaryItems(items, query.Sort)
	if items == nil {
		items = []DepartmentRiskSummaryItem{}
	}

	topDepartments := make([]DepartmentRiskSummaryItem, 0, len(items))
	for _, item := range items {
		if item.IsUnassigned {
			continue
		}
		topDepartments = append(topDepartments, item)
		if len(topDepartments) == 5 {
			break
		}
	}
	if topDepartments == nil {
		topDepartments = []DepartmentRiskSummaryItem{}
	}

	type trendAggregate struct {
		windowStart                 int64
		windowEnd                   int64
		riskEventCount              int64
		totalRequestCount           int64
		unassignedRiskEventCount    int64
		unassignedTotalRequestCount int64
	}
	trendByWindow := make(map[string]*trendAggregate)
	trendWindowOrder := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		pointKey := fmt.Sprintf("%d:%d", snapshot.WindowStart, snapshot.WindowEnd)
		point, ok := trendByWindow[pointKey]
		if !ok {
			point = &trendAggregate{
				windowStart: snapshot.WindowStart,
				windowEnd:   snapshot.WindowEnd,
			}
			trendByWindow[pointKey] = point
			trendWindowOrder = append(trendWindowOrder, pointKey)
		}
		point.totalRequestCount += snapshot.RequestCount
		if snapshot.DeptId == nil {
			point.unassignedTotalRequestCount += snapshot.RequestCount
		}
	}
	for _, parsedEvent := range parsedEvents {
		for _, pointKey := range trendWindowOrder {
			point := trendByWindow[pointKey]
			if point == nil {
				continue
			}
			if parsedEvent.event.CreatedAt < point.windowStart || parsedEvent.event.CreatedAt >= point.windowEnd {
				continue
			}
			if len(parsedEvent.departments) == 0 {
				point.unassignedRiskEventCount++
				point.riskEventCount++
				break
			}
			seen := make(map[int]struct{}, len(parsedEvent.departments))
			for _, dept := range parsedEvent.departments {
				if dept.DepartmentId <= 0 {
					continue
				}
				if _, ok := seen[dept.DepartmentId]; ok {
					continue
				}
				seen[dept.DepartmentId] = struct{}{}
				point.riskEventCount++
			}
			break
		}
	}
	trend := make([]DepartmentRiskTrendPoint, 0, len(trendByWindow))
	for _, pointKey := range trendWindowOrder {
		point := trendByWindow[pointKey]
		if point == nil {
			continue
		}
		if len(trend) > 0 {
			last := trend[len(trend)-1]
			if last.WindowStart == point.windowStart && last.WindowEnd == point.windowEnd {
				continue
			}
		}
		trend = append(trend, DepartmentRiskTrendPoint{
			WindowStart:                 point.windowStart,
			WindowEnd:                   point.windowEnd,
			RiskEventCount:              point.riskEventCount,
			TotalRequestCount:           point.totalRequestCount,
			RiskRate:                    calculateDepartmentRiskRate(point.riskEventCount, point.totalRequestCount),
			UnassignedRiskEventCount:    point.unassignedRiskEventCount,
			UnassignedTotalRequestCount: point.unassignedTotalRequestCount,
		})
	}
	if trend == nil {
		trend = []DepartmentRiskTrendPoint{}
	}

	unassigned := buildDepartmentRiskSummaryItem(nil, entmodel.UsageSnapshotUnassignedDeptName, query.From, query.To, 0, 0, query.TenantId)
	for _, item := range items {
		if item.IsUnassigned {
			unassigned = item
			break
		}
	}

	return DepartmentRiskSummaryResult{
		Items:          items,
		TopDepartments: topDepartments,
		Trend:          trend,
		Unassigned:     unassigned,
		Formula:        defaultDepartmentRiskFormula(),
		DisclaimerKey:  "enterprise.usage.multi_dept_disclaimer",
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

func (s *AlertService) ListAlertDeliveries(query AlertDeliveryQuery) (AlertDeliveryListResult, error) {
	if s == nil || s.db == nil {
		return AlertDeliveryListResult{Items: []AlertDeliveryItem{}}, nil
	}
	if query.TenantId < 0 {
		return AlertDeliveryListResult{}, ErrInvalidAlertDeliveryQuery
	}
	if query.RuleId != nil && *query.RuleId <= 0 {
		return AlertDeliveryListResult{}, ErrInvalidAlertDeliveryQuery
	}
	if query.EventId != nil && *query.EventId <= 0 {
		return AlertDeliveryListResult{}, ErrInvalidAlertDeliveryQuery
	}
	if query.ManualParentId != nil && *query.ManualParentId <= 0 {
		return AlertDeliveryListResult{}, ErrInvalidAlertDeliveryQuery
	}
	page, pageSize := normalizeAlertEventPage(query.Page, query.PageSize)

	db := s.db.Model(&entmodel.AlertDelivery{}).Where("tenant_id = ?", query.TenantId)
	if query.RuleId != nil {
		db = db.Where("rule_id = ?", *query.RuleId)
	}
	if query.EventId != nil {
		db = db.Where("event_id = ?", *query.EventId)
	}
	if query.ManualParentId != nil {
		db = db.Where("manual_parent_id = ?", *query.ManualParentId)
	}
	if channelType := normalizeAlertRuleChannelType(query.ChannelType); channelType != "" {
		db = db.Where("channel_type = ?", channelType)
	} else if strings.TrimSpace(query.ChannelType) != "" {
		return AlertDeliveryListResult{}, ErrInvalidAlertDeliveryQuery
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		switch status {
		case entmodel.AlertDeliveryStatusPending,
			entmodel.AlertDeliveryStatusSent,
			entmodel.AlertDeliveryStatusFailed,
			entmodel.AlertDeliveryStatusFinalFailed,
			entmodel.AlertDeliveryStatusResent:
			db = db.Where("status = ?", status)
		default:
			return AlertDeliveryListResult{}, ErrInvalidAlertDeliveryQuery
		}
	}
	if triggerSource := strings.TrimSpace(query.TriggerSource); triggerSource != "" {
		switch triggerSource {
		case entmodel.AlertDeliveryTriggerRuleMatch, entmodel.AlertDeliveryTriggerManual:
			db = db.Where("trigger_source = ?", triggerSource)
		default:
			return AlertDeliveryListResult{}, ErrInvalidAlertDeliveryQuery
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return AlertDeliveryListResult{}, err
	}

	var deliveries []entmodel.AlertDelivery
	if err := db.Order("created_at DESC, id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&deliveries).Error; err != nil {
		return AlertDeliveryListResult{}, err
	}

	items := make([]AlertDeliveryItem, 0, len(deliveries))
	for _, delivery := range deliveries {
		item, err := mapAlertDeliveryItem(delivery)
		if err != nil {
			return AlertDeliveryListResult{}, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []AlertDeliveryItem{}
	}
	return AlertDeliveryListResult{
		Items:    items,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AlertService) ResendAlertDelivery(tenantId int, deliveryId int, actorId int) (AlertDeliveryResendResult, error) {
	if s == nil || s.db == nil {
		return AlertDeliveryResendResult{}, ErrAlertDeliveryNotFound
	}
	if tenantId < 0 || deliveryId <= 0 || actorId <= 0 {
		return AlertDeliveryResendResult{}, ErrInvalidAlertDeliveryQuery
	}

	var parent entmodel.AlertDelivery
	if err := s.db.Where("tenant_id = ? AND id = ?", tenantId, deliveryId).First(&parent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AlertDeliveryResendResult{}, ErrAlertDeliveryNotFound
		}
		return AlertDeliveryResendResult{}, err
	}
	if parent.Status != entmodel.AlertDeliveryStatusFinalFailed {
		return AlertDeliveryResendResult{}, ErrAlertDeliveryResendNotAllowed
	}

	var existing entmodel.AlertDelivery
	err := s.db.
		Where(
			"tenant_id = ? AND manual_parent_id = ? AND trigger_source = ? AND status IN ?",
			tenantId,
			parent.Id,
			entmodel.AlertDeliveryTriggerManual,
			[]string{entmodel.AlertDeliveryStatusPending, entmodel.AlertDeliveryStatusFailed},
		).
		Order("created_at DESC, id DESC").
		First(&existing).Error
	if err == nil {
		item, mapErr := mapAlertDeliveryItem(existing)
		if mapErr != nil {
			return AlertDeliveryResendResult{}, mapErr
		}
		return AlertDeliveryResendResult{
			Item:         item,
			Created:      false,
			AuditSummary: buildAlertDeliveryResendAuditSummary(parent, existing, false),
			AuditPayload: buildAlertDeliveryResendAuditPayload(parent, existing, actorId, false),
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return AlertDeliveryResendResult{}, err
	}

	round, err := nextManualAlertDeliveryRound(s.db, parent.Id)
	if err != nil {
		return AlertDeliveryResendResult{}, err
	}
	nowUnix := s.now().Unix()
	child := entmodel.AlertDelivery{
		TenantId:       parent.TenantId,
		EventId:        parent.EventId,
		RuleId:         parent.RuleId,
		ChannelType:    parent.ChannelType,
		Status:         entmodel.AlertDeliveryStatusPending,
		AttemptCount:   0,
		MaxAttempts:    parent.MaxAttempts,
		NextRetryAt:    nowUnix,
		LastAttemptAt:  0,
		SentAt:         0,
		FinalFailedAt:  0,
		ErrorReason:    "",
		DedupeKey:      buildManualAlertDeliveryDedupeKey(parent, round),
		TriggerSource:  entmodel.AlertDeliveryTriggerManual,
		ManualParentId: &parent.Id,
		CreatedAt:      nowUnix,
		UpdatedAt:      nowUnix,
		TracePayload:   parent.TracePayload,
	}
	if child.MaxAttempts <= 0 {
		child.MaxAttempts = entmodel.AlertDeliveryDefaultMaxAttempts
	}
	tracePayload, err := parent.ParsedTracePayload()
	if err != nil {
		return AlertDeliveryResendResult{}, err
	}
	normalizeAlertTracePayloadDetailPaths(tracePayload)
	if err := child.SetTracePayload(tracePayload); err != nil {
		return AlertDeliveryResendResult{}, err
	}
	if err := s.db.Create(&child).Error; err != nil {
		return AlertDeliveryResendResult{}, err
	}

	item, err := mapAlertDeliveryItem(child)
	if err != nil {
		return AlertDeliveryResendResult{}, err
	}
	return AlertDeliveryResendResult{
		Item:         item,
		Created:      true,
		AuditSummary: buildAlertDeliveryResendAuditSummary(parent, child, true),
		AuditPayload: buildAlertDeliveryResendAuditPayload(parent, child, actorId, true),
	}, nil
}

func (s *AlertService) EnqueueAlertDeliveriesForPendingEvents(now time.Time, lookbackWindow time.Duration, batchSize int) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	minCreatedAt := int64(0)
	if lookbackWindow > 0 {
		minCreatedAt = now.Add(-lookbackWindow).Unix()
	}

	db := s.db.Model(&entmodel.AlertEvent{})
	if minCreatedAt > 0 {
		db = db.Where("created_at >= ?", minCreatedAt)
	}

	created := 0
	var events []entmodel.AlertEvent
	err := db.Order("created_at ASC, id ASC").FindInBatches(&events, batchSize, func(tx *gorm.DB, batch int) error {
		for _, event := range events {
			count, err := s.enqueueDeliveriesForEvent(event, now)
			if err != nil {
				return err
			}
			created += count
		}
		return nil
	}).Error
	return created, err
}

func (s *AlertService) enqueueDeliveriesForEvent(event entmodel.AlertEvent, now time.Time) (int, error) {
	matchedRules, err := s.MatchAlertRules(event)
	if err != nil {
		return 0, err
	}
	if len(matchedRules) == 0 {
		return 0, nil
	}

	snapshot, err := event.ParsedDepartmentSnapshot()
	if err != nil {
		return 0, err
	}
	created := 0
	for _, matched := range matchedRules {
		bucketSize := matched.Rule.DedupeWindowSeconds
		if bucketSize <= 0 {
			bucketSize = 300
		}
		for _, channel := range matched.Channels {
			if !channel.Enabled {
				continue
			}
			delivery, createdThis, err := s.buildAlertDeliveryFromMatch(event, snapshot, matched, channel, now.Unix(), bucketSize)
			if err != nil {
				return created, err
			}
			if createdThis {
				if err := s.createAlertDeliveryIfAbsent(delivery); err != nil {
					if isAlertDeliveryDuplicateError(err) {
						continue
					}
					return created, err
				}
				created++
			}
		}
	}
	return created, nil
}

func (s *AlertService) MatchAlertRules(event entmodel.AlertEvent) ([]alertMatchedRule, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	snapshot, err := event.ParsedDepartmentSnapshot()
	if err != nil {
		return nil, err
	}
	departmentSet := make(map[int]struct{}, len(snapshot))
	for _, department := range snapshot {
		if department.DepartmentId > 0 {
			departmentSet[department.DepartmentId] = struct{}{}
		}
	}

	var rules []entmodel.AlertRule
	if err := s.db.Where("tenant_id = ? AND enabled = ?", event.TenantId, true).
		Order("updated_at DESC, id DESC").
		Find(&rules).Error; err != nil {
		return nil, err
	}

	result := make([]alertMatchedRule, 0, len(rules))
	for _, rule := range rules {
		riskTypes, err := rule.ParsedRiskTypes()
		if err != nil {
			return nil, err
		}
		if !containsExactString(riskTypes, event.RiskType) {
			continue
		}
		departmentIds, err := rule.ParsedDepartmentIds()
		if err != nil {
			return nil, err
		}
		if len(departmentIds) > 0 && !ruleMatchesAnyDepartment(departmentIds, departmentSet) {
			continue
		}
		item, err := mapAlertRuleItem(rule)
		if err != nil {
			return nil, err
		}
		channels, err := rule.ParsedChannelConfigs()
		if err != nil {
			return nil, err
		}
		result = append(result, alertMatchedRule{
			Rule:     rule,
			Item:     item,
			Channels: channels,
		})
	}
	return result, nil
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
	if query.EventId != nil {
		db = db.Where("id = ?", *query.EventId)
	}
	if query.UnassignedOnly != nil && *query.UnassignedOnly {
		db = db.Where("(department_tokens = ? OR department_tokens = '' OR department_tokens IS NULL)", "|")
	}
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
		userIDs, err := lookupUserIDsByUsername(db, username)
		if err == nil && len(userIDs) > 0 {
			db = db.Where("(username = ? OR user_id IN ?)", username, userIDs)
		} else {
			db = db.Where("username = ?", username)
		}
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
		db = db.Where("created_at < ?", *query.To)
	}
	return db
}

func buildDepartmentRiskSummaryItem(
	deptId *int,
	deptName string,
	from int64,
	to int64,
	riskEventCount int64,
	totalRequestCount int64,
	tenantId int,
) DepartmentRiskSummaryItem {
	name := normalizeUsageDeptName(deptId, deptName)
	isUnassigned := deptId == nil
	entry := DepartmentRiskEventEntry{
		DepartmentId:   deptId,
		DepartmentName: name,
		From:           from,
		To:             to,
		UnassignedOnly: isUnassigned,
	}
	if isUnassigned {
		entry.DetailRoute = fmt.Sprintf("/enterprise-alerts?tab=events&from=%d&to=%d&unassigned_only=true", from, to)
		entry.DetailAPIPath = fmt.Sprintf("/api/enterprise/alerts/events?from=%d&to=%d&unassigned_only=true", from, to)
	} else {
		entry.DetailRoute = fmt.Sprintf("/enterprise-alerts?tab=events&department_id=%d&from=%d&to=%d", *deptId, from, to)
		entry.DetailAPIPath = fmt.Sprintf("/api/enterprise/alerts/events?department_id=%d&from=%d&to=%d", *deptId, from, to)
	}
	if tenantId > 0 {
		if isUnassigned {
			entry.DetailRoute += fmt.Sprintf("&tenant_id=%d", tenantId)
			entry.DetailAPIPath += fmt.Sprintf("&tenant_id=%d", tenantId)
		} else {
			entry.DetailRoute += fmt.Sprintf("&tenant_id=%d", tenantId)
			entry.DetailAPIPath += fmt.Sprintf("&tenant_id=%d", tenantId)
		}
	}
	return DepartmentRiskSummaryItem{
		DeptId:            deptId,
		DeptName:          name,
		IsUnassigned:      isUnassigned,
		WindowStart:       from,
		WindowEnd:         to,
		RiskEventCount:    riskEventCount,
		TotalRequestCount: totalRequestCount,
		RiskRate:          calculateDepartmentRiskRate(riskEventCount, totalRequestCount),
		EventEntry:        entry,
	}
}

func calculateDepartmentRiskRate(riskEventCount int64, totalRequestCount int64) float64 {
	if totalRequestCount <= 0 {
		return 0
	}
	return float64(riskEventCount) / float64(totalRequestCount)
}

func defaultDepartmentRiskFormula() DepartmentRiskFormula {
	return DepartmentRiskFormula{
		Expression:       "Risk rate = risky requests / total requests",
		NumeratorLabel:   "Requests that triggered risk events",
		DenominatorLabel: "Total requests from department members",
	}
}

func sortDepartmentRiskSummaryItems(items []DepartmentRiskSummaryItem, sortConfig UsageSummarySort) []DepartmentRiskSummaryItem {
	sorted := append([]DepartmentRiskSummaryItem{}, items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareDepartmentRiskSummaryItems(sorted[i], sorted[j], sortConfig) < 0
	})
	return sorted
}

func compareDepartmentRiskSummaryItems(a DepartmentRiskSummaryItem, b DepartmentRiskSummaryItem, sortConfig UsageSummarySort) int {
	compareNumeric := func(left int64, right int64) int {
		if left < right {
			return -1
		}
		if left > right {
			return 1
		}
		return 0
	}
	compareFloat := func(left float64, right float64) int {
		if left < right {
			return -1
		}
		if left > right {
			return 1
		}
		return 0
	}
	compareText := func(left string, right string) int {
		return strings.Compare(strings.ToLower(left), strings.ToLower(right))
	}
	applyOrder := func(value int) int {
		if sortConfig.Order == UsageSortOrderAsc {
			return value
		}
		return -value
	}
	var primary int
	switch sortConfig.Field {
	case UsageSummarySortByQuota:
		primary = compareFloat(a.RiskRate, b.RiskRate)
	case UsageSummarySortByUsers:
		primary = compareNumeric(a.RiskEventCount, b.RiskEventCount)
	case UsageSummarySortByName:
		primary = compareText(a.DeptName, b.DeptName)
	default:
		primary = compareNumeric(a.TotalRequestCount, b.TotalRequestCount)
	}
	if primary != 0 {
		return applyOrder(primary)
	}
	if byRate := compareFloat(a.RiskRate, b.RiskRate); byRate != 0 {
		return -byRate
	}
	if byRiskCount := compareNumeric(a.RiskEventCount, b.RiskEventCount); byRiskCount != 0 {
		return -byRiskCount
	}
	if byTotal := compareNumeric(a.TotalRequestCount, b.TotalRequestCount); byTotal != 0 {
		return -byTotal
	}
	if byName := compareText(a.DeptName, b.DeptName); byName != 0 {
		return byName
	}
	return compareOptionalInt(a.DeptId, b.DeptId)
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
				Enabled:   boolPointer(existingEmail.Enabled),
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

func boolPointer(value bool) *bool {
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

func mapAlertDeliveryItem(delivery entmodel.AlertDelivery) (AlertDeliveryItem, error) {
	tracePayload, err := delivery.ParsedTracePayload()
	if err != nil {
		return AlertDeliveryItem{}, err
	}
	var trace *AlertDeliveryTraceItem
	if tracePayload != nil {
		normalizeAlertTracePayloadDetailPaths(tracePayload)
		trace = &AlertDeliveryTraceItem{
			EventId:            tracePayload.EventId,
			RequestId:          tracePayload.RequestId,
			TenantId:           tracePayload.TenantId,
			UserId:             tracePayload.UserId,
			Username:           tracePayload.Username,
			DisplayName:        tracePayload.DisplayName,
			ModelName:          tracePayload.ModelName,
			RiskType:           tracePayload.RiskType,
			ActionResult:       tracePayload.ActionResult,
			EventCreatedAt:     tracePayload.EventCreatedAt,
			DepartmentSnapshot: append([]entmodel.AlertEventDepartmentSnapshot{}, tracePayload.DepartmentSnapshot...),
			DepartmentSummary:  tracePayload.DepartmentSummary,
			EventSummary:       tracePayload.EventSummary,
			RuleId:             tracePayload.RuleId,
			RuleName:           tracePayload.RuleName,
			DetailRoute:        tracePayload.DetailRoute,
			DetailAPIPath:      tracePayload.DetailAPIPath,
		}
		if trace.DepartmentSnapshot == nil {
			trace.DepartmentSnapshot = []entmodel.AlertEventDepartmentSnapshot{}
		}
	}
	return AlertDeliveryItem{
		Id:             delivery.Id,
		TenantId:       delivery.TenantId,
		EventId:        delivery.EventId,
		RuleId:         delivery.RuleId,
		ChannelType:    delivery.ChannelType,
		Status:         delivery.Status,
		AttemptCount:   delivery.AttemptCount,
		MaxAttempts:    delivery.MaxAttempts,
		NextRetryAt:    delivery.NextRetryAt,
		LastAttemptAt:  delivery.LastAttemptAt,
		SentAt:         delivery.SentAt,
		FinalFailedAt:  delivery.FinalFailedAt,
		ErrorReason:    delivery.ErrorReason,
		DedupeKey:      delivery.DedupeKey,
		TriggerSource:  delivery.TriggerSource,
		ManualParentId: delivery.ManualParentId,
		TraceSummary:   buildAlertDeliveryTraceSummary(trace),
		CreatedAt:      delivery.CreatedAt,
		UpdatedAt:      delivery.UpdatedAt,
		Trace:          trace,
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

func containsExactString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func ruleMatchesAnyDepartment(ruleDepartmentIds []int, eventDepartments map[int]struct{}) bool {
	for _, departmentId := range ruleDepartmentIds {
		if _, ok := eventDepartments[departmentId]; ok {
			return true
		}
	}
	return false
}

func (s *AlertService) buildAlertDeliveryFromMatch(
	event entmodel.AlertEvent,
	snapshot []entmodel.AlertEventDepartmentSnapshot,
	matched alertMatchedRule,
	channel entmodel.AlertRuleChannelConfig,
	now int64,
	bucketSeconds int,
) (entmodel.AlertDelivery, bool, error) {
	bucketStart := now
	if bucketSeconds > 0 {
		bucketStart = (event.CreatedAt / int64(bucketSeconds)) * int64(bucketSeconds)
	}
	tracePayload := &entmodel.AlertDeliveryTracePayload{
		EventId:            event.Id,
		RequestId:          event.RequestId,
		TenantId:           event.TenantId,
		UserId:             event.UserId,
		Username:           event.Username,
		DisplayName:        "",
		ModelName:          event.ModelName,
		RiskType:           event.RiskType,
		ActionResult:       event.ActionResult,
		EventCreatedAt:     event.CreatedAt,
		DepartmentSnapshot: append([]entmodel.AlertEventDepartmentSnapshot{}, snapshot...),
		DepartmentSummary:  summarizeAlertDepartments(snapshot),
		EventSummary:       event.Summary,
		RuleId:             matched.Rule.Id,
		RuleName:           matched.Rule.Name,
		DetailRoute:        buildAlertEventDetailRoute(event),
		DetailAPIPath:      buildAlertEventDetailAPIPath(event),
	}
	if event.UserId > 0 {
		if identity, err := loadCurrentUserIdentities(s.db, []int{event.UserId}); err == nil {
			tracePayload.DisplayName = identity[event.UserId].DisplayName
			tracePayload.Username = firstNonEmpty(identity[event.UserId].Username, tracePayload.Username)
		}
	}

	delivery := entmodel.AlertDelivery{
		TenantId:      event.TenantId,
		EventId:       event.Id,
		RuleId:        matched.Rule.Id,
		ChannelType:   channel.Type,
		Status:        entmodel.AlertDeliveryStatusPending,
		AttemptCount:  0,
		MaxAttempts:   entmodel.AlertDeliveryDefaultMaxAttempts,
		NextRetryAt:   now,
		DedupeKey:     buildAlertDeliveryDedupeKey(event.TenantId, event.Id, matched.Rule.Id, channel.Type, bucketStart),
		TriggerSource: entmodel.AlertDeliveryTriggerRuleMatch,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := delivery.SetTracePayload(tracePayload); err != nil {
		return entmodel.AlertDelivery{}, false, err
	}
	return delivery, true, nil
}

func buildAlertDeliveryDedupeKey(tenantId int, eventId int, ruleId int, channelType string, bucketStart int64) string {
	return strings.Join([]string{
		strconv.Itoa(tenantId),
		strconv.Itoa(eventId),
		strconv.Itoa(ruleId),
		channelType,
		strconv.FormatInt(bucketStart, 10),
	}, ":")
}

func buildAlertEventDetailRoute(event entmodel.AlertEvent) string {
	detailRoute := fmt.Sprintf("/enterprise-alerts?tab=events&event_id=%d", event.Id)
	if event.TenantId > 0 {
		detailRoute += fmt.Sprintf("&tenant_id=%d", event.TenantId)
	}
	return detailRoute
}

func buildAlertEventDetailAPIPath(event entmodel.AlertEvent) string {
	detailAPIPath := fmt.Sprintf("/api/enterprise/alerts/events?event_id=%d&page=1&page_size=20", event.Id)
	if event.TenantId > 0 {
		detailAPIPath += fmt.Sprintf("&tenant_id=%d", event.TenantId)
	}
	return detailAPIPath
}

func normalizeAlertTracePayloadDetailPaths(trace *entmodel.AlertDeliveryTracePayload) {
	if trace == nil || trace.EventId <= 0 {
		return
	}
	event := entmodel.AlertEvent{
		Id:       trace.EventId,
		TenantId: trace.TenantId,
	}
	if strings.TrimSpace(trace.DetailRoute) == "" || strings.Contains(trace.DetailRoute, "/enterprise-alerts?event_id=") {
		trace.DetailRoute = buildAlertEventDetailRoute(event)
	}
	if strings.TrimSpace(trace.DetailAPIPath) == "" || !strings.Contains(trace.DetailAPIPath, "event_id=") {
		trace.DetailAPIPath = buildAlertEventDetailAPIPath(event)
	}
}

func buildManualAlertDeliveryDedupeKey(parent entmodel.AlertDelivery, round int64) string {
	return strings.Join([]string{
		strconv.Itoa(parent.TenantId),
		strconv.Itoa(parent.EventId),
		strconv.Itoa(parent.RuleId),
		parent.ChannelType,
		"manual",
		strconv.Itoa(parent.Id),
		strconv.FormatInt(round, 10),
	}, ":")
}

func nextManualAlertDeliveryRound(db *gorm.DB, parentId int) (int64, error) {
	var count int64
	if err := db.Model(&entmodel.AlertDelivery{}).
		Where("manual_parent_id = ? AND trigger_source = ?", parentId, entmodel.AlertDeliveryTriggerManual).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count + 1, nil
}

func buildAlertDeliveryTraceSummary(trace *AlertDeliveryTraceItem) string {
	if trace == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if strings.TrimSpace(trace.DepartmentSummary) != "" {
		parts = append(parts, trace.DepartmentSummary)
	}
	if strings.TrimSpace(trace.RequestId) != "" {
		parts = append(parts, trace.RequestId)
	}
	if strings.TrimSpace(trace.DetailRoute) != "" {
		parts = append(parts, trace.DetailRoute)
	}
	return strings.Join(parts, " · ")
}

func buildAlertDeliveryResendAuditSummary(parent entmodel.AlertDelivery, child entmodel.AlertDelivery, created bool) string {
	if created {
		return fmt.Sprintf(
			"Manual resend created delivery #%d from final_failed delivery #%d (event #%d, rule #%d, %s)",
			child.Id,
			parent.Id,
			parent.EventId,
			parent.RuleId,
			parent.ChannelType,
		)
	}
	return fmt.Sprintf(
		"Manual resend reused delivery #%d for final_failed delivery #%d (event #%d, rule #%d, %s)",
		child.Id,
		parent.Id,
		parent.EventId,
		parent.RuleId,
		parent.ChannelType,
	)
}

func buildAlertDeliveryResendAuditPayload(parent entmodel.AlertDelivery, child entmodel.AlertDelivery, actorId int, created bool) map[string]any {
	return map[string]any{
		"actor_id":             actorId,
		"original_delivery_id": parent.Id,
		"new_delivery_id":      child.Id,
		"event_id":             parent.EventId,
		"rule_id":              parent.RuleId,
		"channel_type":         parent.ChannelType,
		"trigger_source":       child.TriggerSource,
		"manual_parent_id":     child.ManualParentId,
		"created":              created,
	}
}

func summarizeAlertDepartments(snapshot []entmodel.AlertEventDepartmentSnapshot) string {
	if len(snapshot) == 0 {
		return "Unassigned"
	}
	parts := make([]string, 0, len(snapshot))
	for _, department := range snapshot {
		if department.DepartmentId > 0 {
			parts = append(parts, fmt.Sprintf("%s (#%d)", department.DepartmentName, department.DepartmentId))
			continue
		}
		parts = append(parts, department.DepartmentName)
	}
	return strings.Join(parts, ", ")
}

func (s *AlertService) createAlertDeliveryIfAbsent(delivery entmodel.AlertDelivery) error {
	return s.db.Create(&delivery).Error
}

func isAlertDeliveryDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}
