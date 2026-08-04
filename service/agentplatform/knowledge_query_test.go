package agentplatform

import (
	"bytes"
	"context"
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
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)
	require.NoError(t, db.Create(&apmodel.KnowledgeDef{
		ResourceId:          resource.ResourceId,
		ExternalKnowledgeId: "kb_demo",
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

type stubKnowledgeHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (s stubKnowledgeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

func TestKnowledgeQueryServiceReturnsContractInvalidWithoutProviderConfig(t *testing.T) {
	svc, _, clientID, resource := newKnowledgeQueryServiceForTest(t)

	_, err := svc.Query(KnowledgeQueryInput{
		ClientID:   clientID,
		ResourceID: resource.ResourceId,
		Payload:    []byte(`{"query":"how does it work?"}`),
	})
	require.ErrorIs(t, err, ErrOpenCapabilityContractInvalid)
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
