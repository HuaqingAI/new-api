package enterprise

import (
	"sort"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
)

type UsageDashboardOverviewQuery struct {
	TenantId int
	From     int64
	To       int64
	Sort     UsageSummarySort
}

type UsageDashboardMetrics struct {
	RequestCount             int64
	PromptTokens             int64
	CompletionTokens         int64
	Quota                    int64
	UserCount                int64
	DepartmentCount          int64
	ConsumingDepartmentCount int64
	UnassignedRequestCount   int64
	UnassignedQuota          int64
	DataThrough              int64
}

type UsageDashboardOverviewResult struct {
	Metrics          UsageDashboardMetrics
	Items            []UsageDepartmentSummaryItem
	SecondLevelItems []UsageDepartmentSummaryItem
	Trend            []UsageDepartmentTrendPoint
	IsPartial        bool
}

type UsageDepartmentPeersQuery struct {
	TenantId           int
	DepartmentId       int
	From               int64
	To                 int64
	IncludeDescendants bool
	Sort               UsageSummarySort
}

type UsageDepartmentPeersResult struct {
	ParentDepartmentId   *int
	ParentDepartmentName string
	Items                []UsageDepartmentSummaryItem
	DataThrough          int64
	IsPartial            bool
}

type usageSnapshotAggregate struct {
	requestCount     int64
	promptTokens     int64
	completionTokens int64
	quota            int64
	users            map[int]struct{}
	models           []entmodel.UsageSnapshotModelStat
	windowStart      int64
	windowEnd        int64
}

func (s *UsageAggregationService) GetUsageDashboardOverview(query UsageDashboardOverviewQuery) (UsageDashboardOverviewResult, error) {
	if query.From < 0 || query.To <= 0 || query.From >= query.To {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, ErrInvalidUsageSummaryQuery
	}

	departments, err := s.loadUsageDepartments(query.TenantId)
	if err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}
	rootDepartments := make([]entmodel.Department, 0)
	rootDepartmentIDs := make(map[int]struct{})
	for _, department := range departments {
		if department.ParentId == nil {
			rootDepartments = append(rootDepartments, department)
			rootDepartmentIDs[department.Id] = struct{}{}
		}
	}
	secondLevelDepartments := make([]entmodel.Department, 0)
	for _, department := range departments {
		if department.ParentId == nil {
			continue
		}
		if _, isSecondLevelDepartment := rootDepartmentIDs[*department.ParentId]; isSecondLevelDepartment {
			secondLevelDepartments = append(secondLevelDepartments, department)
		}
	}

	var tenantSnapshots []entmodel.UsageScopeSnapshot
	if err := s.db.
		Where("tenant_id = ? AND scope_type = ? AND window_start >= ? AND window_end <= ?", query.TenantId, entmodel.UsageScopeTypeTenant, query.From, query.To).
		Order("window_start ASC, id ASC").
		Find(&tenantSnapshots).Error; err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}
	tenantAggregate, err := aggregateUsageScopeSnapshots(tenantSnapshots)
	if err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}

	rootIDs := make([]int, 0, len(rootDepartments))
	for _, department := range rootDepartments {
		rootIDs = append(rootIDs, department.Id)
	}
	rootAggregates, dataThrough, err := s.loadUsageScopeAggregates(query.TenantId, rootIDs, query.From, query.To)
	if err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}
	items := make([]UsageDepartmentSummaryItem, 0, len(rootDepartments))
	for _, department := range rootDepartments {
		item := usageDepartmentSummaryItemFromAggregate(department.Id, department.Name, rootAggregates[department.Id])
		items = append(items, item)
	}
	secondLevelIDs := make([]int, 0, len(secondLevelDepartments))
	for _, department := range secondLevelDepartments {
		secondLevelIDs = append(secondLevelIDs, department.Id)
	}
	secondLevelAggregates, secondLevelDataThrough, err := s.loadUsageScopeAggregates(query.TenantId, secondLevelIDs, query.From, query.To)
	if err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}
	if secondLevelDataThrough > dataThrough {
		dataThrough = secondLevelDataThrough
	}
	secondLevelItems := make([]UsageDepartmentSummaryItem, 0, len(secondLevelDepartments))
	for _, department := range secondLevelDepartments {
		secondLevelItems = append(secondLevelItems, usageDepartmentSummaryItemFromAggregate(department.Id, department.Name, secondLevelAggregates[department.Id]))
	}

	var directSnapshots []entmodel.UsageSnapshot
	if err := s.db.
		Where("tenant_id = ? AND dept_id IS NULL AND window_start >= ? AND window_end <= ?", query.TenantId, query.From, query.To).
		Order("window_start ASC, id ASC").
		Find(&directSnapshots).Error; err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}
	unassignedAggregate, err := aggregateUsageSnapshots(directSnapshots)
	if err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}

	consumingDepartments, err := s.countConsumingDirectDepartments(query.TenantId, query.From, query.To)
	if err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}
	if tenantAggregate.windowEnd > dataThrough {
		dataThrough = tenantAggregate.windowEnd
	}
	metrics := UsageDashboardMetrics{
		RequestCount:             tenantAggregate.requestCount,
		PromptTokens:             tenantAggregate.promptTokens,
		CompletionTokens:         tenantAggregate.completionTokens,
		Quota:                    tenantAggregate.quota,
		UserCount:                int64(len(tenantAggregate.users)),
		DepartmentCount:          int64(len(departments)),
		ConsumingDepartmentCount: consumingDepartments,
		UnassignedRequestCount:   unassignedAggregate.requestCount,
		UnassignedQuota:          unassignedAggregate.quota,
		DataThrough:              dataThrough,
	}
	trend, err := usageScopeSnapshotsToTrend(tenantSnapshots)
	if err != nil {
		return UsageDashboardOverviewResult{Items: []UsageDepartmentSummaryItem{}}, err
	}
	return UsageDashboardOverviewResult{
		Metrics:          metrics,
		Items:            SortUsageDepartmentSummaryItems(items, query.Sort),
		SecondLevelItems: SortUsageDepartmentSummaryItems(secondLevelItems, query.Sort),
		Trend:            trend,
		IsPartial:        dataThrough > 0 && dataThrough < query.To,
	}, nil
}

