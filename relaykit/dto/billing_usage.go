package dto

import (
	"math"
	"strings"
)

const (
	BillingUsageSourceClaudeMessages = "claude_messages"
	BillingUsageSourceGeminiChat     = "gemini_chat"
	BillingUsageSourceOAIChat        = "oai_chat"
	BillingUsageSourceOAIResponses   = "oai_responses"

	BillingUsageSemanticAnthropic = "anthropic"
	BillingUsageSemanticGemini    = "gemini"
	BillingUsageSemanticOpenAI    = "openai"
)

type BillingUsage struct {
	Source              string               `json:"source,omitempty"`
	Semantic            string               `json:"semantic,omitempty"`
	Estimated           bool                 `json:"estimated,omitempty"`
	OpenAIUsage         *Usage               `json:"openai_usage,omitempty"`
	ClaudeUsage         *ClaudeUsage         `json:"claude_usage,omitempty"`
	GeminiUsageMetadata *GeminiUsageMetadata `json:"gemini_usage_metadata,omitempty"`
}

func NewClaudeMessagesBillingUsage(usage *ClaudeUsage) *BillingUsage {
	if !HasClaudeUsageTokens(usage) {
		return nil
	}
	return &BillingUsage{
		Source:      BillingUsageSourceClaudeMessages,
		Semantic:    BillingUsageSemanticAnthropic,
		ClaudeUsage: cloneClaudeUsage(usage),
	}
}

// HasClaudeUsageTokens mirrors HasOpenAIUsageTokens/HasGeminiUsageMetadataTokens:
// an all-zero ClaudeUsage must not become a BillingUsage, otherwise it would take
// precedence during settlement and zero out a non-zero top-level usage.
func HasClaudeUsageTokens(usage *ClaudeUsage) bool {
	if usage == nil {
		return false
	}
	if usage.InputTokens != 0 ||
		usage.OutputTokens != 0 ||
		usage.CacheCreationInputTokens != 0 ||
		usage.CacheReadInputTokens != 0 ||
		usage.ClaudeCacheCreation5mTokens != 0 ||
		usage.ClaudeCacheCreation1hTokens != 0 {
		return true
	}
	if usage.CacheCreation != nil &&
		(usage.CacheCreation.Ephemeral5mInputTokens != 0 || usage.CacheCreation.Ephemeral1hInputTokens != 0) {
		return true
	}
	return false
}

func NewOpenAIChatBillingUsage(usage *Usage) *BillingUsage {
	return newOpenAIBillingUsage(BillingUsageSourceOAIChat, usage)
}

func NewOpenAIResponsesBillingUsage(usage *Usage) *BillingUsage {
	return newOpenAIBillingUsage(BillingUsageSourceOAIResponses, usage)
}

func newOpenAIBillingUsage(source string, usage *Usage) *BillingUsage {
	if !HasOpenAIUsageTokens(usage) {
		return nil
	}
	return &BillingUsage{
		Source:      source,
		Semantic:    BillingUsageSemanticOpenAI,
		OpenAIUsage: cloneOpenAIUsage(usage),
	}
}

