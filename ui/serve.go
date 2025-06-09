package ui

import (
	"embed"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

//go:embed build/*
var box embed.FS

// VersionInfo 版本信息结构
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

type uiConfig struct {
	Register bool        `json:"register"`
	Version  VersionInfo `json:"version"`
}

// Register registers the ui on the root path.
func Register(r *gin.Engine, version VersionInfo, register bool) {
	uiConfigBytes, err := json.Marshal(uiConfig{Version: version, Register: register})
	if err != nil {
		panic(err)
	}

	replaceConfig := func(content string) string {
		return strings.Replace(content, "%CONFIG%", string(uiConfigBytes), 1)
	}

	// 注册UI路由，使用中间件包装
	r.GET("/", gzip.Gzip(gzip.DefaultCompression), serveFile("index.html", "text/html", replaceConfig))
	r.GET("/index.html", gzip.Gzip(gzip.DefaultCompression), serveFile("index.html", "text/html", replaceConfig))
	r.GET("/manifest.json", gzip.Gzip(gzip.DefaultCompression), serveFile("manifest.json", "application/json", noop))
	r.GET("/asset-manifest.json", gzip.Gzip(gzip.DefaultCompression), serveFile("asset-manifest.json", "application/json", noop))

	// 创建静态文件处理器
	staticHandler := func(c *gin.Context) {
		// 获取文件路径参数
		filepath := c.Param("filepath")

		// 构建完整文件路径
		fullPath := "build/static" + filepath

		// 尝试从build目录读取文件
		content, err := box.ReadFile(fullPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		// 设置适当的Content-Type
		var contentType string
		if strings.HasSuffix(filepath, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(filepath, ".js") {
			contentType = "application/javascript"
		} else if strings.HasSuffix(filepath, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(filepath, ".ico") {
			contentType = "image/x-icon"
		} else {
			contentType = http.DetectContentType(content)
		}

		c.Header("Content-Type", contentType)
		c.Data(http.StatusOK, contentType, content)
	}

	// 使用更具体的路径来避免与其他通配符路由冲突
	r.GET("/static/*filepath", gzip.Gzip(gzip.DefaultCompression), staticHandler)
}

func noop(s string) string {
	return s
}

func serveFile(name, contentType string, convert func(string) string) gin.HandlerFunc {
	content, err := box.ReadFile("build/" + name)
	if err != nil {
		panic(err)
	}
	converted := convert(string(content))
	return func(ctx *gin.Context) {
		ctx.Header("Content-Type", contentType)
		ctx.String(200, converted)
	}
}
