package client

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/types"
	"gopkg.in/yaml.v2"
)

// load users config file
func LoadUsersConfig() error {
	filePath := "user.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("user config file %s not exist", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read user config file failed: %v", err)
	}

	config := &types.UsersConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("parse user config file failed: %v", err)
	}

	types.GoHookUsersConfig = config
	return nil
}

// save users config file
func SaveUsersConfig() error {
	if types.GoHookUsersConfig == nil {
		return fmt.Errorf("user config is nil")
	}

	// create YAML content with comments
	var yamlContent strings.Builder
	yamlContent.WriteString("# GoHook user config file\n")
	yamlContent.WriteString("# user account info\n")
	yamlContent.WriteString("users:\n")

	for _, user := range types.GoHookUsersConfig.Users {
		yamlContent.WriteString(fmt.Sprintf("  - username: %s\n", user.Username))
		yamlContent.WriteString(fmt.Sprintf("    password: %s\n", user.Password))
		yamlContent.WriteString(fmt.Sprintf("    role: %s\n", user.Role))

		// if it is default admin user and password is hashed, add original password comment
		if user.Username == "admin" && strings.HasPrefix(user.Password, "$2a$") {
			// check if it is new created default user (check if only one user)
			if len(types.GoHookUsersConfig.Users) == 1 {
				yamlContent.WriteString("    # default password: admin123 (please change it)\n")
			}
		}
	}

	if err := os.WriteFile("user.yaml", []byte(yamlContent.String()), 0644); err != nil {
		return fmt.Errorf("save user config file failed: %v", err)
	}

	return nil
}

// find user
func FindUser(username string) *types.UserConfig {
	if types.GoHookUsersConfig == nil {
		return nil
	}

	for i := range types.GoHookUsersConfig.Users {
		if types.GoHookUsersConfig.Users[i].Username == username {
			return &types.GoHookUsersConfig.Users[i]
		}
	}

	return nil
}

func Login(c *gin.Context) {
	// get Basic authentication info from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
		return
	}

	// check if it is Basic authentication
	if !strings.HasPrefix(authHeader, "Basic ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization type"})
		return
	}

	// decode Base64 encoded username:password
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization encoding"})
		return
	}

	// split username and password
	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials format"})
		return
	}

	username := credentials[0]
	password := credentials[1]

	// find user
	user := FindUser(username)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// verify password
	if !VerifyPassword(password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// generate JWT token
	token, err := GenerateToken(user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// get client name (from request body name field)
	var requestBody struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&requestBody); err != nil {
		log.Printf("Warning: failed to parse request body: %v", err)
	}

	clientName := requestBody.Name
	if clientName == "" {
		clientName = "unknown client"
	}

	// create client session record
	session := AddClientSession(token, clientName, user.Username)

	c.JSON(http.StatusOK, types.ClientResponse{
		Token: token,
		ID:    session.ID,
		Name:  clientName,
	})
}

// get all users
func GetAllUsers(c *gin.Context) {
	var users []types.UserResponse
	for _, user := range types.GoHookUsersConfig.Users {
		users = append(users, types.UserResponse{
			Username: user.Username,
			Role:     user.Role,
		})
	}
	c.JSON(http.StatusOK, users)
}

// create user
func CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// check if user already exists
	if FindUser(req.Username) != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	// validate role
	if req.Role != "admin" && req.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Must be 'admin' or 'user'"})
		return
	}

	// add new user
	newUser := types.UserConfig{
		Username: req.Username,
		Password: HashPassword(req.Password),
		Role:     req.Role,
	}

	types.GoHookUsersConfig.Users = append(types.GoHookUsersConfig.Users, newUser)

	// save config file
	if err := SaveUsersConfig(); err != nil {
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

// delete user
func DeleteUser(c *gin.Context) {
	username := c.Param("username")
	currentUser, _ := c.Get("username")

	// cannot delete yourself
	if username == currentUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
		return
	}

	// find user index
	userIndex := -1
	for i, user := range types.GoHookUsersConfig.Users {
		if user.Username == username {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// delete user
	types.GoHookUsersConfig.Users = append(types.GoHookUsersConfig.Users[:userIndex], types.GoHookUsersConfig.Users[userIndex+1:]...)

	// save config file
	if err := SaveUsersConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// change password
func ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	username, _ := c.Get("username")
	user := FindUser(username.(string))
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// verify old password
	if !VerifyPassword(req.OldPassword, user.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid old password"})
		return
	}

	// update password
	user.Password = HashPassword(req.NewPassword)

	// save config file
	if err := SaveUsersConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password updated successfully",
	})
}

// admin reset user password
func ResetPassword(c *gin.Context) {
	username := c.Param("username")
	var req struct {
		NewPassword string `json:"newPassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	user := FindUser(username)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// update password
	user.Password = HashPassword(req.NewPassword)

	// save config file
	if err := SaveUsersConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
	})
}

// get current user info
func GetCurrentUser(c *gin.Context) {
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, gin.H{
		"id":       1,
		"name":     username,
		"username": username,
		"role":     role,
		"admin":    role == "admin",
	})
}
