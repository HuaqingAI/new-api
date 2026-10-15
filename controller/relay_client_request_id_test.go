package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureCodexClientRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("chat completion header", func(t *testing.T) {
		ctx := newClientRequestIDTestContext("/v1/chat/completions")
		ctx.Request.Header.Set(common.ClientRequestIdKey, " thread-chat ")

		captureCodexClientRequestID(ctx, types.RelayFormatOpenAI, &dto.GeneralOpenAIRequest{})

		assert.Equal(t, "thread-chat", ctx.GetString(common.InboundRequestIdKey))
		assert.Empty(t, ctx.GetString(common.ClientRequestIdKey))
	})

	t.Run("responses client metadata fallback", func(t *testing.T) {
		ctx := newClientRequestIDTestContext("/v1/responses")
		clientMetadata, err := common.Marshal(map[string]any{"thread_id": "thread-responses"})
		require.NoError(t, err)

		captureCodexClientRequestID(ctx, types.RelayFormatOpenAIResponses, &dto.OpenAIResponsesRequest{
			ClientMetadata: clientMetadata,
		})

		assert.Equal(t, "thread-responses", ctx.GetString(common.InboundRequestIdKey))
	})

	t.Run("ignores other endpoints", func(t *testing.T) {
		ctx := newClientRequestIDTestContext("/v1/completions")
		ctx.Request.Header.Set(common.ClientRequestIdKey, "thread-completions")

		captureCodexClientRequestID(ctx, types.RelayFormatOpenAI, &dto.GeneralOpenAIRequest{})

		assert.Empty(t, ctx.GetString(common.InboundRequestIdKey))
	})

	t.Run("ignores invalid values", func(t *testing.T) {
		ctx := newClientRequestIDTestContext("/v1/chat/completions")
		ctx.Request.Header.Set(common.ClientRequestIdKey, strings.Repeat("x", maxClientRequestIDBytes+1))

		captureCodexClientRequestID(ctx, types.RelayFormatOpenAI, &dto.GeneralOpenAIRequest{})

		assert.Empty(t, ctx.GetString(common.InboundRequestIdKey))
	})
}

func newClientRequestIDTestContext(path string) *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return ctx
}
