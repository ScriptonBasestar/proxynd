// Package mfa provides Multi-Factor Authentication implementation
package mfa

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"

	"proxynd/logging"
)

// MFAMethod MFA 인증 방법 타입
type MFAMethod string

const (
	// MethodTOTP Time-based One-Time Password
	MethodTOTP MFAMethod = "totp"
	// MethodSMS SMS 기반 인증
	MethodSMS MFAMethod = "sms"
	// MethodEmail 이메일 기반 인증
	MethodEmail MFAMethod = "email"
	// MethodBackupCode 백업 코드
	MethodBackupCode MFAMethod = "backup_code"
)

// MFAUserConfig 사용자별 MFA 설정
type MFAUserConfig struct {
	UserID         string                  `json:"user_id"`
	Email          string                  `json:"email"`
	Enabled        bool                    `json:"enabled"`
	Methods        map[MFAMethod]*MFASetup `json:"methods"`
	BackupCodes    []string                `json:"backup_codes"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
	LastUsedAt     *time.Time              `json:"last_used_at,omitempty"`
	FailedAttempts int                     `json:"failed_attempts"`
	LockedUntil    *time.Time              `json:"locked_until,omitempty"`
}

// MFASetup MFA 방법별 설정
type MFASetup struct {
	Method      MFAMethod  `json:"method"`
	Secret      string     `json:"secret,omitempty"`       // TOTP 시크릿
	PhoneNumber string     `json:"phone_number,omitempty"` // SMS 전화번호
	Enabled     bool       `json:"enabled"`
	Verified    bool       `json:"verified"`
	CreatedAt   time.Time  `json:"created_at"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// MFAChallenge MFA 인증 요청
type MFAChallenge struct {
	ChallengeID string     `json:"challenge_id"`
	UserID      string     `json:"user_id"`
	Method      MFAMethod  `json:"method"`
	Code        string     `json:"code,omitempty"` // SMS/Email 코드
	ExpiresAt   time.Time  `json:"expires_at"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	CreatedAt   time.Time  `json:"created_at"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// MFAConfig MFA 서비스 설정
type MFAConfig struct {
	StoragePath         string        `json:"storage_path"`
	Issuer              string        `json:"issuer"`               // TOTP 발급자
	Algorithm           string        `json:"algorithm"`            // TOTP 알고리즘 (SHA1, SHA256, SHA512)
	Digits              int           `json:"digits"`               // TOTP 자릿수
	Period              int           `json:"period"`               // TOTP 갱신 주기 (초)
	BackupCodeCount     int           `json:"backup_code_count"`    // 백업 코드 개수
	BackupCodeLength    int           `json:"backup_code_length"`   // 백업 코드 길이
	CodeExpiration      time.Duration `json:"code_expiration"`      // SMS/Email 코드 만료 시간
	MaxFailedAttempts   int           `json:"max_failed_attempts"`  // 최대 실패 시도 횟수
	LockoutDuration     time.Duration `json:"lockout_duration"`     // 계정 잠금 시간
	ChallengeExpiration time.Duration `json:"challenge_expiration"` // 챌린지 만료 시간
	EnableRecovery      bool          `json:"enable_recovery"`      // 복구 옵션 활성화
}

// DefaultMFAConfig 기본 MFA 설정
func DefaultMFAConfig() *MFAConfig {
	return &MFAConfig{
		StoragePath:         "data/mfa_users.json",
		Issuer:              "ProxyND",
		Algorithm:           "SHA1",
		Digits:              6,
		Period:              30,
		BackupCodeCount:     10,
		BackupCodeLength:    8,
		CodeExpiration:      5 * time.Minute,
		MaxFailedAttempts:   5,
		LockoutDuration:     30 * time.Minute,
		ChallengeExpiration: 10 * time.Minute,
		EnableRecovery:      true,
	}
}

// MFAService MFA 서비스
type MFAService struct {
	config     *MFAConfig
	users      map[string]*MFAUserConfig
	challenges map[string]*MFAChallenge
	mutex      sync.RWMutex
	logger     logging.Logger
}

// NewMFAService MFA 서비스 생성
func NewMFAService(config *MFAConfig) (*MFAService, error) {
	if config == nil {
		config = DefaultMFAConfig()
	}

	service := &MFAService{
		config:     config,
		users:      make(map[string]*MFAUserConfig),
		challenges: make(map[string]*MFAChallenge),
		logger:     logging.GetLogger(),
	}

	// 저장된 사용자 설정 로드
	if err := service.loadUsers(); err != nil {
		service.logger.Warn("Failed to load MFA users", logging.F("error", err))
	}

	// 주기적으로 만료된 챌린지 정리
	go service.cleanupExpiredChallenges()

	return service, nil
}

// EnableMFA 사용자의 MFA 활성화
func (s *MFAService) EnableMFA(userID, email string) (*MFAUserConfig, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 기존 설정 확인
	if existingConfig, exists := s.users[userID]; exists && existingConfig.Enabled {
		return existingConfig, fmt.Errorf("MFA is already enabled for user %s", userID)
	}

	// 새 MFA 설정 생성
	userConfig := &MFAUserConfig{
		UserID:      userID,
		Email:       email,
		Enabled:     false, // 초기에는 비활성화 (검증 후 활성화)
		Methods:     make(map[MFAMethod]*MFASetup),
		BackupCodes: s.generateBackupCodes(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.users[userID] = userConfig

	// 저장
	if err := s.saveUsers(); err != nil {
		delete(s.users, userID)
		return nil, fmt.Errorf("failed to save MFA config: %w", err)
	}

	s.logger.Info("MFA setup initiated for user",
		logging.F("user_id", userID),
		logging.F("email", email))

	return userConfig, nil
}

// SetupTOTP TOTP 설정
func (s *MFAService) SetupTOTP(userID string) (*MFASetup, string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	userConfig, exists := s.users[userID]
	if !exists {
		return nil, "", fmt.Errorf("MFA not enabled for user %s", userID)
	}

	// TOTP 시크릿 생성
	secret, err := s.generateTOTPSecret()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// TOTP 설정 생성
	totpSetup := &MFASetup{
		Method:    MethodTOTP,
		Secret:    secret,
		Enabled:   false, // 검증 후 활성화
		Verified:  false,
		CreatedAt: time.Now(),
	}

	userConfig.Methods[MethodTOTP] = totpSetup
	userConfig.UpdatedAt = time.Now()

	// QR 코드 URL 생성
	qrURL, err := s.generateTOTPURL(userConfig.Email, secret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate QR URL: %w", err)
	}

	// 저장
	if err := s.saveUsers(); err != nil {
		return nil, "", fmt.Errorf("failed to save TOTP setup: %w", err)
	}

	s.logger.Info("TOTP setup created for user",
		logging.F("user_id", userID),
		logging.F("email", userConfig.Email))

	return totpSetup, qrURL, nil
}

// VerifyTOTPSetup TOTP 설정 검증
func (s *MFAService) VerifyTOTPSetup(userID, code string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	userConfig, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("MFA not enabled for user %s", userID)
	}

	totpSetup, exists := userConfig.Methods[MethodTOTP]
	if !exists || totpSetup.Verified {
		return fmt.Errorf("TOTP setup not found or already verified")
	}

	// TOTP 코드 검증
	valid := totp.Validate(code, totpSetup.Secret)
	if !valid {
		return fmt.Errorf("invalid TOTP code")
	}

	// 검증 완료 처리
	now := time.Now()
	totpSetup.Verified = true
	totpSetup.Enabled = true
	totpSetup.VerifiedAt = &now
	totpSetup.LastUsedAt = &now

	// 첫 번째 방법이 검증되면 MFA 활성화
	userConfig.Enabled = true
	userConfig.UpdatedAt = now

	// 저장
	if err := s.saveUsers(); err != nil {
		return fmt.Errorf("failed to save TOTP verification: %w", err)
	}

	s.logger.Info("TOTP verification completed for user",
		logging.F("user_id", userID),
		logging.F("email", userConfig.Email))

	return nil
}

// StartMFAChallenge MFA 인증 챌린지 시작
func (s *MFAService) StartMFAChallenge(userID string, preferredMethod MFAMethod) (*MFAChallenge, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	userConfig, exists := s.users[userID]
	if !exists || !userConfig.Enabled {
		return nil, fmt.Errorf("MFA not enabled for user %s", userID)
	}

	// 계정 잠금 확인
	if userConfig.LockedUntil != nil && time.Now().Before(*userConfig.LockedUntil) {
		return nil, fmt.Errorf("account is locked until %v", userConfig.LockedUntil)
	}

	// 사용 가능한 방법 확인
	var method MFAMethod
	if setup, exists := userConfig.Methods[preferredMethod]; exists && setup.Enabled && setup.Verified {
		method = preferredMethod
	} else {
		// 첫 번째 사용 가능한 방법 선택
		for m, setup := range userConfig.Methods {
			if setup.Enabled && setup.Verified {
				method = m
				break
			}
		}
		if method == "" {
			return nil, fmt.Errorf("no verified MFA method available")
		}
	}

	// 챌린지 ID 생성
	challengeID, err := s.generateChallengeID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge ID: %w", err)
	}

	// 챌린지 생성
	challenge := &MFAChallenge{
		ChallengeID: challengeID,
		UserID:      userID,
		Method:      method,
		ExpiresAt:   time.Now().Add(s.config.ChallengeExpiration),
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
	}

	// SMS/Email 방법의 경우 코드 생성 및 발송
	if method == MethodSMS || method == MethodEmail {
		code, err := s.generateVerificationCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate verification code: %w", err)
		}
		challenge.Code = code

		// 실제 환경에서는 SMS/Email 발송 로직 구현
		s.logger.Info("Verification code generated",
			logging.F("user_id", userID),
			logging.F("method", string(method)),
			logging.F("code", code)) // 프로덕션에서는 제거해야 함
	}

	s.challenges[challengeID] = challenge

	s.logger.Info("MFA challenge started",
		logging.F("user_id", userID),
		logging.F("challenge_id", challengeID),
		logging.F("method", string(method)))

	return challenge, nil
}

// VerifyMFAChallenge MFA 챌린지 검증
func (s *MFAService) VerifyMFAChallenge(challengeID, code string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	challenge, exists := s.challenges[challengeID]
	if !exists {
		return fmt.Errorf("invalid challenge ID")
	}

	// 만료 확인
	if time.Now().After(challenge.ExpiresAt) {
		delete(s.challenges, challengeID)
		return fmt.Errorf("challenge expired")
	}

	// 완료 여부 확인
	if challenge.Completed {
		return fmt.Errorf("challenge already completed")
	}

	// 시도 횟수 확인
	if challenge.Attempts >= challenge.MaxAttempts {
		delete(s.challenges, challengeID)
		s.incrementFailedAttempts(challenge.UserID)
		return fmt.Errorf("maximum attempts exceeded")
	}

	challenge.Attempts++

	userConfig := s.users[challenge.UserID]
	var valid bool

	// 방법별 검증
	switch challenge.Method {
	case MethodTOTP:
		setup := userConfig.Methods[MethodTOTP]
		valid = totp.Validate(code, setup.Secret)
	case MethodSMS, MethodEmail:
		valid = (code == challenge.Code)
	case MethodBackupCode:
		valid = s.validateBackupCode(challenge.UserID, code)
	default:
		return fmt.Errorf("unsupported MFA method: %s", challenge.Method)
	}

	if !valid {
		if challenge.Attempts >= challenge.MaxAttempts {
			delete(s.challenges, challengeID)
			s.incrementFailedAttempts(challenge.UserID)
		}
		return fmt.Errorf("invalid verification code")
	}

	// 검증 성공 처리
	now := time.Now()
	challenge.Completed = true
	challenge.CompletedAt = &now

	// 사용자 설정 업데이트
	userConfig.LastUsedAt = &now
	userConfig.FailedAttempts = 0 // 성공 시 실패 카운트 리셋
	userConfig.LockedUntil = nil

	if method := userConfig.Methods[challenge.Method]; method != nil {
		method.LastUsedAt = &now
	}

	// 백업 코드 사용 시 해당 코드 제거
	if challenge.Method == MethodBackupCode {
		s.removeUsedBackupCode(challenge.UserID, code)
	}

	// 저장
	if err := s.saveUsers(); err != nil {
		s.logger.Error("Failed to save MFA verification result", logging.F("error", err))
	}

	// 챌린지 정리 (성공한 것은 잠시 유지)
	go func() {
		time.Sleep(1 * time.Minute)
		s.mutex.Lock()
		delete(s.challenges, challengeID)
		s.mutex.Unlock()
	}()

	s.logger.Info("MFA challenge verified successfully",
		logging.F("user_id", challenge.UserID),
		logging.F("challenge_id", challengeID),
		logging.F("method", string(challenge.Method)))

	return nil
}

// GetUserMFAStatus 사용자 MFA 상태 조회
func (s *MFAService) GetUserMFAStatus(userID string) (*MFAUserConfig, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	userConfig, exists := s.users[userID]
	if !exists {
		return nil, fmt.Errorf("MFA not configured for user %s", userID)
	}

	// 민감한 정보 제거한 복사본 반환
	safeCopy := *userConfig
	safeCopy.BackupCodes = nil // 백업 코드는 숨김

	// 시크릿 마스킹
	for method, setup := range safeCopy.Methods {
		setupCopy := *setup
		if setupCopy.Secret != "" {
			setupCopy.Secret = "***masked***"
		}
		safeCopy.Methods[method] = &setupCopy
	}

	return &safeCopy, nil
}

// DisableMFA MFA 비활성화
func (s *MFAService) DisableMFA(userID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	userConfig, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("MFA not configured for user %s", userID)
	}

	// MFA 비활성화
	userConfig.Enabled = false
	userConfig.UpdatedAt = time.Now()

	// 모든 방법 비활성화
	for _, method := range userConfig.Methods {
		method.Enabled = false
	}

	// 백업 코드 제거
	userConfig.BackupCodes = nil

	// 저장
	if err := s.saveUsers(); err != nil {
		return fmt.Errorf("failed to save MFA disable: %w", err)
	}

	s.logger.Info("MFA disabled for user",
		logging.F("user_id", userID),
		logging.F("email", userConfig.Email))

	return nil
}

