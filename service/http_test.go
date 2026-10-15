package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCaptureResponseRequestIDsStoresUpstreamAndClientIDsSeparately(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("client request id does not replace upstream request id", func(t *testing.T) {
		ctx := newHTTPHeaderTestContext()
		header := http.Header{}
		header.Set(common.ClientRequestIdKey, " 12ee1873-ba89-403f-bbe6-bc5ebe4b2e9f ")

		CaptureResponseRequestIDs(ctx, header)

		assert.Empty(t, ctx.GetString(common.UpstreamRequestIdKey))
		assert.Equal(t, "12ee1873-ba89-403f-bbe6-bc5ebe4b2e9f", ctx.GetString(common.ClientRequestIdKey))
	})

	t.Run("one api request id remains upstream request id", func(t *testing.T) {
		ctx := newHTTPHeaderTestContext()
		header := http.Header{}
		header.Set(common.ClientRequestIdKey, "client-upstream")
		header.Set(common.RequestIdKey, "one-api-upstream")

		CaptureResponseRequestIDs(ctx, header)

		assert.Equal(t, "one-api-upstream", ctx.GetString(common.UpstreamRequestIdKey))
		assert.Equal(t, "client-upstream", ctx.GetString(common.ClientRequestIdKey))
	})
}

func TestShouldCopyUpstreamHeaderCapturesResponseRequestIDsSeparately(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := newHTTPHeaderTestContext()

	assert.True(t, ShouldCopyUpstreamHeader(ctx, common.ClientRequestIdKey, []string{"client-upstream"}))
	assert.Equal(t, "client-upstream", ctx.GetString(common.ClientRequestIdKey))
	assert.Empty(t, ctx.GetString(common.UpstreamRequestIdKey))

	assert.False(t, ShouldCopyUpstreamHeader(ctx, common.RequestIdKey, []string{"one-api-upstream"}))
	assert.Equal(t, "one-api-upstream", ctx.GetString(common.UpstreamRequestIdKey))

	assert.True(t, ShouldCopyUpstreamHeader(ctx, common.ClientRequestIdKey, []string{"new-client-upstream"}))
	assert.Equal(t, "one-api-upstream", ctx.GetString(common.UpstreamRequestIdKey))
	assert.Equal(t, "new-client-upstream", ctx.GetString(common.ClientRequestIdKey))
}

func newHTTPHeaderTestContext() *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	return ctx
}
