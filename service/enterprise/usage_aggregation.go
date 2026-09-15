package enterprise

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const usageSnapshotUnassignedBucket = "unassigned"

var (
	ErrInvalidUsageAggregationWindow = errors.New("enterprise usage aggregation window invalid")
	ErrInvalidUsageSummaryQuery      = errors.New("enterprise usage summary query invalid")
	ErrInvalidUsageDetailQuery       = errors.New("enterprise usage detail query invalid")
)

type UsageAggregationWindow struct {
	TenantId    int
	WindowStart int64
	WindowEnd   int64
}

type UsageSummaryQuery struct {
	TenantId           int
	DeptId             *int
	From               int64
	To                 int64
	Sort               UsageSummarySort
	IncludeDescendants bool
}

type UsageDepartmentSummaryItem struct {
	DeptId            *int
	DeptName          string
	WindowStart       int64
	WindowEnd         int64
	RequestCount      int64
	PromptTokens      int64
	CompletionTokens  int64
	Quota             int64
	UserCount         int64
	ModelDistribution []entmodel.UsageSnapshotModelStat
}

type UsageDepartmentSummaryResult struct {
	Items []UsageDepartmentSummaryItem
	Scope DepartmentUsageSummaryScope
}

type DepartmentUsageSummaryScope struct {
	DepartmentId       *int
	DepartmentName     string
	IncludeDescendants bool
	DepartmentIds      []int
	RequestCount       int64
	PromptTokens       int64
	CompletionTokens   int64
	Quota              int64
	UserCount          int64
}

type UsageDetailQuery struct {
	TenantId           int
	DeptId             *int
	From               int64
	To                 int64
	IncludeDescendants bool
}

type UsageDepartmentDetailScope struct {
	DepartmentIds            []int
	IncludeDescendants       bool
	MetricBasis              string
	DepartmentCount          int64
	ConsumingDepartmentCount int64
	DataThrough              int64
	IsPartial                bool
}

type UsageDepartmentUserRankItem struct {
	UserId           int
	Username         string
	DisplayName      string
	RequestCount     int64
	PromptTokens     int64
	CompletionTokens int64
	TokenCount       int64
	Quota            int64
}

type UsageRecentLogsUserOption struct {
	UserId      int
	Username    string
	DisplayName string
}

type UsageDepartmentTrendPoint struct {
	WindowStart      int64
	WindowEnd        int64
	RequestCount     int64
	PromptTokens     int64
	CompletionTokens int64
	TokenCount       int64
	Quota            int64
	UserCount        int64
}

type UsageRecentLogsLink struct {
	Path           string
	Section        string
	DepartmentId   *int
	DepartmentName string
	StartTimestamp int64
	EndTimestamp   int64
	Usernames      []string
	UserOptions    []UsageRecentLogsUserOption
}

type UsageDepartmentDetailResult struct {
	DeptId            *int
	DeptName          string
	WindowStart       int64
	WindowEnd         int64
	RequestCount      int64
	PromptTokens      int64
	CompletionTokens  int64
	TokenCount        int64
	Quota             int64
	UserCount         int64
	UserRanking       []UsageDepartmentUserRankItem
	ModelDistribution []entmodel.UsageSnapshotModelStat
	Trend             []UsageDepartmentTrendPoint
	ChildDepartments  []UsageDepartmentSummaryItem
	RecentLogsLink    UsageRecentLogsLink
	Scope             UsageDepartmentDetailScope
}

type UsageAggregationService struct {
	db    *gorm.DB
	logDB *gorm.DB
}

type usageBucketAggregate struct {
	deptId       *int
	deptName     string
	requestCount int64
	promptTokens int64
	completion   int64
	quota        int64
	users        map[int]struct{}
	models       map[string]*entmodel.UsageSnapshotModelStat
}

type usageScopeBucketAggregate struct {
	scopeKey       string
	scopeType      string
	departmentId   *int
	departmentName string
	requestCount   int64
	promptTokens   int64
	completion     int64
	quota          int64
	users          map[int]struct{}
	models         map[string]*entmodel.UsageSnapshotModelStat
}

func NewUsageAggregationService(db *gorm.DB) *UsageAggregationService {
	if db == nil {
		db = model.DB
	}
	logDB := model.LOG_DB
	if logDB == nil {
		logDB = db
	}
	return &UsageAggregationService{db: db, logDB: logDB}
}

func UsageAggregationWatermarkOptionKey(tenantId int) string {
	// Scope snapshots were introduced after direct snapshots. A versioned
	// watermark rebuilds historical windows so existing installations do not
	// permanently serve an empty enterprise overview after upgrade.
	return fmt.Sprintf("EnterpriseUsageAggregationWatermarkV2:%d", tenantId)
}