// RegenerateBackupCodes 백업 코드 재생성
func (s *MFAService) RegenerateBackupCodes(userID string) ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	userConfig, exists := s.users[userID]
	if !exists || !userConfig.Enabled {
		return nil, fmt.Errorf("MFA not enabled for user %s", userID)
	}

	// 새 백업 코드 생성
	newBackupCodes := s.generateBackupCodes()
	userConfig.BackupCodes = newBackupCodes
	userConfig.UpdatedAt = time.Now()

	// 저장
	if err := s.saveUsers(); err != nil {
		return nil, fmt.Errorf("failed to save new backup codes: %w", err)
	}

	s.logger.Info("Backup codes regenerated for user",
		logging.F("user_id", userID),
		logging.F("email", userConfig.Email))

	return newBackupCodes, nil
}

// helper methods

// generateTOTPSecret TOTP 시크릿 생성
func (s *MFAService) generateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(secret), nil
}

// generateTOTPURL TOTP QR 코드 URL 생성
func (s *MFAService) generateTOTPURL(email, secret string) (string, error) {
	u := url.URL{
		Scheme: "otpauth",
		Host:   "totp",
		Path:   "/" + url.PathEscape(s.config.Issuer+":"+email),
	}

	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", s.config.Issuer)
	params.Set("algorithm", s.config.Algorithm)
	params.Set("digits", strconv.Itoa(s.config.Digits))
	params.Set("period", strconv.Itoa(s.config.Period))

	u.RawQuery = params.Encode()
	return u.String(), nil
}

