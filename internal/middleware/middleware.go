package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/client"
)

// wsAuthMiddleware WebSocket auth middleware, support query parameter token and Sec-WebSocket-Protocol header
func WsAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// get token from header first
		tokenString := c.GetHeader("X-GoHook-Key")

		// if no token in header, try to get it from Sec-WebSocket-Protocol header
		if tokenString == "" {
			protocols := c.GetHeader("Sec-WebSocket-Protocol")
			if protocols != "" {
				// parse protocols: "Authorization, <token>"
				parts := strings.Split(protocols, ",")
				if len(parts) >= 2 {
					// trim whitespace and check if first part is "Authorization"
					protocol := strings.TrimSpace(parts[0])
					if protocol == "Authorization" {
						tokenString = strings.TrimSpace(parts[1])
					}
				}
			}
		}

		// if still no token, get it from query parameter (fallback)
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		claims, err := client.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// update session last used time
		client.UpdateSessionLastUsed(tokenString)

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)
		c.Next()
	}
}

// adminMiddleware admin permission middleware
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// authMiddleware JWT auth middleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("X-GoHook-Key")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		claims, err := client.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// update session last used time
		client.UpdateSessionLastUsed(tokenString)

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)
		c.Next()
	}
}

// noLogMiddleware disable logging for the request
func DisableLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// set a flag to indicate that this request should not be logged
		c.Set("disable_log", true)
		c.Next()
	}
}