func HasOpenAIUsageTokens(usage *Usage) bool {
	if usage == nil {
		return false
	}
	if usage.PromptTokens != 0 ||
		usage.CompletionTokens != 0 ||
		usage.TotalTokens != 0 ||
		usage.InputTokens != 0 ||
		usage.OutputTokens != 0 ||
		usage.PromptCacheHitTokens != 0 ||
		usage.ClaudeCacheCreation5mTokens != 0 ||
		usage.ClaudeCacheCreation1hTokens != 0 {
		return true
	}
	if usage.PromptTokensDetails.CachedTokensDetails.HasTokens() ||
		usage.PromptTokensDetails.CachedTokens != 0 ||
		usage.PromptTokensDetails.CachedCreationTokens != 0 ||
		usage.PromptTokensDetails.CacheWriteTokens != 0 ||
		usage.PromptTokensDetails.TextTokens != 0 ||
		usage.PromptTokensDetails.ImageTokens != 0 ||
		usage.PromptTokensDetails.AudioTokens != 0 {
		return true
	}
	if usage.CompletionTokenDetails.ReasoningTokens != 0 ||
		usage.CompletionTokenDetails.TextTokens != 0 ||
		usage.CompletionTokenDetails.ImageTokens != 0 ||
		usage.CompletionTokenDetails.AudioTokens != 0 {
		return true
	}
	if usage.InputTokensDetails == nil {
		return false
	}
	return usage.InputTokensDetails.CachedTokensDetails.HasTokens() ||
		usage.InputTokensDetails.CachedTokens != 0 ||
		usage.InputTokensDetails.CachedCreationTokens != 0 ||
		usage.InputTokensDetails.CacheWriteTokens != 0 ||
		usage.InputTokensDetails.TextTokens != 0 ||
		usage.InputTokensDetails.ImageTokens != 0 ||
		usage.InputTokensDetails.AudioTokens != 0
}

func NewGeminiChatBillingUsage(metadata *GeminiUsageMetadata) *BillingUsage {
	return newGeminiChatBillingUsage(metadata, false)
}

func NewEstimatedGeminiChatBillingUsage(usage *Usage) *BillingUsage {
	if usage == nil {
		return nil
	}
	reasoningTokens := usage.CompletionTokenDetails.ReasoningTokens
	candidateTokens := max(usage.CompletionTokens-reasoningTokens, 0)
	totalTokens := usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	metadata := &GeminiUsageMetadata{
		PromptTokenCount:        usage.PromptTokens,
		CandidatesTokenCount:    candidateTokens,
		TotalTokenCount:         totalTokens,
		ThoughtsTokenCount:      reasoningTokens,
		CachedContentTokenCount: usage.PromptTokensDetails.CachedTokens,
	}
	for _, detail := range []GeminiPromptTokensDetails{
		{Modality: "TEXT", TokenCount: usage.PromptTokensDetails.TextTokens},
		{Modality: "IMAGE", TokenCount: usage.PromptTokensDetails.ImageTokens},
		{Modality: "AUDIO", TokenCount: usage.PromptTokensDetails.AudioTokens},
	} {
		if detail.TokenCount != 0 {
			metadata.PromptTokensDetails = append(metadata.PromptTokensDetails, detail)
		}
	}
	for _, detail := range []GeminiPromptTokensDetails{
		{Modality: "TEXT", TokenCount: usage.CompletionTokenDetails.TextTokens},
		{Modality: "IMAGE", TokenCount: usage.CompletionTokenDetails.ImageTokens},
		{Modality: "AUDIO", TokenCount: usage.CompletionTokenDetails.AudioTokens},
	} {
		if detail.TokenCount != 0 {
			metadata.CandidatesTokensDetails = append(metadata.CandidatesTokensDetails, detail)
		}
	}
	return newGeminiChatBillingUsage(metadata, true)
}

