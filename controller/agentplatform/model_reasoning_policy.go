package agentplatform

import (
	"errors"
	"io"
	"net/http"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	serviceagentplatform "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func GetModelReasoningPolicy(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	policy, err := serviceagentplatform.GetModelReasoningPolicy()
	if err != nil {
		common.SysError(common.LogText("failed to read model reasoning settings"))
		common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to load model reasoning settings"), gin.H{"code": "MODEL_REASONING_POLICY_UNAVAILABLE"})
		return
	}
	common.ApiSuccess(c, policy)
}

func UpdateModelReasoningPolicy(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	const maxRequestBytes = 64 * 1024
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxRequestBytes+1))
	if len(body) > maxRequestBytes {
		common.ApiErrorStatus(c, http.StatusRequestEntityTooLarge, setting.ErrModelReasoningPolicyTooLarge, gin.H{"code": "MODEL_REASONING_POLICY_TOO_LARGE"})
		return
	}
	var request map[string]common.RawMessage
	if err != nil || !utf8.Valid(body) || common.Unmarshal(body, &request) != nil || len(request) != 1 || request["rules"] == nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, setting.ErrModelReasoningPolicyInvalid, gin.H{"code": "MODEL_REASONING_POLICY_INVALID"})
		return
	}
	raw, err := common.Marshal(struct {
		SchemaVersion int               `json:"schema_version"`
		Rules         common.RawMessage `json:"rules"`
	}{1, request["rules"]})
	if err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, setting.ErrModelReasoningPolicyInvalid, gin.H{"code": "MODEL_REASONING_POLICY_INVALID"})
		return
	}
	policy, canonical, err := setting.NormalizeModelReasoningPolicy(string(raw))
	if err != nil {
		if errors.Is(err, setting.ErrModelReasoningPolicyTooLarge) {
			common.ApiErrorStatus(c, http.StatusRequestEntityTooLarge, err, gin.H{"code": "MODEL_REASONING_POLICY_TOO_LARGE"})
			return
		}
		common.ApiErrorStatus(c, http.StatusBadRequest, err, gin.H{"code": "MODEL_REASONING_POLICY_INVALID"})
		return
	}
	if err := model.UpdateOptionsBulk(map[string]string{setting.ModelReasoningPolicyKey: canonical}); err != nil {
		common.SysError(common.LogText("failed to save model reasoning settings"))
		common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to save model reasoning settings"), gin.H{"code": "MODEL_REASONING_POLICY_SAVE_FAILED"})
		return
	}
	effective, err := policy.Effective()
	if err != nil {
		common.ApiErrorStatus(c, http.StatusInternalServerError, common.NewMessage("Failed to load model reasoning settings"), gin.H{"code": "MODEL_REASONING_POLICY_UNAVAILABLE"})
		return
	}
	common.ApiSuccess(c, effective)
}
