package common

import (
	"crypto/sha256"
	"encoding/hex"
)

// AionUiDeviceAuditHash returns a short, non-reversible identifier suitable for logs.
func AionUiDeviceAuditHash(deviceID string) string {
	if deviceID == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(deviceID))
	return hex.EncodeToString(digest[:12])
}
