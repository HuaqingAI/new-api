package agentplatform

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

type httpRetrievalHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPRetrievalProvider struct {
	httpClient httpRetrievalHTTPDoer
}

func NewHTTPRetrievalProvider() *HTTPRetrievalProvider {
	return &HTTPRetrievalProvider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *HTTPRetrievalProvider) WithHTTPClient(client httpRetrievalHTTPDoer) *HTTPRetrievalProvider {
	if client != nil {
		p.httpClient = client
	}
	return p
}

func (p *HTTPRetrievalProvider) ValidateBinding(ctx context.Context, binding KnowledgeProviderBinding) error {
	if strings.TrimSpace(binding.ProviderType) != "http_retrieval" {
		return ErrOpenCapabilityContractInvalid
	}
	url, _ := binding.Config["url"].(string)
	if strings.TrimSpace(url) == "" {
		return ErrOpenCapabilityContractInvalid
	}
	return nil
}

func (p *HTTPRetrievalProvider) Query(ctx context.Context, binding KnowledgeProviderBinding, req KnowledgeProviderQueryRequest) (KnowledgeProviderQueryResponse, error) {
	if err := p.ValidateBinding(ctx, binding); err != nil {
		return KnowledgeProviderQueryResponse{}, err
	}

	url := strings.TrimSpace(binding.Config["url"].(string))
	method, _ := binding.Config["method"].(string)
	if strings.TrimSpace(method) == "" {
		method = http.MethodPost
	}
	payload, err := common.Marshal(map[string]any{
		"query": req.Query,
	})
	if err != nil {
		return KnowledgeProviderQueryResponse{}, ErrOpenCapabilityContractInvalid
	}
	httpReq, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, bytes.NewReader(payload))
	if err != nil {
		return KnowledgeProviderQueryResponse{}, ErrOpenCapabilityContractInvalid
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if headers, ok := binding.Config["headers"].(map[string]any); ok {
		for key, value := range headers {
			httpReq.Header.Set(key, common.Interface2String(value))
		}
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return KnowledgeProviderQueryResponse{}, ErrSkillInvokeTimeout
		}
		return KnowledgeProviderQueryResponse{}, ErrSkillInvokeUpstreamFailed
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return KnowledgeProviderQueryResponse{}, ErrSkillInvokeUpstreamFailed
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return KnowledgeProviderQueryResponse{}, ErrSkillInvokeUpstreamFailed
	}

	var providerResp struct {
		Items []KnowledgeResultItem `json:"items"`
	}
	if err := common.Unmarshal(body, &providerResp); err != nil {
		return KnowledgeProviderQueryResponse{}, ErrOpenCapabilityContractInvalid
	}
	citations := make([]KnowledgeCitation, 0, len(providerResp.Items))
	for _, item := range providerResp.Items {
		citations = append(citations, KnowledgeCitation{
			SourceID: item.ID,
			Title:    item.ID,
			Metadata: item.Metadata,
		})
	}
	return KnowledgeProviderQueryResponse{
		Items:     providerResp.Items,
		Citations: citations,
	}, nil
}

func (p *HTTPRetrievalProvider) Refresh(ctx context.Context, binding KnowledgeProviderBinding) (KnowledgeProviderRefreshState, error) {
	if err := p.ValidateBinding(ctx, binding); err != nil {
		return KnowledgeProviderRefreshState{}, err
	}
	return KnowledgeProviderRefreshState{Status: "ready"}, nil
}

func (p *HTTPRetrievalProvider) Health(ctx context.Context, binding KnowledgeProviderBinding) (KnowledgeProviderHealthState, error) {
	if err := p.ValidateBinding(ctx, binding); err != nil {
		return KnowledgeProviderHealthState{}, err
	}
	return KnowledgeProviderHealthState{Status: "healthy"}, nil
}
