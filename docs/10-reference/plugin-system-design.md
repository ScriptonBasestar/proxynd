# 플러그인 시스템 설계

ProxyND의 플러그인 시스템은 외부 개발자가 ProxyND의 기능을 안전하게 확장할 수 있는 플랫폼을 제공합니다.

## 📋 개요

### 목적
- 서드파티 개발자의 기능 확장 지원
- 새로운 패키지 매니저 지원 추가
- 커스텀 인증 및 권한 시스템
- 특화된 캐시 백엔드 구현

### 설계 원칙
- **안전성**: 플러그인으로 인한 메인 시스템 영향 최소화
- **격리**: 플러그인 간 간섭 방지
- **표준화**: 일관된 개발 및 설치 경험
- **성능**: 플러그인 로딩으로 인한 성능 저하 최소화

## 🏗️ 아키텍처 설계

### 시스템 구성도

```
┌─────────────────────────────────────────────────────────────┐
│                    ProxyND Core                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐  │
│  │ Plugin Manager  │  │ Plugin Registry │  │   Plugin     │  │
│  │                 │  │                 │  │   Loader     │  │
│  └─────────────────┘  └─────────────────┘  └──────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                    Plugin API Layer                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐  │
│  │   gRPC Server   │  │  Security       │  │   Resource   │  │
│  │                 │  │  Sandbox        │  │   Manager    │  │
│  └─────────────────┘  └─────────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │                       │
            ┌───────▼──────┐        ┌──────▼──────┐
            │   Plugin A   │        │   Plugin B  │
            │  (Process)   │        │  (Process)  │
            └──────────────┘        └─────────────┘
```

### 핵심 컴포넌트

#### 1. Plugin Manager
```go
// PluginManager 플러그인 관리자
type PluginManager interface {
    // 플러그인 라이프사이클
    LoadPlugin(pluginPath string) (*Plugin, error)
    UnloadPlugin(pluginID string) error
    ReloadPlugin(pluginID string) error

    // 플러그인 조회
    ListPlugins() ([]*Plugin, error)
    GetPlugin(pluginID string) (*Plugin, error)

    // 플러그인 상태 관리
    EnablePlugin(pluginID string) error
    DisablePlugin(pluginID string) error

    // 설정 관리
    ConfigurePlugin(pluginID string, config map[string]interface{}) error
}

// Plugin 플러그인 정보
type Plugin struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Version     string                 `json:"version"`
    Description string                 `json:"description"`
    Author      string                 `json:"author"`
    Type        PluginType            `json:"type"`
    Status      PluginStatus          `json:"status"`
    Config      map[string]interface{} `json:"config"`
    Dependencies []string              `json:"dependencies"`
    CreatedAt   time.Time             `json:"created_at"`
    UpdatedAt   time.Time             `json:"updated_at"`
}

// PluginType 플러그인 타입
type PluginType string

const (
    PluginTypeProxy    PluginType = "proxy"    // 프록시 확장
    PluginTypeAuth     PluginType = "auth"     // 인증 시스템
    PluginTypeCache    PluginType = "cache"    // 캐시 백엔드
    PluginTypeMonitor  PluginType = "monitor"  // 모니터링
    PluginTypeMiddleware PluginType = "middleware" // 미들웨어
)

// PluginStatus 플러그인 상태
type PluginStatus string

const (
    PluginStatusLoaded    PluginStatus = "loaded"
    PluginStatusEnabled   PluginStatus = "enabled"
    PluginStatusDisabled  PluginStatus = "disabled"
    PluginStatusError     PluginStatus = "error"
)
```

