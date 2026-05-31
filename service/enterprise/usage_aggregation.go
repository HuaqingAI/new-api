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
	TenantId int
	From     int64
	To       int64
	Sort     UsageSummarySort
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
}

type UsageDetailQuery struct {
	TenantId int
	DeptId   *int
	From     int64
	To       int64
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
	RecentLogsLink    UsageRecentLogsLink
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
	return fmt.Sprintf("EnterpriseUsageAggregationWatermark:%d", tenantId)
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
	buckets := aggregateUsageBuckets(logs, memberships, departments)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := deleteUsageSnapshotsForWindow(tx, window); err != nil {
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

		if window.WindowEnd > watermark {
			return s.setWatermarkTx(tx, window.TenantId, window.WindowEnd)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return len(logs), nil
}

func (s *UsageAggregationService) GetDepartmentSummary(query UsageSummaryQuery) (UsageDepartmentSummaryResult, error) {
	if query.From < 0 || query.To <= 0 || query.From >= query.To {
		return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}}, ErrInvalidUsageSummaryQuery
	}

	var snapshots []entmodel.UsageSnapshot
	if err := s.db.
		Where("tenant_id = ? AND window_start >= ? AND window_end <= ?", query.TenantId, query.From, query.To).
		Order("window_start ASC, id ASC").
		Find(&snapshots).Error; err != nil {
		return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}}, err
	}

	grouped := make(map[string]*UsageDepartmentSummaryItem)
	groupedUsers := make(map[string]map[int]struct{})
	for _, snapshot := range snapshots {
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
			return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}}, err
		}
		item.ModelDistribution = mergeUsageModelStats(item.ModelDistribution, stats)

		userIds, err := snapshot.ParsedUserIds()
		if err != nil {
			return UsageDepartmentSummaryResult{Items: []UsageDepartmentSummaryItem{}}, err
		}
		for _, userId := range userIds {
			groupedUsers[key][userId] = struct{}{}
		}
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
	return UsageDepartmentSummaryResult{Items: items}, nil
}

func (s *UsageAggregationService) GetDepartmentDetail(query UsageDetailQuery) (UsageDepartmentDetailResult, error) {
	if query.DeptId == nil || query.From < 0 || query.To <= 0 || query.From >= query.To {
		return UsageDepartmentDetailResult{
			UserRanking:       []UsageDepartmentUserRankItem{},
			ModelDistribution: []entmodel.UsageSnapshotModelStat{},
			Trend:             []UsageDepartmentTrendPoint{},
			RecentLogsLink: UsageRecentLogsLink{
				Path:      "/usage-logs/common",
				Section:   "common",
				Usernames: []string{},
				UserOptions: []UsageRecentLogsUserOption{},
			},
		}, ErrInvalidUsageDetailQuery
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
				Path:      "/usage-logs/common",
				Section:   "common",
				Usernames: []string{},
				UserOptions: []UsageRecentLogsUserOption{},
			},
		}, err
	}

	result := UsageDepartmentDetailResult{
		DeptId:            query.DeptId,
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
	}
	if len(snapshots) == 0 {
		return result, nil
	}

	result.DeptName = normalizeUsageDeptName(query.DeptId, snapshots[0].DeptName)

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
	}
	result.TokenCount = result.PromptTokens + result.CompletionTokens
	result.UserCount = int64(len(allUserIds))
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
	userRanking, usernames, userOptions, err := s.buildDepartmentUserRanking(query, userIds)
	if err != nil {
		return UsageDepartmentDetailResult{}, err
	}
	result.UserRanking = userRanking
	result.RecentLogsLink.DepartmentName = result.DeptName
	result.RecentLogsLink.Usernames = usernames
	result.RecentLogsLink.UserOptions = userOptions
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

func (s *UsageAggregationService) buildDepartmentUserRanking(query UsageDetailQuery, userIds []int) ([]UsageDepartmentUserRankItem, []string, []UsageRecentLogsUserOption, error) {
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

	ranking := make(map[int]*UsageDepartmentUserRankItem, len(userIds))
	for _, logRow := range logs {
		if !usageLogMatchesDepartment(logRow, query.DeptId, memberships[logRow.UserId]) {
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

func usageLogMatchesDepartment(logRow model.Log, deptId *int, memberships []entmodel.UserDepartment) bool {
	if deptId == nil {
		return false
	}
	for _, membership := range memberships {
		if membership.DepartmentId != *deptId {
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