// CloneBillingUsageWithEstimatedCompletion preserves the original upstream
// billing dialect and fills a missing completion count without rebuilding the
// payload from a converted, potentially lossy Usage value.
func CloneBillingUsageWithEstimatedCompletion(usage *BillingUsage, completionTokens int) *BillingUsage {
	clone := CloneBillingUsage(usage)
	if clone == nil || completionTokens <= 0 {
		return clone
	}

	updated := false
	switch {
	case clone.OpenAIUsage != nil:
		openAIUsage := clone.OpenAIUsage
		if openAIUsage.CompletionTokens == 0 && openAIUsage.OutputTokens == 0 {
			openAIUsage.CompletionTokens = completionTokens
			openAIUsage.OutputTokens = completionTokens
			inputTokens := openAIUsage.PromptTokens
			if inputTokens == 0 {
				inputTokens = openAIUsage.InputTokens
			}
			if totalTokens := inputTokens + completionTokens; openAIUsage.TotalTokens < totalTokens {
				openAIUsage.TotalTokens = totalTokens
			}
			updated = true
		}
	case clone.ClaudeUsage != nil:
		if clone.ClaudeUsage.OutputTokens == 0 {
			clone.ClaudeUsage.OutputTokens = completionTokens
			updated = true
		}
	case clone.GeminiUsageMetadata != nil:
		metadata := clone.GeminiUsageMetadata
		if metadata.CandidatesTokenCount == 0 {
			candidateTokens := max(completionTokens-metadata.ThoughtsTokenCount, 0)
			metadata.CandidatesTokenCount = candidateTokens
			totalTokens := metadata.PromptTokenCount + metadata.ToolUsePromptTokenCount + metadata.CandidatesTokenCount + metadata.ThoughtsTokenCount
			if metadata.TotalTokenCount < totalTokens {
				metadata.TotalTokenCount = totalTokens
			}
			updated = true
		}
	}
	if updated {
		clone.Estimated = true
	}
	return clone
}

func newGeminiChatBillingUsage(metadata *GeminiUsageMetadata, estimated bool) *BillingUsage {
	if !HasGeminiUsageMetadataTokens(metadata) {
		return nil
	}
	usageMetadata := cloneGeminiUsageMetadata(*metadata)
	return &BillingUsage{
		Source:              BillingUsageSourceGeminiChat,
		Semantic:            BillingUsageSemanticGemini,
		Estimated:           estimated,
		GeminiUsageMetadata: &usageMetadata,
	}
}

func CloneBillingUsage(usage *BillingUsage) *BillingUsage {
	if usage == nil {
		return nil
	}
	clone := *usage
	clone.OpenAIUsage = cloneOpenAIUsage(usage.OpenAIUsage)
	clone.ClaudeUsage = cloneClaudeUsage(usage.ClaudeUsage)
	if usage.GeminiUsageMetadata != nil {
		metadata := cloneGeminiUsageMetadata(*usage.GeminiUsageMetadata)
		clone.GeminiUsageMetadata = &metadata
	}
	return &clone
}

// CanonicalUsage decodes the original provider usage carried across relay
// hops into the shared accounting shape. The BillingUsage snapshot remains the
// source of truth and is cloned onto the returned value for further relays.
func (usage *BillingUsage) CanonicalUsage() (*Usage, bool) {
	if usage == nil {
		return nil, false
	}
	source := strings.TrimSpace(usage.Source)
	semantic := strings.TrimSpace(usage.Semantic)

	// A structurally recognized but all-zero payload must not become the
	// settlement source of truth; rejecting it lets settlement fall back to a
	// non-zero top-level usage.
	if HasOpenAIUsageTokens(usage.OpenAIUsage) &&
		(strings.EqualFold(source, BillingUsageSourceOAIChat) ||
			strings.EqualFold(source, BillingUsageSourceOAIResponses) ||
			strings.EqualFold(semantic, BillingUsageSemanticOpenAI)) {
		return usage.canonicalOpenAIUsage(), true
	}
	if HasClaudeUsageTokens(usage.ClaudeUsage) &&
		(strings.EqualFold(source, BillingUsageSourceClaudeMessages) ||
			strings.EqualFold(semantic, BillingUsageSemanticAnthropic)) {
		return usage.canonicalClaudeUsage(), true
	}
	if HasGeminiUsageMetadataTokens(usage.GeminiUsageMetadata) &&
		(strings.EqualFold(source, BillingUsageSourceGeminiChat) ||
			strings.EqualFold(semantic, BillingUsageSemanticGemini)) {
		return usage.canonicalGeminiUsage(), true
	}
	return nil, false
}

