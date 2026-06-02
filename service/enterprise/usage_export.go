package enterprise

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const usageExportDisclaimer = "注意：用量按用户当前所属部门重复计入，部门间数值不可加和"

type UsageSummarySortField string
type UsageSortOrder string

const (
	UsageSummarySortByRequests UsageSummarySortField = "requests"
	UsageSummarySortByQuota    UsageSummarySortField = "quota"
	UsageSummarySortByUsers    UsageSummarySortField = "users"
	UsageSummarySortByName     UsageSummarySortField = "dept_name"

	UsageSortOrderAsc  UsageSortOrder = "asc"
	UsageSortOrderDesc UsageSortOrder = "desc"
)

type UsageSummarySort struct {
	Field UsageSummarySortField
	Order UsageSortOrder
}

type DepartmentUsageExportQuery struct {
	TenantId           int
	DepartmentId       *int
	From               int64
	To                 int64
	Sort               UsageSummarySort
	IncludeDescendants bool
}

type DepartmentUsageExportRow struct {
	DeptId           *int
	DeptName         string
	ParentDepartment string
	WindowStart      int64
	WindowEnd        int64
	RequestCount     int64
	PromptTokens     int64
	CompletionTokens int64
	Quota            int64
	UserCount        int64
}

type DepartmentUsageExportResult struct {
	FileName string
	Rows     []DepartmentUsageExportRow
}

type UsageExportService struct {
	db          *gorm.DB
	aggregation *UsageAggregationService
}

func NewUsageExportService(db *gorm.DB) *UsageExportService {
	if db == nil {
		db = model.DB
	}
	return &UsageExportService{
		db:          db,
		aggregation: NewUsageAggregationService(db),
	}
}

func DefaultUsageSummarySort() UsageSummarySort {
	return UsageSummarySort{
		Field: UsageSummarySortByRequests,
		Order: UsageSortOrderDesc,
	}
}

func NormalizeUsageSummarySort(field string, order string) UsageSummarySort {
	result := DefaultUsageSummarySort()

	switch UsageSummarySortField(strings.TrimSpace(strings.ToLower(field))) {
	case UsageSummarySortByRequests, UsageSummarySortByQuota, UsageSummarySortByUsers, UsageSummarySortByName:
		result.Field = UsageSummarySortField(strings.TrimSpace(strings.ToLower(field)))
	}

	switch UsageSortOrder(strings.TrimSpace(strings.ToLower(order))) {
	case UsageSortOrderAsc, UsageSortOrderDesc:
		result.Order = UsageSortOrder(strings.TrimSpace(strings.ToLower(order)))
	}

	return result
}

func SortUsageDepartmentSummaryItems(items []UsageDepartmentSummaryItem, sortConfig UsageSummarySort) []UsageDepartmentSummaryItem {
	sorted := append([]UsageDepartmentSummaryItem{}, items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareUsageSummaryItems(sorted[i], sorted[j], sortConfig) < 0
	})
	return sorted
}

func compareUsageSummaryItems(a UsageDepartmentSummaryItem, b UsageDepartmentSummaryItem, sortConfig UsageSummarySort) int {
	compareNumeric := func(left int64, right int64) int {
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
		primary = compareNumeric(a.Quota, b.Quota)
	case UsageSummarySortByUsers:
		primary = compareNumeric(a.UserCount, b.UserCount)
	case UsageSummarySortByName:
		primary = compareText(usageSummarySortName(a), usageSummarySortName(b))
	default:
		primary = compareNumeric(a.RequestCount, b.RequestCount)
	}
	if primary != 0 {
		return applyOrder(primary)
	}

	if byRequests := compareNumeric(a.RequestCount, b.RequestCount); byRequests != 0 {
		return -byRequests
	}
	if byQuota := compareNumeric(a.Quota, b.Quota); byQuota != 0 {
		return -byQuota
	}
	if byUsers := compareNumeric(a.UserCount, b.UserCount); byUsers != 0 {
		return -byUsers
	}
	if byName := compareText(usageSummarySortName(a), usageSummarySortName(b)); byName != 0 {
		return byName
	}
	return compareOptionalInt(a.DeptId, b.DeptId)
}

func usageSummarySortName(item UsageDepartmentSummaryItem) string {
	if item.DeptName != "" {
		return item.DeptName
	}
	return "未归属"
}

func compareOptionalInt(left *int, right *int) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if *left < *right {
		return -1
	}
	if *left > *right {
		return 1
	}
	return 0
}

