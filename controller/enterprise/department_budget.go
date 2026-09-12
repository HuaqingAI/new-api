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

func CreateDepartmentBudget(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	var req dtoenterprise.CreateDepartmentBudgetRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	input := entservice.CreateDepartmentBudgetInput{
		TenantId:       tenantId,
		Type:           req.Type,
		TotalQuota:     req.TotalQuota,
		CycleQuota:     req.CycleQuota,
		CycleType:      req.CycleType,
		CycleStartedAt: req.CycleStartedAt,
		CustomSeconds:  req.CustomSeconds,
		ExpiresAt:      req.ExpiresAt,
	}

	var item entservice.DepartmentBudgetItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = entservice.NewDepartmentBudgetService(tx).Create(departmentId, input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionDepartmentBudgetCreate,
			ObjectType:  entservice.AdminObjectDepartmentBudget,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: "Created department budget pool",
			Payload: map[string]any{
				"department_id":    item.DepartmentId,
				"type":             item.Type,
				"total_quota":      item.TotalQuota,
				"cycle_quota":      item.CycleQuota,
				"cycle_type":       item.CycleType,
				"cycle_started_at": item.CycleStartedAt,
				"custom_seconds":   item.CustomSeconds,
				"expires_at":       item.ExpiresAt,
			},
		})
	})
	if err != nil {
		if shouldAuditDepartmentBudgetFailure(err) {
			_ = writeAdminAction(model.DB, c, entservice.AdminActionInput{
				TenantId:    input.TenantId,
				ActorId:     c.GetInt("id"),
				ActionType:  entservice.AdminActionDepartmentBudgetReject,
				ObjectType:  entservice.AdminObjectDepartmentBudget,
				ObjectId:    strconv.Itoa(departmentId),
				DiffSummary: "Rejected department budget creation",
				Payload: map[string]any{
					"department_id":    departmentId,
					"type":             input.Type,
					"total_quota":      int64PtrValueOrZero(input.TotalQuota),
					"cycle_quota":      int64PtrValueOrZero(input.CycleQuota),
					"cycle_type":       input.CycleType,
					"cycle_started_at": int64PtrValueOrZero(input.CycleStartedAt),
					"custom_seconds":   int64PtrValueOrZero(input.CustomSeconds),
					"expires_at":       int64PtrValueOrZero(input.ExpiresAt),
					"error":            err.Error(),
				},
			})
		}
		writeDepartmentBudgetError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{
		Item: mapDepartmentBudgetItemDTO(item),
	})
}

func GetDepartmentBudget(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	tenantId, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return
	}

	item, err := entservice.NewDepartmentBudgetService(model.DB).GetByDepartment(departmentId, tenantId)
	if err != nil {
		writeDepartmentBudgetError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{
		Item: mapDepartmentBudgetItemDTOOrNil(item),
	})
}

func PauseDepartmentBudget(c *gin.Context) {
	mutateDepartmentBudgetStatus(c, entservice.AdminActionDepartmentBudgetPause, "Paused department budget pool", entmodelStatusActive(), entmodelStatusPaused())
}

func ResumeDepartmentBudget(c *gin.Context) {
	mutateDepartmentBudgetStatus(c, entservice.AdminActionDepartmentBudgetResume, "Resumed department budget pool", entmodelStatusPaused(), entmodelStatusActive())
}

func ResizeDepartmentBudget(c *gin.Context) {
	departmentId, budgetId, ok := parseDepartmentBudgetPath(c)
	if !ok {
		return
	}
	var req dtoenterprise.ResizeDepartmentBudgetRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	var before *entservice.DepartmentBudgetItem
	var item entservice.DepartmentBudgetItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		detail, err := entservice.NewDepartmentBudgetService(tx).GetDetail(departmentId, budgetId, tenantId)
		if err != nil {
			return err
		}
		before = &detail.Budget
		item, err = entservice.NewDepartmentBudgetService(tx).Resize(departmentId, budgetId, tenantId, entservice.ResizeDepartmentBudgetInput{
			TotalQuota: req.TotalQuota,
			CycleQuota: req.CycleQuota,
		})
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionDepartmentBudgetResize,
			ObjectType:  entservice.AdminObjectDepartmentBudget,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: "Resized department budget pool",
			Payload:     buildDepartmentBudgetResizePayload(departmentId, budgetId, before, item, nil),
		})
	})
	if err != nil {
		auditDepartmentBudgetResizeReject(c, tenantId, departmentId, budgetId, req, err)
		writeDepartmentBudgetError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{
		Item: mapDepartmentBudgetItemDTO(item),
	})
}