func (s *UsageAggregationService) AggregateWindow(window UsageAggregationWindow) (int, error) {
	if window.WindowStart < 0 || window.WindowEnd <= 0 || window.WindowStart >= window.WindowEnd {
		return 0, ErrInvalidUsageAggregationWindow
	}

	watermark, err := s.getWatermark(window.TenantId)
	if err != nil {
		return 0, err
	}
	if watermark >= window.WindowEnd {
		return 0, nil
	}
	if watermark > 0 && watermark < window.WindowStart {
		return 0, ErrInvalidUsageAggregationWindow
	}

	var logs []model.Log
	if err := s.logDB.
		Select("id", "user_id", "username", "model_name", "quota", "prompt_tokens", "completion_tokens", "created_at", "type").
		Where("type = ? AND created_at >= ? AND created_at < ?", model.LogTypeConsume, window.WindowStart, window.WindowEnd).
		Order("created_at ASC, id ASC").
		Find(&logs).Error; err != nil {
		return 0, err
	}

	userIds := uniqueUsageUserIDs(logs)
	memberships, departments, err := s.loadMembershipContext(window.TenantId, userIds, window.WindowStart, window.WindowEnd)
	if err != nil {
		return 0, err
	}
	tenantLogs := filterUsageLogsForTenant(window.TenantId, logs, memberships)
	buckets := aggregateUsageBuckets(tenantLogs, memberships, departments)
	scopeDepartmentNames, ancestorDepartmentIDs, err := s.loadDepartmentAncestors(window.TenantId)
	if err != nil {
		return 0, err
	}
	scopeBuckets := aggregateUsageScopeBuckets(tenantLogs, memberships, scopeDepartmentNames, ancestorDepartmentIDs)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := deleteUsageSnapshotsForWindow(tx, window); err != nil {
			return err
		}
		if err := deleteUsageScopeSnapshotsForWindow(tx, window); err != nil {
			return err
		}
		for _, bucket := range buckets {
			snapshot, err := s.buildSnapshot(window, bucket)
			if err != nil {
				return err
			}
			if err := tx.Create(snapshot).Error; err != nil {
				return err
			}
		}
		for _, bucket := range scopeBuckets {
			snapshot, err := s.buildScopeSnapshot(window, bucket)
			if err != nil {
				return err
			}
			if err := tx.Create(snapshot).Error; err != nil {
				return err
			}
		}

		if window.WindowEnd > watermark {
			return s.setWatermarkTx(tx, window.TenantId, window.WindowEnd)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return len(tenantLogs), nil
}

func (s *UsageAggregationService) GetDepartmentSummary(query UsageSummaryQuery) (UsageDepartmentSummaryResult, error) {
	if query.From < 0 || query.To <= 0 || query.From >= query.To {
		return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}, Scope: DepartmentUsageSummaryScope{DepartmentIds: []int{}}}, ErrInvalidUsageSummaryQuery
	}

	scope := DepartmentUsageSummaryScope{
		DepartmentId:       query.DeptId,
		DepartmentName:     "",
		IncludeDescendants: query.IncludeDescendants,
		DepartmentIds:      []int{},
	}
	var scopeFilter map[int]struct{}
	if query.DeptId != nil {
		resolvedScope, err := ResolveDepartmentScope(s.db, query.TenantId, query.DeptId, query.IncludeDescendants)
		if err != nil {
			return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}, Scope: DepartmentUsageSummaryScope{DepartmentIds: []int{}}}, err
		}
		scope.DepartmentName = resolvedScope.DepartmentName
		scope.DepartmentIds = append([]int{}, resolvedScope.DepartmentIds...)
		scopeFilter = make(map[int]struct{}, len(resolvedScope.DepartmentIds))
		for _, id := range resolvedScope.DepartmentIds {
			scopeFilter[id] = struct{}{}
		}
	}

	var snapshots []entmodel.UsageSnapshot
	if err := s.db.
		Where("tenant_id = ? AND window_start >= ? AND window_end <= ?", query.TenantId, query.From, query.To).
		Order("window_start ASC, id ASC").
		Find(&snapshots).Error; err != nil {
		return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}, Scope: scope}, err
	}

	grouped := make(map[string]*UsageDepartmentSummaryItem)
	groupedUsers := make(map[string]map[int]struct{})
	scopeUsers := map[int]struct{}{}
	for _, snapshot := range snapshots {
		if scopeFilter != nil {
			if snapshot.DeptId == nil {
				continue
			}
			if _, ok := scopeFilter[*snapshot.DeptId]; !ok {
				continue
			}
		}
		key := usageBucketKey(snapshot.DeptId)
		item, ok := grouped[key]
		if !ok {
			item = &UsageDepartmentSummaryItem{
				DeptId:            snapshot.DeptId,
				DeptName:          normalizeUsageDeptName(snapshot.DeptId, snapshot.DeptName),
				WindowStart:       snapshot.WindowStart,
				WindowEnd:         snapshot.WindowEnd,
				ModelDistribution: []entmodel.UsageSnapshotModelStat{},
			}
			grouped[key] = item
			groupedUsers[key] = map[int]struct{}{}
		}
		if snapshot.WindowStart < item.WindowStart || item.WindowStart == 0 {
			item.WindowStart = snapshot.WindowStart
		}
		if snapshot.WindowEnd > item.WindowEnd {
			item.WindowEnd = snapshot.WindowEnd
		}
		item.RequestCount += snapshot.RequestCount
		item.PromptTokens += snapshot.PromptTokens
		item.CompletionTokens += snapshot.CompletionTokens
		item.Quota += snapshot.Quota

		stats, err := snapshot.ParsedModelDistribution()
		if err != nil {
			return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}, Scope: scope}, err
		}
		item.ModelDistribution = mergeUsageModelStats(item.ModelDistribution, stats)

		userIds, err := snapshot.ParsedUserIds()
		if err != nil {
			return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}, Scope: scope}, err
		}
		for _, userId := range userIds {
			groupedUsers[key][userId] = struct{}{}
			scopeUsers[userId] = struct{}{}
		}
		scope.RequestCount += snapshot.RequestCount
		scope.PromptTokens += snapshot.PromptTokens
		scope.CompletionTokens += snapshot.CompletionTokens
		scope.Quota += snapshot.Quota
	}

	items := make([]UsageDepartmentSummaryItem, 0, len(grouped))
	for key, item := range grouped {
		if item.ModelDistribution == nil {
			item.ModelDistribution = []entmodel.UsageSnapshotModelStat{}
		}
		item.UserCount = int64(len(groupedUsers[key]))
		sort.Slice(item.ModelDistribution, func(i, j int) bool {
			return item.ModelDistribution[i].ModelName < item.ModelDistribution[j].ModelName
		})
		items = append(items, *item)
	}
	items = SortUsageDepartmentSummaryItems(items, query.Sort)
	if items == nil {
		items = []UsageDepartmentSummaryItem{}
	}
	scope.UserCount = int64(len(scopeUsers))
	return UsageDepartmentSummaryResult{Items: items, Scope: scope}, nil
}

