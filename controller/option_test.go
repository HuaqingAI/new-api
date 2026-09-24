package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpdateOptionValidatesEnterpriseBudgetThresholds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	common.OptionMap = make(map[string]string)

	oldWarning := operation_setting.GetQuotaSetting().EnterpriseBudgetWarningThreshold
	oldCritical := operation_setting.GetQuotaSetting().EnterpriseBudgetCriticalThreshold
	t.Cleanup(func() {
		operation_setting.GetQuotaSetting().EnterpriseBudgetWarningThreshold = oldWarning
		operation_setting.GetQuotaSetting().EnterpriseBudgetCriticalThreshold = oldCritical
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	operation_setting.GetQuotaSetting().EnterpriseBudgetWarningThreshold = 80
	operation_setting.GetQuotaSetting().EnterpriseBudgetCriticalThreshold = 95

	router := gin.New()
	router.PUT("/api/option/", UpdateOption)

	invalidRecorder := performOptionRequest(t, router, map[string]any{
		"key":   "quota_setting.enterprise_budget_warning_threshold",
		"value": 98,
	})
	require.Equal(t, http.StatusOK, invalidRecorder.Code)
	require.Contains(t, invalidRecorder.Body.String(), `"success":false`)
	require.Contains(t, invalidRecorder.Body.String(), `enterprise.organization.department_budget_threshold_invalid`)

	validRecorder := performOptionRequest(t, router, map[string]any{
		"key":   "quota_setting.enterprise_budget_warning_threshold",
		"value": 75,
	})
	require.Equal(t, http.StatusOK, validRecorder.Code)
	require.Contains(t, validRecorder.Body.String(), `"success":true`)
	require.Equal(t, 75, operation_setting.GetQuotaSetting().EnterpriseBudgetWarningThreshold)
}

func performOptionRequest(t *testing.T, router *gin.Engine, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPut, "/api/option/", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