func ListDepartmentBudgets(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	var req dtoenterprise.DepartmentBudgetListQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	result, err := entservice.NewDepartmentBudgetService(model.DB).ListByDepartment(departmentId, tenantId, entservice.DepartmentBudgetListQuery{
		SortBy:             req.SortBy,
		SortOrder:          req.SortOrder,
		IncludeDescendants: req.IncludeDescendants != nil && *req.IncludeDescendants,
	})
	if err != nil {
		writeDepartmentBudgetError(c, err)
		return
	}

	items := make([]dtoenterprise.DepartmentBudgetItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, *mapDepartmentBudgetItemDTO(item))
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetListResponse{
		Items: items,
		Thresholds: dtoenterprise.DepartmentBudgetThresholds{
			Warning:  result.Thresholds.Warning,
			Critical: result.Thresholds.Critical,
		},
		ScopeDepartmentId:   result.ScopeDepartmentId,
		ScopeDepartmentName: result.ScopeDepartmentName,
		IncludeDescendants:  result.IncludeDescendants,
		ScopeDepartmentIds:  append([]int{}, result.ScopeDepartmentIds...),
	})
}

func GetDepartmentBudgetDetail(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	budgetId, ok := parsePathInt(c, "budget_id")
	if !ok {
		return
	}
	tenantId, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return
	}

	result, err := entservice.NewDepartmentBudgetService(model.DB).GetDetail(departmentId, budgetId, tenantId)
	if err != nil {
		writeDepartmentBudgetError(c, err)
		return
	}

	wallets := make([]dtoenterprise.DepartmentBudgetWalletDetail, 0, len(result.Wallets))
	for _, wallet := range result.Wallets {
		wallets = append(wallets, dtoenterprise.DepartmentBudgetWalletDetail{
			AllocationId:             wallet.AllocationId,
			AllocationStatus:         wallet.AllocationStatus,
			TargetUserId:             wallet.TargetUserId,
			TargetUsername:           wallet.TargetUsername,
			TargetDisplayName:        wallet.TargetDisplayName,
			WalletId:                 wallet.WalletId,
			WalletStatus:             wallet.WalletStatus,
			Quota:                    wallet.Quota,
			RemainQuota:              wallet.RemainQuota,
			CycleType:                wallet.CycleType,
			CycleStartedAt:           wallet.CycleStartedAt,
			NextResetTime:            wallet.NextResetTime,
			ExpiresAt:                wallet.ExpiresAt,
			SourceAllocationId:       wallet.SourceAllocationId,
			SourceParentBudgetId:     wallet.SourceParentBudgetId,
			SourceParentBudgetType:   wallet.SourceParentBudgetType,
			SourceParentBudgetStatus: wallet.SourceParentBudgetStatus,
			CommittedQuota:           wallet.CommittedQuota,
			ProcessedAt:              wallet.ProcessedAt,
			CreatedAt:                wallet.CreatedAt,
			UpdatedAt:                wallet.UpdatedAt,
			Reason:                   wallet.Reason,
		})
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetDetailResponse{
		Budget:  mapDepartmentBudgetItemDTO(result.Budget),
		Wallets: wallets,
		Thresholds: dtoenterprise.DepartmentBudgetThresholds{
			Warning:  result.Thresholds.Warning,
			Critical: result.Thresholds.Critical,
		},
	})
}

func mutateDepartmentBudgetStatus(c *gin.Context, actionType string, diffSummary string, beforeStatus string, afterStatus string) {
	departmentId, budgetId, ok := parseDepartmentBudgetPath(c)
	if !ok {
		return
	}
	var req dtoenterprise.DepartmentBudgetLifecycleRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	var item entservice.DepartmentBudgetItem
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		svc := entservice.NewDepartmentBudgetService(tx)
		var err error
		if actionType == entservice.AdminActionDepartmentBudgetPause {
			item, err = svc.Pause(departmentId, budgetId, tenantId)
		} else {
			item, err = svc.Resume(departmentId, budgetId, tenantId)
		}
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  actionType,
			ObjectType:  entservice.AdminObjectDepartmentBudget,
			ObjectId:    strconv.Itoa(item.Id),
			DiffSummary: diffSummary,
			Payload: map[string]any{
				"department_id": departmentId,
				"budget_id":     budgetId,
				"type":          item.Type,
				"before_status": beforeStatus,
				"after_status":  afterStatus,
			},
		})
	})
	if err != nil {
		writeDepartmentBudgetError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentBudgetResponse{
		Item: mapDepartmentBudgetItemDTO(item),
	})
}

func parseDepartmentBudgetPath(c *gin.Context) (int, int, bool) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return 0, 0, false
	}
	budgetId, ok := parsePathInt(c, "budget_id")
	if !ok {
		return 0, 0, false
	}
	return departmentId, budgetId, true
}