func (s *UsageExportService) ExportDepartmentUsageCSV(query DepartmentUsageExportQuery) (DepartmentUsageExportResult, error) {
	summary, err := s.aggregation.GetDepartmentSummary(UsageSummaryQuery{
		TenantId:           query.TenantId,
		DeptId:             query.DepartmentId,
		From:               query.From,
		To:                 query.To,
		Sort:               query.Sort,
		IncludeDescendants: query.IncludeDescendants,
	})
	if err != nil {
		return DepartmentUsageExportResult{}, err
	}

	parentNames, err := s.loadParentDepartmentNames(query.TenantId, summary.Items)
	if err != nil {
		return DepartmentUsageExportResult{}, err
	}

	rows := make([]DepartmentUsageExportRow, 0, len(summary.Items))
	for _, item := range summary.Items {
		row := DepartmentUsageExportRow{
			DeptId:           item.DeptId,
			DeptName:         item.DeptName,
			ParentDepartment: "",
			WindowStart:      item.WindowStart,
			WindowEnd:        item.WindowEnd,
			RequestCount:     item.RequestCount,
			PromptTokens:     item.PromptTokens,
			CompletionTokens: item.CompletionTokens,
			Quota:            item.Quota,
			UserCount:        item.UserCount,
		}
		if row.DeptId == nil {
			row.DeptName = "未归属"
		}
		if row.DeptId != nil {
			row.ParentDepartment = parentNames[*row.DeptId]
		}
		rows = append(rows, row)
	}

	return DepartmentUsageExportResult{
		FileName: buildDepartmentUsageExportFileName(query.From, query.To),
		Rows:     rows,
	}, nil
}

func (s *UsageExportService) WriteDepartmentUsageCSV(writer io.Writer, result DepartmentUsageExportResult) error {
	csvWriter := csv.NewWriter(writer)

	if err := csvWriter.Write([]string{"# " + usageExportDisclaimer}); err != nil {
		return err
	}
	if err := csvWriter.Write([]string{
		"部门 ID",
		"部门名称",
		"父部门",
		"周期开始",
		"周期结束",
		"请求数",
		"prompt_tokens",
		"completion_tokens",
		"quota",
		"用户数",
	}); err != nil {
		return err
	}

	for _, row := range result.Rows {
		if err := csvWriter.Write([]string{
			formatOptionalInt(row.DeptId),
			row.DeptName,
			row.ParentDepartment,
			formatInt64(row.WindowStart),
			formatInt64(row.WindowEnd),
			formatInt64(row.RequestCount),
			formatInt64(row.PromptTokens),
			formatInt64(row.CompletionTokens),
			formatInt64(row.Quota),
			formatInt64(row.UserCount),
		}); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

func (s *UsageExportService) loadParentDepartmentNames(tenantId int, items []UsageDepartmentSummaryItem) (map[int]string, error) {
	deptIDs := make([]int, 0, len(items))
	for _, item := range items {
		if item.DeptId != nil {
			deptIDs = append(deptIDs, *item.DeptId)
		}
	}
	if len(deptIDs) == 0 {
		return map[int]string{}, nil
	}

	var departments []entmodel.Department
	if err := s.db.
		Select("id", "parent_id").
		Where("tenant_id = ? AND id IN ?", tenantId, deptIDs).
		Find(&departments).Error; err != nil {
		return nil, err
	}

	parentIDs := make([]int, 0)
	parentSeen := map[int]struct{}{}
	deptToParent := make(map[int]*int, len(departments))
	for _, department := range departments {
		deptToParent[department.Id] = department.ParentId
		if department.ParentId != nil {
			if _, ok := parentSeen[*department.ParentId]; !ok {
				parentSeen[*department.ParentId] = struct{}{}
				parentIDs = append(parentIDs, *department.ParentId)
			}
		}
	}
	if len(parentIDs) == 0 {
		return map[int]string{}, nil
	}

	var parentDepartments []entmodel.Department
	if err := s.db.
		Select("id", "name", "status").
		Where("tenant_id = ? AND id IN ?", tenantId, parentIDs).
		Find(&parentDepartments).Error; err != nil {
		return nil, err
	}

	parentNameByID := make(map[int]string, len(parentDepartments))
	for _, parent := range parentDepartments {
		if parent.Status == constant.DepartmentStatusEnabled {
			parentNameByID[parent.Id] = parent.Name
		}
	}

	result := make(map[int]string, len(deptToParent))
	for deptID, parentID := range deptToParent {
		if parentID == nil {
			continue
		}
		if name, ok := parentNameByID[*parentID]; ok {
			result[deptID] = name
		}
	}
	return result, nil
}

func buildDepartmentUsageExportFileName(from int64, to int64) string {
	start := time.Unix(from, 0).UTC().Format("20060102")
	end := time.Unix(to-1, 0).UTC().Format("20060102")
	return fmt.Sprintf("usage-department-%s-%s.csv", start, end)
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func formatInt64(value int64) string {
	return fmt.Sprintf("%d", value)
}
