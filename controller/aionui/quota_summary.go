package aionui

import (
	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
)

func GetQuotaSummary(c *gin.Context) {
	result, err := serviceaionui.BuildQuotaSummary(c.GetInt("aionui_user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, dtoaionui.QuotaSummaryResponse{
		Wallet: dtoaionui.QuotaWalletSummary{
			RemainQuota: result.Wallet.RemainQuota,
			UsedQuota:   result.Wallet.UsedQuota,
			Display:     result.Wallet.Display,
		},
		Subscriptions:         mapQuotaSubscriptionGroups(result.Subscriptions),
		TotalAvailable:        result.TotalAvailable,
		TotalAvailableDisplay: result.TotalAvailableDisplay,
		QuotaApplyURL:         result.QuotaApplyURL,
		RefreshedAt:           result.RefreshedAt,
	})
}

func mapQuotaSubscriptionGroups(groups []serviceaionui.QuotaSubscriptionGroup) []dtoaionui.QuotaSubscriptionGroup {
	out := make([]dtoaionui.QuotaSubscriptionGroup, 0, len(groups))
	for _, group := range groups {
		items := make([]dtoaionui.QuotaSubscriptionItem, 0, len(group.Items))
		for _, item := range group.Items {
			items = append(items, dtoaionui.QuotaSubscriptionItem{
				Id:                     item.Id,
				PlanId:                 item.PlanId,
				AmountTotal:            item.AmountTotal,
				AmountUsed:             item.AmountUsed,
				AmountAvailable:        item.AmountAvailable,
				AmountAvailableDisplay: item.AmountAvailableDisplay,
				EndTime:                item.EndTime,
			})
		}
		out = append(out, dtoaionui.QuotaSubscriptionGroup{
			GroupKey:               group.GroupKey,
			GroupName:              group.GroupName,
			AmountTotal:            group.AmountTotal,
			AmountUsed:             group.AmountUsed,
			AmountAvailable:        group.AmountAvailable,
			AmountAvailableDisplay: group.AmountAvailableDisplay,
			Items:                  items,
		})
	}
	return out
}
