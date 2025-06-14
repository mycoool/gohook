package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/types"
)

// 声明外部函数类型，避免循环导入
var GetAppConfigFunc func() *types.AppConfig
var SaveAppConfigFunc func() error
var FindUserFunc func(string) *types.UserConfig
var HashPasswordFunc func(string) string
var VerifyPasswordFunc func(string, string) bool

// SetUserFunctions 设置用户管理相关函数
func SetUserFunctions(
	getAppConfig func() *types.AppConfig,
	saveAppConfig func() error,
	findUser func(string) *types.UserConfig,
	hashPassword func(string) string,
	verifyPassword func(string, string) bool,
) {
	GetAppConfigFunc = getAppConfig
	SaveAppConfigFunc = saveAppConfig
	FindUserFunc = findUser
	HashPasswordFunc = hashPassword
	VerifyPasswordFunc = verifyPassword
}

// GetUsersHandler 获取所有用户列表 (仅管理员)
func GetUsersHandler(c *gin.Context) {
	if GetAppConfigFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}

	appConfig := GetAppConfigFunc()
	if appConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User config not loaded"})
		return
	}

	var users []types.UserResponse
	for _, user := range appConfig.Users {
		users = append(users, types.UserResponse{
			Username: user.Username,
			Role:     user.Role,
		})
	}
	c.JSON(http.StatusOK, users)
}

// CreateUserHandler 创建用户 (仅管理员)
func CreateUserHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// 检查用户是否已存在
	if FindUserFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	if FindUserFunc(req.Username) != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	// 验证角色
	if req.Role != "admin" && req.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Must be 'admin' or 'user'"})
		return
	}

	// 添加新用户
	if GetAppConfigFunc == nil || HashPasswordFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}

	appConfig := GetAppConfigFunc()
	if appConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User config not loaded"})
		return
	}

	newUser := types.UserConfig{
		Username: req.Username,
		Password: HashPasswordFunc(req.Password),
		Role:     req.Role,
	}

	appConfig.Users = append(appConfig.Users, newUser)

	// 保存配置文件
	if SaveAppConfigFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	if err := SaveAppConfigFunc(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User created successfully",
		"user": types.UserResponse{
			Username: newUser.Username,
			Role:     newUser.Role,
		},
	})
}

// DeleteUserHandler 删除用户 (仅管理员)
func DeleteUserHandler(c *gin.Context) {
	username := c.Param("username")
	currentUser, _ := c.Get("username")

	// 不能删除自己
	if username == currentUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
		return
	}

	if GetAppConfigFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}

	appConfig := GetAppConfigFunc()
	if appConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User config not loaded"})
		return
	}

	// 查找用户索引
	userIndex := -1
	for i, user := range appConfig.Users {
		if user.Username == username {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 删除用户
	appConfig.Users = append(appConfig.Users[:userIndex], appConfig.Users[userIndex+1:]...)

	// 保存配置文件
	if SaveAppConfigFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	if err := SaveAppConfigFunc(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// ChangePasswordHandler 修改密码
func ChangePasswordHandler(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	username, _ := c.Get("username")
	if FindUserFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	user := FindUserFunc(username.(string))
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 验证旧密码
	if VerifyPasswordFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	if !VerifyPasswordFunc(req.OldPassword, user.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid old password"})
		return
	}

	// 更新密码
	if HashPasswordFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	user.Password = HashPasswordFunc(req.NewPassword)

	// 保存配置文件
	if SaveAppConfigFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	if err := SaveAppConfigFunc(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password updated successfully",
	})
}

// ResetPasswordHandler 管理员重置用户密码
func ResetPasswordHandler(c *gin.Context) {
	username := c.Param("username")
	var req struct {
		NewPassword string `json:"newPassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	if FindUserFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	user := FindUserFunc(username)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 更新密码
	if HashPasswordFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	user.Password = HashPasswordFunc(req.NewPassword)

	// 保存配置文件
	if SaveAppConfigFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User service not initialized"})
		return
	}
	if err := SaveAppConfigFunc(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
	})
}
