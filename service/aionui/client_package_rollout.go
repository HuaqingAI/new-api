package aionui

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

func normalizeClientPackageRolloutMode(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		return apmodel.ClientPackageRolloutModeGlobal
	}
	return mode
}

func normalizeClientPackageScopeInputs(scopes []ClientPackageScopeInput) []ClientPackageScopeInput {
	if len(scopes) == 0 {
		return []ClientPackageScopeInput{}
	}
	result := make([]ClientPackageScopeInput, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, ClientPackageScopeInput{
			SubjectType: strings.TrimSpace(strings.ToLower(scope.SubjectType)),
			SubjectId:   strings.TrimSpace(scope.SubjectId),
		})
	}
	return result
}

func validateClientPackageRollout(mode string, scopes []ClientPackageScopeInput) error {
	mode = normalizeClientPackageRolloutMode(mode)
	scopes = normalizeClientPackageScopeInputs(scopes)
	switch mode {
	case apmodel.ClientPackageRolloutModeGlobal:
		if len(scopes) != 0 {
			return ErrClientPackageRolloutInvalid
		}
		return nil
	case apmodel.ClientPackageRolloutModeTargeted:
		if len(scopes) == 0 {
			return ErrClientPackageRolloutInvalid
		}
	default:
		return ErrClientPackageRolloutInvalid
	}
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if scope.SubjectType != apmodel.GrantSubjectTypeUser && scope.SubjectType != apmodel.GrantSubjectTypeDepartment {
			return ErrClientPackageRolloutInvalid
		}
		id, err := strconv.Atoi(scope.SubjectId)
		if err != nil || id <= 0 {
			return ErrClientPackageRolloutInvalid
		}
		key := scope.SubjectType + "\x00" + scope.SubjectId
		if _, exists := seen[key]; exists {
			return ErrClientPackageRolloutInvalid
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateClientPackagePublishedRollout(mode string, scopes []ClientPackageScopeInput) error {
	if err := validateClientPackageRollout(mode, scopes); err != nil {
		return err
	}
	if normalizeClientPackageRolloutMode(mode) == apmodel.ClientPackageRolloutModeTargeted && ClientUpdateAccessMode() != ClientUpdateAccessModeEnforced {
		return ErrClientPackageRolloutInvalid
	}
	return nil
}

func (s *ClientPackageService) createClientPackageWithScopes(pkg *apmodel.ClientPackage, scopes []ClientPackageScopeInput) error {
	if s == nil || s.db == nil || pkg == nil {
		return ErrClientPackageInvalidInput
	}
	if len(scopes) == 0 {
		return s.db.Create(pkg).Error
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pkg).Error; err != nil {
			return err
		}
		return s.replaceScopes(tx, pkg.Id, scopes, pkg.CreatedBy)
	})
}

func (s *ClientPackageService) replaceScopes(tx *gorm.DB, packageID int, scopes []ClientPackageScopeInput, actorUserID int) error {
	if packageID <= 0 || actorUserID <= 0 {
		return ErrClientPackageInvalidInput
	}
	if err := validateClientPackageRollout(apmodel.ClientPackageRolloutModeTargeted, scopes); err != nil && len(scopes) > 0 {
		return err
	}
	if err := tx.Where("client_package_id = ?", packageID).Delete(&apmodel.ClientPackageScope{}).Error; err != nil {
		return err
	}
	if len(scopes) == 0 {
		return nil
	}
	records := make([]apmodel.ClientPackageScope, 0, len(scopes))
	for _, scope := range scopes {
		id, _ := strconv.Atoi(scope.SubjectId)
		switch scope.SubjectType {
		case apmodel.GrantSubjectTypeUser:
			var user model.User
			if err := tx.First(&user, id).Error; err != nil {
				return ErrClientPackageRolloutInvalid
			}
			if user.Status != common.UserStatusEnabled {
				return ErrClientPackageRolloutInvalid
			}
		case apmodel.GrantSubjectTypeDepartment:
			var department entmodel.Department
			if err := tx.Where("id = ? AND status = ?", id, constant.DepartmentStatusEnabled).First(&department).Error; err != nil {
				return ErrClientPackageRolloutInvalid
			}
		}
		records = append(records, apmodel.ClientPackageScope{
			ClientPackageId: packageID,
			SubjectType:     scope.SubjectType,
			SubjectId:       scope.SubjectId,
			CreatedBy:       actorUserID,
		})
	}
	return tx.Create(&records).Error
}

