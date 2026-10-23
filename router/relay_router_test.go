package router

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const aionUiModelFilterTestOption = "AionUiModelFilterExcludedModels"

type aionUiModelFilterTestResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Data    struct {
		TargetTokenName string   `json:"target_token_name"`
		DefaultModel    string   `json:"default_model"`
		ExcludedModels  []string `json:"excluded_models"`
	} `json:"data"`
}

// AIONUI_MODEL_FILTER_TEST_DSN must identify an empty, disposable database.
// Model-list queries use fixed table names, so a table prefix cannot isolate them.
func setupAionUiModelFilterRouterTest(t *testing.T) (*gin.Engine, *model.User, string, string) {
	t.Helper()
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMaster, originalRedis, originalMemory := common.IsMasterNode, common.RedisEnabled, common.MemoryCacheEnabled
	originalSQLite, originalSelfUse := common.SQLitePath, operation_setting.SelfUseModeEnabled
	originalMainType, originalLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.OptionMapRWMutex.RLock()
	originalOptions := maps.Clone(common.OptionMap)
	common.OptionMapRWMutex.RUnlock()
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.IsMasterNode, common.RedisEnabled, common.MemoryCacheEnabled = originalMaster, originalRedis, originalMemory
		common.SQLitePath, operation_setting.SelfUseModeEnabled = originalSQLite, originalSelfUse
		common.SetDatabaseTypes(originalMainType, originalLogType)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})
	dsn := os.Getenv("AIONUI_MODEL_FILTER_TEST_DSN")
	if dsn == "" {
		dsn = "local"
	}
	t.Setenv("SQL_DSN", dsn)
	t.Setenv("LOG_SQL_DSN", "")
	common.IsMasterNode, common.RedisEnabled, common.MemoryCacheEnabled = false, false, false
	common.SQLitePath = filepath.Join(t.TempDir(), "aionui-model-filter.db")
	operation_setting.SelfUseModeEnabled = true
	require.NoError(t, model.InitDB())
	db := model.DB
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	tables, err := db.Migrator().GetTables()
	require.NoError(t, err)
	require.Empty(t, tables, "model filter tests require an empty disposable database")
	fixtures := []any{&model.Option{}, &model.User{}, &model.UserAccessToken{}, &model.Token{}, &model.Channel{}, &model.Ability{}, &model.AuditLog{}}
	t.Cleanup(func() { require.NoError(t, db.Migrator().DropTable(fixtures...)) })
	require.NoError(t, db.AutoMigrate(fixtures...))
	model.LOG_DB = db
	require.NoError(t, model.EnsureLegacyAccessTokenRetireAt(time.Now().Unix()))
	model.InitOptionMap()

	rootBearer, adminBearer := "model-filter-root-pat", "model-filter-admin-pat"
	for _, identity := range []struct {
		name   string
		role   int
		bearer string
	}{{"filter-root", common.RoleRootUser, rootBearer}, {"filter-admin", common.RoleAdminUser, adminBearer}} {
		user := model.User{Username: identity.name, AffCode: identity.name, Role: identity.role, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1}
		user.SetAccessToken(identity.bearer)
		require.NoError(t, db.Create(&user).Error)
	}
	user := &model.User{Username: "filter-relay", AffCode: "filter-relay", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1}
	require.NoError(t, db.Create(user).Error)
	channel := model.Channel{Type: 1, Name: "filter-channel", Key: "fixture", Group: "default", Status: common.ChannelStatusEnabled}
	require.NoError(t, db.Create(&channel).Error)
	abilities := []model.Ability{}
	for _, id := range []string{"model-a", "model-a-preview", "other-model", "model-b"} {
		abilities = append(abilities, model.Ability{Group: "default", Model: id, ChannelId: channel.Id, Enabled: true})
	}
	abilities = append(abilities, model.Ability{Group: "vip", Model: "vip-only", ChannelId: channel.Id, Enabled: true})
	require.NoError(t, db.Create(&abilities).Error)
	engine := gin.New()
	RegisterAgentPlatformRouter(engine.Group("/api"))
	engine.PUT("/api/option/", middleware.RootAuth(), controller.UpdateOption)
	SetRelayRouter(engine)
	return engine, user, rootBearer, adminBearer
}