func (s *UsageAggregationService) GetDepartmentDetail(query UsageDetailQuery) (UsageDepartmentDetailResult, error) {
	if query.DeptId == nil || query.From < 0 || query.To <= 0 || query.From >= query.To {
		return UsageDepartmentDetailResult{
			UserRanking:       []UsageDepartmentUserRankItem{},
			ModelDistribution: []entmodel.UsageSnapshotModelStat{},
			Trend:             []UsageDepartmentTrendPoint{},
			RecentLogsLink: UsageRecentLogsLink{
				Path:        "/usage-logs/common",
				Section:     "common",
				Usernames:   []string{},
				UserOptions: []UsageRecentLogsUserOption{},
			},
		}, ErrInvalidUsageDetailQuery
	}
	resolvedScope := DepartmentScope{
		DepartmentId:       query.DeptId,
		DepartmentIds:      []int{*query.DeptId},
		IncludeDescendants: query.IncludeDescendants,
	}
	resolvedScope, err := ResolveDepartmentScope(s.db, query.TenantId, query.DeptId, query.IncludeDescendants)
	if err != nil {
		return UsageDepartmentDetailResult{}, err
	}
	if query.IncludeDescendants {
		return s.getDepartmentDescendantDetail(query, resolvedScope)
	}

	var snapshots []entmodel.UsageSnapshot
	if err := s.db.
		Where("tenant_id = ? AND dept_id = ? AND window_start >= ? AND window_end <= ?", query.TenantId, *query.DeptId, query.From, query.To).
		Order("window_start ASC, id ASC").
		Find(&snapshots).Error; err != nil {
		return UsageDepartmentDetailResult{
			UserRanking:       []UsageDepartmentUserRankItem{},
			ModelDistribution: []entmodel.UsageSnapshotModelStat{},
			Trend:             []UsageDepartmentTrendPoint{},
			RecentLogsLink: UsageRecentLogsLink{
				Path:        "/usage-logs/common",
				Section:     "common",
				Usernames:   []string{},
				UserOptions: []UsageRecentLogsUserOption{},
			},
		}, err
	}

	result := UsageDepartmentDetailResult{
		DeptId:            query.DeptId,
		DeptName:          resolvedScope.DepartmentName,
		WindowStart:       query.From,
		WindowEnd:         query.To,
		UserRanking:       []UsageDepartmentUserRankItem{},
		ModelDistribution: []entmodel.UsageSnapshotModelStat{},
		Trend:             []UsageDepartmentTrendPoint{},
		RecentLogsLink: UsageRecentLogsLink{
			Path:           "/usage-logs/common",
			Section:        "common",
			DepartmentId:   query.DeptId,
			StartTimestamp: query.From,
			EndTimestamp:   query.To - 1,
			Usernames:      []string{},
			UserOptions:    []UsageRecentLogsUserOption{},
		},
		Scope: UsageDepartmentDetailScope{
			DepartmentIds:      append([]int{}, resolvedScope.DepartmentIds...),
			IncludeDescendants: false,
			MetricBasis:        "direct",
			DepartmentCount:    1,
		},
		ChildDepartments: []UsageDepartmentSummaryItem{},
	}
	if resolvedScope.DepartmentName != "" {
		children, err := s.loadDepartmentChildrenUsage(query)
		if err != nil {
			return UsageDepartmentDetailResult{}, err
		}
		result.ChildDepartments = children
	}
	if len(snapshots) == 0 {
		return result, nil
	}

	if result.DeptName == "" {
		result.DeptName = normalizeUsageDeptName(query.DeptId, snapshots[0].DeptName)
	}

	allUserIds := map[int]struct{}{}
	for _, snapshot := range snapshots {
		result.RequestCount += snapshot.RequestCount
		result.PromptTokens += snapshot.PromptTokens
		result.CompletionTokens += snapshot.CompletionTokens
		result.Quota += snapshot.Quota

		stats, err := snapshot.ParsedModelDistribution()
		if err != nil {
			return UsageDepartmentDetailResult{}, err
		}
		result.ModelDistribution = mergeUsageModelStats(result.ModelDistribution, stats)

		userIds, err := snapshot.ParsedUserIds()
		if err != nil {
			return UsageDepartmentDetailResult{}, err
		}
		for _, userId := range userIds {
			allUserIds[userId] = struct{}{}
		}

		result.Trend = append(result.Trend, UsageDepartmentTrendPoint{
			WindowStart:      snapshot.WindowStart,
			WindowEnd:        snapshot.WindowEnd,
			RequestCount:     snapshot.RequestCount,
			PromptTokens:     snapshot.PromptTokens,
			CompletionTokens: snapshot.CompletionTokens,
			TokenCount:       snapshot.PromptTokens + snapshot.CompletionTokens,
			Quota:            snapshot.Quota,
			UserCount:        snapshot.UserCount,
		})
		if snapshot.WindowEnd > result.Scope.DataThrough {
			result.Scope.DataThrough = snapshot.WindowEnd
		}
	}
	result.TokenCount = result.PromptTokens + result.CompletionTokens
	result.UserCount = int64(len(allUserIds))
	consumingDepartmentCount, err := s.countConsumingDirectDepartmentsInScope(query.TenantId, resolvedScope.DepartmentIds, query.From, query.To)
	if err != nil {
		return UsageDepartmentDetailResult{}, err
	}
	result.Scope.ConsumingDepartmentCount = consumingDepartmentCount
	result.Scope.IsPartial = result.Scope.DataThrough > 0 && result.Scope.DataThrough < query.To
	sort.Slice(result.ModelDistribution, func(i, j int) bool {
		if result.ModelDistribution[i].Quota != result.ModelDistribution[j].Quota {
			return result.ModelDistribution[i].Quota > result.ModelDistribution[j].Quota
		}
		if result.ModelDistribution[i].RequestCount != result.ModelDistribution[j].RequestCount {
			return result.ModelDistribution[i].RequestCount > result.ModelDistribution[j].RequestCount
		}
		return result.ModelDistribution[i].ModelName < result.ModelDistribution[j].ModelName
	})

	userIds := sortedUsageBucketUserIDs(allUserIds)
	userRanking, usernames, userOptions, err := s.buildDepartmentUserRanking(query, userIds, resolvedScope.DepartmentIds)
	if err != nil {
		return UsageDepartmentDetailResult{}, err
	}
	result.UserRanking = userRanking
	result.RecentLogsLink.DepartmentName = result.DeptName
	result.RecentLogsLink.Usernames = usernames
	result.RecentLogsLink.UserOptions = userOptions
	return result, nil
}

