package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mycoool/gohook/internal/types"
	"gopkg.in/yaml.v2"
)

// JWT密钥 - 在生产环境中应该使用环境变量
var jwtSecret = []byte("gohook-secret-key-change-in-production")

// Token有效期
const tokenExpiryDuration = 24 * time.Hour

var appConfig *types.AppConfig

// hashPassword 对密码进行哈希
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// verifyPassword 验证密码
func VerifyPassword(password, hashedPassword string) bool {
	return HashPassword(password) == hashedPassword
}

// generateToken 生成JWT token
func GenerateToken(username, role string) (string, error) {
	expirationTime := time.Now().Add(tokenExpiryDuration)
	claims := &types.Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// validateToken 验证JWT token
func ValidateToken(tokenString string) (*types.Claims, error) {
	claims := &types.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// loadAppConfig 加载用户配置文件
func LoadAppConfig() error {
	data, err := os.ReadFile("user.yaml")
	if err != nil {
		return fmt.Errorf("读取用户配置文件失败: %v", err)
	}

	config := &types.AppConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析用户配置文件失败: %v", err)
	}

	// 如果密码不是哈希格式，则进行哈希处理
	for i := range config.Users {
		if len(config.Users[i].Password) != 64 { // SHA256哈希长度为64字符
			config.Users[i].Password = HashPassword(config.Users[i].Password)
		}
	}

	appConfig = config
	return nil
}

// saveAppConfig 保存用户配置文件
func SaveAppConfig() error {
	if appConfig == nil {
		return fmt.Errorf("用户配置数据为空")
	}

	data, err := yaml.Marshal(appConfig)
	if err != nil {
		return fmt.Errorf("序列化用户配置失败: %v", err)
	}

	// 备份原配置文件
	if _, err := os.Stat("user.yaml"); err == nil {
		if err := os.Rename("user.yaml", "user.yaml.bak"); err != nil {
			log.Printf("Warning: failed to backup user config file: %v", err)
		}
	}

	err = os.WriteFile("user.yaml", data, 0644)
	if err != nil {
		// 如果保存失败，恢复备份
		if _, backupErr := os.Stat("user.yaml.bak"); backupErr == nil {
			if restoreErr := os.Rename("user.yaml.bak", "user.yaml"); restoreErr != nil {
				log.Printf("Error: failed to restore backup user config file: %v", restoreErr)
			}
		}
		return fmt.Errorf("保存用户配置文件失败: %v", err)
	}

	return nil
}

// findUser 查找用户
func FindUser(username string) *types.UserConfig {
	if appConfig == nil {
		return nil
	}
	for i := range appConfig.Users {
		if appConfig.Users[i].Username == username {
			return &appConfig.Users[i]
		}
	}
	return nil
}

// GetAppConfig 获取应用配置
func GetAppConfig() *types.AppConfig {
	return appConfig
}

// SetAppConfig 设置应用配置
func SetAppConfig(config *types.AppConfig) {
	appConfig = config
}

// InitDefaultConfig 初始化默认配置
func InitDefaultConfig() {
	appConfig = &types.AppConfig{
		Users: []types.UserConfig{
			{
				Username: "admin",
				Password: HashPassword("123456"), // 默认密码
				Role:     "admin",
			},
		},
	}
}
