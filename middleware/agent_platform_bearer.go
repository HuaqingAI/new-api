package middleware

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	apservice "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
)

const (
	AgentPlatformClaimsKey = "agent_platform_claims"
)

func AgentPlatformBearer(requiredScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := apservice.NewOAuthTokenService(model.DB).ValidateAccessToken(c.GetHeader("Authorization"))
		if err != nil {
			requestID, _ := c.Get(common.RequestIdKey)
			c.JSON(200, apservice.MapOpenCapabilityError(err, apservice.OpenCapabilityContext{
				RequestID: apservice.OpenCapabilityRequestIDFromGinValue(requestID),
			}))
			c.Abort()
			return
		}
		if !hasAgentPlatformScopes(claims.Scope, requiredScopes...) {
			requestID, _ := c.Get(common.RequestIdKey)
			c.JSON(200, apservice.MapOpenCapabilityError(apservice.ErrOpenCapabilityPermissionDenied, apservice.OpenCapabilityContext{
				RequestID: apservice.OpenCapabilityRequestIDFromGinValue(requestID),
			}))
			c.Abort()
			return
		}

		c.Set(AgentPlatformClaimsKey, claims)
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "agent-platform")
		c.Next()
	}
}

func hasAgentPlatformScopes(granted string, required ...string) bool {
	if len(required) == 0 {
		return true
	}
	scopeSet := map[string]struct{}{}
	for _, value := range strings.Fields(strings.TrimSpace(granted)) {
		scopeSet[strings.TrimSpace(value)] = struct{}{}
	}
	for _, need := range required {
		if _, ok := scopeSet[strings.TrimSpace(need)]; !ok {
			return false
		}
	}
	return true
}
