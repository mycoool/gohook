package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/ui"
)

// 版本信息
var vInfo = &ui.VersionInfo{
	Version:   "2.8.2", // 与app.go中的version常量保持一致
	Commit:    "unknown",
	BuildDate: "unknown",
}

// 配置信息
type Config struct {
	Registration bool
}

var conf = &Config{
	Registration: true, // 允许注册
}

func InitRouter() *gin.Engine {
	g := gin.Default()

	// 注册前端UI路由，这将接管根路径 "/"
	ui.Register(g, *vInfo, conf.Registration)

	g.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 登录接口
	g.POST("/login", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		if username == "admin" && password == "123456" {
			c.JSON(http.StatusOK, gin.H{"message": "登录成功"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "登录失败"})
		}
	})
	// 获取用户列表
	g.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{"admin", "user"}})
	})
	return g
}
