package aionui

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

type QuotaWalletSummary struct {
	RemainQuota int    `json:"remain_quota"`
	UsedQuota   int    `json:"used_quota"`
	Display     string `json:"display"`
}

type QuotaSubscriptionItem struct {
	Id                     int    `json:"id"`
	PlanId                 int    `json:"plan_id"`
	AmountTotal            int64  `json:"amount_total"`
	AmountUsed             int64  `json:"amount_used"`
	AmountAvailable        int64  `json:"amount_available"`
	AmountAvailableDisplay string `json:"amount_available_display"`
	EndTime                int64  `json:"end_time"`
}

type QuotaSubscriptionGroup struct {
	GroupKey               string                  `json:"group_key"`
	GroupName              string                  `json:"group_name"`
	AmountTotal            int64                   `json:"amount_total"`
	AmountUsed             int64                   `json:"amount_used"`
	AmountAvailable        int64                   `json:"amount_available"`
	AmountAvailableDisplay string                  `json:"amount_available_display"`
	Items                  []QuotaSubscriptionItem `json:"items"`
}

type QuotaSummary struct {
	Wallet                QuotaWalletSummary       `json:"wallet"`
	Subscriptions         []QuotaSubscriptionGroup `json:"subscriptions"`
	TotalAvailable        int64                    `json:"total_available"`
	TotalAvailableDisplay string                   `json:"total_available_display"`
	QuotaApplyURL         string                   `json:"quota_apply_url"`
	RefreshedAt           int64                    `json:"refreshed_at"`
}

func BuildQuotaSummary(userID int) (QuotaSummary, error) {
	walletRemain, err := model.GetUserQuota(userID, false)
	if err != nil {
		return QuotaSummary{}, err
	}
	walletUsed, err := model.GetUserUsedQuota(userID)
	if err != nil {
		return QuotaSummary{}, err
	}
	summaries, err := model.GetAllActiveUserSubscriptions(userID)
	if err != nil {
		return QuotaSummary{}, err
	}
	groups := []QuotaSubscriptionGroup{
		{GroupKey: SubscriptionGroupEnterprise, GroupName: subscriptionGroupEnterpriseCN, Items: []QuotaSubscriptionItem{}},
		{GroupKey: SubscriptionGroupPersonal, GroupName: subscriptionGroupPersonalCN, Items: []QuotaSubscriptionItem{}},
	}
	groupIndex := map[string]int{
		SubscriptionGroupEnterprise: 0,
		SubscriptionGroupPersonal:   1,
	}
	for _, summary := range summaries {
		if summary.Subscription == nil {
			continue
		}
		sub := summary.Subscription
		key := SubscriptionGroupPersonal
		if isEnterpriseSubscription(*sub) {
			key = SubscriptionGroupEnterprise
		}
		available := sub.AmountTotal - sub.AmountUsed
		if available < 0 {
			available = 0
		}
		index := groupIndex[key]
		groups[index].AmountTotal += sub.AmountTotal
		groups[index].AmountUsed += sub.AmountUsed
		groups[index].AmountAvailable += available
		groups[index].Items = append(groups[index].Items, QuotaSubscriptionItem{
			Id:                     sub.Id,
			PlanId:                 sub.PlanId,
			AmountTotal:            sub.AmountTotal,
			AmountUsed:             sub.AmountUsed,
			AmountAvailable:        available,
			AmountAvailableDisplay: quotaDisplayInt64(available),
			EndTime:                sub.EndTime,
		})
	}
	total := int64(walletRemain)
	for _, group := range groups {
		total += group.AmountAvailable
	}
	for index := range groups {
		groups[index].AmountAvailableDisplay = quotaDisplayInt64(groups[index].AmountAvailable)
	}
	return QuotaSummary{
		Wallet: QuotaWalletSummary{
			RemainQuota: walletRemain,
			UsedQuota:   walletUsed,
			Display:     quotaDisplay(walletRemain),
		},
		Subscriptions:         groups,
		TotalAvailable:        total,
		TotalAvailableDisplay: quotaDisplayInt64(total),
		QuotaApplyURL:         QuotaApplyURL(),
		RefreshedAt:           model.GetDBTimestamp(),
	}, nil
}

func isEnterpriseSubscription(sub model.UserSubscription) bool {
	sourceType := strings.TrimSpace(sub.SourceType)
	source := strings.TrimSpace(sub.Source)
	return sourceType == model.SubscriptionSourceTypeEnterprise ||
		sourceType == model.SubscriptionSourceTypeEnterpriseV0 ||
		source == model.SubscriptionSourceTypeEnterprise ||
		source == model.SubscriptionSourceTypeEnterpriseV0
}

func QuotaApplyURL() string {
	if configured := strings.TrimSpace(os.Getenv("HTH_QUOTA_APPLY_URL")); configured != "" {
		return configured
	}
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_SERVER_URL")), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("BACKEND_BASE_URL")), "/")
	}
	if base == "" {
		return defaultQuotaApplyPath
	}
	if parsed, err := url.Parse(base); err == nil && parsed.Path == "/v1" {
		parsed.Path = ""
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return strings.TrimRight(parsed.String(), "/") + defaultQuotaApplyPath
	}
	return base + defaultQuotaApplyPath
}

func quotaDisplay(quota int) string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return strconv.Itoa(quota)
	}
	usd := float64(quota) / common.QuotaPerUnit
	value := usd * operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	return fmt.Sprintf("%s%.2f", operation_setting.GetCurrencySymbol(), value)
}

func quotaDisplayInt64(quota int64) string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return strconv.FormatInt(quota, 10)
	}
	usd := float64(quota) / common.QuotaPerUnit
	value := usd * operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	return fmt.Sprintf("%s%.2f", operation_setting.GetCurrencySymbol(), value)
}
