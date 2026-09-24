package controller

import (
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
)

const maxClientRequestIDBytes = 128

func captureCodexClientRequestID(c *gin.Context, relayFormat types.RelayFormat, request dto.Request) {
	if !shouldCaptureCodexClientRequestID(c, relayFormat) {
		return
	}

	clientRequestID := firstValidClientRequestID(
		c.GetHeader(common.ClientRequestIdKey),
		c.GetHeader("Thread-Id"),
		c.GetHeader("Thread_id"),
		c.GetHeader("Session-Id"),
		c.GetHeader("Session_id"),
		clientRequestIDFromRequest(request),
	)
	if clientRequestID == "" {
		return
	}
	c.Set(common.InboundRequestIdKey, clientRequestID)
}

func shouldCaptureCodexClientRequestID(c *gin.Context, relayFormat types.RelayFormat) bool {
	path := c.Request.URL.Path
	return (relayFormat == types.RelayFormatOpenAI && path == "/v1/chat/completions") ||
		(relayFormat == types.RelayFormatOpenAIResponses && path == "/v1/responses")
}

func firstValidClientRequestID(candidates ...string) string {
	for _, candidate := range candidates {
		if value := normalizeClientRequestID(candidate); value != "" {
			return value
		}
	}
	return ""
}

func normalizeClientRequestID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxClientRequestIDBytes || !utf8.ValidString(value) {
		return ""
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return value
}

func clientRequestIDFromRequest(request dto.Request) string {
	switch req := request.(type) {
	case *dto.OpenAIResponsesRequest:
		return clientRequestIDFromMetadata(req.ClientMetadata)
	case *dto.GeneralOpenAIRequest:
		return clientRequestIDFromMetadata(req.Metadata)
	default:
		return ""
	}
}

func clientRequestIDFromMetadata(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}

	var metadata map[string]any
	if err := common.Unmarshal(raw, &metadata); err != nil {
		return ""
	}

	for _, key := range []string{
		"x-client-request-id",
		"client_request_id",
		"clientRequestId",
		"thread_id",
		"threadId",
		"session_id",
		"sessionId",
	} {
		if value, ok := metadata[key].(string); ok {
			return value
		}
	}

	for _, key := range []string{"x-codex-turn-metadata", "turn_metadata", "turnMetadata"} {
		switch value := metadata[key].(type) {
		case map[string]any:
			if threadID, ok := value["thread_id"].(string); ok {
				return threadID
			}
			if threadID, ok := value["threadId"].(string); ok {
				return threadID
			}
		case string:
			var turnMetadata map[string]any
			if err := common.UnmarshalJsonStr(value, &turnMetadata); err != nil {
				continue
			}
			if threadID, ok := turnMetadata["thread_id"].(string); ok {
				return threadID
			}
			if threadID, ok := turnMetadata["threadId"].(string); ok {
				return threadID
			}
		}
	}

	return ""
}
