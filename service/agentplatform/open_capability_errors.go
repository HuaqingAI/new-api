package agentplatform

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	OpenCapabilityCodePermissionDenied = "permissionDenied"
	OpenCapabilityCodeResourceRevoked  = "resourceRevoked"
	OpenCapabilityCodeResourceOffline  = "resourceOffline"
	OpenCapabilityCodeQuotaLimited     = "quotaOrRateLimited"
	OpenCapabilityCodeTimeout          = "timeout"
	OpenCapabilityCodeUpstreamFailed   = "upstreamFailed"
	OpenCapabilityCodeContractInvalid  = "contractInvalid"
)

var (
	ErrOpenCapabilityPermissionDenied = errors.New("agent platform open capability permission denied")
	ErrOpenCapabilityResourceNotFound = errors.New("agent platform open capability resource not found")
	ErrOpenCapabilityResourceRevoked  = errors.New("agent platform open capability resource revoked")
	ErrOpenCapabilityResourceOffline  = errors.New("agent platform open capability resource offline")
	ErrOpenCapabilityContractInvalid  = errors.New("agent platform open capability contract invalid")
)

type OpenCapabilityError struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	Retryable       bool   `json:"retryable"`
	RequestID       string `json:"request_id,omitempty"`
	ResourceID      string `json:"resource_id,omitempty"`
	ResourceVersion string `json:"resource_version,omitempty"`
}

type OpenCapabilityErrorResponse struct {
	Success bool                `json:"success"`
	Error   OpenCapabilityError `json:"error"`
}

type OpenCapabilityContext struct {
	RequestID       string
	ResourceID      string
	ResourceVersion string
}

func MapOpenCapabilityError(err error, ctx OpenCapabilityContext) OpenCapabilityErrorResponse {
	response := OpenCapabilityErrorResponse{
		Success: false,
		Error: OpenCapabilityError{
			Code:            OpenCapabilityCodeUpstreamFailed,
			Message:         "open capability request failed",
			Retryable:       false,
			RequestID:       strings.TrimSpace(ctx.RequestID),
			ResourceID:      strings.TrimSpace(ctx.ResourceID),
			ResourceVersion: strings.TrimSpace(ctx.ResourceVersion),
		},
	}

	switch {
	case errors.Is(err, ErrOpenCapabilityPermissionDenied), errors.Is(err, ErrUnauthorizedClient), errors.Is(err, ErrGrantRevoked):
		response.Error.Code = OpenCapabilityCodePermissionDenied
		response.Error.Message = "permission denied"
	case errors.Is(err, ErrOpenCapabilityResourceNotFound), errors.Is(err, ErrResourceNotFound), errors.Is(err, ErrExposureNotFound), errors.Is(err, ErrResourceVersionNotFound):
		response.Error.Code = OpenCapabilityCodePermissionDenied
		response.Error.Message = "resource not published for client"
	case errors.Is(err, ErrOpenCapabilityResourceRevoked):
		response.Error.Code = OpenCapabilityCodeResourceRevoked
		response.Error.Message = "resource revoked"
	case errors.Is(err, ErrOpenCapabilityResourceOffline):
		response.Error.Code = OpenCapabilityCodeResourceOffline
		response.Error.Message = "resource offline"
	case errors.Is(err, ErrOpenCapabilityContractInvalid), errors.Is(err, ErrInvalidTokenExchangeInput):
		response.Error.Code = OpenCapabilityCodeContractInvalid
		response.Error.Message = "contract invalid"
	case errors.Is(err, ErrSkillInvokeTimeout):
		response.Error.Code = OpenCapabilityCodeTimeout
		response.Error.Message = "skill invoke timeout"
		response.Error.Retryable = true
	case errors.Is(err, ErrSkillInvokeUpstreamFailed):
		response.Error.Code = OpenCapabilityCodeUpstreamFailed
		response.Error.Message = "skill invoke upstream failed"
		response.Error.Retryable = true
	case errors.Is(err, ErrRefreshTokenInvalid):
		response.Error.Code = OpenCapabilityCodePermissionDenied
		response.Error.Message = "permission denied"
	default:
		if err != nil && strings.TrimSpace(err.Error()) != "" {
			response.Error.Message = err.Error()
		}
	}

	return response
}

func OpenCapabilityRequestID(ctx any) string {
	switch value := ctx.(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func OpenCapabilityRequestIDFromGinValue(value any) string {
	return strings.TrimSpace(common.Interface2String(value))
}
