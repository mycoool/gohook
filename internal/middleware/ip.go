package middleware

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRealIP 获取真实客户端IP地址，支持代理环境
// 优先级：X-Forwarded-For > X-Real-IP > X-Client-IP > RemoteAddr
func GetRealIP(c *gin.Context) string {
	// 获取所有可能的IP头部
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	xRealIP := c.GetHeader("X-Real-IP")
	xClientIP := c.GetHeader("X-Client-IP")
	cfConnectingIP := c.GetHeader("CF-Connecting-IP") // Cloudflare
	trueClientIP := c.GetHeader("True-Client-IP")     // Akamai

	// 1. 首先检查 X-Forwarded-For
	if xForwardedFor != "" {
		// X-Forwarded-For 可能包含多个IP，格式：client, proxy1, proxy2
		ips := strings.Split(xForwardedFor, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if isValidIP(ip) && !isPrivateIP(ip) {
				return ip
			}
		}
	}

	// 2. 检查 CF-Connecting-IP (Cloudflare)
	if cfConnectingIP != "" && isValidIP(cfConnectingIP) {
		return cfConnectingIP
	}

	// 3. 检查 True-Client-IP (Akamai)
	if trueClientIP != "" && isValidIP(trueClientIP) {
		return trueClientIP
	}

	// 4. 检查 X-Real-IP
	if xRealIP != "" && isValidIP(xRealIP) {
		return xRealIP
	}

	// 5. 检查 X-Client-IP
	if xClientIP != "" && isValidIP(xClientIP) {
		return xClientIP
	}

	// 6. 如果以上都没有，使用 Gin 的 ClientIP() 方法
	clientIP := c.ClientIP()
	if clientIP != "" {
		return clientIP
	}

	// 7. 最后回退到 RemoteAddr
	if c.Request.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err == nil {
			return host
		}
		return c.Request.RemoteAddr
	}

	return "unknown"
}

// isValidIP 检查是否为有效的IP地址
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// isPrivateIP 检查是否为私有IP地址
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// 检查是否为私有IP地址段
	privateIPBlocks := []*net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},     // 10.0.0.0/8
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},  // 172.16.0.0/12
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)}, // 192.168.0.0/16
		{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},    // 127.0.0.0/8 (loopback)
		{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}, // 169.254.0.0/16 (link-local)
	}

	for _, block := range privateIPBlocks {
		if block.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// IPMiddleware Gin中间件，将真实IP设置到上下文中
func IPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		realIP := GetRealIP(c)
		c.Set("real_ip", realIP)
		c.Next()
	}
}

// GetClientIP 从上下文中获取客户端IP（如果使用了中间件）
// 如果没有使用中间件，则直接调用GetRealIP
func GetClientIP(c *gin.Context) string {
	if realIP, exists := c.Get("real_ip"); exists {
		if ip, ok := realIP.(string); ok {
			return ip
		}
	}
	return GetRealIP(c)
}