// generateBackupCodes 백업 코드 생성
func (s *MFAService) generateBackupCodes() []string {
	codes := make([]string, s.config.BackupCodeCount)
	for i := range codes {
		codes[i] = s.generateRandomCode(s.config.BackupCodeLength)
	}
	return codes
}

// generateRandomCode 랜덤 코드 생성
func (s *MFAService) generateRandomCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// fallback to simple implementation
		return fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes)
}

// generateVerificationCode SMS/Email 인증 코드 생성
func (s *MFAService) generateVerificationCode() (string, error) {
	code := make([]byte, 3)
	if _, err := rand.Read(code); err != nil {
		return "", err
	}

	// 6자리 숫자 코드 생성
	num := int(code[0])<<16 | int(code[1])<<8 | int(code[2])
	return fmt.Sprintf("%06d", num%1000000), nil
}

// generateChallengeID 챌린지 ID 생성
func (s *MFAService) generateChallengeID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(bytes))[:16], nil
}

// validateBackupCode 백업 코드 검증
func (s *MFAService) validateBackupCode(userID, code string) bool {
	userConfig := s.users[userID]
	if userConfig == nil {
		return false
	}

	for _, backupCode := range userConfig.BackupCodes {
		if strings.EqualFold(backupCode, code) {
			return true
		}
	}
	return false
}

