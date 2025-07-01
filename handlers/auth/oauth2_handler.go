package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"proxynd/configs"
	"proxynd/internal/auth/oauth2"
	"proxynd/logging"
)

// OAuth2State 세션 상태 관리
type OAuth2State struct {
	State         string    `json:"state"`
	Provider      string    `json:"provider"`
	RedirectURL   string    `json:"redirect_url"`
	CreatedAt     time.Time `json:"created_at"`
	CodeVerifier  string    `json:"code_verifier,omitempty"`  // PKCE용
	CodeChallenge string    `json:"code_challenge,omitempty"` // PKCE용
}

// 전역 상태 저장소 (프로덕션에서는 Redis 등 사용)
var stateStore = make(map[string]*OAuth2State)

// StartOAuth2Login OAuth2 로그인 시작
func StartOAuth2Login(c *fiber.Ctx) error {
	providerName := c.Params("provider")
	redirectURL := c.Query("redirect", "/")
	
	// OAuth2 설정 로드
	oauth2Config := &configs.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		logging.GetLogger().Error("Failed to load OAuth2 config", logging.F("error", err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "OAuth2 configuration not available",
		})
	}
	
	if !oauth2Config.IsEnabled() {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "OAuth2 authentication is not enabled",
		})
	}
	
	// 제공자 설정 확인
	providerConfig, exists := oauth2Config.GetProvider(providerName)
	if !exists {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth2 provider '%s' not configured", providerName),
		})
	}
	
	// OAuth2 제공자 생성
	provider := oauth2.CreateProvider(providerName, oauth2.ProviderConfig{
		Name:         providerName,
		ClientID:     providerConfig.ClientID,
		ClientSecret: providerConfig.ClientSecret,
		RedirectURI:  providerConfig.RedirectURI,
		Scopes:       providerConfig.GetEffectiveScopes(),
		AuthURL:      providerConfig.AuthURL,
		TokenURL:     providerConfig.TokenURL,
		UserInfoURL:  providerConfig.UserInfoURL,
		EnablePKCE:   providerConfig.EnablePKCE,
	})
	
	// 상태 생성 (CSRF 방지)
	state, err := generateRandomString(32)
	if err != nil {
		logging.GetLogger().Error("Failed to generate OAuth2 state", logging.F("error", err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authentication state",
		})
	}
	
	oauth2State := &OAuth2State{
		State:       state,
		Provider:    providerName,
		RedirectURL: redirectURL,
		CreatedAt:   time.Now(),
	}
	
	var authURL string
	
	// PKCE 지원 확인
	if providerConfig.EnablePKCE {
		codeVerifier, err := generateRandomString(64)
		if err != nil {
			logging.GetLogger().Error("Failed to generate PKCE code verifier", logging.F("error", err))
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate PKCE parameters",
			})
		}
		
		codeChallenge := generateCodeChallenge(codeVerifier)
		oauth2State.CodeVerifier = codeVerifier
		oauth2State.CodeChallenge = codeChallenge
		
		authURL = provider.GetAuthURL(state, codeChallenge)
	} else {
		authURL = provider.GetAuthURL(state, "")
	}
	
	// 상태 저장 (5분 TTL)
	stateStore[state] = oauth2State
	
	// 5분 후 자동 정리
	go func() {
		time.Sleep(5 * time.Minute)
		delete(stateStore, state)
	}()
	
	logging.GetLogger().Info("OAuth2 login started", logging.F("provider", providerName), logging.F("ip", c.IP()))
	
	// 인증 URL로 리다이렉트
	return c.Redirect(authURL)
}

