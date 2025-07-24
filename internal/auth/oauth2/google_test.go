package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test-specific constants
const (
	testProviderGoogle       = "google"
	testPathTokenInfo        = "/tokeninfo"
	testHostOAuth2GoogleAPIs = "oauth2.googleapis.com"
)

func TestNewGoogleProvider(t *testing.T) {
	config := ProviderConfig{
		Name:         testProviderGoogle,
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "https://example.com/callback",
	}

	provider := NewGoogleProvider(config)

	if provider.GetName() != testProviderGoogle {
		t.Errorf("Expected provider name '%s', got '%s'", testProviderGoogle, provider.GetName())
	}

	// Google 기본값이 설정되었는지 확인
	googleProvider := provider.(*GoogleProvider)
	if googleProvider.config.AuthURL != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Error("Google AuthURL should be set to default value")
	}
	if googleProvider.config.TokenURL != "https://oauth2.googleapis.com/token" {
		t.Error("Google TokenURL should be set to default value")
	}
	if googleProvider.config.UserInfoURL != "https://www.googleapis.com/oauth2/v2/userinfo" {
		t.Error("Google UserInfoURL should be set to default value")
	}
	if googleProvider.config.RevokeURL != "https://oauth2.googleapis.com/revoke" {
		t.Error("Google RevokeURL should be set to default value")
	}

	expectedScopes := []string{"openid", "email", "profile"}
	scopes := provider.GetScopes()
	if len(scopes) != len(expectedScopes) {
		t.Errorf("Expected %d scopes, got %d", len(expectedScopes), len(scopes))
	}
	for i, scope := range scopes {
		if scope != expectedScopes[i] {
			t.Errorf("Expected scope '%s', got '%s'", expectedScopes[i], scope)
		}
	}
}

func TestGoogleProvider_GetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v2/userinfo" {
			userResponse := GoogleUser{
				ID:            "123456789",
				Email:         "test@example.com",
				Name:          "Test User",
				GivenName:     "Test",
				FamilyName:    "User",
				Picture:       "https://lh3.googleusercontent.com/photo.jpg",
				Locale:        "en",
				VerifiedEmail: true,
			}
			_ = json.NewEncoder(w).Encode(userResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        "google",
		UserInfoURL: server.URL + "/oauth2/v2/userinfo",
	}

	provider := NewGoogleProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &http.Client{}

	ctx := context.Background()
	userInfo, err := provider.GetUserInfo(ctx, "test_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if userInfo.ID != "123456789" {
		t.Errorf("Expected ID '123456789', got '%s'", userInfo.ID)
	}
	if userInfo.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got '%s'", userInfo.Name)
	}
	if userInfo.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", userInfo.Email)
	}
	if userInfo.Avatar != "https://lh3.googleusercontent.com/photo.jpg" {
		t.Errorf("Expected avatar URL, got '%s'", userInfo.Avatar)
	}
}

func TestGoogleProvider_GetUserInfo_NameFromParts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v2/userinfo" {
			userResponse := GoogleUser{
				ID:    "123456789",
				Email: "test@example.com",
				// Name이 비어있고 GivenName과 FamilyName만 있는 경우
				GivenName:     "John",
				FamilyName:    "Doe",
				Picture:       "https://lh3.googleusercontent.com/photo.jpg",
				VerifiedEmail: true,
			}
			_ = json.NewEncoder(w).Encode(userResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:        "google",
		UserInfoURL: server.URL + "/oauth2/v2/userinfo",
	}

	provider := NewGoogleProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// 테스트용 클라이언트로 교체
	DefaultHTTPClient = &http.Client{}

	ctx := context.Background()
	userInfo, err := provider.GetUserInfo(ctx, "test_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if userInfo.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", userInfo.Name)
	}
}

