package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"proxynd/internal/domain/docker"
	"proxynd/logging"
)

// authManagerImpl Docker 레지스트리 인증 관리 구현
type authManagerImpl struct {
	config     docker.ProxyConfig
	logger     logging.Logger
	tokenCache map[string]*docker.RegistryAuth // 토큰 캐시
	cacheMutex sync.RWMutex                    // 캐시 보호용 뮤텍스
}

// NewAuthenticationManager AuthenticationManager 생성자
func NewAuthenticationManager(
	config docker.ProxyConfig,
	logger logging.Logger,
) docker.AuthenticationManager {
	return &authManagerImpl{
		config:     config,
		logger:     logger,
		tokenCache: make(map[string]*docker.RegistryAuth),
		cacheMutex: sync.RWMutex{},
	}
}

// GetAuthToken 레지스트리별 인증 토큰 조회 및 갱신
func (a *authManagerImpl) GetAuthToken(ctx context.Context, registryURL, repository string) (*docker.RegistryAuth, error) {
	a.logger.Debug("Getting auth token",
		logging.F("registryURL", registryURL),
		logging.F("repository", repository))

	// 설정에서 레지스트리 인증 정보 확인
	registryConfig := a.config.GetRegistryConfig(registryURL)
	if registryConfig == nil {
		// 기본 레지스트리 설정 확인
		defaultRegistry := a.config.GetDefaultRegistry()
		if defaultRegistry != nil && defaultRegistry.Auth.Username != "" {
			return &docker.RegistryAuth{
				Type:     "basic",
				Username: defaultRegistry.Auth.Username,
				Password: defaultRegistry.Auth.Password,
			}, nil
		}
		return nil, fmt.Errorf("no auth config for registry: %s", registryURL)
	}

	// Basic Auth가 설정된 경우
	if registryConfig.Auth.Username != "" && registryConfig.Auth.Password != "" {
		return &docker.RegistryAuth{
			Type:     "basic",
			Username: registryConfig.Auth.Username,
			Password: registryConfig.Auth.Password,
		}, nil
	}

	// Bearer 토큰 방식 처리
	cacheKey := fmt.Sprintf("%s:%s", registryURL, repository)

	a.cacheMutex.RLock()
	if auth, exists := a.tokenCache[cacheKey]; exists {
		if a.ValidateToken(ctx, auth) {
			a.cacheMutex.RUnlock()
			return auth, nil
		}
	}
	a.cacheMutex.RUnlock()

	// 토큰이 없거나 만료된 경우 새로 획득
	return a.acquireToken(ctx, registryURL, repository)
}