func (s *UsageAggregationService) getDepartmentDescendantDetail(query UsageDetailQuery, resolvedScope DepartmentScope) (UsageDepartmentDetailResult, error) {
	var snapshots []entmodel.UsageScopeSnapshot
	if err := s.db.
		Where("tenant_id = ? AND scope_type = ? AND department_id = ? AND window_start >= ? AND window_end <= ?", query.TenantId, entmodel.UsageScopeTypeDepartment, *query.DeptId, query.From, query.To).
		Order("window_start ASC, id ASC").
		Find(&snapshots).Error; err != nil {
		return UsageDepartmentDetailResult{}, err
	}
	if len(snapshots) == 0 {
		var directSnapshots []entmodel.UsageSnapshot
		if err := s.db.
			Where("tenant_id = ? AND dept_id IN ? AND window_start >= ? AND window_end <= ?", query.TenantId, resolvedScope.DepartmentIds, query.From, query.To).
			Order("window_start ASC, id ASC").
			Find(&directSnapshots).Error; err != nil {
			return UsageDepartmentDetailResult{}, err
		}
		convertedSnapshots, convertErr := directSnapshotsToScopeSnapshots(directSnapshots, resolvedScope.DepartmentId, resolvedScope.DepartmentName)
		if convertErr != nil {
			return UsageDepartmentDetailResult{}, convertErr
		}
		snapshots = convertedSnapshots
	}

	result := UsageDepartmentDetailResult{
		DeptId:            query.DeptId,
		DeptName:          resolvedScope.DepartmentName,
		WindowStart:       query.From,
		WindowEnd:         query.To,
		UserRanking:       []UsageDepartmentUserRankItem{},
		ModelDistribution: []entmodel.UsageSnapshotModelStat{},
		Trend:             []UsageDepartmentTrendPoint{},
		RecentLogsLink: UsageRecentLogsLink{
			Path:           "/usage-logs/common",
			Section:        "common",
			DepartmentId:   query.DeptId,
			DepartmentName: resolvedScope.DepartmentName,
			StartTimestamp: query.From,
			EndTimestamp:   query.To - 1,
			Usernames:      []string{},
			UserOptions:    []UsageRecentLogsUserOption{},
		},
		Scope: UsageDepartmentDetailScope{
			DepartmentIds:      append([]int{}, resolvedScope.DepartmentIds...),
			IncludeDescendants: true,
			MetricBasis:        "subtree",
			DepartmentCount:    int64(len(resolvedScope.DepartmentIds)),
		},
		ChildDepartments: []UsageDepartmentSummaryItem{},
	}
	children, err := s.loadDepartmentChildrenUsage(query)
	if err != nil {
		return UsageDepartmentDetailResult{}, err
	}
	result.ChildDepartments = children

	allUserIDs := map[int]struct{}{}
	for _, snapshot := range snapshots {
		result.RequestCount += snapshot.RequestCount
		result.PromptTokens += snapshot.PromptTokens
		result.CompletionTokens += snapshot.CompletionTokens
		result.Quota += snapshot.Quota
		if snapshot.WindowEnd > result.Scope.DataThrough {
			result.Scope.DataThrough = snapshot.WindowEnd
		}
		stats, err := snapshot.ParsedModelDistribution()
		if err != nil {
			return UsageDepartmentDetailResult{}, err
		}
		result.ModelDistribution = mergeUsageModelStats(result.ModelDistribution, stats)
		userIDs, err := snapshot.ParsedUserIds()
		if err != nil {
			return UsageDepartmentDetailResult{}, err
		}
		for _, userID := range userIDs {
			allUserIDs[userID] = struct{}{}
		}
		result.Trend = append(result.Trend, UsageDepartmentTrendPoint{
			WindowStart:      snapshot.WindowStart,
			WindowEnd:        snapshot.WindowEnd,
			RequestCount:     snapshot.RequestCount,
			PromptTokens:     snapshot.PromptTokens,
			CompletionTokens: snapshot.CompletionTokens,
			TokenCount:       snapshot.PromptTokens + snapshot.CompletionTokens,
			Quota:            snapshot.Quota,
			UserCount:        snapshot.UserCount,
		})
	}
	result.TokenCount = result.PromptTokens + result.CompletionTokens
	result.UserCount = int64(len(allUserIDs))
	consumingDepartmentCount, err := s.countConsumingDirectDepartmentsInScope(query.TenantId, resolvedScope.DepartmentIds, query.From, query.To)
	if err != nil {
		return UsageDepartmentDetailResult{}, err
	}
	result.Scope.ConsumingDepartmentCount = consumingDepartmentCount
	result.Scope.IsPartial = result.Scope.DataThrough > 0 && result.Scope.DataThrough < query.To
	sort.Slice(result.ModelDistribution, func(i, j int) bool {
		if result.ModelDistribution[i].Quota != result.ModelDistribution[j].Quota {
			return result.ModelDistribution[i].Quota > result.ModelDistribution[j].Quota
		}
		if result.ModelDistribution[i].RequestCount != result.ModelDistribution[j].RequestCount {
			return result.ModelDistribution[i].RequestCount > result.ModelDistribution[j].RequestCount
		}
		return result.ModelDistribution[i].ModelName < result.ModelDistribution[j].ModelName
	})

	userIDs := sortedUsageBucketUserIDs(allUserIDs)
	userRanking, usernames, userOptions, err := s.buildDepartmentUserRanking(query, userIDs, resolvedScope.DepartmentIds)
	if err != nil {
		return UsageDepartmentDetailResult{}, err
	}
	result.UserRanking = userRanking
	result.RecentLogsLink.Usernames = usernames
	result.RecentLogsLink.UserOptions = userOptions
	return result, nil
}