#### 2. Plugin API Interface
```go
// ProxyPlugin 프록시 플러그인 인터페이스
type ProxyPlugin interface {
    Plugin

    // 프록시 기본 기능
    HandleRequest(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)
    GetProxyType() string
    ValidateConfig(config map[string]interface{}) error

    // 메타데이터
    GetSupportedURLPatterns() []string
    GetCacheConfig() *CacheConfig
}

// AuthPlugin 인증 플러그인 인터페이스
type AuthPlugin interface {
    Plugin

    // 인증 기능
    Authenticate(ctx context.Context, credentials map[string]string) (*User, error)
    Authorize(ctx context.Context, user *User, resource string, action string) (bool, error)

    // 사용자 관리
    CreateUser(ctx context.Context, user *User) error
    UpdateUser(ctx context.Context, user *User) error
    DeleteUser(ctx context.Context, userID string) error
    GetUser(ctx context.Context, userID string) (*User, error)
}

// CachePlugin 캐시 플러그인 인터페이스
type CachePlugin interface {
    Plugin

    // 캐시 기본 기능
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error

    // 고급 기능
    Clear(ctx context.Context, pattern string) error
    Size(ctx context.Context) (int64, error)
    Stats(ctx context.Context) (*CacheStats, error)
}
```

## 🔌 플러그인 개발 가이드

### 플러그인 구조

```
my-plugin/
├── plugin.yaml           # 플러그인 메타데이터
├── main.go              # 플러그인 엔트리포인트
├── handlers/            # 핸들러 구현
│   ├── proxy.go
│   └── auth.go
├── config/              # 설정 스키마
│   └── schema.yaml
├── tests/               # 테스트 코드
├── README.md           # 플러그인 문서
└── Makefile           # 빌드 스크립트
```

### 플러그인 메타데이터 (plugin.yaml)
```yaml
# 기본 정보
id: "custom-git-proxy"
name: "Git Repository Proxy"
version: "1.0.0"
description: "Git repository caching proxy for enterprise environments"
author: "Company Dev Team <dev@company.com>"
license: "MIT"
homepage: "https://github.com/company/proxynd-git-proxy"

# 플러그인 타입 및 의존성
type: "proxy"
api_version: "v1"
min_proxynd_version: "2.0.0"
dependencies:
  - "git"
  - "openssh-client"

# 런타임 설정
runtime:
  memory_limit: "256MB"
  cpu_limit: "1.0"
  network_access: true
  file_system_access: "read-write"
  allowed_paths:
    - "/tmp/git-cache"
    - "/var/lib/proxynd/git"

# 설정 스키마
config_schema:
  type: "object"
  properties:
    upstream_repos:
      type: "array"
      items:
        type: "string"
    cache_ttl:
      type: "string"
      default: "1h"
    auth_method:
      type: "string"
      enum: ["none", "ssh", "token"]
      default: "none"

# 지원하는 URL 패턴
url_patterns:
  - "/proxy/git/*"
  - "/api/git/*"

# 미들웨어 설정 (선택사항)
middleware:
  - name: "rate_limit"
    config:
      requests_per_minute: 100
  - name: "auth_required"
    config:
      methods: ["POST", "PUT", "DELETE"]
```

### 플러그인 구현 예시

#### 기본 구조 (main.go)
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/scriptonbasestar/proxynd/plugin"
)

// GitProxyPlugin Git 프록시 플러그인 구현
type GitProxyPlugin struct {
    config *GitProxyConfig
    client *GitClient
}

// GitProxyConfig 설정 구조체
type GitProxyConfig struct {
    UpstreamRepos []string `yaml:"upstream_repos"`
    CacheTTL      string   `yaml:"cache_ttl"`
    AuthMethod    string   `yaml:"auth_method"`
}

// Plugin interface 구현
func (p *GitProxyPlugin) GetID() string {
    return "custom-git-proxy"
}

func (p *GitProxyPlugin) GetName() string {
    return "Git Repository Proxy"
}

func (p *GitProxyPlugin) GetVersion() string {
    return "1.0.0"
}

func (p *GitProxyPlugin) GetType() plugin.PluginType {
    return plugin.PluginTypeProxy
}

func (p *GitProxyPlugin) Initialize(config map[string]interface{}) error {
    // 설정 파싱
    p.config = &GitProxyConfig{}
    if err := plugin.ParseConfig(config, p.config); err != nil {
        return fmt.Errorf("failed to parse config: %w", err)
    }

    // Git 클라이언트 초기화
    p.client = NewGitClient(p.config)
    return nil
}

