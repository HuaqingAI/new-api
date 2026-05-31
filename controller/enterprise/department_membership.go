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

func ListUserDepartments(c *gin.Context) {
	userId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	query, ok := parseMembershipQuery(c)
	if !ok {
		return
	}
	result, err := departmentMembershipService().ListUserDepartments(userId, query)
	if err != nil {
		writeMembershipError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.UserDepartmentsResponse{
		Items:        mapUserDepartmentItems(result.Items),
		Total:        result.Total,
		IsUnassigned: result.IsUnassigned,
	})
}

func ReplaceUserDepartments(c *gin.Context) {
	userId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	var req dtoenterprise.ReplaceUserDepartmentsRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	input := entservice.ReplaceUserDepartmentsInput{
		TenantId:        valueOrZero(req.TenantId),
		DepartmentIds:   nonNilInts(req.DepartmentIds),
		ExternalUserId:  req.ExternalUserId,
		ExternalSource:  req.ExternalSource,
		JoinedAt:        int64Value(req.JoinedAt),
		DeactivateStale: boolValue(req.DeactivateStale),
	}
	var result entservice.UserDepartmentsResult
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = entservice.NewDepartmentMembershipService(tx).ReplaceUserDepartments(userId, input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    input.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionMembershipReplace,
			ObjectType:  entservice.AdminObjectUserDepartment,
			ObjectId:    strconv.Itoa(userId),
			DiffSummary: "Replaced user department memberships",
			Payload: map[string]any{
				"user_id":          userId,
				"department_ids":   input.DepartmentIds,
				"external_source":  input.ExternalSource,
				"deactivate_stale": input.DeactivateStale,
			},
		})
	}); err != nil {
		writeMembershipError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.UserDepartmentsResponse{
		Items:        mapUserDepartmentItems(result.Items),
		Total:        result.Total,
		IsUnassigned: result.IsUnassigned,
	})
}

func ListDepartmentMembers(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	query, ok := parseMembershipQuery(c)
	if !ok {
		return
	}
	result, err := departmentMembershipService().ListDepartmentMembers(departmentId, query)
	if err != nil {
		writeMembershipError(c, err)
		return
	}

	common.ApiSuccess(c, dtoenterprise.DepartmentMembersResponse{
		Items: mapDepartmentMemberItems(result.Items),
		Total: result.Total,
	})
}

func AddDepartmentMember(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	var req dtoenterprise.AddDepartmentMemberRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	input := entservice.AddDepartmentMemberInput{
		TenantId:       tenantId,
		UserId:         req.UserId,
		ExternalUserId: req.ExternalUserId,
		ExternalSource: req.ExternalSource,
		JoinedAt:       int64Value(req.JoinedAt),
	}
	var item entservice.DepartmentMemberItem
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = entservice.NewDepartmentMembershipService(tx).AddDepartmentMember(departmentId, input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionMembershipAdd,
			ObjectType:  entservice.AdminObjectDepartmentMember,
			ObjectId:    entservice.MembershipObjectId(departmentId, item.UserId),
			DiffSummary: "Added department member",
			Payload: map[string]any{
				"user_id":         item.UserId,
				"department_id":   departmentId,
				"external_source": item.ExternalSource,
				"status":          item.Status,
			},
		})
	}); err != nil {
		writeMembershipError(c, err)
		return
	}

	common.ApiSuccess(c, mapDepartmentMemberItem(item))
}

func DeactivateDepartmentMember(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	userId, ok := parsePathInt(c, "user_id")
	if !ok {
		return
	}

	input, ok := parseOptionalStatusInput(c)
	if !ok {
		return
	}
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := entservice.NewDepartmentMembershipService(tx).DeactivateDepartmentMember(departmentId, userId, input); err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    input.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionMembershipDisable,
			ObjectType:  entservice.AdminObjectDepartmentMember,
			ObjectId:    entservice.MembershipObjectId(departmentId, userId),
			DiffSummary: "Disabled department member",
			Payload: map[string]any{
				"user_id":         userId,
				"department_id":   departmentId,
				"external_source": input.ExternalSource,
				"changed_at":      input.ChangedAt,
			},
		})
	}); err != nil {
		writeMembershipError(c, err)
		return
	}

	common.ApiSuccess(c, nil)
}

func RestoreDepartmentMember(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	userId, ok := parsePathInt(c, "user_id")
	if !ok {
		return
	}

	input, ok := parseOptionalStatusInput(c)
	if !ok {
		return
	}
	var item entservice.DepartmentMemberItem
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = entservice.NewDepartmentMembershipService(tx).RestoreDepartmentMember(departmentId, userId, input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    item.TenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionMembershipRestore,
			ObjectType:  entservice.AdminObjectDepartmentMember,
			ObjectId:    entservice.MembershipObjectId(departmentId, userId),
			DiffSummary: "Restored department member",
			Payload: map[string]any{
				"user_id":         userId,
				"department_id":   departmentId,
				"external_source": item.ExternalSource,
				"status":          item.Status,
			},
		})
	}); err != nil {
		writeMembershipError(c, err)
		return
	}

	common.ApiSuccess(c, mapDepartmentMemberItem(item))
}