func directSnapshotsToScopeSnapshots(snapshots []entmodel.UsageSnapshot, departmentID *int, departmentName string) ([]entmodel.UsageScopeSnapshot, error) {
	byWindow := make(map[int64]*entmodel.UsageScopeSnapshot, len(snapshots))
	usersByWindow := make(map[int64]map[int]struct{}, len(snapshots))
	for _, snapshot := range snapshots {
		scopeSnapshot, exists := byWindow[snapshot.WindowStart]
		if !exists {
			scopeSnapshot = &entmodel.UsageScopeSnapshot{
				TenantId:          snapshot.TenantId,
				ScopeKey:          usageScopeDepartmentKey(*departmentID),
				ScopeType:         entmodel.UsageScopeTypeDepartment,
				DepartmentId:      departmentID,
				DepartmentName:    departmentName,
				WindowStart:       snapshot.WindowStart,
				WindowEnd:         snapshot.WindowEnd,
				ModelDistribution: "[]",
			}
			byWindow[snapshot.WindowStart] = scopeSnapshot
			usersByWindow[snapshot.WindowStart] = map[int]struct{}{}
		}
		if snapshot.WindowEnd > scopeSnapshot.WindowEnd {
			scopeSnapshot.WindowEnd = snapshot.WindowEnd
		}
		scopeSnapshot.RequestCount += snapshot.RequestCount
		scopeSnapshot.PromptTokens += snapshot.PromptTokens
		scopeSnapshot.CompletionTokens += snapshot.CompletionTokens
		scopeSnapshot.Quota += snapshot.Quota
		stats, err := snapshot.ParsedModelDistribution()
		if err != nil {
			return nil, err
		}
		mergedStats, err := scopeSnapshot.ParsedModelDistribution()
		if err != nil {
			return nil, err
		}
		if err := scopeSnapshot.SetModelDistribution(mergeUsageModelStats(mergedStats, stats)); err != nil {
			return nil, err
		}
		userIDs, err := snapshot.ParsedUserIds()
		if err != nil {
			return nil, err
		}
		for _, userID := range userIDs {
			usersByWindow[snapshot.WindowStart][userID] = struct{}{}
		}
	}
	result := make([]entmodel.UsageScopeSnapshot, 0, len(byWindow))
	for windowStart, snapshot := range byWindow {
		userIDs := make([]int, 0, len(usersByWindow[windowStart]))
		for userID := range usersByWindow[windowStart] {
			userIDs = append(userIDs, userID)
		}
		sort.Ints(userIDs)
		if err := snapshot.SetUserIds(userIDs); err != nil {
			return nil, err
		}
		result = append(result, *snapshot)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].WindowStart < result[j].WindowStart })
	return result, nil
}

func (s *UsageAggregationService) getWatermark(tenantId int) (int64, error) {
	key := UsageAggregationWatermarkOptionKey(tenantId)

	common.OptionMapRWMutex.RLock()
	if value, ok := common.OptionMap[key]; ok && value != "" {
		common.OptionMapRWMutex.RUnlock()
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	}
	common.OptionMapRWMutex.RUnlock()

	var option model.Option
	if err := s.db.Where(&model.Option{Key: key}).First(&option).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if option.Value == "" {
		return 0, nil
	}
	return strconv.ParseInt(option.Value, 10, 64)
}

func (s *UsageAggregationService) setWatermarkTx(tx *gorm.DB, tenantId int, value int64) error {
	key := UsageAggregationWatermarkOptionKey(tenantId)
	option := model.Option{Key: key}
	if err := tx.FirstOrCreate(&option, model.Option{Key: key}).Error; err != nil {
		return err
	}
	option.Value = strconv.FormatInt(value, 10)
	if err := tx.Save(&option).Error; err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	common.OptionMap[key] = option.Value
	common.OptionMapRWMutex.Unlock()
	return nil
}

func (s *UsageAggregationService) loadMembershipContext(tenantId int, userIds []int, windowStart int64, windowEnd int64) (map[int][]entmodel.UserDepartment, map[int]string, error) {
	if len(userIds) == 0 {
		return map[int][]entmodel.UserDepartment{}, map[int]string{}, nil
	}

	var memberships []entmodel.UserDepartment
	if err := s.db.
		Where("tenant_id = ? AND user_id IN ? AND (joined_at = 0 OR joined_at < ?) AND (left_at = 0 OR left_at >= ?)", tenantId, userIds, windowEnd, windowStart).
		Order("department_id ASC, id ASC").
		Find(&memberships).Error; err != nil {
		return nil, nil, err
	}

	byUser := make(map[int][]entmodel.UserDepartment)
	departmentIds := make([]int, 0)
	seenDepartment := make(map[int]struct{})
	for _, membership := range memberships {
		byUser[membership.UserId] = append(byUser[membership.UserId], membership)
		if _, ok := seenDepartment[membership.DepartmentId]; !ok {
			seenDepartment[membership.DepartmentId] = struct{}{}
			departmentIds = append(departmentIds, membership.DepartmentId)
		}
	}

	departments := make(map[int]string)
	if len(departmentIds) == 0 {
		return byUser, departments, nil
	}

	var rows []entmodel.Department
	if err := s.db.
		Select("id", "name").
		Where("tenant_id = ? AND id IN ?", tenantId, departmentIds).
		Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	for _, department := range rows {
		departments[department.Id] = department.Name
	}
	return byUser, departments, nil
}

