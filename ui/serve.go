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

// VersionInfo version info structure
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

type uiConfig struct {
	Register bool        `json:"register"`
	Version  VersionInfo `json:"version"`
}

// noLogMiddleware disable logging for the request
func noLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// set a flag to indicate that this request should not be logged
		c.Set("no_log", true)
		c.Next()
	}
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

	// register ui routes, use middleware to disable logging and gzip compression
	r.GET("/", noLogMiddleware(), gzip.Gzip(gzip.DefaultCompression), serveFile("index.html", "text/html", replaceConfig))
	r.GET("/index.html", noLogMiddleware(), gzip.Gzip(gzip.DefaultCompression), serveFile("index.html", "text/html", replaceConfig))
	r.GET("/manifest.json", noLogMiddleware(), gzip.Gzip(gzip.DefaultCompression), serveFile("manifest.json", "application/json", noop))
	r.GET("/asset-manifest.json", noLogMiddleware(), gzip.Gzip(gzip.DefaultCompression), serveFile("asset-manifest.json", "application/json", noop))

	// create static file handler
	staticHandler := func(c *gin.Context) {
		// get file path parameter
		filepath := c.Param("filepath")

		// build full file path
		fullPath := "build/static" + filepath

		// try to read file from build directory
		content, err := box.ReadFile(fullPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		// set appropriate Content-Type
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

	// use more specific path to avoid conflicts with other wildcard routes (disable logging + gzip compression)
	r.GET("/static/*filepath", noLogMiddleware(), gzip.Gzip(gzip.DefaultCompression), staticHandler)
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