func writeDepartmentBudgetError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrDepartmentNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentNotFound)
	case errors.Is(err, entservice.ErrQuotaAllocationBudgetNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetNotFound)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidType):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidType)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidQuota):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidQuota)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleQuota):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidCycleQuota)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleType):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidCycleType)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidCustomSeconds):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidCustomSeconds)
	case errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleStartedAt):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetInvalidCycleStartedAt)
	case errors.Is(err, entservice.ErrDepartmentBudgetTypeImmutable):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetTypeImmutable)
	case errors.Is(err, entservice.ErrDepartmentBudgetThresholdInvalid):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetThresholdInvalid)
	case errors.Is(err, entservice.ErrDepartmentBudgetStatusTransitionInvalid):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetStatusTransition)
	case errors.Is(err, entservice.ErrDepartmentBudgetResizeBelowCommitted):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentBudgetResizeBelowCommitted)
	case errors.Is(err, entservice.ErrInvalidDepartmentBudgetInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func auditDepartmentBudgetResizeReject(c *gin.Context, tenantId int, departmentId int, budgetId int, req dtoenterprise.ResizeDepartmentBudgetRequest, err error) {
	if !shouldAuditDepartmentBudgetResizeFailure(err) {
		return
	}
	_ = writeAdminAction(model.DB, c, entservice.AdminActionInput{
		TenantId:    tenantId,
		ActorId:     c.GetInt("id"),
		ActionType:  entservice.AdminActionDepartmentBudgetResizeReject,
		ObjectType:  entservice.AdminObjectDepartmentBudget,
		ObjectId:    strconv.Itoa(budgetId),
		DiffSummary: "Rejected department budget resize",
		Payload: map[string]any{
			"department_id":         departmentId,
			"budget_id":             budgetId,
			"requested_total_quota": int64PtrValueOrZero(req.TotalQuota),
			"requested_cycle_quota": int64PtrValueOrZero(req.CycleQuota),
			"error":                 err.Error(),
		},
	})
}

func shouldAuditDepartmentBudgetResizeFailure(err error) bool {
	return errors.Is(err, entservice.ErrDepartmentBudgetInvalidQuota) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleQuota) ||
		errors.Is(err, entservice.ErrDepartmentBudgetTypeImmutable) ||
		errors.Is(err, entservice.ErrDepartmentBudgetResizeBelowCommitted)
}

func buildDepartmentBudgetResizePayload(departmentId int, budgetId int, before *entservice.DepartmentBudgetItem, after entservice.DepartmentBudgetItem, err error) map[string]any {
	payload := map[string]any{
		"department_id": departmentId,
		"budget_id":     budgetId,
		"type":          after.Type,
	}
	if before != nil {
		payload["before_total_quota"] = before.TotalQuota
		payload["before_cycle_quota"] = before.CycleQuota
		payload["before_remaining"] = before.Remaining
	}
	payload["after_total_quota"] = after.TotalQuota
	payload["after_cycle_quota"] = after.CycleQuota
	payload["after_remaining"] = after.Remaining
	if err != nil {
		payload["error"] = err.Error()
	}
	return payload
}

func entmodelStatusActive() string {
	return "active"
}

func entmodelStatusPaused() string {
	return "paused"
}

func shouldAuditDepartmentBudgetFailure(err error) bool {
	return errors.Is(err, entservice.ErrDepartmentBudgetInvalidType) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidQuota) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleQuota) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleType) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCustomSeconds) ||
		errors.Is(err, entservice.ErrDepartmentBudgetInvalidCycleStartedAt) ||
		errors.Is(err, entservice.ErrDepartmentBudgetTypeImmutable) ||
		errors.Is(err, entservice.ErrDepartmentBudgetThresholdInvalid)
}

func mapDepartmentBudgetItemDTO(item entservice.DepartmentBudgetItem) *dtoenterprise.DepartmentBudgetItem {
	return &dtoenterprise.DepartmentBudgetItem{
		Id:             item.Id,
		TenantId:       item.TenantId,
		DepartmentId:   item.DepartmentId,
		DepartmentName: item.DepartmentName,
		ScopeType:      item.ScopeType,
		Name:           item.Name,
		IsPublic:       item.IsPublic,
		Type:           item.Type,
		Status:         item.Status,
		TotalQuota:     item.TotalQuota,
		Remaining:      item.Remaining,
		AllocatedTotal: item.AllocatedTotal,
		CycleQuota:     item.CycleQuota,
		CycleType:      item.CycleType,
		CycleStartedAt: item.CycleStartedAt,
		CustomSeconds:  item.CustomSeconds,
		ExpiresAt:      item.ExpiresAt,
		ParentStatus:   item.ParentStatus,
		UsageRatio:     item.UsageRatio,
		ThresholdState: string(item.ThresholdState),
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func mapDepartmentBudgetItemDTOOrNil(item *entservice.DepartmentBudgetItem) *dtoenterprise.DepartmentBudgetItem {
	if item == nil {
		return nil
	}
	return mapDepartmentBudgetItemDTO(*item)
}

func int64PtrValueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