func (s *UsageAggregationService) buildSnapshot(window UsageAggregationWindow, bucket *usageBucketAggregate) (*entmodel.UsageSnapshot, error) {
	stats := make([]entmodel.UsageSnapshotModelStat, 0, len(bucket.models))
	for _, stat := range bucket.models {
		stats = append(stats, *stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].ModelName < stats[j].ModelName
	})

	snapshot := &entmodel.UsageSnapshot{
		TenantId:         window.TenantId,
		DeptId:           bucket.deptId,
		DeptName:         normalizeUsageDeptName(bucket.deptId, bucket.deptName),
		WindowStart:      window.WindowStart,
		WindowEnd:        window.WindowEnd,
		RequestCount:     bucket.requestCount,
		PromptTokens:     bucket.promptTokens,
		CompletionTokens: bucket.completion,
		Quota:            bucket.quota,
	}
	if err := snapshot.SetModelDistribution(stats); err != nil {
		return nil, err
	}
	if err := snapshot.SetUserIds(sortedUsageBucketUserIDs(bucket.users)); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (s *UsageAggregationService) loadDepartmentAncestors(tenantId int) (map[int]string, map[int][]int, error) {
	var departments []entmodel.Department
	if err := s.db.
		Select("id", "name", "parent_id").
		Where("tenant_id = ?", tenantId).
		Order("id ASC").
		Find(&departments).Error; err != nil {
		return nil, nil, err
	}

	names := make(map[int]string, len(departments))
	parentByID := make(map[int]*int, len(departments))
	for _, department := range departments {
		names[department.Id] = department.Name
		parentByID[department.Id] = department.ParentId
	}

	ancestors := make(map[int][]int, len(departments))
	for departmentID := range names {
		seen := map[int]struct{}{}
		path := make([]int, 0)
		currentID := departmentID
		for {
			if _, exists := seen[currentID]; exists {
				break
			}
			seen[currentID] = struct{}{}
			path = append(path, currentID)
			parentID, exists := parentByID[currentID]
			if !exists || parentID == nil {
				break
			}
			if _, exists := names[*parentID]; !exists {
				break
			}
			currentID = *parentID
		}
		ancestors[departmentID] = path
	}
	return names, ancestors, nil
}

func aggregateUsageScopeBuckets(
	logs []model.Log,
	memberships map[int][]entmodel.UserDepartment,
	departmentNames map[int]string,
	ancestorDepartmentIDs map[int][]int,
) map[string]*usageScopeBucketAggregate {
	buckets := make(map[string]*usageScopeBucketAggregate)
	for _, logRow := range logs {
		tenantBucket, exists := buckets[entmodel.UsageScopeTypeTenant]
		if !exists {
			tenantBucket = &usageScopeBucketAggregate{
				scopeKey:  entmodel.UsageScopeTypeTenant,
				scopeType: entmodel.UsageScopeTypeTenant,
				users:     map[int]struct{}{},
				models:    map[string]*entmodel.UsageSnapshotModelStat{},
			}
			buckets[tenantBucket.scopeKey] = tenantBucket
		}
		applyUsageLogToScopeBucket(tenantBucket, logRow)

		scopeDepartmentIDs := map[int]struct{}{}
		for _, membership := range memberships[logRow.UserId] {
			if !usageMembershipEffectiveAt(membership, logRow.CreatedAt) {
				continue
			}
			for _, departmentID := range ancestorDepartmentIDs[membership.DepartmentId] {
				scopeDepartmentIDs[departmentID] = struct{}{}
			}
		}
		for departmentID := range scopeDepartmentIDs {
			key := usageScopeDepartmentKey(departmentID)
			bucket, exists := buckets[key]
			if !exists {
				bucket = &usageScopeBucketAggregate{
					scopeKey:       key,
					scopeType:      entmodel.UsageScopeTypeDepartment,
					departmentId:   intPointer(departmentID),
					departmentName: departmentNames[departmentID],
					users:          map[int]struct{}{},
					models:         map[string]*entmodel.UsageSnapshotModelStat{},
				}
				buckets[key] = bucket
			}
			applyUsageLogToScopeBucket(bucket, logRow)
		}
	}
	return buckets
}

