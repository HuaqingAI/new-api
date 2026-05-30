package enterprise

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	defaultDingTalkOpenAPIBaseURL = "https://oapi.dingtalk.com"
	defaultDingTalkAPIBaseURL     = "https://api.dingtalk.com"
	defaultDingTalkAPITimeout     = 10 * time.Second
)

type DingTalkHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type DingTalkClient struct {
	httpClient     DingTalkHTTPClient
	openAPIBaseURL string
	apiBaseURL     string
}

type DingTalkClientOption func(*DingTalkClient)

type DingTalkAPIError struct {
	Stage      string
	ErrCode    int
	HTTPStatus int
	Summary    string
	Network    bool
}

type dingTalkAPIEnvelope struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
}

type DingTalkOAuthToken struct {
	AccessToken           string
	RefreshToken          string
	ExpiresIn             int64
	RefreshTokenExpiresIn int64
	TokenType             string
	OpenId                string
	UnionId               string
}

type DingTalkOAuthUserInfo struct {
	UnionId string
	OpenId  string
	Nick    string
	Email   string
	Mobile  string
}

type DingTalkContactUserInfo struct {
	UserId  string `json:"userid"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Mobile  string `json:"mobile"`
	Active  *bool  `json:"active"`
	UnionId string `json:"unionid"`
}

type DingTalkDepartmentInfo struct {
	DeptId          int64  `json:"dept_id"`
	Name            string `json:"name"`
	ParentId        int64  `json:"parent_id"`
	AutoAddUser     bool   `json:"auto_add_user"`
	CreateDeptGroup bool   `json:"create_dept_group"`
}

type DingTalkDepartmentUserInfo struct {
	UserId     string  `json:"userid"`
	UnionId    string  `json:"unionid"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	Mobile     string  `json:"mobile"`
	Active     *bool   `json:"active"`
	DeptIdList []int64 `json:"dept_id_list"`
}

type dingTalkOAuthTokenRequest struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Code         string `json:"code"`
	GrantType    string `json:"grantType"`
}

type dingTalkOAuthTokenResponse struct {
	AccessToken                  string `json:"accessToken"`
	AccessTokenLegacy            string `json:"access_token"`
	RefreshToken                 string `json:"refreshToken"`
	RefreshTokenLegacy           string `json:"refresh_token"`
	ExpiresIn                    int64  `json:"expireIn"`
	ExpiresInLegacy              int64  `json:"expires_in"`
	RefreshTokenExpiresIn        int64  `json:"refreshExpireIn"`
	RefreshTokenExpiresInLegacy  int64  `json:"refresh_expires_in"`
	TokenType                    string `json:"tokenType"`
	TokenTypeLegacy              string `json:"token_type"`
	OpenId                       string `json:"openId"`
	OpenIdLegacy                 string `json:"open_id"`
	UnionId                      string `json:"unionId"`
	UnionIdLegacy                string `json:"union_id"`
}

type dingTalkOAuthUserInfoResponse struct {
	UnionId       string `json:"unionId"`
	UnionIdLegacy string `json:"union_id"`
	OpenId        string `json:"openId"`
	OpenIdLegacy  string `json:"open_id"`
	Nick          string `json:"nick"`
	Email         string `json:"email"`
	Mobile        string `json:"mobile"`
}

type dingTalkContactUserRequest struct {
	UnionId string `json:"unionid"`
}

type dingTalkContactUserResponse struct {
	ErrCode int                     `json:"errcode"`
	ErrMsg  string                  `json:"errmsg"`
	Result  DingTalkContactUserInfo `json:"result"`
}

type dingTalkDepartmentListResponse struct {
	ErrCode int                      `json:"errcode"`
	ErrMsg  string                   `json:"errmsg"`
	Result  []DingTalkDepartmentInfo `json:"result"`
}

type dingTalkDepartmentUserListResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Result  struct {
		List       []DingTalkDepartmentUserInfo `json:"list"`
		HasMore    bool                         `json:"has_more"`
		NextCursor int64                        `json:"next_cursor"`
	} `json:"result"`
}

type dingTalkUserGetResponse struct {
	ErrCode int                        `json:"errcode"`
	ErrMsg  string                     `json:"errmsg"`
	Result  DingTalkDepartmentUserInfo `json:"result"`
}

func (e *DingTalkAPIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Summary != "" {
		return e.Summary
	}
	if e.Network {
		return "dingtalk network error"
	}
	return fmt.Sprintf("dingtalk api error at %s", e.Stage)
}

func NewDingTalkClient(options ...DingTalkClientOption) *DingTalkClient {
	client := &DingTalkClient{
		httpClient: &http.Client{
			Timeout: defaultDingTalkAPITimeout,
		},
		openAPIBaseURL: defaultDingTalkOpenAPIBaseURL,
		apiBaseURL:     defaultDingTalkAPIBaseURL,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

func WithDingTalkHTTPClient(httpClient DingTalkHTTPClient) DingTalkClientOption {
	return func(client *DingTalkClient) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithDingTalkOpenAPIBaseURL(baseURL string) DingTalkClientOption {
	return func(client *DingTalkClient) {
		if strings.TrimSpace(baseURL) != "" {
			client.openAPIBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		}
	}
}

func WithDingTalkAPIBaseURL(baseURL string) DingTalkClientOption {
	return func(client *DingTalkClient) {
		if strings.TrimSpace(baseURL) != "" {
			client.apiBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		}
	}
}

func (c *DingTalkClient) GetAccessToken(ctx context.Context, appKey string, appSecret string) (string, error) {
	tokenURL, err := url.Parse(c.openAPIBaseURL + "/gettoken")
	if err != nil {
		return "", err
	}
	q := tokenURL.Query()
	q.Set("appkey", appKey)
	q.Set("appsecret", appSecret)
	tokenURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", &DingTalkAPIError{Stage: "access_token", Summary: "token_request_failed", Network: true}
	}
	defer resp.Body.Close()

	var payload dingTalkAPIEnvelope
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return "", &DingTalkAPIError{Stage: "access_token", HTTPStatus: resp.StatusCode, Summary: "token_response_invalid"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", &DingTalkAPIError{Stage: "access_token", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "token_http_error"}
	}
	if payload.ErrCode != 0 || strings.TrimSpace(payload.AccessToken) == "" {
		return "", &DingTalkAPIError{Stage: "access_token", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "token_invalid_credentials"}
	}
	return strings.TrimSpace(payload.AccessToken), nil
}

func (c *DingTalkClient) ExchangeOAuthCode(ctx context.Context, appKey string, appSecret string, code string) (DingTalkOAuthToken, error) {
	tokenURL, err := url.Parse(c.apiBaseURL + "/v1.0/oauth2/userAccessToken")
	if err != nil {
		return DingTalkOAuthToken{}, err
	}
	requestPayload := dingTalkOAuthTokenRequest{
		ClientId:     appKey,
		ClientSecret: appSecret,
		Code:         code,
		GrantType:    "authorization_code",
	}
	body, err := common.Marshal(requestPayload)
	if err != nil {
		return DingTalkOAuthToken{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL.String(), bytes.NewReader(body))
	if err != nil {
		return DingTalkOAuthToken{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return DingTalkOAuthToken{}, &DingTalkAPIError{Stage: "oauth_token", Summary: "oauth_token_request_failed", Network: true}
	}
	defer resp.Body.Close()

	var payload dingTalkOAuthTokenResponse
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return DingTalkOAuthToken{}, &DingTalkAPIError{Stage: "oauth_token", HTTPStatus: resp.StatusCode, Summary: "oauth_token_response_invalid"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return DingTalkOAuthToken{}, &DingTalkAPIError{Stage: "oauth_token", HTTPStatus: resp.StatusCode, Summary: "oauth_token_http_error"}
	}
	token := DingTalkOAuthToken{
		AccessToken:           firstNonEmpty(payload.AccessToken, payload.AccessTokenLegacy),
		RefreshToken:          firstNonEmpty(payload.RefreshToken, payload.RefreshTokenLegacy),
		ExpiresIn:             payload.ExpiresIn,
		RefreshTokenExpiresIn: payload.RefreshTokenExpiresIn,
		TokenType:             firstNonEmpty(payload.TokenType, payload.TokenTypeLegacy),
		OpenId:                firstNonEmpty(payload.OpenId, payload.OpenIdLegacy),
		UnionId:               firstNonEmpty(payload.UnionId, payload.UnionIdLegacy),
	}
	if token.ExpiresIn == 0 {
		token.ExpiresIn = payload.ExpiresInLegacy
	}
	if token.RefreshTokenExpiresIn == 0 {
		token.RefreshTokenExpiresIn = payload.RefreshTokenExpiresInLegacy
	}
	if token.AccessToken == "" {
		return DingTalkOAuthToken{}, &DingTalkAPIError{Stage: "oauth_token", HTTPStatus: resp.StatusCode, Summary: "oauth_token_empty"}
	}
	return token, nil
}

func (c *DingTalkClient) GetOAuthUserInfo(ctx context.Context, userAccessToken string) (DingTalkOAuthUserInfo, error) {
	userURL, err := url.Parse(c.apiBaseURL + "/v1.0/contact/users/me")
	if err != nil {
		return DingTalkOAuthUserInfo{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL.String(), nil)
	if err != nil {
		return DingTalkOAuthUserInfo{}, err
	}
	req.Header.Set("x-acs-dingtalk-access-token", strings.TrimSpace(userAccessToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return DingTalkOAuthUserInfo{}, &DingTalkAPIError{Stage: "oauth_userinfo", Summary: "oauth_userinfo_request_failed", Network: true}
	}
	defer resp.Body.Close()

	var payload dingTalkOAuthUserInfoResponse
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return DingTalkOAuthUserInfo{}, &DingTalkAPIError{Stage: "oauth_userinfo", HTTPStatus: resp.StatusCode, Summary: "oauth_userinfo_response_invalid"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return DingTalkOAuthUserInfo{}, &DingTalkAPIError{Stage: "oauth_userinfo", HTTPStatus: resp.StatusCode, Summary: "oauth_userinfo_http_error"}
	}
	user := DingTalkOAuthUserInfo{
		UnionId: firstNonEmpty(payload.UnionId, payload.UnionIdLegacy),
		OpenId:  firstNonEmpty(payload.OpenId, payload.OpenIdLegacy),
		Nick:    strings.TrimSpace(payload.Nick),
		Email:   strings.TrimSpace(payload.Email),
		Mobile:  strings.TrimSpace(payload.Mobile),
	}
	if user.UnionId == "" && user.OpenId == "" {
		return DingTalkOAuthUserInfo{}, &DingTalkAPIError{Stage: "oauth_userinfo", HTTPStatus: resp.StatusCode, Summary: "oauth_userinfo_missing_identity"}
	}
	return user, nil
}

func (c *DingTalkClient) GetContactUserByUnionId(ctx context.Context, appAccessToken string, unionId string) (DingTalkContactUserInfo, error) {
	userURL, err := url.Parse(c.openAPIBaseURL + "/topapi/user/getbyunionid")
	if err != nil {
		return DingTalkContactUserInfo{}, err
	}
	q := userURL.Query()
	q.Set("access_token", strings.TrimSpace(appAccessToken))
	userURL.RawQuery = q.Encode()

	body, err := common.Marshal(dingTalkContactUserRequest{UnionId: strings.TrimSpace(unionId)})
	if err != nil {
		return DingTalkContactUserInfo{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, userURL.String(), bytes.NewReader(body))
	if err != nil {
		return DingTalkContactUserInfo{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return DingTalkContactUserInfo{}, &DingTalkAPIError{Stage: "contact_user", Summary: "contact_user_request_failed", Network: true}
	}
	defer resp.Body.Close()

	var payload dingTalkContactUserResponse
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return DingTalkContactUserInfo{}, &DingTalkAPIError{Stage: "contact_user", HTTPStatus: resp.StatusCode, Summary: "contact_user_response_invalid"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return DingTalkContactUserInfo{}, &DingTalkAPIError{Stage: "contact_user", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "contact_user_http_error"}
	}
	if payload.ErrCode != 0 {
		return DingTalkContactUserInfo{}, &DingTalkAPIError{Stage: "contact_user", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "contact_user_lookup_failed"}
	}
	payload.Result.UserId = strings.TrimSpace(payload.Result.UserId)
	payload.Result.Email = strings.TrimSpace(payload.Result.Email)
	payload.Result.Mobile = strings.TrimSpace(payload.Result.Mobile)
	return payload.Result, nil
}

func (c *DingTalkClient) ListSubDepartments(ctx context.Context, accessToken string, departmentId int64) ([]DingTalkDepartmentInfo, error) {
	listURL, err := url.Parse(c.openAPIBaseURL + "/topapi/v2/department/listsub")
	if err != nil {
		return nil, err
	}
	q := listURL.Query()
	q.Set("access_token", strings.TrimSpace(accessToken))
	listURL.RawQuery = q.Encode()

	body, err := common.Marshal(map[string]any{"dept_id": departmentId})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, listURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &DingTalkAPIError{Stage: "department_list", Summary: "department_list_request_failed", Network: true}
	}
	defer resp.Body.Close()

	var payload dingTalkDepartmentListResponse
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return nil, &DingTalkAPIError{Stage: "department_list", HTTPStatus: resp.StatusCode, Summary: "department_list_response_invalid"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &DingTalkAPIError{Stage: "department_list", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "department_list_http_error"}
	}
	if payload.ErrCode != 0 {
		return nil, &DingTalkAPIError{Stage: "department_list", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "department_list_failed"}
	}
	return payload.Result, nil
}

func (c *DingTalkClient) ListDepartmentUsers(ctx context.Context, accessToken string, departmentId int64) ([]DingTalkDepartmentUserInfo, error) {
	users := []DingTalkDepartmentUserInfo{}
	cursor := int64(0)
	for {
		page, hasMore, nextCursor, err := c.listDepartmentUserPage(ctx, accessToken, departmentId, cursor)
		if err != nil {
			return nil, err
		}
		users = append(users, page...)
		if !hasMore {
			return users, nil
		}
		cursor = nextCursor
	}
}

func (c *DingTalkClient) listDepartmentUserPage(ctx context.Context, accessToken string, departmentId int64, cursor int64) ([]DingTalkDepartmentUserInfo, bool, int64, error) {
	listURL, err := url.Parse(c.openAPIBaseURL + "/topapi/v2/user/list")
	if err != nil {
		return nil, false, 0, err
	}
	q := listURL.Query()
	q.Set("access_token", strings.TrimSpace(accessToken))
	listURL.RawQuery = q.Encode()

	body, err := common.Marshal(map[string]any{
		"dept_id": departmentId,
		"cursor":  cursor,
		"size":    100,
	})
	if err != nil {
		return nil, false, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, listURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, false, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, 0, &DingTalkAPIError{Stage: "department_users", Summary: "department_users_request_failed", Network: true}
	}
	defer resp.Body.Close()

	var payload dingTalkDepartmentUserListResponse
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return nil, false, 0, &DingTalkAPIError{Stage: "department_users", HTTPStatus: resp.StatusCode, Summary: "department_users_response_invalid"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, false, 0, &DingTalkAPIError{Stage: "department_users", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "department_users_http_error"}
	}
	if payload.ErrCode != 0 {
		return nil, false, 0, &DingTalkAPIError{Stage: "department_users", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "department_users_failed"}
	}
	return normalizeDingTalkDepartmentUsers(payload.Result.List), payload.Result.HasMore, payload.Result.NextCursor, nil
}

func (c *DingTalkClient) GetUserById(ctx context.Context, accessToken string, userId string) (DingTalkDepartmentUserInfo, error) {
	userURL, err := url.Parse(c.openAPIBaseURL + "/topapi/v2/user/get")
	if err != nil {
		return DingTalkDepartmentUserInfo{}, err
	}
	q := userURL.Query()
	q.Set("access_token", strings.TrimSpace(accessToken))
	userURL.RawQuery = q.Encode()

	body, err := common.Marshal(map[string]any{"userid": strings.TrimSpace(userId)})
	if err != nil {
		return DingTalkDepartmentUserInfo{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, userURL.String(), bytes.NewReader(body))
	if err != nil {
		return DingTalkDepartmentUserInfo{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return DingTalkDepartmentUserInfo{}, &DingTalkAPIError{Stage: "user_get", Summary: "user_get_request_failed", Network: true}
	}
	defer resp.Body.Close()

	var payload dingTalkUserGetResponse
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return DingTalkDepartmentUserInfo{}, &DingTalkAPIError{Stage: "user_get", HTTPStatus: resp.StatusCode, Summary: "user_get_response_invalid"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return DingTalkDepartmentUserInfo{}, &DingTalkAPIError{Stage: "user_get", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "user_get_http_error"}
	}
	if payload.ErrCode != 0 {
		return DingTalkDepartmentUserInfo{}, &DingTalkAPIError{Stage: "user_get", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "user_get_failed"}
	}
	user := payload.Result
	user.UserId = strings.TrimSpace(user.UserId)
	user.UnionId = strings.TrimSpace(user.UnionId)
	user.Email = strings.TrimSpace(user.Email)
	user.Mobile = strings.TrimSpace(user.Mobile)
	return user, nil
}

func (c *DingTalkClient) ProbeAddressBookPermission(ctx context.Context, accessToken string) error {
	probeURL, err := url.Parse(c.openAPIBaseURL + "/topapi/v2/department/listsub")
	if err != nil {
		return err
	}
	q := probeURL.Query()
	q.Set("access_token", accessToken)
	probeURL.RawQuery = q.Encode()

	body, err := common.Marshal(map[string]any{"dept_id": 1})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, probeURL.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &DingTalkAPIError{Stage: "address_book_probe", Summary: "address_book_probe_failed", Network: true}
	}
	defer resp.Body.Close()

	var payload dingTalkAPIEnvelope
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return &DingTalkAPIError{Stage: "address_book_probe", HTTPStatus: resp.StatusCode, Summary: "address_book_response_invalid"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &DingTalkAPIError{Stage: "address_book_probe", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "address_book_http_error"}
	}
	if payload.ErrCode != 0 {
		return &DingTalkAPIError{Stage: "address_book_probe", ErrCode: payload.ErrCode, HTTPStatus: resp.StatusCode, Summary: "address_book_permission_denied"}
	}
	return nil
}

func normalizeDingTalkDepartmentUsers(users []DingTalkDepartmentUserInfo) []DingTalkDepartmentUserInfo {
	normalized := make([]DingTalkDepartmentUserInfo, 0, len(users))
	for _, user := range users {
		user.UserId = strings.TrimSpace(user.UserId)
		user.UnionId = strings.TrimSpace(user.UnionId)
		user.Email = strings.TrimSpace(user.Email)
		user.Mobile = strings.TrimSpace(user.Mobile)
		if user.UserId == "" {
			continue
		}
		normalized = append(normalized, user)
	}
	return normalized
}

func dingTalkAPIError(err error) (*DingTalkAPIError, bool) {
	var apiErr *DingTalkAPIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}
