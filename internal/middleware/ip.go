package middleware

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRealIP get real client IP address, support proxy environment
// priority: X-Forwarded-For > X-Real-IP > X-Client-IP > RemoteAddr
func GetRealIP(c *gin.Context) string {
	// get all possible IP headers
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	xRealIP := c.GetHeader("X-Real-IP")
	xClientIP := c.GetHeader("X-Client-IP")
	cfConnectingIP := c.GetHeader("CF-Connecting-IP") // Cloudflare
	trueClientIP := c.GetHeader("True-Client-IP")     // Akamai

	// 1. first check X-Forwarded-For
	if xForwardedFor != "" {
		// X-Forwarded-For may contain multiple IPs, format: client, proxy1, proxy2
		ips := strings.Split(xForwardedFor, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if isValidIP(ip) && !isPrivateIP(ip) {
				return ip
			}
		}
	}

	// 2. check CF-Connecting-IP (Cloudflare)
	if cfConnectingIP != "" && isValidIP(cfConnectingIP) {
		return cfConnectingIP
	}

	// 3. check True-Client-IP (Akamai)
	if trueClientIP != "" && isValidIP(trueClientIP) {
		return trueClientIP
	}

	// 4. check X-Real-IP
	if xRealIP != "" && isValidIP(xRealIP) {
		return xRealIP
	}

	// 5. check X-Client-IP
	if xClientIP != "" && isValidIP(xClientIP) {
		return xClientIP
	}

	// 6. if none of the above, use Gin's ClientIP() method
	clientIP := c.ClientIP()
	if clientIP != "" {
		return clientIP
	}

	// 7. finally fallback to RemoteAddr
	if c.Request.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err == nil {
			return host
		}
		return c.Request.RemoteAddr
	}

	return "unknown"
}

// isValidIP check if IP is valid
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// isPrivateIP check if IP is private
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// check if IP is private
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

// IPMiddleware Gin middleware, set real IP to context
func IPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		realIP := GetRealIP(c)
		c.Set("real_ip", realIP)
		c.Next()
	}
}

// GetClientIP get client IP from context (if middleware is used)
// if middleware is not used, call GetRealIP directly
func GetClientIP(c *gin.Context) string {
	if realIP, exists := c.Get("real_ip"); exists {
		if ip, ok := realIP.(string); ok {
			return ip
		}
	}
	return GetRealIP(c)
}
