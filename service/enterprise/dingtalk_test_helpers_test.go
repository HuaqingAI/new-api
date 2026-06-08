package enterprise_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

type dingTalkRoundTripFunc func(*http.Request) (*http.Response, error)

func (f dingTalkRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func dingTalkJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func dingTalkStatusResponse(status int, body string) *http.Response {
	resp := dingTalkJSONResponse(body)
	resp.StatusCode = status
	return resp
}

func dingTalkNetworkFailureRoundTripper() http.RoundTripper {
	return dingTalkRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network unavailable")
	})
}
