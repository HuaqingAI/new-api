package aionui

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"gorm.io/gorm"
)

const (
	DesktopRedirectURI          = "aionui://auth/hth-callback"
	DesktopLoopbackCallbackPath = "/hth/callback"
	desktopTokenTTL             = 90 * 24 * time.Hour
	desktopCodeTTL              = 5 * time.Minute
	desktopTokenRedisKeyPrefix  = "aionui:desktop:token:"
)

var (
	ErrInvalidRedirectURI = errors.New("invalid redirect_uri")
	ErrInvalidState       = errors.New("invalid state")
	ErrInvalidCode        = errors.New("invalid or expired desktop code")
	ErrInvalidToken       = errors.New("invalid or expired desktop token")
	ErrEmailRequired      = errors.New("user email is required")
	ErrUserUnavailable    = errors.New("user is disabled or unavailable")
)

var (
	desktopRedisAvailable = func() bool {
		return common.RedisEnabled && common.RDB != nil
	}
	desktopRedisSet = common.RedisSet
	desktopRedisGet = common.RedisGet
)

type DesktopCodeGrant struct {
	Code        string
	UserId      int
	Username    string
	Email       string
	DisplayName string
	State       string
	RedirectURI string
	ExpiresAt   time.Time
	Used        bool
}

type DesktopTokenClaims struct {
	Token      string
	UserId     int
	Username   string
	Email      string
	DeviceId   string
	ExpiresAt  time.Time
	IssuedAt   time.Time
	AppVersion string
}

