package enterprise

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

const (
	usageTenantUserExportDisclaimer     = "注意：导出内容为全公司成员用户排行"
	usageDepartmentUserExportDisclaimer = "注意：导出内容为当前组织范围内的成员用户排行"
)

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

type UsageUserRankSortField string

const (
	UsageUserRankSortByQuota    UsageUserRankSortField = "quota"
	UsageUserRankSortByRequests UsageUserRankSortField = "requests"
	UsageUserRankSortByTokens   UsageUserRankSortField = "tokens"
)

type DepartmentUsageExportQuery struct {
	TenantId           int
	DepartmentId       *int
	From               int64
	To                 int64
	Sort               UsageUserRankSortField
	IncludeDescendants bool
}

type DepartmentUsageExportRow struct {
	MetricBasis      string
	DeptId           *int
	DeptName         string
	WindowStart      int64
	WindowEnd        int64
	UserId           int
	Username         string
	DisplayName      string
	RequestCount     int64
	PromptTokens     int64
	CompletionTokens int64
	TokenCount       int64
	Quota            int64
}

type DepartmentUsageExportResult struct {
	FileName   string
	Disclaimer string
	Rows       []DepartmentUsageExportRow
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

func NormalizeUsageUserRankSort(field string) UsageUserRankSortField {
	switch UsageUserRankSortField(strings.TrimSpace(strings.ToLower(field))) {
	case UsageUserRankSortByRequests, UsageUserRankSortByTokens:
		return UsageUserRankSortField(strings.TrimSpace(strings.ToLower(field)))
	default:
		return UsageUserRankSortByQuota
	}
}

func SortUsageUserRankItems(items []UsageDepartmentUserRankItem, sortField UsageUserRankSortField) []UsageDepartmentUserRankItem {
	sorted := append([]UsageDepartmentUserRankItem{}, items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareUsageUserRankItems(sorted[i], sorted[j], sortField) < 0
	})
	return sorted
}

func compareUsageUserRankItems(a UsageDepartmentUserRankItem, b UsageDepartmentUserRankItem, sortField UsageUserRankSortField) int {
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
	applyDesc := func(value int) int {
		return -value
	}

	var primary int
	switch sortField {
	case UsageUserRankSortByRequests:
		primary = compareNumeric(a.RequestCount, b.RequestCount)
	case UsageUserRankSortByTokens:
		primary = compareNumeric(a.TokenCount, b.TokenCount)
	default:
		primary = compareNumeric(a.Quota, b.Quota)
	}
	if primary != 0 {
		return applyDesc(primary)
	}

	if byQuota := compareNumeric(a.Quota, b.Quota); byQuota != 0 {
		return -byQuota
	}
	if byRequests := compareNumeric(a.RequestCount, b.RequestCount); byRequests != 0 {
		return -byRequests
	}
	if byTokens := compareNumeric(a.TokenCount, b.TokenCount); byTokens != 0 {
		return -byTokens
	}
	if byDisplayName := compareText(a.DisplayName, b.DisplayName); byDisplayName != 0 {
		return byDisplayName
	}
	if byUsername := compareText(a.Username, b.Username); byUsername != 0 {
		return byUsername
	}
	if a.UserId < b.UserId {
		return -1
	}
	if a.UserId > b.UserId {
		return 1
	}
	return 0
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
	ranking := []UsageDepartmentUserRankItem{}
	metricBasis := "tenant"
	deptName := "全公司"
	disclaimer := usageTenantUserExportDisclaimer
	if query.DepartmentId == nil {
		tenantRanking, err := s.buildTenantUserRanking(query)
		if err != nil {
			return DepartmentUsageExportResult{}, err
		}
		ranking = tenantRanking
	} else {
		detail, err := s.aggregation.GetDepartmentDetail(UsageDetailQuery{
			TenantId:           query.TenantId,
			DeptId:             query.DepartmentId,
			From:               query.From,
			To:                 query.To,
			IncludeDescendants: query.IncludeDescendants,
		})
		if err != nil {
			return DepartmentUsageExportResult{}, err
		}
		metricBasis = detail.Scope.MetricBasis
		deptName = detail.DeptName
		disclaimer = usageDepartmentUserExportDisclaimer
		ranking = detail.UserRanking
	}

	ranking = SortUsageUserRankItems(ranking, query.Sort)
	rows := make([]DepartmentUsageExportRow, 0, len(ranking))
	for _, item := range ranking {
		row := DepartmentUsageExportRow{
			MetricBasis:      metricBasis,
			DeptName:         deptName,
			WindowStart:      query.From,
			WindowEnd:        query.To,
			UserId:           item.UserId,
			Username:         item.Username,
			DisplayName:      item.DisplayName,
			RequestCount:     item.RequestCount,
			PromptTokens:     item.PromptTokens,
			CompletionTokens: item.CompletionTokens,
			TokenCount:       item.TokenCount,
			Quota:            item.Quota,
		}
		if query.DepartmentId != nil {
			deptId := *query.DepartmentId
			row.DeptId = &deptId
		}
		rows = append(rows, row)
	}

	return DepartmentUsageExportResult{
		FileName:   buildDepartmentUsageExportFileName(query.From, query.To),
		Disclaimer: disclaimer,
		Rows:       rows,
	}, nil
}