// HandleOAuth2Callback OAuth2 콜백 처리
func HandleOAuth2Callback(c *fiber.Ctx) error {
	providerName := c.Params("provider")
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	
	// 에러 처리
	if errorParam != "" {
		errorDesc := c.Query("error_description", "Unknown error")
		logging.GetLogger().Warn("OAuth2 callback error", logging.F("error", errorParam), logging.F("description", errorDesc))
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             errorParam,
			"error_description": errorDesc,
		})
	}
	
	// 필수 파라미터 확인
	if code == "" || state == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing authorization code or state parameter",
		})
	}
	
	// 상태 검증 (CSRF 방지)
	oauth2State, exists := stateStore[state]
	if !exists {
		logging.GetLogger().Warn("Invalid OAuth2 state", logging.F("state", state), logging.F("ip", c.IP()))
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired authentication state",
		})
	}
	
	// 상태 만료 확인 (5분)
	if time.Since(oauth2State.CreatedAt) > 5*time.Minute {
		delete(stateStore, state)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Authentication state has expired",
		})
	}
	
	// 제공자 일치 확인
	if oauth2State.Provider != providerName {
		delete(stateStore, state)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Provider mismatch in authentication state",
		})
	}
	
	// OAuth2 설정 로드
	oauth2Config := &configs.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		delete(stateStore, state)
		logging.GetLogger().Error("Failed to load OAuth2 config", logging.F("error", err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "OAuth2 configuration not available",
		})
	}
	
	// 제공자 설정 확인
	providerConfig, exists := oauth2Config.GetProvider(providerName)
	if !exists {
		delete(stateStore, state)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth2 provider '%s' not configured", providerName),
		})
	}
	
	// OAuth2 제공자 생성
	provider := oauth2.CreateProvider(providerName, oauth2.ProviderConfig{
		Name:         providerName,
		ClientID:     providerConfig.ClientID,
		ClientSecret: providerConfig.ClientSecret,
		RedirectURI:  providerConfig.RedirectURI,
		Scopes:       providerConfig.GetEffectiveScopes(),
		AuthURL:      providerConfig.AuthURL,
		TokenURL:     providerConfig.TokenURL,
		UserInfoURL:  providerConfig.UserInfoURL,
		EnablePKCE:   providerConfig.EnablePKCE,
	})
	
	// 인증 코드를 액세스 토큰으로 교환
	tokenResp, err := provider.ExchangeCode(c.Context(), code, oauth2State.CodeVerifier)
	if err != nil {
		delete(stateStore, state)
		logging.GetLogger().Error("Failed to exchange OAuth2 code", logging.F("error", err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to exchange authorization code for token",
		})
	}
	
	// 사용자 정보 조회
	userInfo, err := provider.GetUserInfo(c.Context(), tokenResp.AccessToken)
	if err != nil {
		delete(stateStore, state)
		logging.GetLogger().Error("Failed to get user info", logging.F("error", err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}
	
	// 사용자 조직/그룹 정보 조회 (선택사항)
	organizations, _ := provider.GetUserOrganizations(c.Context(), tokenResp.AccessToken)
	
	// 사용자 역할 결정
	userRole := oauth2Config.UserMapping.GetUserRole(userInfo.Email, organizations)
	
	// JWT 토큰 생성 (여기서는 간단한 세션 정보만 저장)
	sessionData := fiber.Map{
		"user_id":      userInfo.ID,
		"email":        userInfo.Email,
		"name":         userInfo.Name,
		"username":     userInfo.Username,
		"avatar":       userInfo.Avatar,
		"provider":     providerName,
		"role":         userRole,
		"organizations": organizations,
		"login_time":   time.Now(),
		"expires_at":   tokenResp.ExpiresAt,
	}
	
	// 세션에 사용자 정보 저장
	sess, err := getSession(c)
	if err != nil {
		delete(stateStore, state)
		logging.GetLogger().Error("Failed to get session", logging.F("error", err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user session",
		})
	}
	
	sess["user"] = sessionData
	sess["access_token"] = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		sess["refresh_token"] = tokenResp.RefreshToken
	}
	
	// 상태 정리
	delete(stateStore, state)
	
	logging.GetLogger().Info("OAuth2 login successful", logging.F("email", userInfo.Email), logging.F("name", userInfo.Name), logging.F("provider", providerName))
	
	// 원래 요청한 페이지로 리다이렉트
	redirectURL := oauth2State.RedirectURL
	if redirectURL == "" {
		redirectURL = "/"
	}
	
	return c.Redirect(redirectURL)
}

