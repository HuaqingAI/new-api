package enterprise

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	dtoenterprise "github.com/QuantumNous/new-api/dto/enterprise"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	entservice "github.com/QuantumNous/new-api/service/enterprise"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDingTalkConfig(c *gin.Context) {
	tenantId := parseTenantIdQuery(c)
	result, err := entservice.NewDingTalkConfigService(model.DB).Get(tenantId)
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func SaveDingTalkConfig(c *gin.Context) {
	var req dtoenterprise.DingTalkConfigRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	input := entservice.DingTalkConfigInput{
		TenantId:     valueOrZero(req.TenantId),
		CorpId:       req.CorpId,
		AppKey:       req.AppKey,
		AppSecret:    req.AppSecret,
		CallbackUrl:  req.CallbackUrl,
		SyncScope:    req.SyncScope,
		LoginEnabled: boolValue(req.LoginEnabled),
		SyncEnabled:  boolValue(req.SyncEnabled),
	}

	var result dtoenterprise.DingTalkConfigResponse
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = entservice.NewDingTalkConfigService(tx).Save(input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    result.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionDingTalkConfigSet,
			ObjectType:  entservice.AdminObjectDingTalkConfig,
			ObjectId:    strconv.Itoa(result.TenantId),
			DiffSummary: "Saved DingTalk enterprise app configuration",
			Payload: map[string]any{
				"tenant_id":      result.TenantId,
				"corp_id":        result.CorpId,
				"app_key":        result.AppKey,
				"app_secret":     req.AppSecret,
				"callback_url":   result.CallbackUrl,
				"sync_scope":     result.SyncScope,
				"login_enabled":  result.LoginEnabled,
				"sync_enabled":   result.SyncEnabled,
				"has_app_secret": result.HasAppSecret,
			},
		})
	})
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}

	common.ApiSuccessI18n(c, i18n.MsgEnterpriseDingTalkConfigSaved, result)
}

func TestDingTalkConnectivity(c *gin.Context) {
	tenantId := parseTenantIdQuery(c)
	result, err := entservice.NewDingTalkConnectivityService(model.DB, nil).Test(c.Request.Context(), tenantId)
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}

	if err := writeAdminAction(model.DB, c, entservice.AdminActionInput{
		TenantId:    result.TenantId,
		ActorId:     c.GetInt("id"),
		ActionType:  entservice.AdminActionDingTalkTest,
		ObjectType:  entservice.AdminObjectDingTalkConfig,
		ObjectId:    strconv.Itoa(result.TenantId),
		DiffSummary: "Tested DingTalk enterprise app connectivity",
		Payload: map[string]any{
			"tenant_id":   result.TenantId,
			"code":        result.Code,
			"stage":       result.Stage,
			"summary":     result.Summary,
			"http_status": result.HTTPStatus,
			"checked_at":  result.CheckedAt,
		},
	}); err != nil {
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	common.ApiSuccessI18n(c, i18n.MsgEnterpriseDingTalkConnectivityTested, result)
}

func StartDingTalkFullSync(c *gin.Context) {
	var req dtoenterprise.DingTalkSyncStartRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId := valueOrZero(req.TenantId)
	result, err := entservice.NewDingTalkSyncService(model.DB, nil).StartFullSync(c.Request.Context(), entservice.DingTalkSyncStartInput{
		TenantId:  tenantId,
		ActorId:   c.GetInt("id"),
		RunInline: boolValue(req.Inline),
	})
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}
	if err := writeAdminAction(model.DB, c, entservice.AdminActionInput{
		TenantId:    result.TenantId,
		ActorId:     c.GetInt("id"),
		ActionType:  entservice.AdminActionDingTalkSyncStart,
		ObjectType:  entservice.AdminObjectDingTalkSyncTask,
		ObjectId:    strconv.Itoa(result.Id),
		DiffSummary: "Started DingTalk address book full sync",
		Payload: map[string]any{
			"tenant_id": result.TenantId,
			"task_id":   result.Id,
			"mode":      result.Mode,
			"status":    result.Status,
			"inline":    boolValue(req.Inline),
		},
	}); err != nil {
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccessI18n(c, i18n.MsgEnterpriseDingTalkSyncStarted, result)
}

func GetDingTalkSyncTask(c *gin.Context) {
	taskId, err := strconv.Atoi(c.Param("id"))
	if err != nil || taskId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidId)
		return
	}
	result, err := entservice.NewDingTalkSyncService(model.DB, nil).GetTask(c.Request.Context(), taskId)
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func ListDingTalkSyncLogs(c *gin.Context) {
	query := entservice.DingTalkSyncLogQuery{
		TenantId:   intQueryPtr(parseTenantIdQuery(c)),
		Status:     c.Query("status"),
		ObjectType: c.Query("object_type"),
		Page:       parsePositiveIntQuery(c, "page", 1),
		PageSize:   parsePositiveIntQuery(c, "page_size", 20),
	}
	if taskId := parsePositiveIntQuery(c, "task_id", 0); taskId > 0 {
		query.TaskId = &taskId
	}
	if startAt := parseInt64Query(c, "start_at"); startAt > 0 {
		query.StartAt = &startAt
	}
	if endAt := parseInt64Query(c, "end_at"); endAt > 0 {
		query.EndAt = &endAt
	}
	result, err := entservice.NewDingTalkSyncService(model.DB, nil).ListLogs(c.Request.Context(), query)
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func ListDingTalkSyncConflicts(c *gin.Context) {
	query := entservice.DingTalkSyncConflictQuery{
		TenantId: intQueryPtr(parseTenantIdQuery(c)),
		Status:   c.Query("status"),
		Page:     parsePositiveIntQuery(c, "page", 1),
		PageSize: parsePositiveIntQuery(c, "page_size", 20),
	}
	result, err := entservice.NewDingTalkSyncService(model.DB, nil).ListConflicts(c.Request.Context(), query)
	if err != nil {
		writeDingTalkConfigError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func parseTenantIdQuery(c *gin.Context) int {
	raw := c.Query("tenant_id")
	if raw == "" {
		return 0
	}
	tenantId, err := strconv.Atoi(raw)
	if err != nil || tenantId < 0 {
		return 0
	}
	return tenantId
}

func intQueryPtr(value int) *int {
	return &value
}

func writeDingTalkConfigError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrDingTalkMissingCredentials):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkMissingCredentials)
	case errors.Is(err, entservice.ErrDingTalkInvalidCallbackURL):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkInvalidCallbackURL)
	case errors.Is(err, entservice.ErrDingTalkConfigNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkConfigNotFound)
	case errors.Is(err, entservice.ErrDingTalkSyncNotEnabled):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkSyncNotEnabled)
	case errors.Is(err, entservice.ErrDingTalkSyncTaskNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDingTalkSyncTaskNotFound)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func parsePositiveIntQuery(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func parseInt64Query(c *gin.Context, key string) int64 {
	raw := c.Query(key)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