func (p *GitProxyPlugin) Shutdown() error {
    if p.client != nil {
        return p.client.Close()
    }
    return nil
}

// ProxyPlugin interface 구현
func (p *GitProxyPlugin) HandleRequest(ctx context.Context, req *plugin.ProxyRequest) (*plugin.ProxyResponse, error) {
    // Git 요청 처리 로직
    switch req.Method {
    case "GET":
        return p.handleGitClone(ctx, req)
    case "POST":
        return p.handleGitPush(ctx, req)
    default:
        return nil, fmt.Errorf("unsupported method: %s", req.Method)
    }
}

func (p *GitProxyPlugin) GetProxyType() string {
    return "git"
}

func (p *GitProxyPlugin) GetSupportedURLPatterns() []string {
    return []string{"/proxy/git/*"}
}

func (p *GitProxyPlugin) ValidateConfig(config map[string]interface{}) error {
    gitConfig := &GitProxyConfig{}
    return plugin.ParseConfig(config, gitConfig)
}

// 요청 처리 구현
func (p *GitProxyPlugin) handleGitClone(ctx context.Context, req *plugin.ProxyRequest) (*plugin.ProxyResponse, error) {
    // Git clone 로직 구현
    repoURL := req.URL.Path[len("/proxy/git/"):]

    // 캐시 확인
    if cached, err := p.getCachedRepo(repoURL); err == nil {
        return &plugin.ProxyResponse{
            StatusCode: 200,
            Headers:    map[string]string{"Content-Type": "application/x-git-upload-pack"},
            Body:       cached,
        }, nil
    }

    // 업스트림에서 가져오기
    data, err := p.client.Clone(ctx, repoURL)
    if err != nil {
        return nil, err
    }

    // 캐시 저장
    _ = p.cacheRepo(repoURL, data)

    return &plugin.ProxyResponse{
        StatusCode: 200,
        Headers:    map[string]string{"Content-Type": "application/x-git-upload-pack"},
        Body:       data,
    }, nil
}

// 플러그인 엔트리포인트
func main() {
    plugin := &GitProxyPlugin{}

    // 플러그인 서버 시작
    server := plugin.NewServer(plugin)
    if err := server.Serve(); err != nil {
        log.Fatal(err)
    }
}
```

## 📦 플러그인 패키징 및 배포

### 빌드 시스템
```makefile
# Makefile
PLUGIN_NAME = custom-git-proxy
VERSION = 1.0.0
BUILD_DIR = build
DIST_DIR = dist

# 플러그인 빌드
build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(PLUGIN_NAME) ./main.go

# 플러그인 패키징
package: build
	mkdir -p $(DIST_DIR)
	tar -czf $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).tar.gz \
		-C $(BUILD_DIR) $(PLUGIN_NAME) \
		-C .. plugin.yaml README.md

# 플러그인 설치 테스트
install: package
	proxyndctl plugin install $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).tar.gz

# 플러그인 테스트
test:
	go test ./...

# 정리
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)

.PHONY: build package install test clean
```

### 배포 방식

#### 1. 로컬 설치
```bash
# 로컬 파일에서 설치
proxyndctl plugin install ./custom-git-proxy-1.0.0.tar.gz

# 압축 해제된 디렉토리에서 설치
proxyndctl plugin install ./custom-git-proxy/
```

#### 2. 원격 저장소에서 설치
```bash
# GitHub 릴리스에서 설치
proxyndctl plugin install github.com/company/proxynd-git-proxy@v1.0.0

# HTTP URL에서 설치
proxyndctl plugin install https://releases.company.com/plugins/git-proxy-1.0.0.tar.gz

# 플러그인 레지스트리에서 설치 (향후 구현)
proxyndctl plugin install --registry official git-proxy
```

## 🔐 보안 및 격리

### 프로세스 격리
```go
// PluginProcess 플러그인 프로세스 관리
type PluginProcess struct {
    plugin   *Plugin
    cmd      *exec.Cmd
    client   PluginClient
    limits   *ResourceLimits
    sandbox  *SecuritySandbox
}