func TestGoogleProvider_GetUserOrganizations(t *testing.T) {
	config := ProviderConfig{
		Name: "google",
	}

	provider := NewGoogleProvider(config)
	ctx := context.Background()

	// Google은 조직 개념이 없으므로 빈 배열 반환
	organizations, err := provider.GetUserOrganizations(ctx, "test_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(organizations) != 0 {
		t.Errorf("Expected empty organizations list, got: %v", organizations)
	}
}

func TestGoogleProvider_ValidateToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathTokenInfo {
			tokenInfo := GoogleTokenInfo{
				Audience:  "test_client_id",
				UserID:    "123456789",
				Scope:     "openid email profile",
				ExpiresIn: 3600,
				Email:     "test@example.com",
			}
			_ = json.NewEncoder(w).Encode(tokenInfo)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:     "google",
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostOAuth2GoogleAPIs {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	tokenInfo, err := provider.ValidateToken(ctx, "valid_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !tokenInfo.Valid {
		t.Error("Expected token to be valid")
	}
	if tokenInfo.UserID != "123456789" {
		t.Errorf("Expected user ID '123456789', got '%s'", tokenInfo.UserID)
	}
	if tokenInfo.Scope != "openid email profile" {
		t.Errorf("Expected scope 'openid email profile', got '%s'", tokenInfo.Scope)
	}
}

func TestGoogleProvider_ValidateToken_WrongAudience(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathTokenInfo {
			tokenInfo := GoogleTokenInfo{
				Audience:  "wrong_client_id", // 잘못된 클라이언트 ID
				UserID:    "123456789",
				Scope:     "openid email profile",
				ExpiresIn: 3600,
			}
			_ = json.NewEncoder(w).Encode(tokenInfo)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:     "google",
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostOAuth2GoogleAPIs {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	tokenInfo, err := provider.ValidateToken(ctx, "invalid_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if tokenInfo.Valid {
		t.Error("Expected token to be invalid due to wrong audience")
	}
}

func TestGoogleProvider_ValidateToken_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_token",
			"error_description": "Invalid Value",
		})
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:     "google",
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostOAuth2GoogleAPIs {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	tokenInfo, err := provider.ValidateToken(ctx, "invalid_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if tokenInfo.Valid {
		t.Error("Expected token to be invalid")
	}
}

func TestGoogleProvider_VerifyIDToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathTokenInfo {
			tokenInfo := GoogleTokenInfo{
				Issuer:         "https://accounts.google.com",
				Audience:       "test_client_id",
				Subject:        "123456789",
				Email:          "test@example.com",
				EmailVerified:  true,
				Name:           "Test User",
				Picture:        "https://lh3.googleusercontent.com/photo.jpg",
				GivenName:      "Test",
				FamilyName:     "User",
				Locale:         "en",
				IssuedAt:       time.Now().Unix(),
				ExpirationTime: time.Now().Add(time.Hour).Unix(),
				HostedDomain:   "example.com",
			}
			_ = json.NewEncoder(w).Encode(tokenInfo)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:     "google",
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostOAuth2GoogleAPIs {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	tokenInfo, err := provider.VerifyIDToken(ctx, "valid_id_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if tokenInfo.Subject != "123456789" {
		t.Errorf("Expected subject '123456789', got '%s'", tokenInfo.Subject)
	}
	if tokenInfo.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", tokenInfo.Email)
	}
	if tokenInfo.HostedDomain != "example.com" {
		t.Errorf("Expected hosted domain 'example.com', got '%s'", tokenInfo.HostedDomain)
	}
}

func TestGoogleProvider_VerifyIDToken_WrongIssuer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathTokenInfo {
			tokenInfo := GoogleTokenInfo{
				Issuer:         "https://evil.com", // 잘못된 발급자
				Audience:       "test_client_id",
				Subject:        "123456789",
				ExpirationTime: time.Now().Add(time.Hour).Unix(),
			}
			_ = json.NewEncoder(w).Encode(tokenInfo)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:     "google",
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostOAuth2GoogleAPIs {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	_, err := provider.VerifyIDToken(ctx, "invalid_id_token")

	if err == nil {
		t.Error("Expected error for wrong issuer")
	}

	if !contains(err.Error(), "issuer mismatch") {
		t.Errorf("Expected issuer mismatch error, got: %v", err)
	}
}

func TestGoogleProvider_GetHostedDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathTokenInfo {
			tokenInfo := GoogleTokenInfo{
				Issuer:         "https://accounts.google.com",
				Audience:       "test_client_id",
				Subject:        "123456789",
				ExpirationTime: time.Now().Add(time.Hour).Unix(),
				HostedDomain:   "example.com",
			}
			_ = json.NewEncoder(w).Encode(tokenInfo)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:     "google",
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostOAuth2GoogleAPIs {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	domain, err := provider.GetHostedDomain(ctx, "valid_id_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", domain)
	}
}

func TestGoogleProvider_CheckGSuiteUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathTokenInfo {
			tokenInfo := GoogleTokenInfo{
				Issuer:         "https://accounts.google.com",
				Audience:       "test_client_id",
				Subject:        "123456789",
				ExpirationTime: time.Now().Add(time.Hour).Unix(),
				HostedDomain:   "company.com",
			}
			_ = json.NewEncoder(w).Encode(tokenInfo)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:     "google",
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostOAuth2GoogleAPIs {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	isGSuite, err := provider.CheckGSuiteUser(ctx, "valid_id_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !isGSuite {
		t.Error("Expected user to be G Suite user")
	}
}

func TestGoogleProvider_GetUserDomains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testPathTokenInfo {
			tokenInfo := GoogleTokenInfo{
				Issuer:         "https://accounts.google.com",
				Audience:       "test_client_id",
				Subject:        "123456789",
				ExpirationTime: time.Now().Add(time.Hour).Unix(),
				HostedDomain:   "company.com",
			}
			_ = json.NewEncoder(w).Encode(tokenInfo)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfig{
		Name:     "google",
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	// 원래 HTTP 클라이언트 백업
	originalClient := DefaultHTTPClient
	defer func() { DefaultHTTPClient = originalClient }()

	// Mock 서버 URL을 사용하도록 설정
	DefaultHTTPClient = &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == testHostOAuth2GoogleAPIs {
				req.URL.Scheme = testSchemeHTTP
				req.URL.Host = server.URL[7:] // "http://" 제거
			}
			return http.DefaultClient.Do(req)
		},
	}

	ctx := context.Background()
	domains, err := provider.GetUserDomains(ctx, "valid_id_token")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedDomains := []string{"company.com"}
	if len(domains) != len(expectedDomains) {
		t.Errorf("Expected %d domains, got %d", len(expectedDomains), len(domains))
	}

	for i, domain := range domains {
		if domain != expectedDomains[i] {
			t.Errorf("Expected domain '%s', got '%s'", expectedDomains[i], domain)
		}
	}
}

func TestGoogleProvider_Registration(t *testing.T) {
	// Google 제공자가 자동으로 등록되었는지 확인
	providers := GetRegisteredProviders()

	found := false
	for _, name := range providers {
		if name == testProviderGoogle {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected Google provider to be automatically registered")
	}

	// 제공자 생성 테스트
	config := ProviderConfig{
		Name:         testProviderGoogle,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		RedirectURI:  "https://example.com/callback",
	}

	provider := CreateProvider(testProviderGoogle, config)
	if provider == nil {
		t.Error("Expected Google provider to be created")
	}

	googleProvider, ok := provider.(*GoogleProvider)
	if !ok {
		t.Error("Expected provider to be GoogleProvider instance")
	}

	if googleProvider.GetName() != testProviderGoogle {
		t.Errorf("Expected provider name '%s', got '%s'", testProviderGoogle, googleProvider.GetName())
	}
}