func (s *ClientPackageService) listScopes(packageID int) ([]ClientPackageScopeInput, error) {
	if s == nil || s.db == nil || packageID <= 0 {
		return nil, ErrClientPackageInvalidInput
	}
	var rows []apmodel.ClientPackageScope
	if err := s.db.Where("client_package_id = ?", packageID).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]ClientPackageScopeInput, 0, len(rows))
	for _, row := range rows {
		result = append(result, ClientPackageScopeInput{SubjectType: row.SubjectType, SubjectId: row.SubjectId})
	}
	return result, nil
}

func (s *ClientPackageService) GetRollout(id int) (dtoaionui.ClientPackageRolloutRequest, error) {
	if s == nil || s.db == nil || id <= 0 {
		return dtoaionui.ClientPackageRolloutRequest{}, ErrClientPackageInvalidInput
	}
	var pkg apmodel.ClientPackage
	if err := s.db.First(&pkg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dtoaionui.ClientPackageRolloutRequest{}, ErrClientPackageNotFound
		}
		return dtoaionui.ClientPackageRolloutRequest{}, err
	}
	scopes, err := s.listScopes(pkg.Id)
	if err != nil {
		return dtoaionui.ClientPackageRolloutRequest{}, err
	}
	response := dtoaionui.ClientPackageRolloutRequest{
		RolloutMode: normalizeClientPackageRolloutMode(pkg.RolloutMode),
		Scopes:      make([]dtoaionui.ClientPackageScopeInput, 0, len(scopes)),
	}
	for _, scope := range scopes {
		response.Scopes = append(response.Scopes, dtoaionui.ClientPackageScopeInput{SubjectType: scope.SubjectType, SubjectId: scope.SubjectId})
	}
	return response, nil
}

func (s *ClientPackageService) UpdateRollout(id int, mode string, scopes []ClientPackageScopeInput, actorUserID int) (dtoaionui.ClientPackageRolloutRequest, error) {
	if s == nil || s.db == nil || id <= 0 || actorUserID <= 0 {
		return dtoaionui.ClientPackageRolloutRequest{}, ErrClientPackageInvalidInput
	}
	mode = normalizeClientPackageRolloutMode(mode)
	scopes = normalizeClientPackageScopeInputs(scopes)
	if err := validateClientPackageRollout(mode, scopes); err != nil {
		return dtoaionui.ClientPackageRolloutRequest{}, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var pkg apmodel.ClientPackage
		if err := tx.First(&pkg, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClientPackageNotFound
			}
			return err
		}
		if pkg.Status == apmodel.ClientPackageStatusPublished && mode == apmodel.ClientPackageRolloutModeTargeted && ClientUpdateAccessMode() != ClientUpdateAccessModeEnforced {
			return ErrClientPackageRolloutInvalid
		}
		if err := tx.Model(&apmodel.ClientPackage{}).Where("id = ?", id).Update("rollout_mode", mode).Error; err != nil {
			return err
		}
		return s.replaceScopes(tx, id, scopes, actorUserID)
	}); err != nil {
		return dtoaionui.ClientPackageRolloutRequest{}, err
	}
	return s.GetRollout(id)
}

