package config

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// RequiredEnvVars 필수 환경 변수 목록
var RequiredEnvVars = []string{
	"JWT_SECRET",
}

// ConditionalEnvVars 조건부 필수 환경 변수 (OAuth2 활성화 시)
var ConditionalEnvVars = map[string][]string{
	"oauth2_enabled": {
		"OAUTH_GITHUB_CLIENT_ID",
		"OAUTH_GITHUB_CLIENT_SECRET",
	},
}

// ValidateRequiredEnvVars 필수 환경 변수 검증
func ValidateRequiredEnvVars() error {
	var missing []string

	for _, env := range RequiredEnvVars {
		if os.Getenv(env) == "" {
			missing = append(missing, env)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", 
			strings.Join(missing, ", "))
	}

	return nil
}

// ValidateConditionalEnvVars 조건부 환경 변수 검증
func ValidateConditionalEnvVars() error {
	// OAuth2 활성화 여부 확인
	if os.Getenv("OAUTH_ENABLED") == "true" {
		var missing []string
		
		for _, env := range ConditionalEnvVars["oauth2_enabled"] {
			if os.Getenv(env) == "" {
				missing = append(missing, env)
			}
		}
		
		if len(missing) > 0 {
			return fmt.Errorf("OAuth2 is enabled but missing required variables: %s", 
				strings.Join(missing, ", "))
		}
	}

	return nil
}

// LoadSecureConfig 보안 설정 로드 및 검증
func LoadSecureConfig() error {
	// 필수 환경 변수 검증
	if err := ValidateRequiredEnvVars(); err != nil {
		return err
	}

	// 조건부 환경 변수 검증
	if err := ValidateConditionalEnvVars(); err != nil {
		return err
	}

	// JWT 시크릿 길이 검증
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters")
	}

	// 개발 환경에서 기본 비밀번호 사용 확인
	if strings.Contains(jwtSecret, "your-jwt-secret") || 
	   strings.Contains(jwtSecret, "default") ||
	   strings.Contains(jwtSecret, "example") {
		return errors.New("please change JWT_SECRET from default value")
	}

	return nil
}

// GenerateSecureKey 안전한 키 생성 도구
func GenerateSecureKey() (string, error) {
	// 시간 기반 시드로 랜덤 생성
	rand.Seed(time.Now().UnixNano())
	
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	key := make([]byte, 64) // 64자 길이의 키 생성
	
	for i := range key {
		key[i] = charset[rand.Intn(len(charset))]
	}
	
	return string(key), nil
}

// ValidateEnvironment 전체 환경 검증
func ValidateEnvironment() error {
	// 환경 설정 로드
	if err := LoadSecureConfig(); err != nil {
		return fmt.Errorf("환경 검증 실패: %w", err)
	}

	// 추가 보안 검사들
	if err := validateSecuritySettings(); err != nil {
		return fmt.Errorf("보안 설정 검증 실패: %w", err)
	}

	return nil
}

// validateSecuritySettings 보안 설정 검증
func validateSecuritySettings() error {
	// 프로덕션 환경에서 디버그 모드 비활성화 확인
	if os.Getenv("PROXYND_ENV") == "production" {
		if os.Getenv("LOG_LEVEL") == "debug" {
			return errors.New("production 환경에서는 LOG_LEVEL을 debug로 설정하면 안됩니다")
		}
		
		if os.Getenv("TLS_ENABLED") != "true" {
			return errors.New("production 환경에서는 TLS를 활성화해야 합니다")
		}
	}

	return nil
}