// ResourceLimits 리소스 제한
type ResourceLimits struct {
    MaxMemory   int64         // 최대 메모리 사용량
    MaxCPU      float64       // 최대 CPU 사용률 (0.0-1.0)
    MaxFileSize int64         // 최대 파일 크기
    Timeout     time.Duration // 최대 실행 시간
}

// SecuritySandbox 보안 샌드박스
type SecuritySandbox struct {
    AllowedPaths    []string // 접근 가능한 경로
    NetworkAccess   bool     // 네트워크 접근 권한
    FileSystemMode  string   // 파일시스템 접근 모드 (read-only, read-write, none)
    EnvironmentVars []string // 전달할 환경 변수
}
```

### 권한 관리
```yaml
# 플러그인 권한 설정 예시
permissions:
  # 파일 시스템 접근
  filesystem:
    read:
      - "/var/lib/proxynd/cache"
      - "/tmp"
    write:
      - "/var/lib/proxynd/plugins/${plugin.id}"

  # 네트워크 접근
  network:
    outbound:
      - "github.com:443"
      - "gitlab.com:443"
    inbound: false

  # 시스템 호출 제한
  syscalls:
    allow:
      - "read"
      - "write"
      - "open"
      - "close"
    deny:
      - "exec"
      - "fork"
      - "socket"

  # API 접근 권한
  api:
    - "cache.read"
    - "cache.write"
    - "config.read"
```

### 통신 보안
```go
// 플러그인과 메인 프로세스 간 gRPC 통신
// TLS 인증서를 사용한 암호화 통신
func setupSecureConnection(pluginID string) (*grpc.ClientConn, error) {
    // 플러그인별 고유한 인증서 생성
    cert, err := generatePluginCertificate(pluginID)
    if err != nil {
        return nil, err
    }

    // TLS 설정
    creds := credentials.NewTLS(&tls.Config{
        Certificates: []tls.Certificate{cert},
        ServerName:   pluginID,
    })

    // gRPC 연결
    conn, err := grpc.Dial(
        getPluginAddress(pluginID),
        grpc.WithTransportCredentials(creds),
        grpc.WithTimeout(30*time.Second),
    )

    return conn, err
}
```

## 🔧 CLI 인터페이스

### 플러그인 관리 명령어

```bash
# 플러그인 설치
proxyndctl plugin install <source>
proxyndctl plugin install github.com/company/git-proxy@v1.0.0
proxyndctl plugin install ./local-plugin.tar.gz

# 플러그인 목록 조회
proxyndctl plugin list
proxyndctl plugin list --type proxy
proxyndctl plugin list --status enabled

# 플러그인 정보 조회
proxyndctl plugin info git-proxy
proxyndctl plugin info --detailed git-proxy

# 플러그인 활성화/비활성화
proxyndctl plugin enable git-proxy
proxyndctl plugin disable git-proxy

# 플러그인 설정
proxyndctl plugin config git-proxy --set cache_ttl=2h
proxyndctl plugin config git-proxy --file config.yaml

# 플러그인 업데이트
proxyndctl plugin update git-proxy
proxyndctl plugin update --all

# 플러그인 제거
proxyndctl plugin remove git-proxy
proxyndctl plugin remove --force git-proxy

# 플러그인 로그 조회
proxyndctl plugin logs git-proxy
proxyndctl plugin logs --tail 100 git-proxy

# 플러그인 상태 모니터링
proxyndctl plugin status git-proxy
proxyndctl plugin status --watch git-proxy
```

### 플러그인 개발 지원 명령어

```bash
# 플러그인 스캐폴딩 생성
proxyndctl plugin create --type proxy --name my-proxy
proxyndctl plugin create --template basic my-plugin

# 플러그인 검증
proxyndctl plugin validate ./my-plugin/
proxyndctl plugin validate --strict ./my-plugin/

# 플러그인 테스트
proxyndctl plugin test ./my-plugin/
proxyndctl plugin test --integration ./my-plugin/

# 플러그인 빌드
proxyndctl plugin build ./my-plugin/
proxyndctl plugin build --target linux/amd64 ./my-plugin/