// acquireToken 새로운 Bearer 토큰 획득
func (a *authManagerImpl) acquireToken(ctx context.Context, registryURL, repository string) (*docker.RegistryAuth, error) {
	// Docker Hub의 경우 토큰 서비스 URL
	tokenURL := "https://auth.docker.io/token"
	service := "registry.docker.io"

	// 다른 레지스트리의 경우 /v2/ 엔드포인트에서 챌린지 확인
	if !strings.Contains(registryURL, "docker.io") {
		challenge, err := a.getAuthChallenge(ctx, registryURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get auth challenge: %w", err)
		}

		if challenge.Realm != "" {
			tokenURL = challenge.Realm
			service = challenge.Service
		}
	}

	// 토큰 요청
	reqURL := fmt.Sprintf("%s?service=%s&scope=repository:%s:pull", tokenURL, service, repository)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	// Basic Auth 설정 (있는 경우)
	registryConfig := a.config.GetRegistryConfig(registryURL)
	if registryConfig != nil && registryConfig.Auth.Username != "" {
		req.SetBasicAuth(registryConfig.Auth.Username, registryConfig.Auth.Password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	// 토큰 응답 파싱
	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	token := tokenResp.Token
	if token == "" {
		token = tokenResp.AccessToken
	}

	if token == "" {
		return nil, fmt.Errorf("no token in response")
	}

	// 만료 시간 계산 (기본값: 1시간)
	expiresIn := tokenResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}

	auth := &docker.RegistryAuth{
		Type:      "bearer",
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
		Realm:     tokenURL,
		Service:   service,
		Scope:     fmt.Sprintf("repository:%s:pull", repository),
	}

	// 캐시에 저장
	cacheKey := fmt.Sprintf("%s:%s", registryURL, repository)
	a.cacheMutex.Lock()
	a.tokenCache[cacheKey] = auth
	a.cacheMutex.Unlock()

	return auth, nil
}

// getAuthChallenge 인증 챌린지 정보 획득
func (a *authManagerImpl) getAuthChallenge(ctx context.Context, registryURL string) (*docker.RegistryAuth, error) {
	url := fmt.Sprintf("%s/v2/", strings.TrimRight(registryURL, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return a.ProcessAuthChallenge(ctx, resp.Header)
	}

	return nil, fmt.Errorf("no auth challenge received")
}

// RefreshToken 만료된 토큰 갱신
func (a *authManagerImpl) RefreshToken(ctx context.Context, auth *docker.RegistryAuth) error {
	a.logger.Debug("Refreshing expired token", logging.F("realm", auth.Realm))

	// 리프레시 토큰이 있는 경우 사용
	if auth.RefreshToken != "" {
		return a.refreshWithRefreshToken(ctx, auth)
	}

	// 새로운 토큰 획득 (기존 토큰으로는 갱신 불가)
	return fmt.Errorf("token refresh not supported without refresh token")
}

// refreshWithRefreshToken 리프레시 토큰으로 갱신
func (a *authManagerImpl) refreshWithRefreshToken(ctx context.Context, auth *docker.RegistryAuth) error {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", auth.RefreshToken)
	data.Set("service", auth.Service)
	data.Set("scope", auth.Scope)

	req, err := http.NewRequestWithContext(ctx, "POST", auth.Realm, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		Token        string `json:"token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	// 토큰 정보 업데이트
	token := tokenResp.Token
	if token == "" {
		token = tokenResp.AccessToken
	}

	auth.Token = token
	if tokenResp.RefreshToken != "" {
		auth.RefreshToken = tokenResp.RefreshToken
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}
	auth.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)

	return nil
}

// ValidateToken 토큰 유효성 검증
func (a *authManagerImpl) ValidateToken(ctx context.Context, auth *docker.RegistryAuth) bool {
	if auth == nil {
		return false
	}

	// Basic Auth는 항상 유효
	if auth.Type == "basic" {
		return auth.Username != "" && auth.Password != ""
	}

	// Bearer 토큰 만료 확인
	if auth.Type == "bearer" {
		if auth.Token == "" {
			return false
		}

		// 만료 시간 확인 (5분 여유)
		return time.Now().Before(auth.ExpiresAt.Add(-5 * time.Minute))
	}

	return false
}

// ProcessAuthChallenge 인증 챌린지 처리 (401 응답)
func (a *authManagerImpl) ProcessAuthChallenge(ctx context.Context, headers http.Header) (*docker.RegistryAuth, error) {
	authHeader := headers.Get("Www-Authenticate")
	if authHeader == "" {
		return nil, fmt.Errorf("no auth challenge in response")
	}

	// Bearer 챌린지 파싱
	if strings.HasPrefix(authHeader, "Bearer ") {
		return a.parseBearerChallenge(authHeader)
	}

	// Basic 챌린지 파싱
	if strings.HasPrefix(authHeader, "Basic ") {
		return &docker.RegistryAuth{
			Type: "basic",
		}, nil
	}

	return nil, fmt.Errorf("unsupported auth challenge: %s", authHeader)
}

// parseBearerChallenge Bearer 챌린지 파싱
func (a *authManagerImpl) parseBearerChallenge(authHeader string) (*docker.RegistryAuth, error) {
	// Bearer realm="https://auth.docker.io/token",service="registry.docker.io"
	parts := strings.TrimPrefix(authHeader, "Bearer ")

	auth := &docker.RegistryAuth{
		Type: "bearer",
	}

	// 파라미터 파싱
	params := strings.Split(parts, ",")
	for _, param := range params {
		param = strings.TrimSpace(param)
		if strings.HasPrefix(param, "realm=") {
			auth.Realm = strings.Trim(strings.TrimPrefix(param, "realm="), "\"")
		} else if strings.HasPrefix(param, "service=") {
			auth.Service = strings.Trim(strings.TrimPrefix(param, "service="), "\"")
		} else if strings.HasPrefix(param, "scope=") {
			auth.Scope = strings.Trim(strings.TrimPrefix(param, "scope="), "\"")
		}
	}

	if auth.Realm == "" {
		return nil, fmt.Errorf("no realm in bearer challenge")
	}

	return auth, nil
}

// SetBasicAuth Basic 인증 설정
func (a *authManagerImpl) SetBasicAuth(request *http.Request, username, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("username and password required for basic auth")
	}

	request.SetBasicAuth(username, password)
	return nil
}

// SetBearerAuth Bearer 토큰 인증 설정
func (a *authManagerImpl) SetBearerAuth(request *http.Request, token string) error {
	if token == "" {
		return fmt.Errorf("token required for bearer auth")
	}

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return nil
}

// ClearExpiredTokens 만료된 토큰 정리
func (a *authManagerImpl) ClearExpiredTokens(ctx context.Context) error {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, auth := range a.tokenCache {
		if auth.Type == "bearer" && now.After(auth.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(a.tokenCache, key)
		a.logger.Debug("Cleared expired token", logging.F("key", key))
	}

	a.logger.Debug("Token cleanup completed", logging.F("expired_count", len(expiredKeys)))
	return nil
}
