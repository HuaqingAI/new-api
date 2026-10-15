package aionui

import (
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	aionuimodel "github.com/QuantumNous/new-api/model/aionui"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const (
	ClientHeartbeatInterval     = 30 * time.Minute
	ClientHeartbeatActiveWindow = 90 * time.Minute
)

var ErrClientHeartbeatInvalidInput = errors.New("aionui client heartbeat input invalid")

type ClientHeartbeatValidationError struct {
	Message string
}

func (e ClientHeartbeatValidationError) Error() string {
	return e.Message
}

func (e ClientHeartbeatValidationError) Unwrap() error {
	return ErrClientHeartbeatInvalidInput
}

type ClientHeartbeatInput struct {
	UserId        int
	DeviceId      string
	ClientVersion string
	Platform      string
	LanIP         string
	User          dtoaionui.ClientHeartbeatUserInfo
}

type ClientInstallationQuery struct {
	Page       int
	PageSize   int
	Keyword    string
	Platform   string
	Version    string
	ActiveOnly bool
}

type ClientHeartbeatService struct {
	db  *gorm.DB
	now func() time.Time
}

func NewClientHeartbeatService(db *gorm.DB) *ClientHeartbeatService {
	return &ClientHeartbeatService{db: db, now: time.Now}
}

func (s *ClientHeartbeatService) Report(input ClientHeartbeatInput) (dtoaionui.ClientHeartbeatResponse, error) {
	if s == nil || s.db == nil {
		return dtoaionui.ClientHeartbeatResponse{}, ErrClientHeartbeatInvalidInput
	}
	input, err := validateClientHeartbeatInput(input)
	if err != nil {
		return dtoaionui.ClientHeartbeatResponse{}, err
	}

	user, departmentNames, err := s.currentUserSnapshot(input.UserId)
	if err != nil {
		return dtoaionui.ClientHeartbeatResponse{}, err
	}
	departmentData, err := common.Marshal(departmentNames)
	if err != nil {
		return dtoaionui.ClientHeartbeatResponse{}, err
	}

	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = user.Username
	}
	now := s.now().UTC()
	values := map[string]any{
		"user_id":               user.Id,
		"username":              user.Username,
		"email":                 model.NormalizeEmail(user.Email),
		"display_name":          displayName,
		"department_names_json": string(departmentData),
		"client_version":        input.ClientVersion,
		"platform":              input.Platform,
		"lan_ip":                input.LanIP,
		"last_heartbeat_at":     now,
	}

	var installation aionuimodel.ClientInstallation
	err = s.db.Where("device_id = ?", input.DeviceId).First(&installation).Error
	if err == nil {
		if err := s.db.Model(&aionuimodel.ClientInstallation{}).Where("id = ?", installation.Id).Updates(values).Error; err != nil {
			return dtoaionui.ClientHeartbeatResponse{}, err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dtoaionui.ClientHeartbeatResponse{}, err
	} else {
		installation = aionuimodel.ClientInstallation{
			DeviceId:            input.DeviceId,
			UserId:              user.Id,
			Username:            user.Username,
			Email:               model.NormalizeEmail(user.Email),
			DisplayName:         displayName,
			DepartmentNamesJSON: string(departmentData),
			ClientVersion:       input.ClientVersion,
			Platform:            input.Platform,
			LanIP:               input.LanIP,
			FirstHeartbeatAt:    now,
			LastHeartbeatAt:     now,
		}
		if err := s.db.Create(&installation).Error; err != nil {
			if lookupErr := s.db.Where("device_id = ?", input.DeviceId).First(&installation).Error; lookupErr != nil {
				return dtoaionui.ClientHeartbeatResponse{}, err
			}
			if updateErr := s.db.Model(&aionuimodel.ClientInstallation{}).Where("id = ?", installation.Id).Updates(values).Error; updateErr != nil {
				return dtoaionui.ClientHeartbeatResponse{}, updateErr
			}
		}
	}

	return dtoaionui.ClientHeartbeatResponse{
		AcceptedAt:           now.Format(time.RFC3339),
		NextHeartbeatSeconds: int(ClientHeartbeatInterval.Seconds()),
	}, nil
}

func (s *ClientHeartbeatService) List(query ClientInstallationQuery) (dtoaionui.ClientInstallationListResponse, error) {
	if s == nil || s.db == nil {
		return dtoaionui.ClientInstallationListResponse{}, ErrClientHeartbeatInvalidInput
	}
	query, err := validateClientInstallationQuery(query)
	if err != nil {
		return dtoaionui.ClientInstallationListResponse{}, err
	}
	activeSince := s.now().UTC().Add(-ClientHeartbeatActiveWindow)
	dbQuery := s.db.Model(&aionuimodel.ClientInstallation{})
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		dbQuery = dbQuery.Where("username LIKE ? OR email LIKE ? OR display_name LIKE ?", keyword, keyword, keyword)
	}
	if query.Platform != "" {
		dbQuery = dbQuery.Where("platform = ?", query.Platform)
	}
	if query.Version != "" {
		dbQuery = dbQuery.Where("client_version = ?", query.Version)
	}
	if query.ActiveOnly {
		dbQuery = dbQuery.Where("last_heartbeat_at >= ?", activeSince)
	}

	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return dtoaionui.ClientInstallationListResponse{}, err
	}
	var rows []aionuimodel.ClientInstallation
	if err := dbQuery.Order("last_heartbeat_at desc").Order("id desc").Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).Find(&rows).Error; err != nil {
		return dtoaionui.ClientInstallationListResponse{}, err
	}

	items := make([]dtoaionui.ClientInstallationItem, 0, len(rows))
	for _, row := range rows {
		departments := []string{}
		if err := common.UnmarshalJsonStr(row.DepartmentNamesJSON, &departments); err != nil {
			return dtoaionui.ClientInstallationListResponse{}, fmt.Errorf("decode aionui client installation departments: %w", err)
		}
		if departments == nil {
			departments = []string{}
		}
		items = append(items, dtoaionui.ClientInstallationItem{
			Id:               row.Id,
			ClientVersion:    row.ClientVersion,
			Platform:         row.Platform,
			LanIP:            row.LanIP,
			UserId:           row.UserId,
			Username:         row.Username,
			Email:            row.Email,
			DisplayName:      row.DisplayName,
			Departments:      departments,
			FirstHeartbeatAt: row.FirstHeartbeatAt.UTC().Format(time.RFC3339),
			LastHeartbeatAt:  row.LastHeartbeatAt.UTC().Format(time.RFC3339),
			IsActive:         !row.LastHeartbeatAt.Before(activeSince),
		})
	}

	summary, err := s.summary(activeSince)
	if err != nil {
		return dtoaionui.ClientInstallationListResponse{}, err
	}
	return dtoaionui.ClientInstallationListResponse{
		Items:    items,
		Total:    int(total),
		Page:     query.Page,
		PageSize: query.PageSize,
		Summary:  summary,
	}, nil
}

