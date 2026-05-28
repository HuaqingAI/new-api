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
)

func ListUserDepartments(c *gin.Context) {
	userId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	query := parseMembershipQuery(c)
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
	result, err := departmentMembershipService().ReplaceUserDepartments(userId, input)
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

func ListDepartmentMembers(c *gin.Context) {
	departmentId, ok := parsePathInt(c, "id")
	if !ok {
		return
	}

	query := parseMembershipQuery(c)
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

	item, err := departmentMembershipService().AddDepartmentMember(departmentId, entservice.AddDepartmentMemberInput{
		TenantId:       valueOrZero(req.TenantId),
		UserId:         req.UserId,
		ExternalUserId: req.ExternalUserId,
		ExternalSource: req.ExternalSource,
		JoinedAt:       int64Value(req.JoinedAt),
	})
	if err != nil {
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
	if err := departmentMembershipService().DeactivateDepartmentMember(departmentId, userId, input); err != nil {
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
	item, err := departmentMembershipService().RestoreDepartmentMember(departmentId, userId, input)
	if err != nil {
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

func parseMembershipQuery(c *gin.Context) entservice.MembershipQuery {
	var query dtoenterprise.DepartmentMembershipQuery
	_ = c.ShouldBindQuery(&query)
	return entservice.MembershipQuery{
		TenantId:       query.TenantId,
		Status:         query.Status,
		ExternalSource: query.ExternalSource,
		Page:           valueOrZero(query.Page),
		PageSize:       valueOrZero(query.PageSize),
	}
}

func parseOptionalStatusInput(c *gin.Context) (entservice.MembershipMutationInput, bool) {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return entservice.MembershipMutationInput{}, true
	}
	var req dtoenterprise.MembershipStatusRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return entservice.MembershipMutationInput{}, false
	}
	return entservice.MembershipMutationInput{
		TenantId:       valueOrZero(req.TenantId),
		ExternalSource: req.ExternalSource,
		ChangedAt:      int64Value(req.ChangedAt),
	}, true
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