func (usage *BillingUsage) canonicalOpenAIUsage() *Usage {
	canonical := cloneOpenAIUsage(usage.OpenAIUsage)
	if canonical.InputTokensDetails != nil {
		// InputTokensDetails fills fields that PromptTokensDetails omitted;
		// existing PromptTokensDetails values stay canonical on overlap.
		filled := *canonical.InputTokensDetails
		mergeInputTokenDetails(&filled, canonical.PromptTokensDetails)
		canonical.PromptTokensDetails = filled
	}
	if canonical.PromptTokensDetails.CachedTokens == 0 && canonical.PromptCacheHitTokens > 0 {
		canonical.PromptTokensDetails.CachedTokens = canonical.PromptCacheHitTokens
	}
	if canonical.PromptTokens == 0 && canonical.InputTokens > 0 {
		canonical.PromptTokens = canonical.InputTokens
	}
	if canonical.CompletionTokens == 0 && canonical.OutputTokens > 0 {
		canonical.CompletionTokens = canonical.OutputTokens
	}
	if canonical.InputTokens == 0 && canonical.PromptTokens > 0 {
		canonical.InputTokens = canonical.PromptTokens
	}
	if canonical.OutputTokens == 0 && canonical.CompletionTokens > 0 {
		canonical.OutputTokens = canonical.CompletionTokens
	}
	if canonical.TotalTokens == 0 {
		canonical.TotalTokens = addTokenCounts(canonical.PromptTokens, canonical.CompletionTokens)
	}
	normalizeUsageTokenCounts(canonical)
	canonical.UsageSemantic = BillingUsageSemanticOpenAI
	canonical.UsageSource = usage.Source
	canonical.BillingUsage = CloneBillingUsage(usage)
	return canonical
}

func (usage *BillingUsage) canonicalClaudeUsage() *Usage {
	claudeUsage := usage.ClaudeUsage
	// Flat legacy fields are a fallback only when this snapshot never carried
	// a CacheCreation sub-object. Presence (non-nil), not zero vs non-zero,
	// is the discriminator — a later sub-object that zeros 1h must win.
	var cacheCreation5m, cacheCreation1h int
	if claudeUsage.CacheCreation != nil {
		cacheCreation5m = clampTokenCount(claudeUsage.GetCacheCreation5mTokens())
		cacheCreation1h = clampTokenCount(claudeUsage.GetCacheCreation1hTokens())
	} else {
		cacheCreation5m = clampTokenCount(claudeUsage.ClaudeCacheCreation5mTokens)
		cacheCreation1h = clampTokenCount(claudeUsage.ClaudeCacheCreation1hTokens)
	}

	canonical := &Usage{
		PromptTokens:                clampTokenCount(claudeUsage.InputTokens),
		CompletionTokens:            clampTokenCount(claudeUsage.OutputTokens),
		TotalTokens:                 addTokenCounts(claudeUsage.InputTokens, claudeUsage.OutputTokens),
		InputTokens:                 addTokenCounts(claudeUsage.InputTokens, claudeUsage.CacheReadInputTokens, claudeUsage.CacheCreationInputTokens),
		OutputTokens:                clampTokenCount(claudeUsage.OutputTokens),
		UsageSemantic:               BillingUsageSemanticAnthropic,
		UsageSource:                 BillingUsageSourceClaudeMessages,
		BillingUsage:                CloneBillingUsage(usage),
		ClaudeCacheCreation5mTokens: cacheCreation5m,
		ClaudeCacheCreation1hTokens: cacheCreation1h,
	}
	canonical.PromptTokensDetails.CachedTokens = clampTokenCount(claudeUsage.CacheReadInputTokens)
	canonical.PromptTokensDetails.CachedCreationTokens = clampTokenCount(claudeUsage.CacheCreationInputTokens)
	return canonical
}