func (s *UsageAggregationService) GetDepartmentPeers(query UsageDepartmentPeersQuery) (UsageDepartmentPeersResult, error) {
	if query.DepartmentId <= 0 || query.From < 0 || query.To <= 0 || query.From >= query.To {
		return UsageDepartmentPeersResult{Items: []UsageDepartmentSummaryItem{}}, ErrInvalidUsageSummaryQuery
	}

	departments, err := s.loadUsageDepartments(query.TenantId)
	if err != nil {
		return UsageDepartmentPeersResult{Items: []UsageDepartmentSummaryItem{}}, err
	}
	departmentByID := make(map[int]entmodel.Department, len(departments))
	for _, department := range departments {
		departmentByID[department.Id] = department
	}
	current, exists := departmentByID[query.DepartmentId]
	if !exists {
		return UsageDepartmentPeersResult{Items: []UsageDepartmentSummaryItem{}}, ErrDepartmentNotFound
	}

	peers := make([]entmodel.Department, 0)
	for _, department := range departments {
		if sameUsageParent(department.ParentId, current.ParentId) {
			peers = append(peers, department)
		}
	}
	peerIDs := make([]int, 0, len(peers))
	for _, department := range peers {
		peerIDs = append(peerIDs, department.Id)
	}

	aggregates := map[int]usageSnapshotAggregate{}
	dataThrough := int64(0)
	if query.IncludeDescendants {
		aggregates, dataThrough, err = s.loadUsageScopeAggregates(query.TenantId, peerIDs, query.From, query.To)
	} else {
		aggregates, dataThrough, err = s.loadDirectUsageAggregates(query.TenantId, peerIDs, query.From, query.To)
	}
	if err != nil {
		return UsageDepartmentPeersResult{Items: []UsageDepartmentSummaryItem{}}, err
	}

	items := make([]UsageDepartmentSummaryItem, 0, len(peers))
	for _, department := range peers {
		items = append(items, usageDepartmentSummaryItemFromAggregate(department.Id, department.Name, aggregates[department.Id]))
	}
	result := UsageDepartmentPeersResult{
		ParentDepartmentId: current.ParentId,
		Items:              SortUsageDepartmentSummaryItems(items, query.Sort),
		DataThrough:        dataThrough,
		IsPartial:          dataThrough > 0 && dataThrough < query.To,
	}
	if current.ParentId != nil {
		result.ParentDepartmentName = departmentByID[*current.ParentId].Name
	}
	return result, nil
}

func (s *UsageAggregationService) loadDepartmentChildrenUsage(query UsageDetailQuery) ([]UsageDepartmentSummaryItem, error) {
	departments, err := s.loadUsageDepartments(query.TenantId)
	if err != nil {
		return nil, err
	}
	children := make([]entmodel.Department, 0)
	for _, department := range departments {
		if sameUsageParent(department.ParentId, query.DeptId) {
			children = append(children, department)
		}
	}
	if len(children) == 0 {
		return []UsageDepartmentSummaryItem{}, nil
	}
	childIDs := make([]int, 0, len(children))
	for _, child := range children {
		childIDs = append(childIDs, child.Id)
	}
	var aggregates map[int]usageSnapshotAggregate
	if query.IncludeDescendants {
		aggregates, _, err = s.loadUsageScopeAggregates(query.TenantId, childIDs, query.From, query.To)
	} else {
		aggregates, _, err = s.loadDirectUsageAggregates(query.TenantId, childIDs, query.From, query.To)
	}
	if err != nil {
		return nil, err
	}
	items := make([]UsageDepartmentSummaryItem, 0, len(children))
	for _, child := range children {
		items = append(items, usageDepartmentSummaryItemFromAggregate(child.Id, child.Name, aggregates[child.Id]))
	}
	return SortUsageDepartmentSummaryItems(items, DefaultUsageSummarySort()), nil
}

