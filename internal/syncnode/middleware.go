package syncnode

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AgentTokenMiddleware validates sync agent token for node-scoped endpoints.
// It accepts token from:
// - X-Sync-Token header
// - Authorization: Bearer <token>
// - JSON body field "token" (fallback, body is restored for handlers)
func AgentTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseIDParam(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		token := extractAgentToken(c)
		node, err := defaultService.ValidateAgentToken(c.Request.Context(), id, token)
		if err != nil {
			status := http.StatusInternalServerError
			if err == ErrInvalidToken {
				status = http.StatusUnauthorized
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("sync_node", node)
		c.Next()
	}
}

func extractAgentToken(c *gin.Context) string {
	token := strings.TrimSpace(c.GetHeader("X-Sync-Token"))
	if token != "" {
		return token
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token = strings.TrimSpace(auth[7:])
	}
	if token != "" {
		return token
	}

	// Fallback: attempt to read token from JSON body.
	if c.Request.Body == nil {
		return ""
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	_ = c.Request.Body.Close()
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var payload struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(bodyBytes, &payload)
	return strings.TrimSpace(payload.Token)
}