func RenameDepartmentMember(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}
	userId, ok := parsePathInt(c, "user_id")
	if !ok {
		return
	}

	var req dtoenterprise.RenameDepartmentMemberRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	tenantId, ok := requestTenantId(c, req.TenantId)
	if !ok {
		return
	}

	input := entservice.RenameDepartmentMemberInput{
		TenantId:    tenantId,
		ActorId:     c.GetInt("id"),
		NewUsername: req.NewUsername,
		ChangedAt:   int64Value(req.ChangedAt),
	}
	var (
		item             entservice.DepartmentMemberItem
		previousUsername string
	)
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		item, previousUsername, err = entservice.NewDepartmentMembershipService(tx).RenameDepartmentMember(departmentId, userId, input)
		if err != nil {
			return err
		}
		return writeAdminAction(tx, c, entservice.AdminActionInput{
			TenantId:    tenantId,
			ActorId:     c.GetInt("id"),
			ActionType:  entservice.AdminActionMembershipRename,
			ObjectType:  entservice.AdminObjectDepartmentMember,
			ObjectId:    entservice.MembershipObjectId(departmentId, userId),
			DiffSummary: "Renamed department member username",
			Payload: map[string]any{
				"user_id":           userId,
				"department_id":     departmentId,
				"previous_username": previousUsername,
				"new_username":      item.Username,
			},
		})
	}); err != nil {
		writeMembershipError(c, err)
		return
	}

	common.ApiSuccess(c, mapDepartmentMemberItem(item))
}

func departmentMembershipService() *entservice.DepartmentMembershipService {
	return entservice.NewDepartmentMembershipService(model.DB)
}

func parsePathInt(c *gin.Context, name string) (int, bool) {
	value, err := strconv.Atoi(c.Param(name))
	if err != nil || value <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return 0, false
	}
	return value, true
}

func parseMembershipQuery(c *gin.Context) (entservice.MembershipQuery, bool) {
	var query dtoenterprise.DepartmentMembershipQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return entservice.MembershipQuery{}, false
	}
	return entservice.MembershipQuery{
		TenantId:       query.TenantId,
		Status:         query.Status,
		ExternalSource: query.ExternalSource,
		Page:           valueOrZero(query.Page),
		PageSize:       valueOrZero(query.PageSize),
	}, true
}

func parseOptionalStatusInput(c *gin.Context) (entservice.MembershipMutationInput, bool) {
	queryTenantId, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return entservice.MembershipMutationInput{}, false
	}
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return entservice.MembershipMutationInput{TenantId: queryTenantId}, true
	}
	var req dtoenterprise.MembershipStatusRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return entservice.MembershipMutationInput{}, false
	}
	tenantId := queryTenantId
	if c.Query("tenant_id") == "" && req.TenantId != nil {
		if *req.TenantId < 0 {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return entservice.MembershipMutationInput{}, false
		}
		tenantId = valueOrZero(req.TenantId)
	}
	return entservice.MembershipMutationInput{
		TenantId:       tenantId,
		ExternalSource: req.ExternalSource,
		ChangedAt:      int64Value(req.ChangedAt),
	}, true
}

func parseMembershipTenantIdQuery(c *gin.Context) (int, bool) {
	raw := c.Query("tenant_id")
	if raw == "" {
		return 0, true
	}
	tenantId, err := strconv.Atoi(raw)
	if err != nil || tenantId < 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return 0, false
	}
	return tenantId, true
}

func requestTenantId(c *gin.Context, bodyTenantId *int) (int, bool) {
	queryTenantId, ok := parseMembershipTenantIdQuery(c)
	if !ok {
		return 0, false
	}
	if c.Query("tenant_id") != "" {
		return queryTenantId, true
	}
	if bodyTenantId != nil && *bodyTenantId < 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return 0, false
	}
	return valueOrZero(bodyTenantId), true
}

func writeMembershipError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entservice.ErrDepartmentNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDepartmentNotFound)
	case errors.Is(err, entservice.ErrUserNotFound):
		common.ApiErrorI18n(c, i18n.MsgUserNotExists)
	case errors.Is(err, entservice.ErrMembershipNotFound):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseMembershipNotFound)
	case errors.Is(err, entservice.ErrMembershipAlreadyExists):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseMembershipAlreadyExists)
	case errors.Is(err, entservice.ErrDuplicateDepartment):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseDuplicateDepartment)
	case errors.Is(err, entservice.ErrEnterpriseUsernameInvalid):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseUsernameInvalid)
	case errors.Is(err, entservice.ErrEnterpriseUsernameExists):
		common.ApiErrorI18n(c, i18n.MsgEnterpriseUsernameExists)
	case errors.Is(err, entservice.ErrInvalidMembershipInput):
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	default:
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
	}
}

func mapUserDepartmentItems(items []entservice.UserDepartmentItem) []dtoenterprise.UserDepartmentItem {
	out := make([]dtoenterprise.UserDepartmentItem, 0, len(items))
	for _, item := range items {
		out = append(out, dtoenterprise.UserDepartmentItem(item))
	}
	return out
}

func mapDepartmentMemberItems(items []entservice.DepartmentMemberItem) []dtoenterprise.DepartmentMemberItem {
	out := make([]dtoenterprise.DepartmentMemberItem, 0, len(items))
	for _, item := range items {
		out = append(out, mapDepartmentMemberItem(item))
	}
	return out
}

func mapDepartmentMemberItem(item entservice.DepartmentMemberItem) dtoenterprise.DepartmentMemberItem {
	return dtoenterprise.DepartmentMemberItem(item)
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func nonNilInts(values []int) []int {
	if values == nil {
		return []int{}
	}
	return values
}

func writeAdminAction(db *gorm.DB, c *gin.Context, input entservice.AdminActionInput) error {
	return entservice.NewAdminActionService(db).Write(input)
}