func requestAionUiModelFilter(t *testing.T, engine *gin.Engine, method, target, bearer, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func decodeAionUiModelFilterResponse(t *testing.T, response *httptest.ResponseRecorder) aionUiModelFilterTestResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var payload aionUiModelFilterTestResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.True(t, payload.Success, response.Body.String())
	return payload
}

func decodeAionUiAvailableModels(t *testing.T, response *httptest.ResponseRecorder) ([]string, map[string]any) {
	t.Helper()
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var payload map[string]any
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	items, ok := payload["data"].([]any)
	idKey := "id"
	if !ok {
		items, ok = payload["models"].([]any)
		idKey = "name"
	}
	require.True(t, ok, response.Body.String())
	ids := make([]string, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		require.True(t, ok)
		id, ok := item[idKey].(string)
		require.True(t, ok)
		ids = append(ids, id)
	}
	return ids, payload
}

func TestAionUiModelFilter(t *testing.T) {
	engine, user, rootBearer, adminBearer := setupAionUiModelFilterRouterTest(t)
	const route = "/api/agent-platform/model-filter"
	allModels := []string{"model-a", "model-a-preview", "other-model", "model-b"}
	remaining := []string{"model-a-preview", "other-model", "model-b"}

	t.Run("root permission and empty default", func(t *testing.T) {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			response := requestAionUiModelFilter(t, engine, method, route, "", `{ "excluded_models": [] }`, nil)
			assert.Equal(t, http.StatusUnauthorized, response.Code)
			response = requestAionUiModelFilter(t, engine, method, route, adminBearer, `{ "excluded_models": [] }`, nil)
			assert.Equal(t, http.StatusForbidden, response.Code)
			var denied aionUiModelFilterTestResponse
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &denied))
			assert.Equal(t, "AUTH_INSUFFICIENT_PRIVILEGE", denied.Code)
		}
		payload := decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodGet, route, rootBearer, "", nil))
		assert.Equal(t, serviceaionui.DefaultPersonalAPIKeyName, payload.Data.TargetTokenName)
		assert.Equal(t, serviceaionui.DefaultAgentModel, payload.Data.DefaultModel)
		assert.Equal(t, []string{}, payload.Data.ExcludedModels)
	})

	t.Run("canonical save and authenticated model scope", func(t *testing.T) {
		response := requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, `{"excluded_models":[" unavailable-id "," model-a ","model-a","MODEL-B",""]}`, nil)
		payload := decodeAionUiModelFilterResponse(t, response)
		assert.Equal(t, []string{"MODEL-B", "model-a", "unavailable-id"}, payload.Data.ExcludedModels)
		var persisted model.Option
		require.NoError(t, model.DB.First(&persisted, &model.Option{Key: aionUiModelFilterTestOption}).Error)
		assert.JSONEq(t, `["MODEL-B","model-a","unavailable-id"]`, persisted.Value)
		cases := []struct {
			name, tokenName, target string
			limits                  string
			headers                 map[string]string
			expected                []string
		}{
			{name: "exact name", tokenName: serviceaionui.DefaultPersonalAPIKeyName, target: "/v1/models", expected: remaining},
			{name: "ordinary name with spoofed metadata", tokenName: "ordinary", target: "/v1/models?token_name=hth-default-apikey", headers: map[string]string{"token_name": "hth-default-apikey", "X-AionUi": "true"}, expected: allModels},
			{name: "empty name", target: "/v1/models", expected: allModels},
			{name: "different case", tokenName: "HTH-DEFAULT-APIKEY", target: "/v1/models", expected: allModels},
			{name: "surrounding whitespace", tokenName: " hth-default-apikey ", target: "/v1/models", expected: allModels},
			{name: "name suffix", tokenName: "hth-default-apikey-test", target: "/v1/models", expected: allModels},
			{name: "token model restriction", tokenName: serviceaionui.DefaultPersonalAPIKeyName, target: "/v1/models", limits: "model-a,model-b,vip-only", expected: []string{"model-b"}},
			{name: "Gemini path", tokenName: serviceaionui.DefaultPersonalAPIKeyName, target: "/v1beta/models", expected: allModels},
			{name: "Gemini OpenAI path", tokenName: serviceaionui.DefaultPersonalAPIKeyName, target: "/v1beta/openai/models", expected: allModels},
			{name: "Anthropic format", tokenName: serviceaionui.DefaultPersonalAPIKeyName, target: "/v1/models", headers: map[string]string{"anthropic-version": "2023-06-01"}, expected: remaining},
			{name: "Gemini query", tokenName: serviceaionui.DefaultPersonalAPIKeyName, target: "/v1/models?key=", expected: remaining},
		}
		for index, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				key := fmt.Sprintf("aionmodelkey%d", index)
				token := model.Token{UserId: user.Id, Key: key, Name: tc.tokenName, Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true, ModelLimitsEnabled: tc.limits != "", ModelLimits: tc.limits}
				require.NoError(t, model.DB.Create(&token).Error)
				if tc.name == "Anthropic format" {
					tc.headers["x-api-key"] = key
				}
				if tc.name == "Gemini query" {
					tc.target += key
				}
				ids, payload := decodeAionUiAvailableModels(t, requestAionUiModelFilter(t, engine, http.MethodGet, tc.target, key, "", tc.headers))
				assert.ElementsMatch(t, tc.expected, ids)
				if tc.name == "exact name" {
					assert.Equal(t, "list", payload["object"])
					for _, raw := range payload["data"].([]any) {
						item := raw.(map[string]any)
						assert.Equal(t, "model", item["object"])
						assert.Equal(t, "openai", item["owned_by"])
						assert.Contains(t, item, "created")
						assert.Contains(t, item, "supported_endpoint_types")
					}
				}
				if tc.name == "Anthropic format" {
					assert.Equal(t, false, payload["has_more"])
				}
				if tc.name == "Gemini query" {
					assert.Contains(t, payload, "nextPageToken")
				}
			})
		}
		response = requestAionUiModelFilter(t, engine, http.MethodGet, "/v1/models", "missing-key", "", nil)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("validation preserves saved configuration", func(t *testing.T) {
		oversizedItems := make([]string, 1001)
		for i := range oversizedItems {
			oversizedItems[i] = "same-id"
		}
		oversizedJSON, err := common.Marshal(map[string]any{"excluded_models": oversizedItems})
		require.NoError(t, err)
		cases := []struct {
			name, body string
			status     int
		}{
			{"missing", `{}`, http.StatusBadRequest},
			{"null array", `{"excluded_models":null}`, http.StatusBadRequest},
			{"wrong type", `{"excluded_models":"model-a"}`, http.StatusBadRequest},
			{"null member", `{"excluded_models":["model-a",null]}`, http.StatusBadRequest},
			{"numeric member", `{"excluded_models":[1]}`, http.StatusBadRequest},
			{"control character", `{"excluded_models":["bad\nmodel"]}`, http.StatusBadRequest},
			{"leading newline", `{"excluded_models":["\nmodel-a"]}`, http.StatusBadRequest},
			{"trailing tab", `{"excluded_models":["model-a\t"]}`, http.StatusBadRequest},
			{"only control characters", `{"excluded_models":["\n\t"]}`, http.StatusBadRequest},
			{"rune length", `{"excluded_models":["` + strings.Repeat("模", 256) + `"]}`, http.StatusBadRequest},
			{"raw item count before deduplication", string(oversizedJSON), http.StatusBadRequest},
			{"body size", `{"excluded_models":[]}` + strings.Repeat(" ", 64*1024), http.StatusRequestEntityTooLarge},
			{"trailing JSON", `{"excluded_models":[]} {}`, http.StatusBadRequest},
			{"invalid UTF-8", `{"excluded_models":["` + string([]byte{0xff}) + `"]}`, http.StatusBadRequest},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				response := requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, tc.body, nil)
				assert.Equal(t, tc.status, response.Code, response.Body.String())
				payload := decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodGet, route, rootBearer, "", nil))
				assert.Equal(t, []string{"MODEL-B", "model-a", "unavailable-id"}, payload.Data.ExcludedModels)
			})
		}
	})

	t.Run("input limits", func(t *testing.T) {
		models := make([]string, 1000)
		// Use short duplicates for the count boundary so body size is independent.
		for i := range models {
			models[i] = "model-a"
		}
		body, err := common.Marshal(map[string]any{"excluded_models": models})
		require.NoError(t, err)
		payload := decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, string(body), nil))
		assert.Equal(t, []string{"model-a"}, payload.Data.ExcludedModels)
		longID := strings.Repeat("模", 255)
		body, err = common.Marshal(map[string]any{"excluded_models": []string{longID}})
		require.NoError(t, err)
		payload = decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, string(body), nil))
		assert.Equal(t, []string{longID}, payload.Data.ExcludedModels)

		// 238 IDs of 255 bytes plus one of 32 bytes serialize to exactly 60 KiB.
		models = make([]string, 239)
		for i := range 238 {
			models[i] = fmt.Sprintf("m%04d", i) + strings.Repeat("x", 250)
		}
		models[238] = "extra" + strings.Repeat("x", 27)
		canonical, err := common.Marshal(models)
		require.NoError(t, err)
		require.Len(t, canonical, 60*1024)
		body, err = common.Marshal(map[string]any{"excluded_models": models})
		require.NoError(t, err)
		payload = decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, string(body), nil))
		assert.Len(t, payload.Data.ExcludedModels, 239)
		models[238] += "x"
		body, err = common.Marshal(map[string]any{"excluded_models": models})
		require.NoError(t, err)
		response := requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, string(body), nil)
		assert.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		unchanged := decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodGet, route, rootBearer, "", nil))
		assert.Equal(t, payload.Data.ExcludedModels, unchanged.Data.ExcludedModels)
	})

	t.Run("clear all exclude all and preserve protocol envelopes", func(t *testing.T) {
		body, err := common.Marshal(map[string]any{"excluded_models": allModels})
		require.NoError(t, err)
		decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, string(body), nil))
		for _, headers := range []map[string]string{nil, {"x-api-key": "aionmodelkey0", "anthropic-version": "2023-06-01"}, {"x-goog-api-key": "aionmodelkey0"}} {
			ids, payload := decodeAionUiAvailableModels(t, requestAionUiModelFilter(t, engine, http.MethodGet, "/v1/models", "aionmodelkey0", "", headers))
			assert.Empty(t, ids)
			if headers != nil && headers["anthropic-version"] != "" {
				assert.Equal(t, "", payload["first_id"])
				assert.Equal(t, "", payload["last_id"])
				assert.Equal(t, false, payload["has_more"])
			}
		}
		payload := decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, `{"excluded_models":[]}`, nil))
		assert.Equal(t, []string{}, payload.Data.ExcludedModels)
		ids, _ := decodeAionUiAvailableModels(t, requestAionUiModelFilter(t, engine, http.MethodGet, "/v1/models", "aionmodelkey0", "", nil))
		assert.ElementsMatch(t, allModels, ids)
	})

	t.Run("generic option path persistence reload and write failure", func(t *testing.T) {
		require.NoError(t, model.DB.Create(&model.Option{Key: "filter-unrelated-option", Value: "preserved"}).Error)
		body, err := common.Marshal(map[string]string{"key": aionUiModelFilterTestOption, "value": `[" model-b ","model-a","model-b"]`})
		require.NoError(t, err)
		response := requestAionUiModelFilter(t, engine, http.MethodPut, "/api/option/", rootBearer, string(body), nil)
		require.Equal(t, http.StatusOK, response.Code)
		var generic struct {
			Success bool `json:"success"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &generic))
		require.True(t, generic.Success, response.Body.String())
		var persisted model.Option
		require.NoError(t, model.DB.First(&persisted, &model.Option{Key: aionUiModelFilterTestOption}).Error)
		assert.Equal(t, `["model-a","model-b"]`, persisted.Value)
		require.Error(t, model.UpdateOption(aionUiModelFilterTestOption, `[null]`))
		require.NoError(t, model.UpdateOption(aionUiModelFilterTestOption, `["model-b","model-a"]`))
		model.InitOptionMap()
		model.InitOptionMap()
		payload := decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodGet, route, rootBearer, "", nil))
		assert.Equal(t, []string{"model-a", "model-b"}, payload.Data.ExcludedModels)
		var unrelated model.Option
		require.NoError(t, model.DB.First(&unrelated, &model.Option{Key: "filter-unrelated-option"}).Error)
		assert.Equal(t, "preserved", unrelated.Value)
		require.NoError(t, model.DB.Model(&model.Option{Key: aionUiModelFilterTestOption}).Update("value", `[null]`).Error)
		model.InitOptionMap()
		payload = decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodGet, route, rootBearer, "", nil))
		assert.Equal(t, []string{}, payload.Data.ExcludedModels)
		require.NoError(t, model.UpdateOption(aionUiModelFilterTestOption, `["model-a","model-b"]`))
		require.NoError(t, model.DB.Migrator().DropTable(&model.Option{}))
		response = requestAionUiModelFilter(t, engine, http.MethodPut, route, rootBearer, `{"excluded_models":[]}`, nil)
		assert.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
		payload = decodeAionUiModelFilterResponse(t, requestAionUiModelFilter(t, engine, http.MethodGet, route, rootBearer, "", nil))
		assert.Equal(t, []string{"model-a", "model-b"}, payload.Data.ExcludedModels)
		var audit []model.AuditLog
		require.NoError(t, model.LOG_DB.Where("category = ? AND route = ?", model.AuditCategoryOperation, route).Find(&audit).Error)
		require.NotEmpty(t, audit)
		var successful, failed bool
		for _, entry := range audit {
			successful = successful || entry.Success
			failed = failed || !entry.Success
			encoded, err := common.Marshal(entry)
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), rootBearer)
			assert.NotContains(t, string(encoded), "Bearer ")
		}
		assert.True(t, successful)
		assert.True(t, failed)
	})
}

func TestListModelsSupportsOpenAIAndGeminiAuthentication(t *testing.T) {
	setupRelayRouterTestDB(t)

	user := model.User{
		Username: "models-user",
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    100,
	}
	require.NoError(t, model.DB.Create(&user).Error)
	require.NoError(t, model.DB.Create(&model.Token{
		UserId:         user.Id,
		Key:            "modelstestkey",
		Status:         common.TokenStatusEnabled,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
	}).Error)

	engine := gin.New()
	SetRelayRouter(engine)

	tests := []struct {
		name           string
		path           string
		headerName     string
		expectedObject string
		expectedField  string
	}{
		{
			name:           "OpenAI bearer token",
			path:           "/v1/models",
			headerName:     "Authorization",
			expectedObject: "list",
			expectedField:  "data",
		},
		{
			name:          "Gemini API key header",
			path:          "/v1/models",
			headerName:    "x-goog-api-key",
			expectedField: "models",
		},
		{
			name:          "Gemini API key query",
			path:          "/v1/models?key=modelstestkey",
			expectedField: "models",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			if test.headerName != "" {
				value := "modelstestkey"
				if test.headerName == "Authorization" {
					value = "Bearer " + value
				}
				request.Header.Set(test.headerName, value)
			}

			engine.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			var payload map[string]any
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
			assert.Contains(t, payload, test.expectedField)
			assert.NotContains(t, payload, "error")
			if test.expectedObject != "" {
				assert.Equal(t, test.expectedObject, payload["object"])
			}
		})
	}
}

func setupRelayRouterTestDB(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	originalIsMasterNode := common.IsMasterNode
	originalRedisEnabled := common.RedisEnabled
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")

	common.IsMasterNode = false
	common.RedisEnabled = false
	common.SQLitePath = fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))
	require.NoError(t, model.InitDB())
	model.LOG_DB = model.DB
	require.NoError(t, model.DB.AutoMigrate(&model.User{}, &model.Token{}, &model.Ability{}))

	t.Cleanup(func() {
		if sqlDB, err := model.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		common.IsMasterNode = originalIsMasterNode
		common.RedisEnabled = originalRedisEnabled
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	})
}