func (s *UsageAggregationService) loadUsageDepartments(tenantId int) ([]entmodel.Department, error) {
	var departments []entmodel.Department
	if err := s.db.
		Select("id", "name", "parent_id", "status").
		Where("tenant_id = ?", tenantId).
		Order("id ASC").
		Find(&departments).Error; err != nil {
		return nil, err
	}
	return departments, nil
}

func (s *UsageAggregationService) loadUsageScopeAggregates(tenantId int, departmentIDs []int, from int64, to int64) (map[int]usageSnapshotAggregate, int64, error) {
	if len(departmentIDs) == 0 {
		return map[int]usageSnapshotAggregate{}, 0, nil
	}
	var snapshots []entmodel.UsageScopeSnapshot
	if err := s.db.
		Where("tenant_id = ? AND scope_type = ? AND department_id IN ? AND window_start >= ? AND window_end <= ?", tenantId, entmodel.UsageScopeTypeDepartment, departmentIDs, from, to).
		Order("window_start ASC, id ASC").
		Find(&snapshots).Error; err != nil {
		return nil, 0, err
	}
	byDepartment := make(map[int][]entmodel.UsageScopeSnapshot, len(departmentIDs))
	for _, snapshot := range snapshots {
		if snapshot.DepartmentId != nil {
			byDepartment[*snapshot.DepartmentId] = append(byDepartment[*snapshot.DepartmentId], snapshot)
		}
	}
	result := make(map[int]usageSnapshotAggregate, len(departmentIDs))
	dataThrough := int64(0)
	for _, departmentID := range departmentIDs {
		aggregate, err := aggregateUsageScopeSnapshots(byDepartment[departmentID])
		if err != nil {
			return nil, 0, err
		}
		result[departmentID] = aggregate
		if aggregate.windowEnd > dataThrough {
			dataThrough = aggregate.windowEnd
		}
	}
	return result, dataThrough, nil
}

func (s *UsageAggregationService) loadDirectUsageAggregates(tenantId int, departmentIDs []int, from int64, to int64) (map[int]usageSnapshotAggregate, int64, error) {
	if len(departmentIDs) == 0 {
		return map[int]usageSnapshotAggregate{}, 0, nil
	}
	var snapshots []entmodel.UsageSnapshot
	if err := s.db.
		Where("tenant_id = ? AND dept_id IN ? AND window_start >= ? AND window_end <= ?", tenantId, departmentIDs, from, to).
		Order("window_start ASC, id ASC").
		Find(&snapshots).Error; err != nil {
		return nil, 0, err
	}
	byDepartment := make(map[int][]entmodel.UsageSnapshot, len(departmentIDs))
	for _, snapshot := range snapshots {
		if snapshot.DeptId != nil {
			byDepartment[*snapshot.DeptId] = append(byDepartment[*snapshot.DeptId], snapshot)
		}
	}
	result := make(map[int]usageSnapshotAggregate, len(departmentIDs))
	dataThrough := int64(0)
	for _, departmentID := range departmentIDs {
		aggregate, err := aggregateUsageSnapshots(byDepartment[departmentID])
		if err != nil {
			return nil, 0, err
		}
		result[departmentID] = aggregate
		if aggregate.windowEnd > dataThrough {
			dataThrough = aggregate.windowEnd
		}
	}
	return result, dataThrough, nil
}

func (s *UsageAggregationService) countConsumingDirectDepartments(tenantId int, from int64, to int64) (int64, error) {
	return s.countConsumingDirectDepartmentsInScope(tenantId, nil, from, to)
}

func (s *UsageAggregationService) countConsumingDirectDepartmentsInScope(tenantId int, departmentIDs []int, from int64, to int64) (int64, error) {
	var snapshots []entmodel.UsageSnapshot
	db := s.db.
		Select("dept_id", "request_count").
		Where("tenant_id = ? AND dept_id IS NOT NULL AND window_start >= ? AND window_end <= ?", tenantId, from, to)
	if len(departmentIDs) > 0 {
		db = db.Where("dept_id IN ?", departmentIDs)
	}
	if err := db.Find(&snapshots).Error; err != nil {
		return 0, err
	}
	consuming := map[int]struct{}{}
	for _, snapshot := range snapshots {
		if snapshot.DeptId != nil && snapshot.RequestCount > 0 {
			consuming[*snapshot.DeptId] = struct{}{}
		}
	}
	return int64(len(consuming)), nil
}

