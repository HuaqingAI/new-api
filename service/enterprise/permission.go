package enterprise

import (
	"errors"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

type PermissionService struct {
	db *gorm.DB
}

const readableUserDepartmentBatchSize = 500

type DepartmentAdminRoleInput struct {
	TenantId     int
	UserId       int
	DepartmentId int
}

type DepartmentOwnerResolution struct {
	TenantId         int
	DepartmentId     int
	Facts            []DepartmentOwnerFact
	EffectiveOwners  []EffectiveDepartmentOwner
	OwnerCount       int
	Fallback         string
	LocalDenyUserIds map[int]struct{}
}

type DepartmentOwnerFact struct {
	Id                        int
	TenantId                  int
	UserId                    int
	DepartmentId              int
	Role                      int
	Source                    string
	Effect                    string
	ExternalSource            string
	Status                    int
	InheritedFromDepartmentId int
	CreatedAt                 int64
	UpdatedAt                 int64
}

type EffectiveDepartmentOwner struct {
	UserId                    int
	DepartmentId              int
	Source                    string
	Effect                    string
	InheritedFromDepartmentId int
	RoleFactId                int
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

func (s *PermissionService) GrantDepartmentAdmin(input DepartmentAdminRoleInput) (entmodel.DepartmentRole, error) {
	if input.UserId <= 0 || input.DepartmentId <= 0 {
		return entmodel.DepartmentRole{}, ErrInvalidAdminActionInput
	}
	if err := ensureEnterpriseUserExists(s.db, input.UserId); err != nil {
		return entmodel.DepartmentRole{}, err
	}
	if err := ensureEnterpriseDepartmentExists(s.db, input.TenantId, input.DepartmentId); err != nil {
		return entmodel.DepartmentRole{}, err
	}

	return s.upsertDepartmentRoleFact(input, constant.EnterpriseDepartmentRoleSourceManualGrant, constant.EnterpriseDepartmentRoleEffectAllow, "")
}

func (s *PermissionService) RevokeDepartmentAdmin(input DepartmentAdminRoleInput) error {
	return s.revokeDepartmentRoleFact(input, constant.EnterpriseDepartmentRoleSourceManualGrant)
}

func (s *PermissionService) DenyDepartmentOwner(input DepartmentAdminRoleInput) (entmodel.DepartmentRole, error) {
	if input.UserId <= 0 || input.DepartmentId <= 0 {
		return entmodel.DepartmentRole{}, ErrInvalidAdminActionInput
	}
	if err := ensureEnterpriseUserExists(s.db, input.UserId); err != nil {
		return entmodel.DepartmentRole{}, err
	}
	if err := ensureEnterpriseDepartmentExists(s.db, input.TenantId, input.DepartmentId); err != nil {
		return entmodel.DepartmentRole{}, err
	}
	return s.upsertDepartmentRoleFact(input, constant.EnterpriseDepartmentRoleSourceManualDenyOverride, constant.EnterpriseDepartmentRoleEffectDeny, "")
}

func (s *PermissionService) RevokeDepartmentOwnerDeny(input DepartmentAdminRoleInput) error {
	return s.revokeDepartmentRoleFact(input, constant.EnterpriseDepartmentRoleSourceManualDenyOverride)
}

func (s *PermissionService) UpsertDingTalkDepartmentOwner(input DepartmentAdminRoleInput) (entmodel.DepartmentRole, error) {
	if input.UserId <= 0 || input.DepartmentId <= 0 {
		return entmodel.DepartmentRole{}, ErrInvalidAdminActionInput
	}
	return s.upsertDepartmentRoleFact(input, constant.EnterpriseDepartmentRoleSourceDingTalkOwner, constant.EnterpriseDepartmentRoleEffectAllow, constant.EnterpriseExternalSourceDingTalk)
}

func (s *PermissionService) DeactivateDingTalkDepartmentOwner(input DepartmentAdminRoleInput) error {
	return s.revokeDepartmentRoleFact(input, constant.EnterpriseDepartmentRoleSourceDingTalkOwner)
}

func (s *PermissionService) upsertDepartmentRoleFact(input DepartmentAdminRoleInput, source string, effect string, externalSource string) (entmodel.DepartmentRole, error) {
	var role entmodel.DepartmentRole
	err := s.db.Where(
		"tenant_id = ? AND user_id = ? AND department_id = ? AND role = ? AND source = ?",
		input.TenantId,
		input.UserId,
		input.DepartmentId,
		constant.EnterpriseDepartmentRoleDeptAdmin,
		source,
	).First(&role).Error
	if err == nil {
		updates := map[string]any{
			"effect":          effect,
			"external_source": externalSource,
			"status":          constant.EnterpriseDepartmentRoleStatusActive,
			"updated_at":      time.Now().Unix(),
		}
		if err := s.db.Model(&role).Updates(updates).Error; err != nil {
			return entmodel.DepartmentRole{}, err
		}
		role.Effect = effect
		role.ExternalSource = externalSource
		role.Status = constant.EnterpriseDepartmentRoleStatusActive
		return role, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return entmodel.DepartmentRole{}, err
	}

	role = entmodel.DepartmentRole{
		TenantId:       input.TenantId,
		UserId:         input.UserId,
		DepartmentId:   input.DepartmentId,
		Role:           constant.EnterpriseDepartmentRoleDeptAdmin,
		Source:         source,
		Effect:         effect,
		ExternalSource: externalSource,
		Status:         constant.EnterpriseDepartmentRoleStatusActive,
	}
	if err := s.db.Create(&role).Error; err != nil {
		return entmodel.DepartmentRole{}, err
	}
	return role, nil
}

func (s *PermissionService) revokeDepartmentRoleFact(input DepartmentAdminRoleInput, source string) error {
	if input.UserId <= 0 || input.DepartmentId <= 0 {
		return ErrInvalidAdminActionInput
	}
	tx := s.db.Model(&entmodel.DepartmentRole{}).Where(
		"tenant_id = ? AND user_id = ? AND department_id = ? AND role = ? AND source = ? AND status = ?",
		input.TenantId,
		input.UserId,
		input.DepartmentId,
		constant.EnterpriseDepartmentRoleDeptAdmin,
		source,
		constant.EnterpriseDepartmentRoleStatusActive,
	).Updates(map[string]any{
		"status":     constant.EnterpriseDepartmentRoleStatusInactive,
		"updated_at": time.Now().Unix(),
	})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrDepartmentRoleNotFound
	}
	return nil
}

func (s *PermissionService) ResolveEffectiveDepartmentOwners(tenantId int, departmentId int) (DepartmentOwnerResolution, error) {
	if departmentId <= 0 {
		return DepartmentOwnerResolution{TenantId: tenantId, DepartmentId: departmentId, Facts: []DepartmentOwnerFact{}, EffectiveOwners: []EffectiveDepartmentOwner{}, Fallback: "admin"}, ErrDepartmentNotFound
	}
	ids, inheritedFrom, err := s.departmentLineage(tenantId, departmentId)
	if err != nil {
		return DepartmentOwnerResolution{}, err
	}
	var roles []entmodel.DepartmentRole
	if err := s.db.Where(
		"tenant_id = ? AND department_id IN ? AND role = ?",
		tenantId,
		ids,
		constant.EnterpriseDepartmentRoleDeptAdmin,
	).Find(&roles).Error; err != nil {
		return DepartmentOwnerResolution{}, err
	}

	resolution := DepartmentOwnerResolution{
		TenantId:         tenantId,
		DepartmentId:     departmentId,
		Facts:            make([]DepartmentOwnerFact, 0, len(roles)),
		EffectiveOwners:  []EffectiveDepartmentOwner{},
		Fallback:         "admin",
		LocalDenyUserIds: map[int]struct{}{},
	}
	activeByUser := map[int][]entmodel.DepartmentRole{}
	for _, role := range roles {
		if role.Source == "" {
			role.Source = constant.EnterpriseDepartmentRoleSourceManualGrant
		}
		if role.Effect == "" {
			if role.Source == constant.EnterpriseDepartmentRoleSourceManualDenyOverride {
				role.Effect = constant.EnterpriseDepartmentRoleEffectDeny
			} else {
				role.Effect = constant.EnterpriseDepartmentRoleEffectAllow
			}
		}
		resolution.Facts = append(resolution.Facts, mapDepartmentOwnerFact(role))
		if role.DepartmentId == departmentId && role.Status == constant.EnterpriseDepartmentRoleStatusActive && role.Source == constant.EnterpriseDepartmentRoleSourceManualDenyOverride && role.Effect == constant.EnterpriseDepartmentRoleEffectDeny {
			resolution.LocalDenyUserIds[role.UserId] = struct{}{}
			continue
		}
		if role.Status != constant.EnterpriseDepartmentRoleStatusActive || role.Effect != constant.EnterpriseDepartmentRoleEffectAllow {
			continue
		}
		if role.Source != constant.EnterpriseDepartmentRoleSourceManualGrant && role.Source != constant.EnterpriseDepartmentRoleSourceDingTalkOwner {
			continue
		}
		activeByUser[role.UserId] = append(activeByUser[role.UserId], role)
	}
	sort.SliceStable(resolution.Facts, func(i, j int) bool {
		if resolution.Facts[i].DepartmentId != resolution.Facts[j].DepartmentId {
			return inheritedFrom[resolution.Facts[i].DepartmentId] < inheritedFrom[resolution.Facts[j].DepartmentId]
		}
		if resolution.Facts[i].UserId != resolution.Facts[j].UserId {
			return resolution.Facts[i].UserId < resolution.Facts[j].UserId
		}
		return ownerSourcePrecedence(resolution.Facts[i].Source) < ownerSourcePrecedence(resolution.Facts[j].Source)
	})

	userIds := make([]int, 0, len(activeByUser))
	for userId := range activeByUser {
		if _, denied := resolution.LocalDenyUserIds[userId]; denied {
			continue
		}
		userIds = append(userIds, userId)
	}
	sort.Ints(userIds)
	for _, userId := range userIds {
		best := bestEffectiveOwnerFact(activeByUser[userId], inheritedFrom)
		resolution.EffectiveOwners = append(resolution.EffectiveOwners, EffectiveDepartmentOwner{
			UserId:                    userId,
			DepartmentId:              departmentId,
			Source:                    best.Source,
			Effect:                    best.Effect,
			InheritedFromDepartmentId: best.DepartmentId,
			RoleFactId:                best.Id,
		})
	}
	resolution.OwnerCount = len(resolution.EffectiveOwners)
	if resolution.OwnerCount == 0 {
		return resolution, ErrDepartmentOwnerNotFound
	}
	return resolution, nil
}

func (s *PermissionService) CanManageDepartment(userId int, tenantId int, departmentId int) (bool, error) {
	return s.CanGovernDepartment(userId, tenantId, departmentId)
}

func (s *PermissionService) CanGovernDepartment(userId int, tenantId int, departmentId int) (bool, error) {
	if userId <= 0 || departmentId <= 0 {
		return false, nil
	}
	resolution, err := s.ResolveEffectiveDepartmentOwners(tenantId, departmentId)
	if err != nil {
		if errors.Is(err, ErrDepartmentOwnerNotFound) {
			if _, denied := resolution.LocalDenyUserIds[userId]; denied {
				return false, ErrDepartmentOwnerDeniedByLocalRule
			}
			return false, nil
		}
		return false, err
	}
	if _, denied := resolution.LocalDenyUserIds[userId]; denied {
		return false, ErrDepartmentOwnerDeniedByLocalRule
	}
	for _, owner := range resolution.EffectiveOwners {
		if owner.UserId == userId {
			return true, nil
		}
	}
	return false, nil
}

func (s *PermissionService) ListManageableDepartmentIds(userId int, tenantId int) ([]int, error) {
	if userId <= 0 {
		return []int{}, nil
	}

	var roles []entmodel.DepartmentRole
	if err := s.db.Where(
		"tenant_id = ? AND user_id = ? AND role = ? AND status = ?",
		tenantId,
		userId,
		constant.EnterpriseDepartmentRoleDeptAdmin,
		constant.EnterpriseDepartmentRoleStatusActive,
	).Find(&roles).Error; err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return []int{}, nil
	}

	var departments []entmodel.Department
	if err := s.db.Where("tenant_id = ?", tenantId).Find(&departments).Error; err != nil {
		return nil, err
	}

	existing := make(map[int]struct{}, len(departments))
	childrenByParent := make(map[int][]int, len(departments))
	for _, department := range departments {
		existing[department.Id] = struct{}{}
		if department.ParentId != nil {
			childrenByParent[*department.ParentId] = append(childrenByParent[*department.ParentId], department.Id)
		}
	}

	manageable := make(map[int]struct{})
	for _, role := range roles {
		if role.Source == "" {
			role.Source = constant.EnterpriseDepartmentRoleSourceManualGrant
		}
		if role.Effect == "" {
			role.Effect = constant.EnterpriseDepartmentRoleEffectAllow
		}
		if role.Source != constant.EnterpriseDepartmentRoleSourceManualGrant && role.Source != constant.EnterpriseDepartmentRoleSourceDingTalkOwner {
			continue
		}
		if _, ok := existing[role.DepartmentId]; !ok {
			continue
		}
		addDepartmentAndDescendants(role.DepartmentId, childrenByParent, manageable, map[int]bool{})
	}
	var deniedRoles []entmodel.DepartmentRole
	if err := s.db.Where(
		"tenant_id = ? AND user_id = ? AND role = ? AND source = ? AND effect = ? AND status = ?",
		tenantId,
		userId,
		constant.EnterpriseDepartmentRoleDeptAdmin,
		constant.EnterpriseDepartmentRoleSourceManualDenyOverride,
		constant.EnterpriseDepartmentRoleEffectDeny,
		constant.EnterpriseDepartmentRoleStatusActive,
	).Find(&deniedRoles).Error; err != nil {
		return nil, err
	}
	for _, role := range deniedRoles {
		delete(manageable, role.DepartmentId)
	}

	ids := make([]int, 0, len(manageable))
	for id := range manageable {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

func (s *PermissionService) ReadableUserDepartmentIds(actorUserId int, targetUserId int, tenantId int) ([]int, error) {
	if actorUserId <= 0 || targetUserId <= 0 {
		return []int{}, nil
	}
	if actorUserId == targetUserId {
		return []int{}, nil
	}

	manageableIds, err := s.ListManageableDepartmentIds(actorUserId, tenantId)
	if err != nil {
		return nil, err
	}
	if len(manageableIds) == 0 {
		return []int{}, nil
	}

	memberships := []entmodel.UserDepartment{}
	for start := 0; start < len(manageableIds); start += readableUserDepartmentBatchSize {
		end := start + readableUserDepartmentBatchSize
		if end > len(manageableIds) {
			end = len(manageableIds)
		}
		var batch []entmodel.UserDepartment
		if err := s.db.Select("department_id").
			Where("tenant_id = ? AND user_id = ? AND status = ? AND department_id IN ?", tenantId, targetUserId, constant.EnterpriseMembershipStatusActive, manageableIds[start:end]).
			Find(&batch).Error; err != nil {
			return nil, err
		}
		memberships = append(memberships, batch...)
	}
	departmentIds := make([]int, 0, len(memberships))
	seen := make(map[int]struct{}, len(memberships))
	for _, membership := range memberships {
		if _, ok := seen[membership.DepartmentId]; ok {
			continue
		}
		seen[membership.DepartmentId] = struct{}{}
		departmentIds = append(departmentIds, membership.DepartmentId)
	}
	sort.Ints(departmentIds)
	return departmentIds, nil
}

func (s *PermissionService) departmentLineage(tenantId int, departmentId int) ([]int, map[int]int, error) {
	var departments []entmodel.Department
	if err := s.db.Where("tenant_id = ?", tenantId).Find(&departments).Error; err != nil {
		return nil, nil, err
	}
	parentById := make(map[int]*int, len(departments))
	for _, department := range departments {
		parentById[department.Id] = department.ParentId
	}
	if _, ok := parentById[departmentId]; !ok {
		return nil, nil, ErrDepartmentNotFound
	}
	ids := []int{}
	inheritedFrom := map[int]int{}
	seen := map[int]bool{}
	current := departmentId
	distance := 0
	for current > 0 {
		if seen[current] {
			break
		}
		seen[current] = true
		ids = append(ids, current)
		inheritedFrom[current] = distance
		parent, ok := parentById[current]
		if !ok || parent == nil {
			break
		}
		current = *parent
		distance++
	}
	return ids, inheritedFrom, nil
}

func mapDepartmentOwnerFact(role entmodel.DepartmentRole) DepartmentOwnerFact {
	return DepartmentOwnerFact{
		Id:                        role.Id,
		TenantId:                  role.TenantId,
		UserId:                    role.UserId,
		DepartmentId:              role.DepartmentId,
		Role:                      role.Role,
		Source:                    role.Source,
		Effect:                    role.Effect,
		ExternalSource:            role.ExternalSource,
		Status:                    role.Status,
		InheritedFromDepartmentId: role.DepartmentId,
		CreatedAt:                 role.CreatedAt,
		UpdatedAt:                 role.UpdatedAt,
	}
}

func bestEffectiveOwnerFact(roles []entmodel.DepartmentRole, inheritedFrom map[int]int) entmodel.DepartmentRole {
	sort.SliceStable(roles, func(i, j int) bool {
		leftPrecedence := ownerSourcePrecedence(roles[i].Source)
		rightPrecedence := ownerSourcePrecedence(roles[j].Source)
		if leftPrecedence != rightPrecedence {
			return leftPrecedence < rightPrecedence
		}
		leftDistance := inheritedFrom[roles[i].DepartmentId]
		rightDistance := inheritedFrom[roles[j].DepartmentId]
		if leftDistance != rightDistance {
			return leftDistance < rightDistance
		}
		return roles[i].Id < roles[j].Id
	})
	return roles[0]
}

func ownerSourcePrecedence(source string) int {
	switch source {
	case constant.EnterpriseDepartmentRoleSourceManualDenyOverride:
		return 0
	case constant.EnterpriseDepartmentRoleSourceManualGrant:
		return 1
	case constant.EnterpriseDepartmentRoleSourceDingTalkOwner:
		return 2
	default:
		return 99
	}
}

func addDepartmentAndDescendants(id int, childrenByParent map[int][]int, out map[int]struct{}, path map[int]bool) {
	if path[id] {
		return
	}
	path[id] = true
	out[id] = struct{}{}
	for _, childId := range childrenByParent[id] {
		addDepartmentAndDescendants(childId, childrenByParent, out, path)
	}
	delete(path, id)
}

func ensureEnterpriseUserExists(db *gorm.DB, userId int) error {
	var count int64
	if err := db.Model(&model.User{}).Where("id = ?", userId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrUserNotFound
	}
	return nil
}

func ensureEnterpriseDepartmentExists(db *gorm.DB, tenantId int, departmentId int) error {
	var count int64
	if err := db.Model(&entmodel.Department{}).Where("tenant_id = ? AND id = ?", tenantId, departmentId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrDepartmentNotFound
	}
	return nil
}