func (s *UsageAggregationService) buildScopeSnapshot(window UsageAggregationWindow, bucket *usageScopeBucketAggregate) (*entmodel.UsageScopeSnapshot, error) {
	stats := make([]entmodel.UsageSnapshotModelStat, 0, len(bucket.models))
	for _, stat := range bucket.models {
		stats = append(stats, *stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].ModelName < stats[j].ModelName
	})

	snapshot := &entmodel.UsageScopeSnapshot{
		TenantId:         window.TenantId,
		ScopeKey:         bucket.scopeKey,
		ScopeType:        bucket.scopeType,
		DepartmentId:     bucket.departmentId,
		DepartmentName:   bucket.departmentName,
		WindowStart:      window.WindowStart,
		WindowEnd:        window.WindowEnd,
		RequestCount:     bucket.requestCount,
		PromptTokens:     bucket.promptTokens,
		CompletionTokens: bucket.completion,
		Quota:            bucket.quota,
	}
	if err := snapshot.SetModelDistribution(stats); err != nil {
		return nil, err
	}
	if err := snapshot.SetUserIds(sortedUsageBucketUserIDs(bucket.users)); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func applyUsageLogToScopeBucket(bucket *usageScopeBucketAggregate, logRow model.Log) {
	bucket.requestCount++
	bucket.promptTokens += int64(logRow.PromptTokens)
	bucket.completion += int64(logRow.CompletionTokens)
	bucket.quota += int64(logRow.Quota)
	bucket.users[logRow.UserId] = struct{}{}

	stat, exists := bucket.models[logRow.ModelName]
	if !exists {
		stat = &entmodel.UsageSnapshotModelStat{ModelName: logRow.ModelName}
		bucket.models[logRow.ModelName] = stat
	}
	stat.RequestCount++
	stat.PromptTokens += int64(logRow.PromptTokens)
	stat.CompletionTokens += int64(logRow.CompletionTokens)
	stat.Quota += int64(logRow.Quota)
}

func usageScopeDepartmentKey(departmentID int) string {
	return fmt.Sprintf("department:%d", departmentID)
}

func intPointer(value int) *int {
	return &value
}

func aggregateUsageBuckets(logs []model.Log, memberships map[int][]entmodel.UserDepartment, departments map[int]string) map[string]*usageBucketAggregate {
	buckets := make(map[string]*usageBucketAggregate)
	for _, logRow := range logs {
		matched := false
		for _, membership := range memberships[logRow.UserId] {
			if !usageMembershipEffectiveAt(membership, logRow.CreatedAt) {
				continue
			}
			deptId := membership.DepartmentId
			key := usageBucketKey(&deptId)
			bucket, ok := buckets[key]
			if !ok {
				bucket = &usageBucketAggregate{
					deptId:   &deptId,
					deptName: departments[deptId],
					users:    map[int]struct{}{},
					models:   map[string]*entmodel.UsageSnapshotModelStat{},
				}
				buckets[key] = bucket
			}
			applyUsageLogToBucket(bucket, logRow)
			matched = true
		}
		if matched {
			continue
		}
		bucket, ok := buckets[usageSnapshotUnassignedBucket]
		if !ok {
			bucket = &usageBucketAggregate{
				deptId:   nil,
				deptName: entmodel.UsageSnapshotUnassignedDeptName,
				users:    map[int]struct{}{},
				models:   map[string]*entmodel.UsageSnapshotModelStat{},
			}
			buckets[usageSnapshotUnassignedBucket] = bucket
		}
		applyUsageLogToBucket(bucket, logRow)
	}
	return buckets
}

func applyUsageLogToBucket(bucket *usageBucketAggregate, logRow model.Log) {
	bucket.requestCount++
	bucket.promptTokens += int64(logRow.PromptTokens)
	bucket.completion += int64(logRow.CompletionTokens)
	bucket.quota += int64(logRow.Quota)
	bucket.users[logRow.UserId] = struct{}{}

	stat, ok := bucket.models[logRow.ModelName]
	if !ok {
		stat = &entmodel.UsageSnapshotModelStat{ModelName: logRow.ModelName}
		bucket.models[logRow.ModelName] = stat
	}
	stat.RequestCount++
	stat.PromptTokens += int64(logRow.PromptTokens)
	stat.CompletionTokens += int64(logRow.CompletionTokens)
	stat.Quota += int64(logRow.Quota)
}

func uniqueUsageUserIDs(logs []model.Log) []int {
	seen := make(map[int]struct{})
	ids := make([]int, 0, len(logs))
	for _, logRow := range logs {
		if _, ok := seen[logRow.UserId]; ok {
			continue
		}
		seen[logRow.UserId] = struct{}{}
		ids = append(ids, logRow.UserId)
	}
	return ids
}

func filterUsageLogsForTenant(tenantId int, logs []model.Log, memberships map[int][]entmodel.UserDepartment) []model.Log {
	// Tenant 0 preserves the existing global enterprise behavior, including
	// consumption by users that have no current department assignment. Other
	// tenants have no tenant field on the consume log, so a log belongs to them
	// only when it has an effective membership in that tenant at event time.
	if tenantId == 0 {
		return logs
	}
	filtered := make([]model.Log, 0, len(logs))
	for _, logRow := range logs {
		for _, membership := range memberships[logRow.UserId] {
			if usageMembershipEffectiveAt(membership, logRow.CreatedAt) {
				filtered = append(filtered, logRow)
				break
			}
		}
	}
	return filtered
}

func sortedUsageBucketUserIDs(users map[int]struct{}) []int {
	ids := make([]int, 0, len(users))
	for userId := range users {
		ids = append(ids, userId)
	}
	sort.Ints(ids)
	return ids
}

func usageMembershipEffectiveAt(membership entmodel.UserDepartment, ts int64) bool {
	if membership.Status == constant.EnterpriseMembershipStatusPending {
		return false
	}
	if membership.JoinedAt > 0 && ts < membership.JoinedAt {
		return false
	}
	if membership.LeftAt > 0 && ts > membership.LeftAt {
		return false
	}
	return true
}

func usageBucketKey(deptId *int) string {
	if deptId == nil {
		return usageSnapshotUnassignedBucket
	}
	return fmt.Sprintf("dept:%d", *deptId)
}

func normalizeUsageDeptName(deptId *int, deptName string) string {
	if deptId == nil {
		return entmodel.UsageSnapshotUnassignedDeptName
	}
	return deptName
}

