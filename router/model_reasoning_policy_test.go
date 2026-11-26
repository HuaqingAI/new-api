package router

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestModelReasoningPolicyNormalization(t *testing.T) {
	raw := `{"schema_version":1,"rules":[{"match_type":"model","match_value":" gpt-special ","efforts":["max","none","max"]},{"match_type":"family","match_value":" GPT ","efforts":["high","low"]},{"match_type":"family","match_value":"deepseek","efforts":["none"]}]}`
	policy, canonical, err := setting.NormalizeModelReasoningPolicy(raw)
	require.NoError(t, err)
	assert.Equal(t, []setting.ModelReasoningRule{
		{MatchType: "family", MatchValue: "deepseek", Efforts: []string{"none"}},
		{MatchType: "family", MatchValue: "gpt", Efforts: []string{"low", "high"}},
		{MatchType: "model", MatchValue: "gpt-special", Efforts: []string{"none", "max"}},
	}, policy.Rules)
	effective, err := policy.Effective()
	require.NoError(t, err)
	roundTrip, encoded, err := setting.NormalizeModelReasoningPolicy(canonical)
	require.NoError(t, err)
	roundTripEffective, err := roundTrip.Effective()
	require.NoError(t, err)
	assert.Equal(t, canonical, encoded)
	assert.Equal(t, effective.Revision, roundTripEffective.Revision)
	assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, effective.Revision)
	assert.Equal(t, []string{"low", "medium", "high"}, effective.DefaultEfforts)

	cases := []struct{ name, raw string }{
		{"missing schema", `{"rules":[]}`},
		{"unknown schema", `{"schema_version":2,"rules":[]}`},
		{"unknown top level field", `{"schema_version":1,"rules":[],"rule":[]}`},
		{"null rules", `{"schema_version":1,"rules":null}`},
		{"null rule", `{"schema_version":1,"rules":[null]}`},
		{"wrong rule type", `{"schema_version":1,"rules":["gpt"]}`},
		{"unknown field", `{"schema_version":1,"rules":[{"match_type":"family","match_value":"gpt","efforts":["high"],"enabled":true}]}`},
		{"unknown family", `{"schema_version":1,"rules":[{"match_type":"family","match_value":"claude","efforts":["high"]}]}`},
		{"unknown match type", `{"schema_version":1,"rules":[{"match_type":"prefix","match_value":"gpt","efforts":["high"]}]}`},
		{"empty model", `{"schema_version":1,"rules":[{"match_type":"model","match_value":" ","efforts":["high"]}]}`},
		{"control character", `{"schema_version":1,"rules":[{"match_type":"model","match_value":"\ngpt","efforts":["high"]}]}`},
		{"unpaired high surrogate", `{"schema_version":1,"rules":[{"match_type":"model","match_value":"gpt\ud800","efforts":["high"]}]}`},
		{"unpaired low surrogate", `{"schema_version":1,"rules":[{"match_type":"model","match_value":"gpt\udfff","efforts":["high"]}]}`},
		{"empty efforts", `{"schema_version":1,"rules":[{"match_type":"family","match_value":"gpt","efforts":[]}]}`},
		{"null efforts", `{"schema_version":1,"rules":[{"match_type":"family","match_value":"gpt","efforts":null}]}`},
		{"null effort", `{"schema_version":1,"rules":[{"match_type":"family","match_value":"gpt","efforts":[null]}]}`},
		{"unknown effort", `{"schema_version":1,"rules":[{"match_type":"family","match_value":"gpt","efforts":["ultra"]}]}`},
		{"duplicate target after normalization", `{"schema_version":1,"rules":[{"match_type":"family","match_value":"GPT","efforts":["low"]},{"match_type":"family","match_value":" gpt ","efforts":["high"]}]}`},
		{"trailing JSON", `{"schema_version":1,"rules":[]} {}`},
		{"invalid UTF-8", `{"schema_version":1,"rules":[{"match_type":"model","match_value":"` + string([]byte{0xff}) + `","efforts":["low"]}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := setting.NormalizeModelReasoningPolicy(tc.raw)
			require.ErrorIs(t, err, setting.ErrModelReasoningPolicyInvalid)
		})
	}
	for _, id := range []string{strings.Repeat("模", 255), "gpt\uFFFD", "gpt\\ud800", "gpt😀"} {
		body, err := common.Marshal(setting.ModelReasoningPolicy{SchemaVersion: 1, Rules: []setting.ModelReasoningRule{{MatchType: "model", MatchValue: id, Efforts: []string{"none", "low", "medium", "high", "xhigh", "max"}}}})
		require.NoError(t, err)
		parsed, _, err := setting.NormalizeModelReasoningPolicy(string(body))
		require.NoError(t, err)
		assert.Equal(t, id, parsed.Rules[0].MatchValue)
	}
	for _, id := range []string{"gpt", "GPT"} {
		body, err := common.Marshal(setting.ModelReasoningPolicy{SchemaVersion: 1, Rules: []setting.ModelReasoningRule{{MatchType: "family", MatchValue: "gpt", Efforts: []string{"low"}}, {MatchType: "model", MatchValue: id, Efforts: []string{"max"}}}})
		require.NoError(t, err)
		_, _, err = setting.NormalizeModelReasoningPolicy(string(body))
		require.NoError(t, err, "a model ID must not be confused with a family")
	}
	limits := setting.ModelReasoningPolicy{SchemaVersion: 1, Rules: make([]setting.ModelReasoningRule, 1000)}
	for i := range limits.Rules {
		limits.Rules[i] = setting.ModelReasoningRule{MatchType: "model", MatchValue: fmt.Sprint(i), Efforts: []string{"low"}}
	}
	encodedLimits, err := common.Marshal(limits)
	require.NoError(t, err)
	_, _, err = setting.NormalizeModelReasoningPolicy(string(encodedLimits))
	require.NoError(t, err)
	limits.Rules = append(limits.Rules, setting.ModelReasoningRule{MatchType: "model", MatchValue: "extra", Efforts: []string{"low"}})
	encodedLimits, err = common.Marshal(limits)
	require.NoError(t, err)
	_, _, err = setting.NormalizeModelReasoningPolicy(string(encodedLimits))
	require.ErrorIs(t, err, setting.ErrModelReasoningPolicyInvalid)
	limits.Rules = []setting.ModelReasoningRule{{MatchType: "model", MatchValue: strings.Repeat("模", 256), Efforts: []string{"low"}}}
	encodedLimits, err = common.Marshal(limits)
	require.NoError(t, err)
	_, _, err = setting.NormalizeModelReasoningPolicy(string(encodedLimits))
	require.ErrorIs(t, err, setting.ErrModelReasoningPolicyInvalid)
}

func TestModelReasoningPolicyAPI(t *testing.T) {
	engine, user, rootBearer, adminBearer := setupAionUiModelFilterRouterTest(t)
	var version string
	versionQuery := "SELECT version()"
	if common.UsingMainDatabase(common.DatabaseTypeSQLite) {
		versionQuery = "SELECT sqlite_version()"
	}
	require.NoError(t, model.DB.Raw(versionQuery).Scan(&version).Error)
	t.Logf("database version: %s", version)
	RegisterAionUiRouter(engine.Group("/api"))
	const management = "/api/agent-platform/model-reasoning-policy"
	const consumer = "/api/aionui/model-reasoning-policy"
	read := func(path, bearer string) setting.EffectiveModelReasoningPolicy {
		t.Helper()
		response := requestAionUiModelFilter(t, engine, http.MethodGet, path, bearer, "", nil)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assert.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
		var payload struct {
			Success bool                                  `json:"success"`
			Data    setting.EffectiveModelReasoningPolicy `json:"data"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
		require.True(t, payload.Success, response.Body.String())
		return payload.Data
	}

	t.Run("root permission and default", func(t *testing.T) {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			response := requestAionUiModelFilter(t, engine, method, management, "", `{"rules":[]}`, nil)
			assert.Equal(t, http.StatusUnauthorized, response.Code)
			response = requestAionUiModelFilter(t, engine, method, management, adminBearer, `{"rules":[]}`, nil)
			assert.Equal(t, http.StatusForbidden, response.Code)
		}
		policy := read(management, rootBearer)
		assert.Equal(t, 1, policy.SchemaVersion)
		assert.Equal(t, "global", policy.Scope)
		assert.Equal(t, []string{"low", "medium", "high"}, policy.DefaultEfforts)
		assert.Empty(t, policy.Rules)
	})

	t.Run("scoped dashboard tokens", func(t *testing.T) {
		var root model.User
		require.NoError(t, model.DB.Where("username = ?", "filter-root").First(&root).Error)
		for i, scope := range []string{"user:read", "option:read", "option:write"} {
			raw := model.AccessTokenPrefix + fmt.Sprintf("reasoning-scope-%d-", i) + strings.Repeat("a", 43)
			token := model.UserAccessToken{UserId: root.Id, Name: "reasoning fixture", TokenHash: model.AccessTokenFingerprint(raw)}
			require.NoError(t, token.SetScopes([]string{scope}))
			require.NoError(t, model.DB.Create(&token).Error)
			for _, method := range []string{http.MethodGet, http.MethodPut} {
				response := requestAionUiModelFilter(t, engine, method, management, raw, `{"rules":[]}`, nil)
				expected := http.StatusForbidden
				if method == http.MethodGet && scope == "option:read" || method == http.MethodPut && scope == "option:write" {
					expected = http.StatusOK
				}
				assert.Equal(t, expected, response.Code, response.Body.String())
			}
		}
	})

	t.Run("save reload and desktop credentials", func(t *testing.T) {
		response := requestAionUiModelFilter(t, engine, http.MethodPut, management, rootBearer, `{"rules":[{"match_type":"family","match_value":"GPT","efforts":["high","low"]},{"match_type":"model","match_value":" gpt-special ","efforts":["max","none"]}]}`, nil)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		policy := read(management, rootBearer)
		require.Len(t, policy.Rules, 2)
		assert.Equal(t, []string{"none", "max"}, policy.Rules[1].Efforts)
		model.InitOptionMap()
		model.InitOptionMap()
		assert.Equal(t, policy, read(management, rootBearer))

		// Simulate a second node whose periodic option cache has not refreshed.
		common.OptionMapRWMutex.Lock()
		common.OptionMap[setting.ModelReasoningPolicyKey] = setting.DefaultModelReasoningPolicy
		common.OptionMapRWMutex.Unlock()
		assert.Equal(t, policy, read(management, rootBearer))
		for _, bearer := range []string{"", rootBearer, "invalid-desktop-token"} {
			response := requestAionUiModelFilter(t, engine, http.MethodGet, consumer, bearer, "", nil)
			assert.Equal(t, http.StatusUnauthorized, response.Code)
		}
		require.NoError(t, model.DB.Model(user).Update("email", "reasoning-user@example.com").Error)
		user.Email = "reasoning-user@example.com"
		serviceaionui.ResetDesktopAuthServiceForTest()
		t.Cleanup(serviceaionui.ResetDesktopAuthServiceForTest)
		code, err := serviceaionui.DefaultDesktopAuthService().IssueCode(user, "http://127.0.0.1:49152/hth/callback", "reasoning-test-state")
		require.NoError(t, err)
		desktop, err := serviceaionui.DefaultDesktopAuthService().ExchangeCode(dtoaionui.DesktopTokenRequest{Code: code, DeviceId: "reasoning-fixture"})
		require.NoError(t, err)
		assert.Equal(t, policy, read(consumer, desktop.AccessToken))
		response = requestAionUiModelFilter(t, engine, http.MethodGet, consumer, desktop.PersonalAPIKey.Key, "", nil)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
		require.NoError(t, model.DB.Model(user).Update("status", common.UserStatusDisabled).Error)
		response = requestAionUiModelFilter(t, engine, http.MethodGet, consumer, desktop.AccessToken, "", nil)
		assert.Equal(t, http.StatusForbidden, response.Code)
		require.NoError(t, model.DB.Model(user).Update("status", common.UserStatusEnabled).Error)
	})

	t.Run("invalid updates retain saved rules", func(t *testing.T) {
		require.NoError(t, model.UpdateOption(setting.ModelReasoningPolicyKey, setting.DefaultModelReasoningPolicy))
		before := read(management, rootBearer)
		for _, body := range []string{`{}`, `{"rules":null}`, `{"rules":[],"schema_version":1}`, `{"rules":[{"match_type":"model","match_value":"gpt","efforts":[]}]}`, `{"rules":[{"match_type":"family","match_value":"glm","efforts":["ultra"]}]}`} {
			response := requestAionUiModelFilter(t, engine, http.MethodPut, management, rootBearer, body, nil)
			assert.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			assert.Equal(t, before, read(management, rootBearer))
		}
		response := requestAionUiModelFilter(t, engine, http.MethodPut, management, rootBearer, `{"rules":[]}`+strings.Repeat(" ", 64*1024), nil)
		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
		assert.Equal(t, before, read(management, rootBearer))
		// HTML escaping can make the canonical value exceed 60 KiB even if
		// the original request body fits within the transport limit.
		rules := make([]string, 200)
		for i := range rules {
			rules[i] = fmt.Sprintf(`{"match_type":"model","match_value":"m%d%s","efforts":["low"]}`, i, strings.Repeat("<", 60))
		}
		body := `{"rules":[` + strings.Join(rules, ",") + `]}`
		require.Less(t, len(body), 64*1024)
		response = requestAionUiModelFilter(t, engine, http.MethodPut, management, rootBearer, body, nil)
		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code, response.Body.String())
		assert.Equal(t, before, read(management, rootBearer))
	})

	t.Run("generic option validates normalizes and preserves unrelated data", func(t *testing.T) {
		require.NoError(t, model.DB.Create(&model.Option{Key: "reasoning-unrelated", Value: "preserved"}).Error)
		raw := `{"schema_version":1,"rules":[{"match_type":"family","match_value":"QWEN","efforts":["high","low","high"]}]}`
		body, err := common.Marshal(map[string]string{"key": setting.ModelReasoningPolicyKey, "value": raw})
		require.NoError(t, err)
		response := requestAionUiModelFilter(t, engine, http.MethodPut, "/api/option/", rootBearer, string(body), nil)
		require.Equal(t, http.StatusOK, response.Code)
		var result struct {
			Success bool `json:"success"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
		require.True(t, result.Success, response.Body.String())
		assert.Equal(t, []setting.ModelReasoningRule{{MatchType: "family", MatchValue: "qwen", Efforts: []string{"low", "high"}}}, read(management, rootBearer).Rules)
		require.Error(t, model.UpdateOption(setting.ModelReasoningPolicyKey, `{"schema_version":1,"rules":null}`))
		require.Error(t, model.UpdateOptionsBulk(map[string]string{setting.ModelReasoningPolicyKey: `{"schema_version":1,"rules":null}`, "reasoning-unrelated": "changed"}))
		var unrelated model.Option
		require.NoError(t, model.DB.First(&unrelated, &model.Option{Key: "reasoning-unrelated"}).Error)
		assert.Equal(t, "preserved", unrelated.Value)
	})

	t.Run("database save failure preserves committed policy and cache", func(t *testing.T) {
		require.NoError(t, model.UpdateOption(setting.ModelReasoningPolicyKey, setting.DefaultModelReasoningPolicy))
		before := read(management, rootBearer)
		require.NoError(t, model.DB.Callback().Update().Before("gorm:update").Register("reasoning-fixture-failure", func(tx *gorm.DB) {
			if _, ok := tx.Statement.Model.(*model.Option); ok {
				tx.AddError(errors.New("fixture write failed"))
			}
		}))
		response := requestAionUiModelFilter(t, engine, http.MethodPut, management, rootBearer, `{"rules":[{"match_type":"family","match_value":"gpt","efforts":["max"]}]}`, nil)
		require.NoError(t, model.DB.Callback().Update().Remove("reasoning-fixture-failure"))
		assert.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
		assert.Contains(t, response.Body.String(), "MODEL_REASONING_POLICY_SAVE_FAILED")
		assert.Equal(t, before, read(management, rootBearer))
		common.OptionMapRWMutex.RLock()
		cached := common.OptionMap[setting.ModelReasoningPolicyKey]
		common.OptionMapRWMutex.RUnlock()
		assert.Equal(t, setting.DefaultModelReasoningPolicy, cached)
	})

	t.Run("stored corruption is unavailable rather than an empty policy", func(t *testing.T) {
		require.NoError(t, model.DB.Model(&model.Option{}).Where(&model.Option{Key: setting.ModelReasoningPolicyKey}).Update("value", "broken").Error)
		response := requestAionUiModelFilter(t, engine, http.MethodGet, management, rootBearer, "", nil)
		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Contains(t, response.Body.String(), "MODEL_REASONING_POLICY_UNAVAILABLE")
		require.NoError(t, model.UpdateOption(setting.ModelReasoningPolicyKey, setting.DefaultModelReasoningPolicy))
		assert.Empty(t, read(management, rootBearer).Rules)
	})
}
