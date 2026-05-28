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
	defaultDingTalkAPITimeout     = 10 * time.Second
)

type DingTalkHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type DingTalkClient struct {
	httpClient     DingTalkHTTPClient
	openAPIBaseURL string
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

func dingTalkAPIError(err error) (*DingTalkAPIError, bool) {
	var apiErr *DingTalkAPIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}