// removeUsedBackupCode 사용된 백업 코드 제거
func (s *MFAService) removeUsedBackupCode(userID, code string) {
	userConfig := s.users[userID]
	if userConfig == nil {
		return
	}

	for i, backupCode := range userConfig.BackupCodes {
		if strings.EqualFold(backupCode, code) {
			// 슬라이스에서 제거
			userConfig.BackupCodes = append(userConfig.BackupCodes[:i], userConfig.BackupCodes[i+1:]...)
			break
		}
	}
}

// incrementFailedAttempts 실패 시도 횟수 증가
func (s *MFAService) incrementFailedAttempts(userID string) {
	userConfig := s.users[userID]
	if userConfig == nil {
		return
	}

	userConfig.FailedAttempts++
	userConfig.UpdatedAt = time.Now()

	// 최대 실패 횟수 도달 시 계정 잠금
	if userConfig.FailedAttempts >= s.config.MaxFailedAttempts {
		lockUntil := time.Now().Add(s.config.LockoutDuration)
		userConfig.LockedUntil = &lockUntil

		s.logger.Warn("User account locked due to excessive MFA failures",
			logging.F("user_id", userID),
			logging.F("failed_attempts", userConfig.FailedAttempts),
			logging.F("locked_until", lockUntil))
	}

	// 저장
	if err := s.saveUsers(); err != nil {
		s.logger.Error("Failed to save failed attempt increment", logging.F("error", err))
	}
}

