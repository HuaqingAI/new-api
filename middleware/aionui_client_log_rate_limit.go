package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/gin-gonic/gin"
)

const (
	aionUiClientLogUploadRateLimit     = 3
	aionUiClientLogUploadRateWindowSec = int64(15 * 60)
	aionUiClientLogUploadRateMark      = "AIONUI_CLIENT_LOG_UPLOAD"
)

var aionUiClientLogUploadMemory = struct {
	entries     map[int]aionUiClientLogUploadRateEntry
	lastCleanup int64
	mu          sync.Mutex
}{entries: make(map[int]aionUiClientLogUploadRateEntry)}

type aionUiClientLogUploadRateEntry struct {
	count       int
	windowStart int64
}

func AionUiClientLogUploadRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt("aionui_user_id")
		if userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized", "code": "unauthorized"})
			c.Abort()
			return
		}
		if common.RedisEnabled {
			allowed, _, ttl, err := redisFixedWindowTake(c.Request.Context(), redisUserRateLimitKey(aionUiClientLogUploadRateMark, userID), aionUiClientLogUploadRateLimit, aionUiClientLogUploadRateWindowSec)
			if err == nil {
				if !allowed {
					writeAionUiClientLogUploadRateLimited(c, ttl)
					return
				}
				c.Next()
				return
			}
			// Redis is an optimization for the fixed-window counter. If it is
			// unavailable, preserve the endpoint's availability with the expiring
			// process-local fallback instead of failing every upload closed.
			logger.LogError(c.Request.Context(), "AionUI client log upload rate limit check failed; using in-memory fallback")
		}
		if !takeAionUiClientLogUploadMemory(userID, time.Now().Unix()) {
			writeAionUiClientLogUploadRateLimited(c, aionUiClientLogUploadRateWindowSec)
			return
		}
		c.Next()
	}
}

func takeAionUiClientLogUploadMemory(userID int, now int64) bool {
	windowStart := now - now%aionUiClientLogUploadRateWindowSec
	aionUiClientLogUploadMemory.mu.Lock()
	defer aionUiClientLogUploadMemory.mu.Unlock()
	if aionUiClientLogUploadMemory.lastCleanup == 0 ||
		(now >= aionUiClientLogUploadMemory.lastCleanup && now-aionUiClientLogUploadMemory.lastCleanup >= aionUiClientLogUploadRateWindowSec) {
		for key, entry := range aionUiClientLogUploadMemory.entries {
			if now >= entry.windowStart && now-entry.windowStart >= aionUiClientLogUploadRateWindowSec {
				delete(aionUiClientLogUploadMemory.entries, key)
			}
		}
		aionUiClientLogUploadMemory.lastCleanup = now
	}
	entry := aionUiClientLogUploadMemory.entries[userID]
	if entry.windowStart != windowStart {
		entry = aionUiClientLogUploadRateEntry{windowStart: windowStart}
	}
	if entry.count >= aionUiClientLogUploadRateLimit {
		aionUiClientLogUploadMemory.entries[userID] = entry
		return false
	}
	entry.count++
	aionUiClientLogUploadMemory.entries[userID] = entry
	return true
}

func writeAionUiClientLogUploadRateLimited(c *gin.Context, retryAfterSeconds int64) {
	if retryAfterSeconds <= 0 {
		retryAfterSeconds = aionUiClientLogUploadRateWindowSec
	}
	c.Header("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
	logger.LogInfo(c.Request.Context(), fmt.Sprintf(
		"event=aionui_client_log_upload user_id=%d device_hash=%s client_version= log_date= file_count=0 total_bytes=0 result=rate_limited duration_ms=0",
		c.GetInt("aionui_user_id"),
		common.AionUiDeviceAuditHash(c.GetString("aionui_device_id")),
	))
	c.JSON(http.StatusTooManyRequests, gin.H{"success": false, "message": "rate_limited", "code": "rate_limited"})
	c.Abort()
}