# 플러그인 패키징
proxyndctl plugin package ./my-plugin/
proxyndctl plugin package --output my-plugin-1.0.0.tar.gz ./my-plugin/
```

## 📊 모니터링 및 관찰성

### 플러그인 메트릭
```go
// PluginMetrics 플러그인 메트릭
type PluginMetrics struct {
    PluginID     string    `json:"plugin_id"`
    RequestCount int64     `json:"request_count"`
    ErrorCount   int64     `json:"error_count"`
    AvgLatency   float64   `json:"avg_latency_ms"`
    MemoryUsage  int64     `json:"memory_usage_bytes"`
    CPUUsage     float64   `json:"cpu_usage_percent"`
    Uptime       int64     `json:"uptime_seconds"`
    LastActivity time.Time `json:"last_activity"`
}
```

### 건강성 검사
```bash
# 플러그인 헬스체크
proxyndctl plugin health git-proxy

# 모든 플러그인 헬스체크
proxyndctl plugin health --all

# 헬스체크 자동화 (향후 구현)
proxyndctl plugin monitor --interval 30s --alert-webhook https://hooks.slack.com/...
```

### 로깅 표준화
```json
{
  "timestamp": "2025-01-01T10:30:45Z",
  "plugin_id": "git-proxy",
  "plugin_version": "1.0.0",
  "level": "info",
  "message": "Git repository cloned successfully",
  "request_id": "req-123456",
  "duration_ms": 1250,
  "metadata": {
    "repository": "github.com/company/repo",
    "size_bytes": 1048576,
    "cache_hit": false
  }
}
```

## 🧪 테스트 전략

### 플러그인 개발자용 테스트 프레임워크
```go
// 플러그인 테스트 헬퍼
package plugintest

import (
    "testing"
    "github.com/scriptonbasestar/proxynd/plugin"
)

// TestPluginSuite 플러그인 테스트 스위트
type TestPluginSuite struct {
    plugin   plugin.Plugin
    server   *plugin.TestServer
    client   *plugin.TestClient
}

// NewTestSuite 테스트 스위트 생성
func NewTestSuite(t *testing.T, p plugin.Plugin) *TestPluginSuite {
    return &TestPluginSuite{
        plugin: p,
        server: plugin.NewTestServer(p),
        client: plugin.NewTestClient(),
    }
}

// 사용 예시
func TestGitProxy(t *testing.T) {
    plugin := &GitProxyPlugin{}
    suite := plugintest.NewTestSuite(t, plugin)
    defer suite.Cleanup()

    // 플러그인 초기화 테스트
    err := plugin.Initialize(map[string]interface{}{
        "upstream_repos": []string{"https://github.com/example/repo"},
        "cache_ttl": "1h",
    })
    assert.NoError(t, err)

    // 요청 처리 테스트
    req := &plugin.ProxyRequest{
        Method: "GET",
        URL:    "/proxy/git/example/repo",
    }

    resp, err := plugin.HandleRequest(context.Background(), req)
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

## 🚀 구현 로드맵

### Phase 1: 기본 인프라 (3주)
- [ ] 플러그인 매니저 구현
- [ ] gRPC 기반 플러그인 통신
- [ ] 기본 프로세스 격리
- [ ] CLI 인터페이스 구현

### Phase 2: 보안 및 격리 (2주)
- [ ] 리소스 제한 구현
- [ ] 보안 샌드박스
- [ ] 권한 관리 시스템
- [ ] 암호화 통신

### Phase 3: 개발자 도구 (2주)
- [ ] 플러그인 스캐폴딩
- [ ] 테스트 프레임워크
- [ ] 문서 생성 도구
- [ ] 예제 플러그인

### Phase 4: 운영 기능 (1주)
- [ ] 모니터링 및 메트릭
- [ ] 로깅 표준화
- [ ] 자동 업데이트
- [ ] 플러그인 레지스트리

---

**📅 작성일**: 2025-01-01  
**📝 버전**: v1.0  
**🎯 구현 우선순위**: 낮음 (장기 프로젝트)
