package example

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"proxynd/internal/services/proxy"
	proxymocks "proxynd/internal/services/proxy/mocks"
)

// ExampleService 예제 서비스 - Mock 사용 데모
type ExampleService struct {
	cache    proxy.CacheService
	upstream proxy.UpstreamClient
	config   proxy.ConfigService
}

// NewExampleService 예제 서비스 생성
func NewExampleService(cache proxy.CacheService, upstream proxy.UpstreamClient, config proxy.ConfigService) *ExampleService {
	return &ExampleService{
		cache:    cache,
		upstream: upstream,
		config:   config,
	}
}

// ProcessRequest 요청 처리 예제
func (s *ExampleService) ProcessRequest(ctx context.Context, key string) (string, error) {
	// 캐시에서 조회
	reader, exists, err := s.cache.Get(ctx, key)
	if err != nil {
		return "", err
	}
	
	if exists {
		data, _ := io.ReadAll(reader)
		reader.Close()
		return string(data), nil
	}
	
	// 업스트림에서 가져오기
	resp, err := s.upstream.Fetch(ctx, "http://example.com/"+key, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	// 캐시에 저장
	err = s.cache.Put(ctx, key, strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}

// TestExampleService_ProcessRequest Mock을 사용한 테스트 예제
func TestExampleService_ProcessRequest(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		setupMocks    func(*proxymocks.MockCacheService, *proxymocks.MockUpstreamClient, *proxymocks.MockConfigService)
		expectedValue string
		expectedError error
	}{
		{
			name: "캐시 히트",
			key:  "test-key",
			setupMocks: func(cache *proxymocks.MockCacheService, upstream *proxymocks.MockUpstreamClient, config *proxymocks.MockConfigService) {
				// 캐시에서 데이터 반환
				reader := io.NopCloser(strings.NewReader("cached data"))
				cache.EXPECT().Get(mock.Anything, "test-key").Return(reader, true, nil)
			},
			expectedValue: "cached data",
			expectedError: nil,
		},
		{
			name: "캐시 미스 - 업스트림에서 가져오기",
			key:  "test-key",
			setupMocks: func(cache *proxymocks.MockCacheService, upstream *proxymocks.MockUpstreamClient, config *proxymocks.MockConfigService) {
				// 캐시 미스
				cache.EXPECT().Get(mock.Anything, "test-key").Return(nil, false, nil)
				
				// 업스트림에서 데이터 가져오기
				resp := &proxy.ProxyResponse{
					Body:       io.NopCloser(strings.NewReader("upstream data")),
					StatusCode: 200,
				}
				upstream.EXPECT().Fetch(mock.Anything, "http://example.com/test-key", nil).Return(resp, nil)
				
				// 캐시에 저장
				cache.EXPECT().Put(mock.Anything, "test-key", mock.AnythingOfType("*strings.Reader")).Return(nil)
			},
			expectedValue: "upstream data",
			expectedError: nil,
		},
		{
			name: "캐시 조회 실패",
			key:  "test-key",
			setupMocks: func(cache *proxymocks.MockCacheService, upstream *proxymocks.MockUpstreamClient, config *proxymocks.MockConfigService) {
				cache.EXPECT().Get(mock.Anything, "test-key").Return(nil, false, errors.New("cache error"))
			},
			expectedValue: "",
			expectedError: errors.New("cache error"),
		},
		{
			name: "업스트림 요청 실패",
			key:  "test-key",
			setupMocks: func(cache *proxymocks.MockCacheService, upstream *proxymocks.MockUpstreamClient, config *proxymocks.MockConfigService) {
				// 캐시 미스
				cache.EXPECT().Get(mock.Anything, "test-key").Return(nil, false, nil)
				
				// 업스트림 요청 실패
				upstream.EXPECT().Fetch(mock.Anything, "http://example.com/test-key", nil).Return(nil, errors.New("upstream error"))
			},
			expectedValue: "",
			expectedError: errors.New("upstream error"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 생성
			mockCache := proxymocks.NewMockCacheService(t)
			mockUpstream := proxymocks.NewMockUpstreamClient(t)
			mockConfig := proxymocks.NewMockConfigService(t)
			
			// Mock 설정
			tt.setupMocks(mockCache, mockUpstream, mockConfig)
			
			// 서비스 생성
			service := NewExampleService(mockCache, mockUpstream, mockConfig)
			
			// 테스트 실행
			ctx := context.Background()
			result, err := service.ProcessRequest(ctx, tt.key)
			
			// 검증
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, result)
			}
		})
	}
}

// TestExampleService_ProcessRequest_WithExpectations expectation 검증 예제
func TestExampleService_ProcessRequest_WithExpectations(t *testing.T) {
	// Mock 생성
	mockCache := proxymocks.NewMockCacheService(t)
	mockUpstream := proxymocks.NewMockUpstreamClient(t)
	mockConfig := proxymocks.NewMockConfigService(t)
	
	// 특정 횟수 호출 검증
	mockCache.EXPECT().Get(mock.Anything, "key1").Return(nil, false, nil).Times(1)
	mockCache.EXPECT().Get(mock.Anything, "key2").Return(nil, false, nil).Times(1)
	
	// 순서 검증
	inOrder := mock.InOrder(
		mockCache.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil),
		mockUpstream.EXPECT().Fetch(mock.Anything, mock.Anything, mock.Anything).Return(&proxy.ProxyResponse{
			Body: io.NopCloser(strings.NewReader("data")),
		}, nil),
		mockCache.EXPECT().Put(mock.Anything, mock.Anything, mock.Anything).Return(nil),
	)
	_ = inOrder
	
	// 서비스 생성 및 실행
	service := NewExampleService(mockCache, mockUpstream, mockConfig)
	ctx := context.Background()
	
	// 첫 번째 요청
	_, err := service.ProcessRequest(ctx, "key1")
	require.NoError(t, err)
}

// TestExampleService_ProcessRequest_PartialMocking 부분 Mock 예제
func TestExampleService_ProcessRequest_PartialMocking(t *testing.T) {
	// Mock 생성
	mockCache := proxymocks.NewMockCacheService(t)
	mockUpstream := proxymocks.NewMockUpstreamClient(t)
	mockConfig := proxymocks.NewMockConfigService(t)
	
	// Match 함수를 사용한 유연한 매칭
	mockCache.EXPECT().Get(mock.Anything, mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "test-")
	})).Return(nil, false, nil)
	
	// 반환값을 동적으로 생성
	mockUpstream.EXPECT().Fetch(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, url string, headers map[string]string) (*proxy.ProxyResponse, error) {
			// URL에 따라 다른 응답 반환
			if strings.Contains(url, "error") {
				return nil, errors.New("simulated error")
			}
			return &proxy.ProxyResponse{
				Body: io.NopCloser(strings.NewReader("dynamic response for " + url)),
			}, nil
		})
	
	// 서비스 테스트
	service := NewExampleService(mockCache, mockUpstream, mockConfig)
	ctx := context.Background()
	
	result, err := service.ProcessRequest(ctx, "test-dynamic")
	require.NoError(t, err)
	assert.Contains(t, result, "dynamic response")
}