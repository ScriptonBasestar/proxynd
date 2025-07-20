package routers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"

	"proxynd/logging"
)

// UserInfo 사용자 정보 구조체
type UserInfo struct {
	Username    string     `json:"username" yaml:"username"`
	Password    string     `json:"password,omitempty" yaml:"password"`
	Role        string     `json:"role" yaml:"role"`
	CreatedAt   time.Time  `json:"created_at" yaml:"created_at"`
	LastLogin   *time.Time `json:"last_login,omitempty" yaml:"last_login,omitempty"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Active      bool       `json:"active" yaml:"active"`
}

// UserListResponse 사용자 목록 응답
type UserListResponse struct {
	Users     []UserInfo `json:"users"`
	Total     int        `json:"total"`
	Timestamp time.Time  `json:"timestamp"`
}

// UserAddRequest 사용자 추가 요청
type UserAddRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Role        string `json:"role,omitempty"`
	Description string `json:"description,omitempty"`
}

// UserDeleteRequest 사용자 삭제 요청
type UserDeleteRequest struct {
	Username string `json:"username" binding:"required"`
}

// UserConfig 사용자 설정 파일 구조체
type UserConfig struct {
	Users []UserInfo `yaml:"users"`
}

// UserRouter 사용자 관리 API 라우터 설정
func UserRouter(app *fiber.App) {
	api := app.Group("/api/user")

	// 사용자 목록 조회
	api.Get("/list", getUserList)

	// 사용자 추가
	api.Post("/add", addUser)

	// 사용자 삭제
	api.Delete("/delete", deleteUser)

	// 사용자 정보 조회
	api.Get("/:username", getUserInfo)

	// 사용자 정보 수정
	api.Put("/:username", updateUser)

	// 사용자 비밀번호 변경
	api.Post("/:username/password", changePassword)

	// 사용자 활성화/비활성화
	api.Post("/:username/toggle", toggleUserStatus)
}

// getUserList 사용자 목록 조회 핸들러
func getUserList(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	users, err := loadUserConfig()
	if err != nil {
		logger.Error("Failed to load user config", logging.F("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).SendString("사용자 설정 로딩 실패")
	}

	// 패스워드 제거 (보안상)
	for i := range users {
		users[i].Password = ""
	}

	response := UserListResponse{
		Users:     users,
		Total:     len(users),
		Timestamp: time.Now(),
	}

	logger.Info("User list requested")
	return c.JSON(response)
}

// addUser 사용자 추가 핸들러
func addUser(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	var req UserAddRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("잘못된 요청 형식")
	}

	// 입력 검증
	if req.Username == "" {
		return c.Status(fiber.StatusBadRequest).SendString("사용자명이 필요합니다")
	}
	if req.Password == "" {
		return c.Status(fiber.StatusBadRequest).SendString("비밀번호가 필요합니다")
	}
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).SendString("비밀번호는 최소 6자 이상이어야 합니다")
	}

	// 기본 역할 설정
	if req.Role == "" {
		req.Role = "user"
	}

	// 기존 사용자 로딩
	users, err := loadUserConfig()
	if err != nil {
		logger.Error("Failed to load user config", logging.F("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).SendString("사용자 설정 로딩 실패")
	}

	// 중복 사용자 확인
	for _, user := range users {
		if user.Username == req.Username {
			return c.Status(fiber.StatusConflict).SendString("이미 존재하는 사용자입니다")
		}
	}

	// 비밀번호 해시
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		logger.Error("Failed to hash password", logging.F("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).SendString("비밀번호 처리 실패")
	}

	// 새 사용자 생성
	newUser := UserInfo{
		Username:    req.Username,
		Password:    hashedPassword,
		Role:        req.Role,
		CreatedAt:   time.Now(),
		Description: req.Description,
		Active:      true,
	}

	// 사용자 추가
	users = append(users, newUser)

	// 설정 파일 저장
	if err := saveUserConfig(users); err != nil {
		logger.Error("Failed to save user config", logging.F("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).SendString("사용자 설정 저장 실패")
	}

	logger.Info("User added", logging.F("username", req.Username))

	// 응답에서 패스워드 제거
	newUser.Password = ""
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "사용자가 성공적으로 추가되었습니다",
		"user":    newUser,
	})
}

// deleteUser 사용자 삭제 핸들러
func deleteUser(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	var req UserDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("잘못된 요청 형식")
	}

	if req.Username == "" {
		return c.Status(fiber.StatusBadRequest).SendString("사용자명이 필요합니다")
	}

	// 기존 사용자 로딩
	users, err := loadUserConfig()
	if err != nil {
		logger.Error("Failed to load user config", logging.F("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).SendString("사용자 설정 로딩 실패")
	}

	// 사용자 찾기 및 삭제
	found := false
	newUsers := []UserInfo{}
	for _, user := range users {
		if user.Username != req.Username {
			newUsers = append(newUsers, user)
		} else {
			found = true
		}
	}

	if !found {
		return c.Status(fiber.StatusNotFound).SendString("사용자를 찾을 수 없습니다")
	}

	// 설정 파일 저장
	if err := saveUserConfig(newUsers); err != nil {
		logger.Error("Failed to save user config", logging.F("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).SendString("사용자 설정 저장 실패")
	}

	logger.Info("User deleted", logging.F("username", req.Username))

	return c.JSON(fiber.Map{
		"message": "사용자가 성공적으로 삭제되었습니다",
	})
}

// getUserInfo 사용자 정보 조회 핸들러
func getUserInfo(c *fiber.Ctx) error {
	logger := logging.GetLogger()
	username := c.Params("username")

	if username == "" {
		return c.Status(fiber.StatusBadRequest).SendString("사용자명이 필요합니다")
	}

	users, err := loadUserConfig()
	if err != nil {
		logger.Error("Failed to load user config", logging.F("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).SendString("사용자 설정 로딩 실패")
	}

	// 사용자 찾기
	for _, user := range users {
		if user.Username == username {
			// 패스워드 제거
			user.Password = ""
			logger.Info("User info requested", logging.F("username", username))
			return c.JSON(user)
		}
	}

	return c.Status(fiber.StatusNotFound).SendString("사용자를 찾을 수 없습니다")
}

// updateUser 사용자 정보 수정 핸들러 (간단한 구현)
func updateUser(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).SendString("사용자 정보 수정 기능은 향후 구현 예정입니다")
}

// changePassword 비밀번호 변경 핸들러 (간단한 구현)
func changePassword(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).SendString("비밀번호 변경 기능은 향후 구현 예정입니다")
}

// toggleUserStatus 사용자 활성화/비활성화 핸들러 (간단한 구현)
func toggleUserStatus(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).SendString("사용자 상태 변경 기능은 향후 구현 예정입니다")
}

// loadUserConfig 사용자 설정 파일 로딩
func loadUserConfig() ([]UserInfo, error) {
	configDir := getUserConfigDir()
	configPath := filepath.Join(configDir, "users.yaml")

	// 파일이 없으면 빈 배열 반환
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return []UserInfo{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read user config file: %v", err)
	}

	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse user config: %v", err)
	}

	return config.Users, nil
}

// saveUserConfig 사용자 설정 파일 저장
func saveUserConfig(users []UserInfo) error {
	configDir := getUserConfigDir()

	// 디렉토리 생성
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	config := UserConfig{
		Users: users,
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal user config: %v", err)
	}

	configPath := filepath.Join(configDir, "users.yaml")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write user config file: %v", err)
	}

	return nil
}

// getUserConfigDir 사용자 설정 디렉토리 경로 반환
func getUserConfigDir() string {
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		return configDir
	}
	return "/config"
}

// hashPassword 비밀번호 해시 (간단한 구현)
func hashPassword(password string) (string, error) {
	// 솔트 생성
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// SHA256 해시
	hash := sha256.Sum256([]byte(password + hex.EncodeToString(salt)))

	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash[:]), nil
}