func usageScopeSnapshotsToTrend(snapshots []entmodel.UsageScopeSnapshot) ([]UsageDepartmentTrendPoint, error) {
	byWindow := make(map[int64]*UsageDepartmentTrendPoint, len(snapshots))
	for _, snapshot := range snapshots {
		point, exists := byWindow[snapshot.WindowStart]
		if !exists {
			point = &UsageDepartmentTrendPoint{
				WindowStart: snapshot.WindowStart,
				WindowEnd:   snapshot.WindowEnd,
			}
			byWindow[snapshot.WindowStart] = point
		}
		point.RequestCount += snapshot.RequestCount
		point.PromptTokens += snapshot.PromptTokens
		point.CompletionTokens += snapshot.CompletionTokens
		point.Quota += snapshot.Quota
		point.TokenCount = point.PromptTokens + point.CompletionTokens
		point.UserCount = snapshot.UserCount
	}
	trend := make([]UsageDepartmentTrendPoint, 0, len(byWindow))
	for _, point := range byWindow {
		trend = append(trend, *point)
	}
	sort.Slice(trend, func(i, j int) bool {
		return trend[i].WindowStart < trend[j].WindowStart
	})
	return trend, nil
}

func aggregateUsageScopeSnapshots(snapshots []entmodel.UsageScopeSnapshot) (usageSnapshotAggregate, error) {
	result := usageSnapshotAggregate{users: map[int]struct{}{}, models: []entmodel.UsageSnapshotModelStat{}}
	for _, snapshot := range snapshots {
		result.requestCount += snapshot.RequestCount
		result.promptTokens += snapshot.PromptTokens
		result.completionTokens += snapshot.CompletionTokens
		result.quota += snapshot.Quota
		if result.windowStart == 0 || snapshot.WindowStart < result.windowStart {
			result.windowStart = snapshot.WindowStart
		}
		if snapshot.WindowEnd > result.windowEnd {
			result.windowEnd = snapshot.WindowEnd
		}
		stats, err := snapshot.ParsedModelDistribution()
		if err != nil {
			return usageSnapshotAggregate{}, err
		}
		result.models = mergeUsageModelStats(result.models, stats)
		userIDs, err := snapshot.ParsedUserIds()
		if err != nil {
			return usageSnapshotAggregate{}, err
		}
		for _, userID := range userIDs {
			result.users[userID] = struct{}{}
		}
	}
	sort.Slice(result.models, func(i, j int) bool {
		return result.models[i].ModelName < result.models[j].ModelName
	})
	return result, nil
}

func aggregateUsageSnapshots(snapshots []entmodel.UsageSnapshot) (usageSnapshotAggregate, error) {
	result := usageSnapshotAggregate{users: map[int]struct{}{}, models: []entmodel.UsageSnapshotModelStat{}}
	for _, snapshot := range snapshots {
		result.requestCount += snapshot.RequestCount
		result.promptTokens += snapshot.PromptTokens
		result.completionTokens += snapshot.CompletionTokens
		result.quota += snapshot.Quota
		if result.windowStart == 0 || snapshot.WindowStart < result.windowStart {
			result.windowStart = snapshot.WindowStart
		}
		if snapshot.WindowEnd > result.windowEnd {
			result.windowEnd = snapshot.WindowEnd
		}
		stats, err := snapshot.ParsedModelDistribution()
		if err != nil {
			return usageSnapshotAggregate{}, err
		}
		result.models = mergeUsageModelStats(result.models, stats)
		userIDs, err := snapshot.ParsedUserIds()
		if err != nil {
			return usageSnapshotAggregate{}, err
		}
		for _, userID := range userIDs {
			result.users[userID] = struct{}{}
		}
	}
	sort.Slice(result.models, func(i, j int) bool {
		return result.models[i].ModelName < result.models[j].ModelName
	})
	return result, nil
}

func usageDepartmentSummaryItemFromAggregate(departmentID int, departmentName string, aggregate usageSnapshotAggregate) UsageDepartmentSummaryItem {
	return UsageDepartmentSummaryItem{
		DeptId:            intPointer(departmentID),
		DeptName:          departmentName,
		WindowStart:       aggregate.windowStart,
		WindowEnd:         aggregate.windowEnd,
		RequestCount:      aggregate.requestCount,
		PromptTokens:      aggregate.promptTokens,
		CompletionTokens:  aggregate.completionTokens,
		Quota:             aggregate.quota,
		UserCount:         int64(len(aggregate.users)),
		ModelDistribution: aggregate.models,
	}
}

func sameUsageParent(left *int, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