// cleanupExpiredChallenges 만료된 챌린지 정리
func (s *MFAService) cleanupExpiredChallenges() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mutex.Lock()
		now := time.Now()
		cleanedCount := 0

		for challengeID, challenge := range s.challenges {
			if now.After(challenge.ExpiresAt) {
				delete(s.challenges, challengeID)
				cleanedCount++
			}
		}

		if cleanedCount > 0 {
			s.logger.Debug("Cleaned up expired MFA challenges",
				logging.F("count", cleanedCount))
		}

		s.mutex.Unlock()
	}
}

// loadUsers 저장된 사용자 설정 로드
func (s *MFAService) loadUsers() error {
	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(s.config.StoragePath), 0o755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 파일이 없으면 무시
	if _, err := os.Stat(s.config.StoragePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(s.config.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to read MFA users file: %w", err)
	}

	var users map[string]*MFAUserConfig
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to parse MFA users file: %w", err)
	}

	s.users = users

	s.logger.Info("Loaded MFA user configurations", logging.F("count", len(users)))
	return nil
}

// saveUsers 사용자 설정 저장
func (s *MFAService) saveUsers() error {
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MFA users: %w", err)
	}

	// 임시 파일에 쓰고 원자적으로 이동
	tempPath := s.config.StoragePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write MFA users file: %w", err)
	}

	if err := os.Rename(tempPath, s.config.StoragePath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to move MFA users file: %w", err)
	}

	return nil
}

// GenerateQRCode QR 코드 이미지 생성 (util 함수)
func GenerateQRCode(totpURL string, size int) ([]byte, error) {
	qr, err := qrcode.New(totpURL, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}

	return qr.PNG(size)
}
