package service

import (
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

const (
	usageBillingPathLocal              = "local"
	usageBillingPathUpstream           = "upstream"
	usageBillingPathOpenAI             = "billing-usage-openai"
	usageBillingPathOpenAIEstimated    = "billing-usage-openai-estimated"
	usageBillingPathAnthropic          = "billing-usage-anthropic"
	usageBillingPathAnthropicEstimated = "billing-usage-anthropic-estimated"
	usageBillingPathGemini             = "billing-usage-gemini"
	usageBillingPathGeminiEstimated    = "billing-usage-gemini-estimated"
)

func effectiveBillingUsage(usage *dto.Usage) *dto.Usage {
	if billingUsage, ok := usageFromBillingUsage(usage); ok {
		return billingUsage
	}
	return usage
}

func usageBillingPathForLog(isLocalCountTokens bool, usage *dto.Usage) string {
	effectiveUsage, ok := usageFromBillingUsage(usage)
	if !ok {
		if isLocalCountTokens {
			return usageBillingPathLocal
		}
		return usageBillingPathUpstream
	}

	switch effectiveUsage.UsageSemantic {
	case dto.BillingUsageSemanticOpenAI:
		if usage.BillingUsage.Estimated {
			return usageBillingPathOpenAIEstimated
		}
		return usageBillingPathOpenAI
	case dto.BillingUsageSemanticAnthropic:
		if usage.BillingUsage.Estimated {
			return usageBillingPathAnthropicEstimated
		}
		return usageBillingPathAnthropic
	case dto.BillingUsageSemanticGemini:
		if usage.BillingUsage.Estimated {
			return usageBillingPathGeminiEstimated
		}
		return usageBillingPathGemini
	}

	return usageBillingPathUpstream
}

func appendUsageBillingPathForLog(other map[string]interface{}, isLocalCountTokens bool, usage *dto.Usage) {
	if other == nil {
		return
	}
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = make(map[string]interface{})
		other["admin_info"] = adminInfo
	}
	adminInfo["usage_billing_path"] = usageBillingPathForLog(isLocalCountTokens, usage)
}

func usageFromBillingUsage(usage *dto.Usage) (*dto.Usage, bool) {
	if usage == nil || usage.BillingUsage == nil {
		return nil, false
	}
	billingUsage := usage.BillingUsage
	source := strings.TrimSpace(billingUsage.Source)
	semantic := strings.TrimSpace(billingUsage.Semantic)

	if billingUsage.OpenAIUsage != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceOAIChat) ||
			strings.EqualFold(source, dto.BillingUsageSourceOAIResponses) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticOpenAI)) {
		return usageFromOpenAIBillingUsage(billingUsage), true
	}

	if billingUsage.ClaudeUsage != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceClaudeMessages) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticAnthropic)) {
		return usageFromClaudeBillingUsage(billingUsage), true
	}

	if billingUsage.GeminiUsageMetadata != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceGeminiChat) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticGemini)) {
		return usageFromGeminiBillingUsage(billingUsage), true
	}

	return nil, false
}

func usageFromOpenAIBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	usage := *billingUsage.OpenAIUsage
	usage.PromptTokens = billingTokenCount("openai prompt tokens", usage.PromptTokens)
	usage.CompletionTokens = billingTokenCount("openai completion tokens", usage.CompletionTokens)
	usage.TotalTokens = billingTokenCount("openai total tokens", usage.TotalTokens)
	usage.InputTokens = billingTokenCount("openai input tokens", usage.InputTokens)
	usage.OutputTokens = billingTokenCount("openai output tokens", usage.OutputTokens)
	usage.PromptCacheHitTokens = billingTokenCount("openai prompt cache hit tokens", usage.PromptCacheHitTokens)
	usage.ClaudeCacheCreation5mTokens = billingTokenCount("openai claude cache creation 5m tokens", usage.ClaudeCacheCreation5mTokens)
	usage.ClaudeCacheCreation1hTokens = billingTokenCount("openai claude cache creation 1h tokens", usage.ClaudeCacheCreation1hTokens)
	usage.PromptTokensDetails.CachedTokens = billingTokenCount("openai cached prompt tokens", usage.PromptTokensDetails.CachedTokens)
	usage.PromptTokensDetails.CachedCreationTokens = billingTokenCount("openai cached creation prompt tokens", usage.PromptTokensDetails.CachedCreationTokens)
	usage.PromptTokensDetails.CacheWriteTokens = billingTokenCount("openai cache write prompt tokens", usage.PromptTokensDetails.CacheWriteTokens)
	usage.PromptTokensDetails.TextTokens = billingTokenCount("openai text prompt tokens", usage.PromptTokensDetails.TextTokens)
	usage.PromptTokensDetails.AudioTokens = billingTokenCount("openai audio prompt tokens", usage.PromptTokensDetails.AudioTokens)
	usage.PromptTokensDetails.ImageTokens = billingTokenCount("openai image prompt tokens", usage.PromptTokensDetails.ImageTokens)
	usage.CompletionTokenDetails.TextTokens = billingTokenCount("openai text completion tokens", usage.CompletionTokenDetails.TextTokens)
	usage.CompletionTokenDetails.AudioTokens = billingTokenCount("openai audio completion tokens", usage.CompletionTokenDetails.AudioTokens)
	usage.CompletionTokenDetails.ImageTokens = billingTokenCount("openai image completion tokens", usage.CompletionTokenDetails.ImageTokens)
	usage.CompletionTokenDetails.ReasoningTokens = billingTokenCount("openai reasoning tokens", usage.CompletionTokenDetails.ReasoningTokens)
	if usage.InputTokensDetails != nil {
		inputTokensDetails := *usage.InputTokensDetails
		inputTokensDetails.CachedTokens = billingTokenCount("openai cached input tokens", inputTokensDetails.CachedTokens)
		inputTokensDetails.CachedCreationTokens = billingTokenCount("openai cached creation input tokens", inputTokensDetails.CachedCreationTokens)
		inputTokensDetails.CacheWriteTokens = billingTokenCount("openai cache write input tokens", inputTokensDetails.CacheWriteTokens)
		inputTokensDetails.TextTokens = billingTokenCount("openai text input tokens", inputTokensDetails.TextTokens)
		inputTokensDetails.AudioTokens = billingTokenCount("openai audio input tokens", inputTokensDetails.AudioTokens)
		inputTokensDetails.ImageTokens = billingTokenCount("openai image input tokens", inputTokensDetails.ImageTokens)
		usage.InputTokensDetails = &inputTokensDetails
	}
	if usage.PromptTokens == 0 && usage.InputTokens > 0 {
		usage.PromptTokens = usage.InputTokens
	}
	if usage.CompletionTokens == 0 && usage.OutputTokens > 0 {
		usage.CompletionTokens = usage.OutputTokens
	}
	if usage.InputTokens == 0 && usage.PromptTokens > 0 {
		usage.InputTokens = usage.PromptTokens
	}
	if usage.OutputTokens == 0 && usage.CompletionTokens > 0 {
		usage.OutputTokens = usage.CompletionTokens
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = billingTokenSum("openai total tokens", usage.PromptTokens, usage.CompletionTokens)
	}
	usage.UsageSemantic = dto.BillingUsageSemanticOpenAI
	usage.UsageSource = billingUsage.Source
	usage.BillingUsage = dto.CloneBillingUsage(billingUsage)
	return &usage
}

func usageFromClaudeBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	claudeUsage := billingUsage.ClaudeUsage
	inputTokens := billingTokenCount("claude prompt tokens", claudeUsage.InputTokens)
	outputTokens := billingTokenCount("claude completion tokens", claudeUsage.OutputTokens)
	cacheReadTokens := billingTokenCount("claude cache read tokens", claudeUsage.CacheReadInputTokens)
	cacheCreationTokens := billingTokenCount("claude cache creation tokens", claudeUsage.CacheCreationInputTokens)
	cacheCreation5m := billingTokenCount("claude cache creation 5m tokens", claudeUsage.GetCacheCreation5mTokens())
	if cacheCreation5m == 0 {
		cacheCreation5m = billingTokenCount("claude legacy cache creation 5m tokens", claudeUsage.ClaudeCacheCreation5mTokens)
	}
	cacheCreation1h := billingTokenCount("claude cache creation 1h tokens", claudeUsage.GetCacheCreation1hTokens())
	if cacheCreation1h == 0 {
		cacheCreation1h = billingTokenCount("claude legacy cache creation 1h tokens", claudeUsage.ClaudeCacheCreation1hTokens)
	}

	usage := &dto.Usage{
		PromptTokens:                inputTokens,
		CompletionTokens:            outputTokens,
		TotalTokens:                 billingTokenSum("claude total tokens", inputTokens, outputTokens),
		InputTokens:                 billingTokenSum("claude input tokens", inputTokens, cacheReadTokens, cacheCreationTokens),
		OutputTokens:                outputTokens,
		UsageSemantic:               dto.BillingUsageSemanticAnthropic,
		UsageSource:                 dto.BillingUsageSourceClaudeMessages,
		BillingUsage:                dto.CloneBillingUsage(billingUsage),
		ClaudeCacheCreation5mTokens: cacheCreation5m,
		ClaudeCacheCreation1hTokens: cacheCreation1h,
	}
	usage.PromptTokensDetails.CachedTokens = cacheReadTokens
	usage.PromptTokensDetails.CachedCreationTokens = cacheCreationTokens
	return usage
}

func usageFromGeminiBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	metadata := *billingUsage.GeminiUsageMetadata
	promptTokens := billingTokenSum("gemini prompt tokens", metadata.PromptTokenCount, metadata.ToolUsePromptTokenCount)
	completionTokens := billingTokenSum("gemini completion tokens", metadata.CandidatesTokenCount, metadata.ThoughtsTokenCount)
	reportedTotalTokens := billingTokenCount("gemini reported total tokens", metadata.TotalTokenCount)
	if completionTokens == 0 && reportedTotalTokens > promptTokens {
		completionTokens = reportedTotalTokens - promptTokens
	}
	componentTotalTokens := billingTokenSum("gemini component total tokens", promptTokens, completionTokens)
	usage := &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      max(reportedTotalTokens, componentTotalTokens),
		UsageSemantic:    dto.BillingUsageSemanticGemini,
		UsageSource:      dto.BillingUsageSourceGeminiChat,
		BillingUsage:     dto.CloneBillingUsage(billingUsage),
	}
	usage.CompletionTokenDetails.ReasoningTokens = billingTokenCount("gemini reasoning tokens", metadata.ThoughtsTokenCount)
	usage.PromptTokensDetails.CachedTokens = billingTokenCount("gemini cached input tokens", metadata.CachedContentTokenCount)

	for _, detail := range metadata.PromptTokensDetails {
		addGeminiInputTokenDetail(&usage.PromptTokensDetails, detail)
	}
	for _, detail := range metadata.ToolUsePromptTokensDetails {
		addGeminiInputTokenDetail(&usage.PromptTokensDetails, detail)
	}
	for _, detail := range metadata.CandidatesTokensDetails {
		switch detail.Modality {
		case "IMAGE":
			usage.CompletionTokenDetails.ImageTokens = billingTokenSum("gemini image completion tokens", usage.CompletionTokenDetails.ImageTokens, detail.TokenCount)
		case "AUDIO":
			usage.CompletionTokenDetails.AudioTokens = billingTokenSum("gemini audio completion tokens", usage.CompletionTokenDetails.AudioTokens, detail.TokenCount)
		case "TEXT":
			usage.CompletionTokenDetails.TextTokens = billingTokenSum("gemini text completion tokens", usage.CompletionTokenDetails.TextTokens, detail.TokenCount)
		}
	}

	if usage.PromptTokens > 0 && usage.PromptTokensDetails.TextTokens == 0 && usage.PromptTokensDetails.AudioTokens == 0 && usage.PromptTokensDetails.ImageTokens == 0 {
		usage.PromptTokensDetails.TextTokens = usage.PromptTokens
	}
	return usage
}

func billingTokenCount(operation string, value int) int {
	if value >= 0 {
		return value
	}
	common.SysError(fmt.Sprintf("%s received negative token count %d, clamped to 0", operation, value))
	return 0
}

func billingTokenSum(operation string, values ...int) int {
	total := 0
	for _, value := range values {
		value = billingTokenCount(operation, value)
		if value > math.MaxInt-total {
			common.SysError(fmt.Sprintf("%s overflow, clamped to %d", operation, math.MaxInt))
			return math.MaxInt
		}
		total += value
	}
	return total
}

func addGeminiInputTokenDetail(details *dto.InputTokenDetails, detail dto.GeminiPromptTokensDetails) {
	switch detail.Modality {
	case "AUDIO":
		details.AudioTokens = billingTokenSum("gemini audio input tokens", details.AudioTokens, detail.TokenCount)
	case "IMAGE":
		details.ImageTokens = billingTokenSum("gemini image input tokens", details.ImageTokens, detail.TokenCount)
	case "TEXT":
		details.TextTokens = billingTokenSum("gemini text input tokens", details.TextTokens, detail.TokenCount)
	}
}
