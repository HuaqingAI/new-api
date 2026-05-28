package enterprise

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type DepartmentMembershipService struct {
	db *gorm.DB
}

type MembershipQuery struct {
	TenantId       *int
	Status         *int
	ExternalSource string
	Page           int
	PageSize       int
}

type ReplaceUserDepartmentsInput struct {
	TenantId        int
	DepartmentIds   []int
	ExternalUserId  string
	ExternalSource  string
	JoinedAt        int64
	DeactivateStale bool
}

type AddDepartmentMemberInput struct {
	TenantId       int
	UserId         int
	ExternalUserId string
	ExternalSource string
	JoinedAt       int64
}

type MembershipMutationInput struct {
	TenantId       int
	ExternalSource string
	ChangedAt      int64
}

type UserDepartmentItem struct {
	Id             int    `json:"id"`
	TenantId       int    `json:"tenant_id"`
	UserId         int    `json:"user_id"`
	DepartmentId   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
	ExternalUserId string `json:"external_user_id"`
	ExternalSource string `json:"external_source"`
	Status         int    `json:"status"`
	JoinedAt       int64  `json:"joined_at"`
	LeftAt         int64  `json:"left_at"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type DepartmentMemberItem struct {
	Id             int    `json:"id"`
	TenantId       int    `json:"tenant_id"`
	UserId         int    `json:"user_id"`
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`
	DepartmentId   int    `json:"department_id"`
	ExternalUserId string `json:"external_user_id"`
	ExternalSource string `json:"external_source"`
	Status         int    `json:"status"`
	JoinedAt       int64  `json:"joined_at"`
	LeftAt         int64  `json:"left_at"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type UserDepartmentsResult struct {
	Items        []UserDepartmentItem `json:"items"`
	Total        int                  `json:"total"`
	IsUnassigned bool                 `json:"is_unassigned"`
}

type DepartmentMembersResult struct {
	Items []DepartmentMemberItem `json:"items"`
	Total int                    `json:"total"`
}

func NewDepartmentMembershipService(db *gorm.DB) *DepartmentMembershipService {
	return &DepartmentMembershipService{db: db}
}

func (s *DepartmentMembershipService) ListUserDepartments(userId int, query MembershipQuery) (UserDepartmentsResult, error) {
	if userId <= 0 {
		return emptyUserDepartments(true), ErrInvalidMembershipInput
	}
	if err := s.ensureUserExists(userId); err != nil {
		return emptyUserDepartments(true), err
	}

	var rows []struct {
		Id             int
		TenantId       int
		UserId         int
		DepartmentId   int
		DepartmentName string
		ExternalUserId string
		ExternalSource string
		Status         int
		JoinedAt       int64
		LeftAt         int64
		CreatedAt      int64
		UpdatedAt      int64
	}

	db := s.db.Table("enterprise_user_departments AS ud").
		Select("ud.id, ud.tenant_id, ud.user_id, ud.department_id, d.name AS department_name, ud.external_user_id, ud.external_source, ud.status, ud.joined_at, ud.left_at, ud.created_at, ud.updated_at").
		Joins("LEFT JOIN enterprise_departments AS d ON d.id = ud.department_id AND d.tenant_id = ud.tenant_id").
		Where("ud.user_id = ?", userId)
	if query.Status == nil {
		query.Status = intPtr(constant.EnterpriseMembershipStatusActive)
	}
	db = applyMembershipQuery(db, query, "ud")
	if err := db.Order("ud.department_id ASC").Scan(&rows).Error; err != nil {
		return emptyUserDepartments(true), err
	}

	items := make([]UserDepartmentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserDepartmentItem(row))
	}
	return UserDepartmentsResult{
		Items:        items,
		Total:        len(items),
		IsUnassigned: len(items) == 0,
	}, nil
}

func (s *DepartmentMembershipService) ListDepartmentMembers(departmentId int, query MembershipQuery) (DepartmentMembersResult, error) {
	if departmentId <= 0 {
		return DepartmentMembersResult{Items: []DepartmentMemberItem{}}, ErrInvalidMembershipInput
	}
	tenantId := tenantFromQuery(query)
	if err := s.ensureDepartmentExists(tenantId, departmentId); err != nil {
		return DepartmentMembersResult{Items: []DepartmentMemberItem{}}, err
	}

	var rows []struct {
		Id             int
		TenantId       int
		UserId         int
		Username       string
		DisplayName    string
		DepartmentId   int
		ExternalUserId string
		ExternalSource string
		Status         int
		JoinedAt       int64
		LeftAt         int64
		CreatedAt      int64
		UpdatedAt      int64
	}

	db := s.db.Table("enterprise_user_departments AS ud").
		Select("ud.id, ud.tenant_id, ud.user_id, u.username, u.display_name, ud.department_id, ud.external_user_id, ud.external_source, ud.status, ud.joined_at, ud.left_at, ud.created_at, ud.updated_at").
		Joins("LEFT JOIN users AS u ON u.id = ud.user_id").
		Where("ud.department_id = ?", departmentId)
	if query.Status == nil {
		query.Status = intPtr(constant.EnterpriseMembershipStatusActive)
	}
	db = applyMembershipQuery(db, query, "ud")
	if err := db.Order("ud.user_id ASC").Scan(&rows).Error; err != nil {
		return DepartmentMembersResult{Items: []DepartmentMemberItem{}}, err
	}

	items := make([]DepartmentMemberItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, DepartmentMemberItem(row))
	}
	return DepartmentMembersResult{Items: items, Total: len(items)}, nil
}

func (s *DepartmentMembershipService) ReplaceUserDepartments(userId int, input ReplaceUserDepartmentsInput) (UserDepartmentsResult, error) {
	if userId <= 0 {
		return emptyUserDepartments(true), ErrInvalidMembershipInput
	}
	if err := rejectDuplicateDepartmentIds(input.DepartmentIds); err != nil {
		return emptyUserDepartments(true), err
	}
	if err := s.ensureUserExists(userId); err != nil {
		return emptyUserDepartments(true), err
	}

	now := time.Now().Unix()
	source := normalizeExternalSource(input.ExternalSource)
	joinedAt := input.JoinedAt
	if joinedAt == 0 {
		joinedAt = now
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, departmentId := range input.DepartmentIds {
			if departmentId <= 0 {
				return ErrInvalidMembershipInput
			}
			if err := (&DepartmentMembershipService{db: tx}).ensureDepartmentExists(input.TenantId, departmentId); err != nil {
				return err
			}
		}

		if len(input.DepartmentIds) == 0 || input.DeactivateStale {
			update := map[string]any{
				"status":     constant.EnterpriseMembershipStatusInactive,
				"left_at":    now,
				"updated_at": now,
			}
			q := tx.Model(&entmodel.UserDepartment{}).
				Where("tenant_id = ? AND user_id = ? AND external_source = ?", input.TenantId, userId, source)
			if len(input.DepartmentIds) > 0 {
				q = q.Where("department_id NOT IN ?", input.DepartmentIds)
			}
			if err := q.Updates(update).Error; err != nil {
				return err
			}
		}

		for _, departmentId := range input.DepartmentIds {
			membership := entmodel.UserDepartment{
				TenantId:       input.TenantId,
				UserId:         userId,
				DepartmentId:   departmentId,
				ExternalUserId: input.ExternalUserId,
				ExternalSource: source,
				Status:         constant.EnterpriseMembershipStatusActive,
				JoinedAt:       joinedAt,
				LeftAt:         0,
			}
			if err := upsertMembership(tx, membership, true); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return emptyUserDepartments(true), err
	}

	return s.ListUserDepartments(userId, MembershipQuery{
		TenantId:       &input.TenantId,
		ExternalSource: source,
		Status:         intPtr(constant.EnterpriseMembershipStatusActive),
	})
}

func (s *DepartmentMembershipService) AddDepartmentMember(departmentId int, input AddDepartmentMemberInput) (DepartmentMemberItem, error) {
	if departmentId <= 0 || input.UserId <= 0 {
		return DepartmentMemberItem{}, ErrInvalidMembershipInput
	}
	if err := s.ensureUserExists(input.UserId); err != nil {
		return DepartmentMemberItem{}, err
	}
	if err := s.ensureDepartmentExists(input.TenantId, departmentId); err != nil {
		return DepartmentMemberItem{}, err
	}

	source := normalizeExternalSource(input.ExternalSource)
	joinedAt := input.JoinedAt
	if joinedAt == 0 {
		joinedAt = time.Now().Unix()
	}

	membership := entmodel.UserDepartment{
		TenantId:       input.TenantId,
		UserId:         input.UserId,
		DepartmentId:   departmentId,
		ExternalUserId: input.ExternalUserId,
		ExternalSource: source,
		Status:         constant.EnterpriseMembershipStatusActive,
		JoinedAt:       joinedAt,
		LeftAt:         0,
	}
	if err := upsertMembership(s.db, membership, false); err != nil {
		return DepartmentMemberItem{}, err
	}

	result, err := s.ListDepartmentMembers(departmentId, MembershipQuery{
		TenantId:       &input.TenantId,
		ExternalSource: source,
		Status:         intPtr(constant.EnterpriseMembershipStatusActive),
	})
	if err != nil {
		return DepartmentMemberItem{}, err
	}
	for _, item := range result.Items {
		if item.UserId == input.UserId {
			return item, nil
		}
	}
	return DepartmentMemberItem{}, ErrMembershipNotFound
}

func (s *DepartmentMembershipService) DeactivateDepartmentMember(departmentId int, userId int, input MembershipMutationInput) error {
	return s.updateMembershipStatus(departmentId, userId, input, constant.EnterpriseMembershipStatusInactive)
}

func (s *DepartmentMembershipService) RestoreDepartmentMember(departmentId int, userId int, input MembershipMutationInput) (DepartmentMemberItem, error) {
	if err := s.updateMembershipStatus(departmentId, userId, input, constant.EnterpriseMembershipStatusActive); err != nil {
		return DepartmentMemberItem{}, err
	}
	result, err := s.ListDepartmentMembers(departmentId, MembershipQuery{
		TenantId:       &input.TenantId,
		ExternalSource: normalizeExternalSource(input.ExternalSource),
		Status:         intPtr(constant.EnterpriseMembershipStatusActive),
	})
	if err != nil {
		return DepartmentMemberItem{}, err
	}
	for _, item := range result.Items {
		if item.UserId == userId {
			return item, nil
		}
	}
	return DepartmentMemberItem{}, ErrMembershipNotFound
}

func (s *DepartmentMembershipService) updateMembershipStatus(departmentId int, userId int, input MembershipMutationInput, status int) error {
	if departmentId <= 0 || userId <= 0 {
		return ErrInvalidMembershipInput
	}
	changedAt := input.ChangedAt
	if changedAt == 0 {
		changedAt = time.Now().Unix()
	}
	update := map[string]any{
		"status":     status,
		"updated_at": changedAt,
	}
	if status == constant.EnterpriseMembershipStatusActive {
		update["left_at"] = int64(0)
	} else {
		update["left_at"] = changedAt
	}

	tx := s.db.Model(&entmodel.UserDepartment{}).
		Where("tenant_id = ? AND user_id = ? AND department_id = ? AND external_source = ?", input.TenantId, userId, departmentId, normalizeExternalSource(input.ExternalSource)).
		Updates(update)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrMembershipNotFound
	}
	return nil
}

func (s *DepartmentMembershipService) ensureUserExists(userId int) error {
	var count int64
	if err := s.db.Model(&model.User{}).Where("id = ?", userId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *DepartmentMembershipService) ensureDepartmentExists(tenantId int, departmentId int) error {
	var count int64
	if err := s.db.Model(&entmodel.Department{}).Where("tenant_id = ? AND id = ?", tenantId, departmentId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrDepartmentNotFound
	}
	return nil
}

func upsertMembership(db *gorm.DB, membership entmodel.UserDepartment, allowExisting bool) error {
	var existing entmodel.UserDepartment
	err := db.Where(
		"tenant_id = ? AND user_id = ? AND department_id = ? AND external_source = ?",
		membership.TenantId,
		membership.UserId,
		membership.DepartmentId,
		membership.ExternalSource,
	).First(&existing).Error
	if err == nil {
		if !allowExisting && existing.Status == constant.EnterpriseMembershipStatusActive {
			return ErrMembershipAlreadyExists
		}
		return db.Model(&existing).Updates(map[string]any{
			"external_user_id": membership.ExternalUserId,
			"status":           constant.EnterpriseMembershipStatusActive,
			"joined_at":        membership.JoinedAt,
			"left_at":          int64(0),
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := db.Create(&membership).Error; err != nil {
		return err
	}
	return nil
}

func applyMembershipQuery(db *gorm.DB, query MembershipQuery, alias string) *gorm.DB {
	if query.TenantId != nil {
		db = db.Where(alias+".tenant_id = ?", *query.TenantId)
	}
	if query.Status != nil {
		db = db.Where(alias+".status = ?", *query.Status)
	}
	if query.ExternalSource != "" {
		db = db.Where(alias+".external_source = ?", query.ExternalSource)
	}
	if query.PageSize > 0 {
		page := query.Page
		if page < 1 {
			page = 1
		}
		db = db.Limit(query.PageSize).Offset((page - 1) * query.PageSize)
	}
	return db
}

func rejectDuplicateDepartmentIds(ids []int) error {
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return ErrInvalidMembershipInput
		}
		if _, ok := seen[id]; ok {
			return ErrDuplicateDepartment
		}
		seen[id] = struct{}{}
	}
	return nil
}

func emptyUserDepartments(isUnassigned bool) UserDepartmentsResult {
	return UserDepartmentsResult{Items: []UserDepartmentItem{}, IsUnassigned: isUnassigned}
}

func normalizeExternalSource(source string) string {
	if source == "" {
		return constant.EnterpriseExternalSourceManual
	}
	return source
}

func tenantFromQuery(query MembershipQuery) int {
	if query.TenantId == nil {
		return 0
	}
	return *query.TenantId
}

func intPtr(v int) *int {
	return &v
}