func (s *ClientHeartbeatService) currentUserSnapshot(userID int) (model.User, []string, error) {
	var user model.User
	if err := s.db.Select("id", "username", "display_name", "email", "status").First(&user, userID).Error; err != nil {
		return model.User{}, nil, err
	}
	if user.Status != common.UserStatusEnabled || model.NormalizeEmail(user.Email) == "" {
		return model.User{}, nil, ErrClientHeartbeatInvalidInput
	}

	var memberships []entmodel.UserDepartment
	if err := s.db.Where("user_id = ? AND status = ?", userID, constant.EnterpriseMembershipStatusActive).Find(&memberships).Error; err != nil {
		return model.User{}, nil, err
	}
	departmentIDs := make([]int, 0, len(memberships))
	seen := make(map[int]struct{}, len(memberships))
	for _, membership := range memberships {
		if membership.DepartmentId <= 0 {
			continue
		}
		if _, exists := seen[membership.DepartmentId]; exists {
			continue
		}
		seen[membership.DepartmentId] = struct{}{}
		departmentIDs = append(departmentIDs, membership.DepartmentId)
	}
	sort.Ints(departmentIDs)
	if len(departmentIDs) == 0 {
		return user, []string{}, nil
	}

	var departments []entmodel.Department
	if err := s.db.Select("id", "name").Where("id IN ?", departmentIDs).Find(&departments).Error; err != nil {
		return model.User{}, nil, err
	}
	namesByID := make(map[int]string, len(departments))
	for _, department := range departments {
		namesByID[department.Id] = strings.TrimSpace(department.Name)
	}
	departmentNames := make([]string, 0, len(departmentIDs))
	for _, departmentID := range departmentIDs {
		if name := namesByID[departmentID]; name != "" {
			departmentNames = append(departmentNames, name)
		}
	}
	return user, departmentNames, nil
}

