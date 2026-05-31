package agentplatform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newKnowledgeQueryServiceForTest(t *testing.T) (*KnowledgeQueryService, *gorm.DB, string, apmodel.Resource) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:                  "knowledge-client",
		DisplayName:           "Knowledge Client",
		ClientType:            "desktop",
		Status:                "active",
		AllowedGrantTypesJSON: `["client_credentials"]`,
		AllowedScopesJSON:     `["ap.resources.read","ap.knowledge.query"]`,
		ContractVersion:       "2026-06",
		CapabilitiesJSON:      `{"discovery":true}`,
	}
	require.NoError(t, db.Create(&client).Error)

	resource := apmodel.Resource{
		ResourceType:  apmodel.ResourceTypeKnowledge,
		DisplayName:   "Knowledge",
		OwnerUserId:   100,
		Status:        apmodel.ResourceStatusPublished,
		LatestVersion: "1.0.0",
	}
	require.NoError(t, db.Create(&resource).Error)
	require.NoError(t, db.Create(&apmodel.ResourceVersion{
		ResourceId:      resource.ResourceId,
		Version:         "1.0.0",
		ContractVersion: "2026-06",
		SchemaJSON:      `{"type":"object"}`,
		DetailJSON:      `{"knowledge":{"mode":"retrieval"}}`,
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)
	require.NoError(t, db.Create(&apmodel.KnowledgeDef{
		ResourceId:               resource.ResourceId,
		ResourceVersion:          "1.0.0",
		KnowledgeMode:            "retrieval",
		ProviderType:             "http_retrieval",
		ProviderAdapterKey:       "provider-a",
		ProviderConfigJSON:       `{"url":"https://example.com/query","method":"POST"}`,
		QuerySchemaJSON:          `{"type":"object"}`,
		CitationSchemaJSON:       `{"items":{"type":"object"}}`,
		FreshnessRulesJSON:       `{"ttl":300}`,
		ProviderCapabilitiesJSON: `{"citations":true}`,
	}).Error)
	now := time.Now().UTC()
	require.NoError(t, db.Create(&apmodel.Exposure{
		ResourceId:          resource.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           client.ClientId,
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-knowledge",
		PublishedAt:         &now,
	}).Error)

	return NewKnowledgeQueryService(db), db, client.ClientId, resource
}

type stubKnowledgeProvider struct {
	validate func(ctx context.Context, binding KnowledgeProviderBinding) error
	query    func(ctx context.Context, binding KnowledgeProviderBinding, req KnowledgeProviderQueryRequest) (KnowledgeProviderQueryResponse, error)
}

type stubKnowledgeHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (s stubKnowledgeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

func (s stubKnowledgeProvider) ValidateBinding(ctx context.Context, binding KnowledgeProviderBinding) error {
	if s.validate != nil {
		return s.validate(ctx, binding)
	}
	return nil
}

func (s stubKnowledgeProvider) Query(ctx context.Context, binding KnowledgeProviderBinding, req KnowledgeProviderQueryRequest) (KnowledgeProviderQueryResponse, error) {
	if s.query != nil {
		return s.query(ctx, binding, req)
	}
	return KnowledgeProviderQueryResponse{}, nil
}

func (s stubKnowledgeProvider) Refresh(ctx context.Context, binding KnowledgeProviderBinding) (KnowledgeProviderRefreshState, error) {
	return KnowledgeProviderRefreshState{Status: "ready"}, nil
}

func (s stubKnowledgeProvider) Health(ctx context.Context, binding KnowledgeProviderBinding) (KnowledgeProviderHealthState, error) {
	return KnowledgeProviderHealthState{Status: "healthy"}, nil
}

func TestKnowledgeQueryServiceReturnsStructuredRetrievalResult(t *testing.T) {
	svc, _, clientID, resource := newKnowledgeQueryServiceForTest(t)
	svc = svc.WithProvider(stubKnowledgeProvider{
		query: func(ctx context.Context, binding KnowledgeProviderBinding, req KnowledgeProviderQueryRequest) (KnowledgeProviderQueryResponse, error) {
			return KnowledgeProviderQueryResponse{
				Items: []KnowledgeResultItem{
					{
						ID:      "doc-1",
						Score:   0.92,
						Snippet: "Result for " + req.Query,
					},
				},
				Citations: []KnowledgeCitation{
					{SourceID: "doc-1", Title: "Doc 1"},
				},
			}, nil
		},
	})

	result, err := svc.Query(KnowledgeQueryInput{
		ClientID:   clientID,
		ResourceID: resource.ResourceId,
		Payload:    []byte(`{"query":"how does it work?"}`),
	})
	require.NoError(t, err)
	require.Equal(t, resource.ResourceId, result.ResourceID)
	require.Len(t, result.Items, 1)
	require.Len(t, result.Citations, 1)
	require.Contains(t, result.Items[0].Snippet, "how does it work?")
}

func TestKnowledgeQueryServiceRejectsInvalidPayload(t *testing.T) {
	svc, _, clientID, resource := newKnowledgeQueryServiceForTest(t)

	_, err := svc.Query(KnowledgeQueryInput{
		ClientID:   clientID,
		ResourceID: resource.ResourceId,
		Payload:    []byte(`{"query":""}`),
	})
	require.ErrorIs(t, err, ErrOpenCapabilityContractInvalid)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(`{"query":"ok"}`), &payload))
}

func TestHTTPRetrievalProviderMapsHTTPErrors(t *testing.T) {
	provider := NewHTTPRetrievalProvider().WithHTTPClient(stubKnowledgeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		},
	})
	_, err := provider.Query(context.Background(), KnowledgeProviderBinding{
		ProviderType: "http_retrieval",
		Config: map[string]any{
			"url": "https://example.com/query",
		},
	}, KnowledgeProviderQueryRequest{Query: "demo"})
	require.ErrorIs(t, err, ErrSkillInvokeTimeout)

	provider = NewHTTPRetrievalProvider().WithHTTPClient(stubKnowledgeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":"bad gateway"}`)),
			}, nil
		},
	})
	_, err = provider.Query(context.Background(), KnowledgeProviderBinding{
		ProviderType: "http_retrieval",
		Config: map[string]any{
			"url": "https://example.com/query",
		},
	}, KnowledgeProviderQueryRequest{Query: "demo"})
	require.ErrorIs(t, err, ErrSkillInvokeUpstreamFailed)
}

func TestKnowledgeQueryServiceMapsProviderFailures(t *testing.T) {
	svc, _, clientID, resource := newKnowledgeQueryServiceForTest(t)
	svc = svc.WithProvider(stubKnowledgeProvider{
		query: func(ctx context.Context, binding KnowledgeProviderBinding, req KnowledgeProviderQueryRequest) (KnowledgeProviderQueryResponse, error) {
			return KnowledgeProviderQueryResponse{}, errors.New("provider broke")
		},
	})

	_, err := svc.Query(KnowledgeQueryInput{
		ClientID:   clientID,
		ResourceID: resource.ResourceId,
		Payload:    []byte(`{"query":"demo"}`),
	})
	require.Error(t, err)
}
