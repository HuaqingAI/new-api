package agentplatform

import (
	"bytes"
	"context"
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

type stubSkillHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (s stubSkillHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

func newSkillInvokeServiceForTest(t *testing.T) (*SkillInvokeService, *gorm.DB, string, apmodel.Resource) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))

	client := apmodel.Client{
		Slug:                  "invoke-client",
		DisplayName:           "Invoke Client",
		ClientType:            "desktop",
		Status:                "active",
		AllowedGrantTypesJSON: `["client_credentials"]`,
		AllowedScopesJSON:     `["ap.resources.read","ap.skills.invoke"]`,
		ContractVersion:       "2026-06",
		CapabilitiesJSON:      `{"discovery":true}`,
	}
	require.NoError(t, db.Create(&client).Error)

	resource := apmodel.Resource{
		ResourceType:  apmodel.ResourceTypeSkill,
		DisplayName:   "Invoke Skill",
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
		DetailJSON:      `{"skill":{"invoke_mode":"sync"}}`,
		Status:          apmodel.ResourceStatusPublished,
		CreatedBy:       100,
	}).Error)
	require.NoError(t, db.Create(&apmodel.SkillDef{
		ResourceId:        resource.ResourceId,
		ResourceVersion:   "1.0.0",
		InvokeSchemaJSON:  `{"type":"object"}`,
		OutputSchemaJSON:  `{"type":"object"}`,
		InvokeMode:        "sync",
		TimeoutSeconds:    1,
		BindingConfigJSON: `{"method":"POST","url":"https://example.com/invoke","headers":{"X-Test":"yes"}}`,
	}).Error)
	now := time.Now().UTC()
	require.NoError(t, db.Create(&apmodel.Exposure{
		ResourceId:          resource.ResourceId,
		ResourceVersion:     "1.0.0",
		ClientKey:           client.ClientId,
		VisibilityState:     apmodel.ExposureVisibilityVisible,
		CallableState:       apmodel.ExposureCallableEnabled,
		FreshnessTTLSeconds: 300,
		ETag:                "etag-skill",
		PublishedAt:         &now,
	}).Error)

	return NewSkillInvokeService(db), db, client.ClientId, resource
}

func TestSkillInvokeServiceInvokesConfiguredHTTPBinding(t *testing.T) {
	svc, _, clientID, resource := newSkillInvokeServiceForTest(t)
	svc = svc.WithHTTPClient(stubSkillHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "POST", req.Method)
			require.Equal(t, "https://example.com/invoke", req.URL.String())
			require.Equal(t, "yes", req.Header.Get("X-Test"))
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.JSONEq(t, `{"input":"demo"}`, string(body))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
			}, nil
		},
	})

	result, err := svc.Invoke(SkillInvokeInput{
		ClientID:   clientID,
		ResourceID: resource.ResourceId,
		Payload:    []byte(`{"input":"demo"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(result.Output))
}

func TestSkillInvokeServiceMapsTimeoutAndUpstreamFailure(t *testing.T) {
	svc, _, clientID, resource := newSkillInvokeServiceForTest(t)

	timeoutSvc := svc.WithHTTPClient(stubSkillHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		},
	})
	_, err := timeoutSvc.Invoke(SkillInvokeInput{
		ClientID:   clientID,
		ResourceID: resource.ResourceId,
		Payload:    []byte(`{"input":"demo"}`),
	})
	require.ErrorIs(t, err, ErrSkillInvokeTimeout)

	failSvc := svc.WithHTTPClient(stubSkillHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("boom")
		},
	})
	_, err = failSvc.Invoke(SkillInvokeInput{
		ClientID:   clientID,
		ResourceID: resource.ResourceId,
		Payload:    []byte(`{"input":"demo"}`),
	})
	require.ErrorIs(t, err, ErrSkillInvokeUpstreamFailed)
}