func (usage *BillingUsage) canonicalGeminiUsage() *Usage {
	metadata := usage.GeminiUsageMetadata
	promptTokens := addTokenCounts(metadata.PromptTokenCount, metadata.ToolUsePromptTokenCount)
	canonical := &Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: addTokenCounts(metadata.CandidatesTokenCount, metadata.ThoughtsTokenCount),
		TotalTokens:      clampTokenCount(metadata.TotalTokenCount),
		UsageSemantic:    BillingUsageSemanticGemini,
		UsageSource:      BillingUsageSourceGeminiChat,
		BillingUsage:     CloneBillingUsage(usage),
	}
	canonical.CompletionTokenDetails.ReasoningTokens = clampTokenCount(metadata.ThoughtsTokenCount)
	canonical.PromptTokensDetails.CachedTokens = clampTokenCount(metadata.CachedContentTokenCount)

	for _, detail := range metadata.PromptTokensDetails {
		addGeminiInputTokenDetail(&canonical.PromptTokensDetails, detail)
	}
	for _, detail := range metadata.ToolUsePromptTokensDetails {
		addGeminiInputTokenDetail(&canonical.PromptTokensDetails, detail)
	}
	for _, detail := range metadata.CandidatesTokensDetails {
		switch normalizeGeminiModality(detail.Modality) {
		case "IMAGE":
			addTokenCount(&canonical.CompletionTokenDetails.ImageTokens, detail.TokenCount)
		case "AUDIO":
			addTokenCount(&canonical.CompletionTokenDetails.AudioTokens, detail.TokenCount)
		case "TEXT":
			addTokenCount(&canonical.CompletionTokenDetails.TextTokens, detail.TokenCount)
		}
	}

	if canonical.TotalTokens == 0 {
		canonical.TotalTokens = addTokenCounts(canonical.PromptTokens, canonical.CompletionTokens)
	} else if canonical.CompletionTokens <= 0 {
		canonical.CompletionTokens = max(canonical.TotalTokens-canonical.PromptTokens, 0)
	}
	minTotalTokens := addTokenCounts(canonical.PromptTokens, canonical.CompletionTokens)
	if canonical.TotalTokens < minTotalTokens {
		canonical.TotalTokens = minTotalTokens
	}
	if canonical.PromptTokens > 0 && canonical.PromptTokensDetails.TextTokens == 0 &&
		canonical.PromptTokensDetails.AudioTokens == 0 && canonical.PromptTokensDetails.ImageTokens == 0 {
		canonical.PromptTokensDetails.TextTokens = canonical.PromptTokens
	}
	return canonical
}

func addGeminiInputTokenDetail(details *InputTokenDetails, detail GeminiPromptTokensDetails) {
	switch normalizeGeminiModality(detail.Modality) {
	case "AUDIO":
		addTokenCount(&details.AudioTokens, detail.TokenCount)
	case "IMAGE":
		addTokenCount(&details.ImageTokens, detail.TokenCount)
	case "TEXT":
		addTokenCount(&details.TextTokens, detail.TokenCount)
	}
}

