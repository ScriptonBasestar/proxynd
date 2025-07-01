package oauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient HTTP 클라이언트 인터페이스
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient 기본 HTTP 클라이언트
var DefaultHTTPClient HTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// exchangeCodeGeneric Generic 코드 교환 구현
func exchangeCodeGeneric(ctx context.Context, config ProviderConfig, code string, codeVerifier string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURI)
	
	// PKCE 지원
	if config.EnablePKCE && codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// 만료 시간 계산
	if tokenResp.ExpiresIn > 0 {
		tokenResp.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	
	return &tokenResp, nil
}

// getUserInfoGeneric Generic 사용자 정보 조회 구현
func getUserInfoGeneric(ctx context.Context, config ProviderConfig, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", config.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var rawUserInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawUserInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	
	// 필드 매핑을 통한 사용자 정보 추출
	userInfo := &UserInfo{}
	
	if id, ok := getStringField(rawUserInfo, config.UserIDField); ok {
		userInfo.ID = id
	}
	if email, ok := getStringField(rawUserInfo, config.EmailField); ok {
		userInfo.Email = email
	}
	if name, ok := getStringField(rawUserInfo, config.NameField); ok {
		userInfo.Name = name
	}
	if username, ok := getStringField(rawUserInfo, config.UsernameField); ok {
		userInfo.Username = username
	}
	if avatar, ok := getStringField(rawUserInfo, config.AvatarField); ok {
		userInfo.Avatar = avatar
	}
	
	// 추가 정보 추출 (선택사항)
	if company, ok := getStringField(rawUserInfo, "company"); ok {
		userInfo.Company = company
	}
	if location, ok := getStringField(rawUserInfo, "location"); ok {
		userInfo.Location = location
	}
	if bio, ok := getStringField(rawUserInfo, "bio"); ok {
		userInfo.Bio = bio
	}
	
	// 생성/수정 시간 파싱 시도
	if createdAt, ok := getTimeField(rawUserInfo, "created_at"); ok {
		userInfo.CreatedAt = createdAt
	}
	if updatedAt, ok := getTimeField(rawUserInfo, "updated_at"); ok {
		userInfo.UpdatedAt = updatedAt
	}
	
	return userInfo, nil
}

// refreshTokenGeneric Generic 토큰 갱신 구현
func refreshTokenGeneric(ctx context.Context, config ProviderConfig, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("refresh_token", refreshToken)
	
	req, err := http.NewRequestWithContext(ctx, "POST", config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}
	
	// 만료 시간 계산
	if tokenResp.ExpiresIn > 0 {
		tokenResp.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	
	return &tokenResp, nil
}

// revokeTokenGeneric Generic 토큰 취소 구현
func revokeTokenGeneric(ctx context.Context, config ProviderConfig, token string) error {
	if config.RevokeURL == "" {
		return nil // 토큰 취소를 지원하지 않는 제공자
	}
	
	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	
	req, err := http.NewRequestWithContext(ctx, "POST", config.RevokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token revocation failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// validateTokenGeneric Generic 토큰 검증 구현
func validateTokenGeneric(ctx context.Context, config ProviderConfig, token string) (*TokenInfo, error) {
	// 대부분의 OAuth2 제공자는 토큰 검증을 위해 사용자 정보 조회를 사용
	// 실제 토큰 검증 엔드포인트가 있는 경우 각 제공자에서 오버라이드
	userInfo, err := getUserInfoGeneric(ctx, config, token)
	if err != nil {
		return &TokenInfo{Valid: false}, nil
	}
	
	return &TokenInfo{
		Valid:  true,
		UserID: userInfo.ID,
		Scope:  strings.Join(config.Scopes, " "),
	}, nil
}

// getStringField map에서 문자열 필드 추출
func getStringField(data map[string]interface{}, fieldName string) (string, bool) {
	if fieldName == "" {
		return "", false
	}
	
	value, exists := data[fieldName]
	if !exists {
		return "", false
	}
	
	str, ok := value.(string)
	return str, ok
}

// getTimeField map에서 시간 필드 추출
func getTimeField(data map[string]interface{}, fieldName string) (time.Time, bool) {
	if fieldName == "" {
		return time.Time{}, false
	}
	
	value, exists := data[fieldName]
	if !exists {
		return time.Time{}, false
	}
	
	// 문자열로 저장된 시간 파싱 시도
	if str, ok := value.(string); ok {
		// ISO 8601 형식 파싱 시도
		if t, err := time.Parse(time.RFC3339, str); err == nil {
			return t, true
		}
		// 다른 형식들 시도
		formats := []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
			"2006-01-02 15:04:05",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, str); err == nil {
				return t, true
			}
		}
	}
	
	return time.Time{}, false
}

// CreateAPIRequest API 요청 생성 헬퍼
func CreateAPIRequest(ctx context.Context, method, url string, body interface{}, accessToken string) (*http.Request, error) {
	var reqBody io.Reader
	
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	return req, nil
}

// ParseAPIResponse API 응답 파싱 헬퍼
func ParseAPIResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	if target == nil {
		return nil
	}
	
	return json.NewDecoder(resp.Body).Decode(target)
}