func (s *ClientPackageService) latestEligiblePublished(userID int, platform string, requireUpdate bool) (apmodel.ClientPackage, bool, error) {
	if s == nil || s.db == nil || userID <= 0 {
		return apmodel.ClientPackage{}, false, ErrClientPackageInvalidInput
	}
	rows, err := s.publishedPackages(platform, requireUpdate)
	if err != nil || len(rows) == 0 {
		return apmodel.ClientPackage{}, false, err
	}
	subjects, err := platformGrantSubjectsForDB(s.db, userID)
	if err != nil {
		return apmodel.ClientPackage{}, false, err
	}
	packageIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		packageIDs = append(packageIDs, row.Id)
	}
	var scopes []apmodel.ClientPackageScope
	if err := s.db.Where("client_package_id IN ?", packageIDs).Find(&scopes).Error; err != nil {
		return apmodel.ClientPackage{}, false, err
	}
	eligible := clientPackageEligibleRows(rows, scopes, subjects)
	if len(eligible) == 0 {
		return apmodel.ClientPackage{}, false, nil
	}
	sortClientPackages(eligible)
	return eligible[0], true, nil
}

func (s *ClientPackageService) eligiblePublishedPackageByID(userID int, platform string, packageID int, requireUpdate bool) (apmodel.ClientPackage, bool, error) {
	if s == nil || s.db == nil || userID <= 0 || packageID <= 0 {
		return apmodel.ClientPackage{}, false, ErrClientPackageInvalidInput
	}
	var pkg apmodel.ClientPackage
	query := s.db.Where("id = ? AND platform = ? AND status = ?", packageID, platform, apmodel.ClientPackageStatusPublished)
	if requireUpdate {
		query = query.Where("update_metadata_file_path <> ?", "")
		if platform != apmodel.ClientPackagePlatformWindowsX64 {
			query = query.Where("update_file_path <> ?", "")
		}
	}
	if err := query.First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apmodel.ClientPackage{}, false, nil
		}
		return apmodel.ClientPackage{}, false, err
	}
	subjects, err := platformGrantSubjectsForDB(s.db, userID)
	if err != nil {
		return apmodel.ClientPackage{}, false, err
	}
	var scopes []apmodel.ClientPackageScope
	if err := s.db.Where("client_package_id = ?", packageID).Find(&scopes).Error; err != nil {
		return apmodel.ClientPackage{}, false, err
	}
	eligible := clientPackageEligibleRows([]apmodel.ClientPackage{pkg}, scopes, subjects)
	if len(eligible) == 0 {
		return apmodel.ClientPackage{}, false, nil
	}
	return eligible[0], true, nil
}

func (s *ClientPackageService) publishedPackages(platform string, requireUpdate bool) ([]apmodel.ClientPackage, error) {
	query := s.db.Where("platform = ? AND status = ?", platform, apmodel.ClientPackageStatusPublished)
	if requireUpdate {
		query = query.Where("update_metadata_file_path <> ?", "")
		if platform != apmodel.ClientPackagePlatformWindowsX64 {
			query = query.Where("update_file_path <> ?", "")
		}
	}
	var rows []apmodel.ClientPackage
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func clientPackageEligibleRows(rows []apmodel.ClientPackage, scopes []apmodel.ClientPackageScope, subjects []grantSubject) []apmodel.ClientPackage {
	subjectSet := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		subjectSet[subject.Type+"\x00"+subject.Id] = struct{}{}
	}
	scopeMatched := make(map[int]bool)
	for _, scope := range scopes {
		if _, ok := subjectSet[scope.SubjectType+"\x00"+scope.SubjectId]; ok {
			scopeMatched[scope.ClientPackageId] = true
		}
	}
	eligible := make([]apmodel.ClientPackage, 0, len(rows))
	for _, row := range rows {
		if normalizeClientPackageRolloutMode(row.RolloutMode) == apmodel.ClientPackageRolloutModeGlobal || scopeMatched[row.Id] {
			eligible = append(eligible, row)
		}
	}
	return eligible
}

func sortClientPackages(rows []apmodel.ClientPackage) {
	sort.Slice(rows, func(i int, j int) bool {
		cmp := compareClientPackageVersion(rows[i].Version, rows[j].Version)
		if cmp != 0 {
			return cmp > 0
		}
		if rows[i].PublishedAt != nil && rows[j].PublishedAt != nil && !rows[i].PublishedAt.Equal(*rows[j].PublishedAt) {
			return rows[i].PublishedAt.After(*rows[j].PublishedAt)
		}
		return rows[i].Id > rows[j].Id
	})
}