type desktopTokenRedisPayload struct {
	UserId     int    `json:"user_id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	DeviceId   string `json:"device_id"`
	ExpiresAt  int64  `json:"expires_at"`
	IssuedAt   int64  `json:"issued_at"`
	AppVersion string `json:"app_version"`
}

type DesktopAuthService struct {
	mu     sync.Mutex
	codes  map[string]DesktopCodeGrant
	tokens map[string]DesktopTokenClaims
	now    func() time.Time
}

var defaultDesktopAuthService = NewDesktopAuthService()

func NewDesktopAuthService() *DesktopAuthService {
	return &DesktopAuthService{
		codes:  map[string]DesktopCodeGrant{},
		tokens: map[string]DesktopTokenClaims{},
		now:    time.Now,
	}
}

func DefaultDesktopAuthService() *DesktopAuthService {
	return defaultDesktopAuthService
}

func ResetDesktopAuthServiceForTest() {
	defaultDesktopAuthService = NewDesktopAuthService()
}

func ValidateDesktopRedirectURI(redirectURI string) error {
	value := strings.TrimSpace(redirectURI)
	if value == DesktopRedirectURI || isDesktopLoopbackRedirectURI(value) {
		return nil
	}
	return ErrInvalidRedirectURI
}

func isDesktopLoopbackRedirectURI(redirectURI string) bool {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" || parsed.Path != DesktopLoopbackCallbackPath || parsed.User != nil {
		return false
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname != "127.0.0.1" && hostname != "localhost" {
		return false
	}
	if parsed.Port() == "" {
		return false
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return true
}

func ValidateDesktopState(state string) error {
	if len(strings.TrimSpace(state)) < 12 {
		return ErrInvalidState
	}
	return nil
}

func (s *DesktopAuthService) IssueCode(user *model.User, redirectURI string, state string) (string, error) {
	if err := ValidateDesktopRedirectURI(redirectURI); err != nil {
		return "", err
	}
	if err := ValidateDesktopState(state); err != nil {
		return "", err
	}
	if user == nil || user.Id <= 0 || user.Status != common.UserStatusEnabled {
		return "", ErrUserUnavailable
	}
	email := model.NormalizeEmail(user.Email)
	if email == "" {
		return "", ErrEmailRequired
	}

	code, err := desktopSecret("aionui_code_", 40)
	if err != nil {
		return "", err
	}
	now := s.now()
	grant := DesktopCodeGrant{
		Code:        code,
		UserId:      user.Id,
		Username:    user.Username,
		Email:       email,
		DisplayName: user.DisplayName,
		State:       state,
		RedirectURI: redirectURI,
		ExpiresAt:   now.Add(desktopCodeTTL),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code] = grant
	return code, nil
}

func (s *DesktopAuthService) ExchangeCode(req dtoaionui.DesktopTokenRequest) (dtoaionui.DesktopTokenResponse, error) {
	code := strings.TrimSpace(req.Code)
	deviceId := strings.TrimSpace(req.DeviceId)
	if code == "" || deviceId == "" || len(deviceId) > 128 {
		return dtoaionui.DesktopTokenResponse{}, ErrInvalidCode
	}

	now := s.now()
	s.mu.Lock()
	grant, ok := s.codes[code]
	if !ok || grant.Used || now.After(grant.ExpiresAt) {
		s.mu.Unlock()
		return dtoaionui.DesktopTokenResponse{}, ErrInvalidCode
	}
	grant.Used = true
	delete(s.codes, code)
	s.mu.Unlock()

	user, err := model.GetUserById(grant.UserId, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dtoaionui.DesktopTokenResponse{}, ErrUserUnavailable
		}
		return dtoaionui.DesktopTokenResponse{}, err
	}
	if user.Status != common.UserStatusEnabled {
		return dtoaionui.DesktopTokenResponse{}, ErrUserUnavailable
	}
	email := model.NormalizeEmail(user.Email)
	if email == "" {
		return dtoaionui.DesktopTokenResponse{}, ErrEmailRequired
	}

	token, err := desktopSecret("aionui_", 48)
	if err != nil {
		return dtoaionui.DesktopTokenResponse{}, err
	}
	expiresAt := now.Add(desktopTokenTTL)
	claims := DesktopTokenClaims{
		Token:      token,
		UserId:     user.Id,
		Username:   user.Username,
		Email:      email,
		DeviceId:   deviceId,
		ExpiresAt:  expiresAt,
		IssuedAt:   now,
		AppVersion: strings.TrimSpace(req.AppVersion),
	}

	s.mu.Lock()
	s.tokens[token] = claims
	s.mu.Unlock()
	if err := s.saveToken(claims); err != nil {
		return dtoaionui.DesktopTokenResponse{}, err
	}
	departments, err := desktopUserDepartmentNames(user.Id)
	if err != nil {
		return dtoaionui.DesktopTokenResponse{}, err
	}
	personalAPIKey, err := EnsurePersonalAPIKey(user.Id)
	if err != nil {
		return dtoaionui.DesktopTokenResponse{}, err
	}

	return dtoaionui.DesktopTokenResponse{
		AccessToken: token,
		ExpiresAt:   expiresAt.Unix(),
		User: dtoaionui.DesktopUser{
			Id:          user.Id,
			Username:    user.Username,
			Email:       email,
			DisplayName: user.DisplayName,
			Departments: departments,
		},
		PersonalAPIKey: dtoaionui.DesktopPersonalAPIKey{
			Name:      personalAPIKey.Name,
			Key:       personalAPIKey.Key,
			MaskedKey: personalAPIKey.MaskedKey,
		},
		QuotaApplyURL: QuotaApplyURL(),
	}, nil
}

func desktopUserDepartmentNames(userID int) ([]string, error) {
	var memberships []entmodel.UserDepartment
	if err := model.DB.Where("user_id = ? AND status = ?", userID, constant.EnterpriseMembershipStatusActive).Find(&memberships).Error; err != nil {
		if isOptionalDesktopDepartmentLookupError(err) {
			return []string{}, nil
		}
		return nil, err
	}
	departmentIds := make([]int, 0, len(memberships))
	seen := map[int]struct{}{}
	for _, membership := range memberships {
		if membership.DepartmentId <= 0 {
			continue
		}
		if _, ok := seen[membership.DepartmentId]; ok {
			continue
		}
		seen[membership.DepartmentId] = struct{}{}
		departmentIds = append(departmentIds, membership.DepartmentId)
	}
	sort.Ints(departmentIds)
	if len(departmentIds) == 0 {
		return []string{}, nil
	}
	var departments []entmodel.Department
	if err := model.DB.Select("id", "name").Where("id IN ?", departmentIds).Find(&departments).Error; err != nil {
		if isOptionalDesktopDepartmentLookupError(err) {
			return []string{}, nil
		}
		return nil, err
	}
	nameById := map[int]string{}
	for _, department := range departments {
		nameById[department.Id] = strings.TrimSpace(department.Name)
	}
	names := make([]string, 0, len(departmentIds))
	for _, id := range departmentIds {
		if name := nameById[id]; name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func isOptionalDesktopDepartmentLookupError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table") ||
		strings.Contains(message, "doesn't exist") ||
		strings.Contains(message, "does not exist") ||
		strings.Contains(message, "undefined table") ||
		strings.Contains(message, "42p01")
}

func desktopSecret(prefix string, length int) (string, error) {
	value, err := common.GenerateRandomCharsKey(length)
	if err != nil {
		return "", err
	}
	return prefix + value, nil
}

func (s *DesktopAuthService) saveToken(claims DesktopTokenClaims) error {
	if !desktopRedisAvailable() {
		return nil
	}
	ttl := claims.ExpiresAt.Sub(s.now())
	if ttl <= 0 {
		return nil
	}
	payload := desktopTokenRedisPayload{
		UserId:     claims.UserId,
		Username:   claims.Username,
		Email:      claims.Email,
		DeviceId:   claims.DeviceId,
		ExpiresAt:  claims.ExpiresAt.Unix(),
		IssuedAt:   claims.IssuedAt.Unix(),
		AppVersion: claims.AppVersion,
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	return desktopRedisSet(desktopTokenRedisKey(claims.Token), string(data), ttl)
}

func (s *DesktopAuthService) ValidateAuthorization(header string) (DesktopTokenClaims, error) {
	token := strings.TrimSpace(header)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		return DesktopTokenClaims{}, ErrInvalidToken
	}

	now := s.now()
	s.mu.Lock()
	claims, ok := s.tokens[token]
	if !ok || now.After(claims.ExpiresAt) {
		if ok {
			delete(s.tokens, token)
		}
		s.mu.Unlock()
		if ok {
			return DesktopTokenClaims{}, ErrInvalidToken
		}
		var err error
		claims, err = s.loadToken(token, now)
		if err != nil {
			return DesktopTokenClaims{}, err
		}
		if claims.Token == "" {
			return DesktopTokenClaims{}, ErrInvalidToken
		}
		s.mu.Lock()
		s.tokens[token] = claims
		s.mu.Unlock()
	} else {
		s.mu.Unlock()
	}

	user, err := model.GetUserById(claims.UserId, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DesktopTokenClaims{}, ErrUserUnavailable
		}
		return DesktopTokenClaims{}, err
	}
	if user.Status != common.UserStatusEnabled {
		return DesktopTokenClaims{}, ErrUserUnavailable
	}
	email := model.NormalizeEmail(user.Email)
	if email == "" {
		return DesktopTokenClaims{}, ErrEmailRequired
	}
	claims.Username = user.Username
	claims.Email = email
	return claims, nil
}

func (s *DesktopAuthService) loadToken(token string, now time.Time) (DesktopTokenClaims, error) {
	if !desktopRedisAvailable() {
		return DesktopTokenClaims{}, nil
	}
	data, err := desktopRedisGet(desktopTokenRedisKey(token))
	if err != nil {
		return DesktopTokenClaims{}, nil
	}
	var payload desktopTokenRedisPayload
	if err := common.Unmarshal([]byte(data), &payload); err != nil {
		return DesktopTokenClaims{}, err
	}
	expiresAt := time.Unix(payload.ExpiresAt, 0)
	if now.After(expiresAt) {
		return DesktopTokenClaims{}, nil
	}
	return DesktopTokenClaims{
		Token:      token,
		UserId:     payload.UserId,
		Username:   payload.Username,
		Email:      payload.Email,
		DeviceId:   payload.DeviceId,
		ExpiresAt:  expiresAt,
		IssuedAt:   time.Unix(payload.IssuedAt, 0),
		AppVersion: payload.AppVersion,
	}, nil
}

func desktopTokenRedisKey(token string) string {
	return desktopTokenRedisKeyPrefix + desktopTokenHash(token)
}

func desktopTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func BuildDesktopCallbackURL(redirectURI string, code string, state string) (string, error) {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("code", code)
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