func clampTokenCount(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func addTokenCounts(values ...int) int {
	total := 0
	for _, value := range values {
		value = clampTokenCount(value)
		if value > math.MaxInt-total {
			return math.MaxInt
		}
		total += value
	}
	return total
}

func addTokenCount(target *int, value int) {
	if target == nil {
		return
	}
	*target = addTokenCounts(*target, value)
}

func normalizeUsageTokenCounts(usage *Usage) {
	if usage == nil {
		return
	}
	usage.PromptTokens = clampTokenCount(usage.PromptTokens)
	usage.CompletionTokens = clampTokenCount(usage.CompletionTokens)
	usage.TotalTokens = clampTokenCount(usage.TotalTokens)
	usage.InputTokens = clampTokenCount(usage.InputTokens)
	usage.OutputTokens = clampTokenCount(usage.OutputTokens)
	usage.PromptCacheHitTokens = clampTokenCount(usage.PromptCacheHitTokens)
	usage.ClaudeCacheCreation5mTokens = clampTokenCount(usage.ClaudeCacheCreation5mTokens)
	usage.ClaudeCacheCreation1hTokens = clampTokenCount(usage.ClaudeCacheCreation1hTokens)
	normalizeInputTokenDetails(&usage.PromptTokensDetails)
	normalizeOutputTokenDetails(&usage.CompletionTokenDetails)
	if usage.InputTokensDetails != nil {
		normalizeInputTokenDetails(usage.InputTokensDetails)
	}
}

func normalizeInputTokenDetails(details *InputTokenDetails) {
	if details == nil {
		return
	}
	details.CachedTokens = clampTokenCount(details.CachedTokens)
	details.CachedCreationTokens = clampTokenCount(details.CachedCreationTokens)
	details.CacheWriteTokens = clampTokenCount(details.CacheWriteTokens)
	details.TextTokens = clampTokenCount(details.TextTokens)
	details.AudioTokens = clampTokenCount(details.AudioTokens)
	details.ImageTokens = clampTokenCount(details.ImageTokens)
	if details.CachedTokensDetails == nil {
		return
	}
	clampTokenPointer(details.CachedTokensDetails.TextTokens)
	clampTokenPointer(details.CachedTokensDetails.ImageTokens)
	clampTokenPointer(details.CachedTokensDetails.AudioTokens)
}

func normalizeOutputTokenDetails(details *OutputTokenDetails) {
	if details == nil {
		return
	}
	details.TextTokens = clampTokenCount(details.TextTokens)
	details.AudioTokens = clampTokenCount(details.AudioTokens)
	details.ImageTokens = clampTokenCount(details.ImageTokens)
	details.ReasoningTokens = clampTokenCount(details.ReasoningTokens)
}

func clampTokenPointer(value *int) {
	if value != nil && *value < 0 {
		*value = 0
	}
}

func cloneOpenAIUsage(usage *Usage) *Usage {
	if usage == nil {
		return nil
	}
	clone := *usage
	clone.BillingUsage = nil
	clone.PromptTokensDetails = usage.PromptTokensDetails.Clone()
	if usage.InputTokensDetails != nil {
		inputTokensDetails := usage.InputTokensDetails.Clone()
		clone.InputTokensDetails = &inputTokensDetails
	}
	return &clone
}

func cloneClaudeUsage(usage *ClaudeUsage) *ClaudeUsage {
	if usage == nil {
		return nil
	}
	clone := *usage
	clone.BillingUsage = nil
	if usage.CacheCreation != nil {
		cacheCreation := *usage.CacheCreation
		clone.CacheCreation = &cacheCreation
	}
	if usage.ServerToolUse != nil {
		serverToolUse := *usage.ServerToolUse
		clone.ServerToolUse = &serverToolUse
	}
	return &clone
}

func cloneGeminiUsageMetadata(metadata GeminiUsageMetadata) GeminiUsageMetadata {
	metadata.PromptTokensDetails = append([]GeminiPromptTokensDetails{}, metadata.PromptTokensDetails...)
	metadata.ToolUsePromptTokensDetails = append([]GeminiPromptTokensDetails{}, metadata.ToolUsePromptTokensDetails...)
	metadata.CandidatesTokensDetails = append([]GeminiPromptTokensDetails{}, metadata.CandidatesTokensDetails...)
	metadata.BillingUsage = nil
	return metadata
}

func HasGeminiUsageMetadataTokens(metadata *GeminiUsageMetadata) bool {
	if metadata == nil {
		return false
	}
	if metadata.PromptTokenCount != 0 ||
		metadata.ToolUsePromptTokenCount != 0 ||
		metadata.CandidatesTokenCount != 0 ||
		metadata.TotalTokenCount != 0 ||
		metadata.ThoughtsTokenCount != 0 ||
		metadata.CachedContentTokenCount != 0 {
		return true
	}
	for _, detail := range metadata.PromptTokensDetails {
		if detail.TokenCount != 0 {
			return true
		}
	}
	for _, detail := range metadata.ToolUsePromptTokensDetails {
		if detail.TokenCount != 0 {
			return true
		}
	}
	for _, detail := range metadata.CandidatesTokensDetails {
		if detail.TokenCount != 0 {
			return true
		}
	}
	return false
}