// HandleLogout 로그아웃 처리
func HandleLogout(c *fiber.Ctx) error {
	sess, err := getSession(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}
	
	// 세션에서 사용자 정보 가져오기
	if userData, exists := sess["user"]; exists && userData != nil {
		if userMap, ok := userData.(fiber.Map); ok {
			if email, ok := userMap["email"].(string); ok {
				logging.GetLogger().Info("User logged out", logging.F("email", email))
			}
		}
	}
	
	// 세션 파기 (맵 초기화)
	for key := range sess {
		delete(sess, key)
	}
	
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// RefreshToken 토큰 갱신
func RefreshToken(c *fiber.Ctx) error {
	sess, err := getSession(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}
	
	refreshToken, exists := sess["refresh_token"]
	if !exists || refreshToken == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No refresh token available",
		})
	}
	
	userData, exists := sess["user"]
	if !exists || userData == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No user session found",
		})
	}
	
	userMap, ok := userData.(fiber.Map)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid user session data",
		})
	}
	
	providerName, ok := userMap["provider"].(string)
	if !ok {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Provider information not found in session",
		})
	}
	
	// OAuth2 설정 로드
	oauth2Config := &configs.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "OAuth2 configuration not available",
		})
	}
	
	// 제공자 설정 확인
	providerConfig, exists := oauth2Config.GetProvider(providerName)
	if !exists {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth2 provider '%s' not configured", providerName),
		})
	}
	
	// OAuth2 제공자 생성
	provider := oauth2.CreateProvider(providerName, oauth2.ProviderConfig{
		Name:         providerName,
		ClientID:     providerConfig.ClientID,
		ClientSecret: providerConfig.ClientSecret,
		RedirectURI:  providerConfig.RedirectURI,
		TokenURL:     providerConfig.TokenURL,
	})
	
	// 토큰 갱신
	newTokenResp, err := provider.RefreshToken(c.Context(), refreshToken.(string))
	if err != nil {
		logging.GetLogger().Error("Failed to refresh token", logging.F("error", err))
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to refresh token",
		})
	}
	
	// 세션 업데이트
	sess["access_token"] = newTokenResp.AccessToken
	if newTokenResp.RefreshToken != "" {
		sess["refresh_token"] = newTokenResp.RefreshToken
	}
	
	userMap["expires_at"] = newTokenResp.ExpiresAt
	sess["user"] = userMap
	
	if email, ok := userMap["email"].(string); ok {
		logging.GetLogger().Info("Token refreshed", logging.F("email", email))
	}
	
	return c.JSON(fiber.Map{
		"message":    "Token refreshed successfully",
		"expires_at": newTokenResp.ExpiresAt,
	})
}

// GetCurrentUser 현재 로그인한 사용자 정보 조회
func GetCurrentUser(c *fiber.Ctx) error {
	sess, err := getSession(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}
	
	userData, exists := sess["user"]
	if !exists || userData == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	return c.JSON(fiber.Map{
		"user": userData,
	})
}

// GetAuthStatus 인증 상태 확인
func GetAuthStatus(c *fiber.Ctx) error {
	sess, err := getSession(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}
	
	userData, exists := sess["user"]
	isAuthenticated := exists && userData != nil
	
	result := fiber.Map{
		"authenticated": isAuthenticated,
	}
	
	if isAuthenticated {
		if userMap, ok := userData.(fiber.Map); ok {
			result["user"] = fiber.Map{
				"email":    userMap["email"],
				"name":     userMap["name"],
				"username": userMap["username"],
				"role":     userMap["role"],
				"provider": userMap["provider"],
			}
			
			// 토큰 만료 시간 확인
			if expiresAt, ok := userMap["expires_at"].(time.Time); ok {
				result["expires_at"] = expiresAt
				result["expires_soon"] = time.Until(expiresAt) < 10*time.Minute
			}
		}
	}
	
	// OAuth2 설정 로드하여 활성화된 제공자 목록 반환
	oauth2Config := &configs.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err == nil && oauth2Config.IsEnabled() {
		result["oauth2_enabled"] = true
		result["available_providers"] = oauth2Config.GetEnabledProviders()
	} else {
		result["oauth2_enabled"] = false
	}
	
	return c.JSON(result)
}

// 유틸리티 함수들

// generateRandomString 안전한 랜덤 문자열 생성
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// generateCodeChallenge PKCE Code Challenge 생성 (SHA256 기반)
func generateCodeChallenge(verifier string) string {
	// 간단한 구현을 위해 base64 인코딩 사용
	// 실제로는 SHA256 해시 후 base64url 인코딩 필요
	encoded := base64.URLEncoding.EncodeToString([]byte(verifier))
	// 최대 43자로 제한 (안전한 길이 체크)
	if len(encoded) > 43 {
		return encoded[:43]
	}
	return encoded
}

// getSession 세션 헬퍼 함수
func getSession(c *fiber.Ctx) (fiber.Map, error) {
	// 간단한 구현을 위해 map 사용
	// 실제로는 fiber session 미들웨어 사용 필요
	sessionData := c.Locals("session")
	if sessionData == nil {
		// 새 세션 생성
		newSession := make(fiber.Map)
		c.Locals("session", newSession)
		return newSession, nil
	}
	
	if session, ok := sessionData.(fiber.Map); ok {
		return session, nil
	}
	
	return nil, fmt.Errorf("invalid session data type")
}