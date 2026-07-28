package middleware

import (
	"errors"
	"net/http"

	serviceaionui "github.com/QuantumNous/new-api/service/aionui"
	"github.com/gin-gonic/gin"
)

func AionUiDesktopAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := serviceaionui.DefaultDesktopAuthService().ValidateAuthorization(c.GetHeader("Authorization"))
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, serviceaionui.ErrUserUnavailable) || errors.Is(err, serviceaionui.ErrEmailRequired) {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{
				"success": false,
				"message": err.Error(),
			})
			c.Abort()
			return
		}
		c.Set("aionui_user_id", claims.UserId)
		c.Set("aionui_username", claims.Username)
		c.Set("aionui_email", claims.Email)
		c.Set("aionui_device_id", claims.DeviceId)
		c.Set("aionui_desktop_token", claims.Token)
		c.Next()
	}
}
