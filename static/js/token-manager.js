/**
 * ProxyND Token Manager
 * 클라이언트 측 JWT 토큰 자동 갱신 관리 라이브러리
 */

class TokenManager {
    constructor(options = {}) {
        this.options = {
            refreshEndpoint: '/auth/refresh',
            statusEndpoint: '/auth/status',
            refreshMargin: 10 * 60 * 1000, // 10분 전에 갱신
            retryInterval: 5 * 1000, // 5초 후 재시도
            maxRetries: 3,
            onTokenRefreshed: null,
            onAuthRequired: null,
            onError: null,
            debug: false,
            ...options
        };
        
        this.isRefreshing = false;
        this.refreshPromise = null;
        this.intervalId = null;
        this.retryCount = 0;
        
        this.init();
    }
    
    /**
     * 토큰 매니저 초기화
     */
    init() {
        this.log('TokenManager initialized');
        this.startMonitoring();
        this.setupResponseInterceptor();
    }
    
    /**
     * 토큰 만료 모니터링 시작
     */
    startMonitoring() {
        this.checkTokenStatus();
        
        // 1분마다 토큰 상태 확인
        this.intervalId = setInterval(() => {
            this.checkTokenStatus();
        }, 60 * 1000);
    }
    
    /**
     * 토큰 상태 확인
     */
    async checkTokenStatus() {
        try {
            const response = await fetch(this.options.statusEndpoint, {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json'
                }
            });
            
            if (!response.ok) {
                if (response.status === 401) {
                    this.handleAuthRequired();
                }
                return;
            }
            
            const data = await response.json();
            
            if (!data.authenticated) {
                this.handleAuthRequired();
                return;
            }
            
            // 토큰 만료 시간 확인
            if (data.expires_at) {
                const expiresAt = new Date(data.expires_at);
                const now = new Date();
                const timeUntilExpiry = expiresAt.getTime() - now.getTime();
                
                this.log(`Token expires in ${Math.round(timeUntilExpiry / 1000)} seconds`);
                
                // 토큰이 곧 만료되거나 이미 만료된 경우
                if (timeUntilExpiry <= this.options.refreshMargin) {
                    this.log('Token needs refresh');
                    this.refreshToken();
                }
            }
            
            // 서버에서 토큰 곧 만료 헤더를 보낸 경우
            if (data.expires_soon) {
                this.log('Server indicates token expires soon');
                this.refreshToken();
            }
            
        } catch (error) {
            this.log('Error checking token status:', error);
            this.handleError(error);
        }
    }
    
    /**
     * 토큰 갱신
     */
    async refreshToken() {
        if (this.isRefreshing) {
            this.log('Token refresh already in progress, waiting...');
            return this.refreshPromise;
        }
        
        this.isRefreshing = true;
        this.refreshPromise = this.performRefresh();
        
        try {
            const result = await this.refreshPromise;
            this.retryCount = 0; // 성공 시 재시도 카운트 리셋
            return result;
        } catch (error) {
            this.handleRefreshError(error);
            throw error;
        } finally {
            this.isRefreshing = false;
            this.refreshPromise = null;
        }
    }
    
    /**
     * 실제 토큰 갱신 수행
     */
    async performRefresh() {
        this.log('Performing token refresh...');
        
        const response = await fetch(this.options.refreshEndpoint, {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                this.handleAuthRequired();
                throw new Error('Authentication required');
            }
            
            const errorData = await response.json().catch(() => ({}));
            throw new Error(errorData.error || 'Token refresh failed');
        }
        
        const data = await response.json();
        
        this.log('Token refreshed successfully');
        
        // 갱신 성공 콜백 호출
        if (this.options.onTokenRefreshed) {
            this.options.onTokenRefreshed(data);
        }
        
        return data;
    }
    
    /**
     * HTTP 응답 인터셉터 설정
     */
    setupResponseInterceptor() {
        // Fetch API 래핑
        const originalFetch = window.fetch;
        
        window.fetch = async (url, options = {}) => {
            const response = await originalFetch(url, options);
            
            // X-Token-Expires-Soon 헤더 확인
            if (response.headers.get('X-Token-Expires-Soon') === 'true') {
                this.log('Server indicates token expires soon via header');
                this.refreshToken().catch(error => {
                    this.log('Auto-refresh failed:', error);
                });
            }
            
            // 401 Unauthorized 처리
            if (response.status === 401 && !url.includes(this.options.refreshEndpoint)) {
                this.log('Received 401, attempting token refresh');
                
                try {
                    await this.refreshToken();
                    
                    // 원래 요청 재시도
                    return originalFetch(url, options);
                } catch (refreshError) {
                    this.log('Token refresh failed, auth required');
                    this.handleAuthRequired();
                }
            }
            
            return response;
        };
    }
    
    /**
     * 토큰 갱신 실패 처리
     */
    handleRefreshError(error) {
        this.retryCount++;
        
        if (this.retryCount < this.options.maxRetries) {
            this.log(`Token refresh failed, retrying in ${this.options.retryInterval}ms (attempt ${this.retryCount}/${this.options.maxRetries})`);
            
            setTimeout(() => {
                this.refreshToken();
            }, this.options.retryInterval);
        } else {
            this.log('Max refresh retries exceeded, auth required');
            this.handleAuthRequired();
        }
    }
    
    /**
     * 인증 필요 처리
     */
    handleAuthRequired() {
        this.log('Authentication required');
        this.stop();
        
        if (this.options.onAuthRequired) {
            this.options.onAuthRequired();
        } else {
            // 기본 동작: 로그인 페이지로 리다이렉트
            window.location.href = '/auth/login/github';
        }
    }
    
    /**
     * 에러 처리
     */
    handleError(error) {
        this.log('Error occurred:', error);
        
        if (this.options.onError) {
            this.options.onError(error);
        }
    }
    
    /**
     * 모니터링 중지
     */
    stop() {
        if (this.intervalId) {
            clearInterval(this.intervalId);
            this.intervalId = null;
        }
        
        this.log('TokenManager stopped');
    }
    
    /**
     * 수동 토큰 갱신
     */
    async forceRefresh() {
        this.log('Force refresh requested');
        return this.refreshToken();
    }
    
    /**
     * 현재 토큰 상태 조회
     */
    async getStatus() {
        const response = await fetch(this.options.statusEndpoint, {
            method: 'GET',
            credentials: 'include'
        });
        
        if (!response.ok) {
            throw new Error('Failed to get auth status');
        }
        
        return response.json();
    }
    
    /**
     * 디버그 로그
     */
    log(...args) {
        if (this.options.debug) {
            console.log('[TokenManager]', ...args);
        }
    }
}

/**
 * 글로벌 토큰 매니저 인스턴스 생성 및 초기화
 */
window.TokenManager = TokenManager;

// 페이지 로드 시 자동 초기화
document.addEventListener('DOMContentLoaded', () => {
    // 전역 설정 확인
    const config = window.proxynd?.tokenManager || {};
    
    // 토큰 매니저 인스턴스 생성
    window.tokenManager = new TokenManager({
        debug: config.debug || false,
        onTokenRefreshed: (data) => {
            console.log('Token refreshed:', data);
        },
        onAuthRequired: () => {
            console.log('Authentication required');
            // 사용자 정의 로그인 페이지로 리다이렉트
            if (config.loginUrl) {
                window.location.href = config.loginUrl;
            }
        },
        onError: (error) => {
            console.error('TokenManager error:', error);
        }
    });
});

/**
 * API 호출 헬퍼 함수
 */
window.apiCall = async (url, options = {}) => {
    const defaultOptions = {
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        }
    };
    
    return fetch(url, { ...defaultOptions, ...options });
};

// CommonJS와 AMD 지원
if (typeof module !== 'undefined' && module.exports) {
    module.exports = TokenManager;
} else if (typeof define === 'function' && define.amd) {
    define([], () => TokenManager);
}