func (s *UsageExportService) WriteDepartmentUsageCSV(writer io.Writer, result DepartmentUsageExportResult) error {
	csvWriter := csv.NewWriter(writer)

	if err := csvWriter.Write([]string{"# " + result.Disclaimer}); err != nil {
		return err
	}
	if err := csvWriter.Write([]string{
		"部门 ID",
		"部门名称",
		"周期开始",
		"周期结束",
		"用户 ID",
		"用户名",
		"显示名称",
		"请求数",
		"输入 Tokens",
		"输出 Tokens",
		"总 Tokens",
		"额度",
	}); err != nil {
		return err
	}

	for _, row := range result.Rows {
		if err := csvWriter.Write([]string{
			formatOptionalInt(row.DeptId),
			row.DeptName,
			formatUnixDate(row.WindowStart),
			formatUnixDate(row.WindowEnd - 1),
			formatInt(row.UserId),
			row.Username,
			row.DisplayName,
			formatInt64(row.RequestCount),
			formatInt64(row.PromptTokens),
			formatInt64(row.CompletionTokens),
			formatInt64(row.TokenCount),
			formatDisplayQuota(row.Quota),
		}); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

func (s *UsageExportService) buildTenantUserRanking(query DepartmentUsageExportQuery) ([]UsageDepartmentUserRankItem, error) {
	var logs []model.Log
	if err := s.aggregation.logDB.
		Select("user_id", "username", "quota", "prompt_tokens", "completion_tokens", "created_at").
		Where("type = ? AND created_at >= ? AND created_at < ?", model.LogTypeConsume, query.From, query.To).
		Order("created_at ASC, id ASC").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	userIDs := uniqueUsageUserIDs(logs)
	memberships, _, err := s.aggregation.loadMembershipContext(query.TenantId, userIDs, query.From, query.To)
	if err != nil {
		return nil, err
	}
	ranking, _, _, err := buildUsageUserRankingItems(s.db, filterUsageLogsForTenant(query.TenantId, logs, memberships))
	if err != nil {
		return nil, err
	}
	return ranking, nil
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

func formatInt(value int) string {
	return fmt.Sprintf("%d", value)
}

func formatUnixDate(value int64) string {
	return time.Unix(value, 0).Format("2006-01-02")
}

func formatInt64(value int64) string {
	return fmt.Sprintf("%d", value)
}

func formatDisplayQuota(quota int64) string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return formatQuotaTokens(quota)
	}
	if common.QuotaPerUnit <= 0 || math.IsInf(common.QuotaPerUnit, 0) || math.IsNaN(common.QuotaPerUnit) {
		return formatInt64(quota)
	}

	usd := float64(quota) / common.QuotaPerUnit
	value := usd * operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	number := formatQuotaAmount(value)
	symbol := operation_setting.GetCurrencySymbol()
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeCustom {
		return strings.TrimSpace(symbol + " " + number)
	}
	return symbol + number
}

func formatQuotaTokens(quota int64) string {
	if quota == 0 {
		return "0"
	}

	value := float64(quota)
	if math.Abs(value) >= 1000 {
		return formatQuotaDecimal(value/1000, 1) + "k"
	}
	return strconv.FormatInt(quota, 10)
}

func formatQuotaAmount(value float64) string {
	digits := 4
	if math.Abs(value) >= 1 {
		digits = 2
	}
	threshold := math.Pow10(-digits)
	if value != 0 && math.Abs(value) < threshold {
		if value > 0 {
			value = threshold
		} else {
			value = -threshold
		}
	}
	return formatQuotaDecimal(value, digits)
}

func formatQuotaDecimal(value float64, digits int) string {
	formatted := strconv.FormatFloat(value, 'f', digits, 64)
	if strings.Contains(formatted, ".") {
		formatted = strings.TrimRight(formatted, "0")
		formatted = strings.TrimRight(formatted, ".")
	}
	return formatted
}