func (s *ClientHeartbeatService) summary(activeSince time.Time) (dtoaionui.ClientInstallationSummary, error) {
	var totalInstallations int64
	if err := s.db.Model(&aionuimodel.ClientInstallation{}).Count(&totalInstallations).Error; err != nil {
		return dtoaionui.ClientInstallationSummary{}, err
	}
	var totalUsers int64
	if err := s.db.Model(&aionuimodel.ClientInstallation{}).Distinct("user_id").Count(&totalUsers).Error; err != nil {
		return dtoaionui.ClientInstallationSummary{}, err
	}
	activeQuery := s.db.Model(&aionuimodel.ClientInstallation{}).Where("last_heartbeat_at >= ?", activeSince)
	var activeInstallations int64
	if err := activeQuery.Count(&activeInstallations).Error; err != nil {
		return dtoaionui.ClientInstallationSummary{}, err
	}
	var activeUsers int64
	if err := s.db.Model(&aionuimodel.ClientInstallation{}).Where("last_heartbeat_at >= ?", activeSince).Distinct("user_id").Count(&activeUsers).Error; err != nil {
		return dtoaionui.ClientInstallationSummary{}, err
	}
	return dtoaionui.ClientInstallationSummary{
		ActiveUsers:         int(activeUsers),
		ActiveInstallations: int(activeInstallations),
		TotalUsers:          int(totalUsers),
		TotalInstallations:  int(totalInstallations),
		ActiveSince:         activeSince.UTC().Format(time.RFC3339),
	}, nil
}

func validateClientHeartbeatInput(input ClientHeartbeatInput) (ClientHeartbeatInput, error) {
	input.DeviceId = strings.TrimSpace(input.DeviceId)
	input.ClientVersion = strings.TrimSpace(input.ClientVersion)
	input.Platform = strings.TrimSpace(input.Platform)
	input.LanIP = strings.TrimSpace(input.LanIP)
	if input.UserId <= 0 || input.DeviceId == "" || len(input.DeviceId) > 128 {
		return ClientHeartbeatInput{}, clientHeartbeatValidationError("invalid client identity")
	}
	if !validClientHeartbeatVersion(input.ClientVersion) {
		return ClientHeartbeatInput{}, clientHeartbeatValidationError("invalid client version")
	}
	if !validClientPackagePlatform(input.Platform) {
		return ClientHeartbeatInput{}, clientHeartbeatValidationError("invalid client platform")
	}
	if !validClientHeartbeatLanIP(input.LanIP) {
		return ClientHeartbeatInput{}, clientHeartbeatValidationError("invalid client lan ip")
	}
	if len(strings.TrimSpace(input.User.Email)) == 0 || len(input.User.Email) > 320 || len(strings.TrimSpace(input.User.Name)) == 0 || len(input.User.Name) > 128 || input.User.Departments == nil || len(input.User.Departments) > 50 {
		return ClientHeartbeatInput{}, clientHeartbeatValidationError("invalid client user snapshot")
	}
	for _, department := range input.User.Departments {
		if strings.TrimSpace(department) == "" || len(department) > 255 {
			return ClientHeartbeatInput{}, clientHeartbeatValidationError("invalid client department snapshot")
		}
	}
	return input, nil
}

func validateClientInstallationQuery(query ClientInstallationQuery) (ClientInstallationQuery, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Platform = strings.TrimSpace(query.Platform)
	query.Version = strings.TrimSpace(query.Version)
	if len(query.Keyword) > 128 {
		return ClientInstallationQuery{}, clientHeartbeatValidationError("invalid client keyword")
	}
	if query.Platform != "" && !validClientPackagePlatform(query.Platform) {
		return ClientInstallationQuery{}, clientHeartbeatValidationError("invalid client platform")
	}
	if query.Version != "" && !validClientHeartbeatVersion(query.Version) {
		return ClientInstallationQuery{}, clientHeartbeatValidationError("invalid client version")
	}
	return query, nil
}

func validClientHeartbeatVersion(version string) bool {
	if len(version) == 0 || len(version) > 64 {
		return false
	}
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

func validClientHeartbeatLanIP(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > 45 {
		return false
	}
	address, err := netip.ParseAddr(value)
	return err == nil && address.Is4() && address.IsPrivate()
}

func clientHeartbeatValidationError(message string) ClientHeartbeatValidationError {
	return ClientHeartbeatValidationError{Message: message}
}