func mergeUsageModelStats(base []entmodel.UsageSnapshotModelStat, extra []entmodel.UsageSnapshotModelStat) []entmodel.UsageSnapshotModelStat {
	byModel := make(map[string]*entmodel.UsageSnapshotModelStat, len(base)+len(extra))
	for i := range base {
		stat := base[i]
		copied := stat
		byModel[stat.ModelName] = &copied
	}
	for _, stat := range extra {
		current, ok := byModel[stat.ModelName]
		if !ok {
			copied := stat
			byModel[stat.ModelName] = &copied
			continue
		}
		current.RequestCount += stat.RequestCount
		current.PromptTokens += stat.PromptTokens
		current.CompletionTokens += stat.CompletionTokens
		current.Quota += stat.Quota
	}

	merged := make([]entmodel.UsageSnapshotModelStat, 0, len(byModel))
	for _, stat := range byModel {
		merged = append(merged, *stat)
	}
	return merged
}

func (s *UsageAggregationService) buildDepartmentUserRanking(query UsageDetailQuery, userIds []int, departmentIDs []int) ([]UsageDepartmentUserRankItem, []string, []UsageRecentLogsUserOption, error) {
	if len(userIds) == 0 {
		return []UsageDepartmentUserRankItem{}, []string{}, []UsageRecentLogsUserOption{}, nil
	}

	var logs []model.Log
	if err := s.logDB.
		Select("user_id", "username", "model_name", "quota", "prompt_tokens", "completion_tokens", "created_at").
		Where("type = ? AND created_at >= ? AND created_at < ? AND user_id IN ?", model.LogTypeConsume, query.From, query.To, userIds).
		Order("created_at ASC, id ASC").
		Find(&logs).Error; err != nil {
		return nil, nil, nil, err
	}

	memberships, _, err := s.loadMembershipContext(query.TenantId, userIds, query.From, query.To)
	if err != nil {
		return nil, nil, nil, err
	}
	currentUsers, err := loadCurrentUserIdentities(s.db, userIds)
	if err != nil {
		return nil, nil, nil, err
	}

	allowedDepartments := make(map[int]struct{}, len(departmentIDs))
	for _, departmentID := range departmentIDs {
		allowedDepartments[departmentID] = struct{}{}
	}
	ranking := make(map[int]*UsageDepartmentUserRankItem, len(userIds))
	for _, logRow := range logs {
		if !usageLogMatchesDepartment(logRow, allowedDepartments, memberships[logRow.UserId]) {
			continue
		}

		item, ok := ranking[logRow.UserId]
		if !ok {
			currentUser := currentUsers[logRow.UserId]
			item = &UsageDepartmentUserRankItem{
				UserId:      logRow.UserId,
				Username:    firstNonEmpty(currentUser.Username, logRow.Username),
				DisplayName: currentUser.DisplayName,
			}
			ranking[logRow.UserId] = item
		}
		if item.Username == "" {
			item.Username = firstNonEmpty(currentUsers[logRow.UserId].Username, logRow.Username)
		}
		if item.DisplayName == "" {
			item.DisplayName = currentUsers[logRow.UserId].DisplayName
		}
		item.RequestCount++
		item.PromptTokens += int64(logRow.PromptTokens)
		item.CompletionTokens += int64(logRow.CompletionTokens)
		item.TokenCount += int64(logRow.PromptTokens + logRow.CompletionTokens)
		item.Quota += int64(logRow.Quota)
	}

	items := make([]UsageDepartmentUserRankItem, 0, len(ranking))
	usernames := make([]string, 0, len(ranking))
	userOptions := make([]UsageRecentLogsUserOption, 0, len(ranking))
	for _, item := range ranking {
		items = append(items, *item)
		if item.Username != "" {
			usernames = append(usernames, item.Username)
			userOptions = append(userOptions, UsageRecentLogsUserOption{
				UserId:      item.UserId,
				Username:    item.Username,
				DisplayName: item.DisplayName,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Quota != items[j].Quota {
			return items[i].Quota > items[j].Quota
		}
		if items[i].RequestCount != items[j].RequestCount {
			return items[i].RequestCount > items[j].RequestCount
		}
		if items[i].TokenCount != items[j].TokenCount {
			return items[i].TokenCount > items[j].TokenCount
		}
		if items[i].DisplayName != items[j].DisplayName {
			return items[i].DisplayName < items[j].DisplayName
		}
		if items[i].Username != items[j].Username {
			return items[i].Username < items[j].Username
		}
		return items[i].UserId < items[j].UserId
	})
	sort.Strings(usernames)
	sort.Slice(userOptions, func(i, j int) bool {
		if userOptions[i].DisplayName != userOptions[j].DisplayName {
			return userOptions[i].DisplayName < userOptions[j].DisplayName
		}
		if userOptions[i].Username != userOptions[j].Username {
			return userOptions[i].Username < userOptions[j].Username
		}
		return userOptions[i].UserId < userOptions[j].UserId
	})
	return items, usernames, userOptions, nil
}

func usageLogMatchesDepartment(logRow model.Log, allowedDepartments map[int]struct{}, memberships []entmodel.UserDepartment) bool {
	for _, membership := range memberships {
		if _, exists := allowedDepartments[membership.DepartmentId]; !exists {
			continue
		}
		if usageMembershipEffectiveAt(membership, logRow.CreatedAt) {
			return true
		}
	}
	return false
}

func deleteUsageSnapshotsForWindow(tx *gorm.DB, window UsageAggregationWindow) error {
	return tx.Where("tenant_id = ? AND window_start = ? AND window_end = ?", window.TenantId, window.WindowStart, window.WindowEnd).
		Delete(&entmodel.UsageSnapshot{}).Error
}

func deleteUsageScopeSnapshotsForWindow(tx *gorm.DB, window UsageAggregationWindow) error {
	return tx.Where("tenant_id = ? AND window_start = ? AND window_end = ?", window.TenantId, window.WindowStart, window.WindowEnd).
		Delete(&entmodel.UsageScopeSnapshot{}).Error
}
