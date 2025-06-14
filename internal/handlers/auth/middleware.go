package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 声明外部函数类型，避免循环导入
var UpdateSessionLastUsedFunc func(string)

// SetUpdateSessionLastUsedFunc 设置会话更新函数
func SetUpdateSessionLastUsedFunc(fn func(string)) {
	UpdateSessionLastUsedFunc = fn
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("X-GoHook-Key")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 更新会话最后使用时间
		if UpdateSessionLastUsedFunc != nil {
			UpdateSessionLastUsedFunc(tokenString)
		}

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)
		c.Next()
	}
}

// WSAuthMiddleware WebSocket专用认证中间件，支持查询参数token
func WSAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从Header获取token
		tokenString := c.GetHeader("X-GoHook-Key")

		// 如果Header中没有token，从查询参数获取
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 更新会话最后使用时间
		if UpdateSessionLastUsedFunc != nil {
			UpdateSessionLastUsedFunc(tokenString)
		}

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)
		c.Next()
	}
}

// AdminMiddleware 管理员权限中间件
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
