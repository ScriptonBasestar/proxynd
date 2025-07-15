# ProxyND Hot Reload 가이드

## 개요

ProxyND는 서비스 재시작 없이 설정을 동적으로 변경할 수 있는 핫 리로드 기능을 제공합니다. 이 가이드는 핫 리로드 시스템의 사용법과 구현 방법을 설명합니다.

## 주요 특징

- **자동 파일 감시**: 설정 파일 변경 시 자동으로 감지하여 리로드
- **수동 리로드**: SIGHUP 시그널을 통한 수동 리로드 지원
- **디바운싱**: 연속적인 파일 변경 시 마지막 변경만 처리
- **트랜잭션 리로드**: 모든 핸들러가 성공해야 설정 적용
- **Viper 통합**: Viper 기반 설정과 레거시 설정 모두 지원

## 사용 방법

### 기본 사용법

```go
// 핫 리로드 매니저 생성
hr, err := configs.NewUnifiedHotReload("config.yaml", false) // false = 레거시 모드
if err != nil {
    log.Fatal(err)
}

// 리로드 핸들러 등록
hr.RegisterHandler(&MyReloadHandler{})

// 핫 리로드 시작
if err := hr.Start(); err != nil {
    log.Fatal(err)
}

// 설정 사용
config := hr.GetConfig()

// 종료 시
defer hr.Stop()
```

### Viper 모드 사용

```go
// Viper 모드로 핫 리로드 생성
hr, err := configs.NewUnifiedHotReload("config.yaml", true) // true = Viper 모드
if err != nil {
    log.Fatal(err)
}
```

### 빠른 설정

```go
// 기본 핸들러가 포함된 빠른 설정
hr, err := configs.QuickSetupHotReload("config.yaml", useViper)
if err != nil {
    log.Fatal(err)
}
// 로깅, 캐시, 메트릭 핸들러가 자동으로 등록됨
```

## 리로드 핸들러 구현

### 핸들러 인터페이스

```go
type ReloadHandler interface {
    OnConfigReload(oldConfig, newConfig *UnifiedConfig) error
    Name() string
}
```

### 커스텀 핸들러 예제

```go
type DatabaseReloadHandler struct {
    db *sql.DB
}

func (h *DatabaseReloadHandler) OnConfigReload(old, new *UnifiedConfig) error {
    // 데이터베이스 연결 풀 크기 변경
    if old.Database.MaxConnections != new.Database.MaxConnections {
        h.db.SetMaxOpenConns(new.Database.MaxConnections)
        log.Printf("Updated max DB connections: %d", new.Database.MaxConnections)
    }
    
    // 재시작이 필요한 변경은 에러 반환
    if old.Database.Host != new.Database.Host {
        return fmt.Errorf("database host change requires restart")
    }
    
    return nil
}

func (h *DatabaseReloadHandler) Name() string {
    return "DatabaseReloadHandler"
}
```

### 간편 핸들러 생성

```go
// 함수 기반 핸들러
hr.RegisterHandler(&genericReloadHandler{
    name: "SimpleHandler",
    fn: func(old, new *UnifiedConfig) error {
        if old.Server.Port != new.Server.Port {
            log.Printf("Port changed: %d -> %d", old.Server.Port, new.Server.Port)
        }
        return nil
    },
})
```

## 설정 변경 감지

### 자동 감지

설정 파일이 변경되면 자동으로 감지됩니다:
- 파일 쓰기 (Write)
- 파일 생성 (Create)
- 파일 이름 변경 (Rename)

### 수동 리로드

SIGHUP 시그널을 보내서 수동으로 리로드:

```bash
# 프로세스 ID 찾기
ps aux | grep proxynd

# SIGHUP 시그널 보내기
kill -HUP <PID>
```

### 디바운싱

연속적인 파일 변경 시 마지막 변경 후 500ms 후에 리로드됩니다:

```go
// 디바운스 시간 변경
hr.SetDebounceTime(1 * time.Second)
```

## 리로드 가능한 설정

### 즉시 적용 가능

- 로그 레벨 및 포맷
- 캐시 TTL 및 크기 제한
- 메트릭 활성화/비활성화
- 인증 설정
- IP 화이트리스트
- 타임아웃 설정

### 재시작 필요

- 서버 포트
- 캐시 백엔드 타입 (file → redis)
- TLS 설정
- 데이터베이스 연결 정보

## 핸들러 실행 순서

1. 설정 파일 변경 감지
2. 새 설정 로드 및 검증
3. 모든 핸들러 순차 실행
4. 모든 핸들러 성공 시 설정 적용
5. 하나라도 실패 시 기존 설정 유지

## 베스트 프랙티스

### 1. 핸들러 설계

```go
func (h *MyHandler) OnConfigReload(old, new *UnifiedConfig) error {
    // 1. 변경 사항 확인
    if old.MyConfig == new.MyConfig {
        return nil // 변경 없음
    }
    
    // 2. 검증
    if !isValid(new.MyConfig) {
        return fmt.Errorf("invalid config: %v", new.MyConfig)
    }
    
    // 3. 적용
    if err := h.applyConfig(new.MyConfig); err != nil {
        return fmt.Errorf("failed to apply: %w", err)
    }
    
    // 4. 로깅
    log.Printf("[%s] Config updated: %v", h.Name(), new.MyConfig)
    
    return nil
}
```

### 2. 에러 처리

```go
// 재시작이 필요한 변경
if old.Critical != new.Critical {
    return fmt.Errorf("critical config change requires restart")
}

// 부분 실패 허용
if err := h.updateOptional(new); err != nil {
    log.Printf("Warning: optional update failed: %v", err)
    // 에러를 반환하지 않음
}

// 필수 변경 실패
if err := h.updateRequired(new); err != nil {
    return fmt.Errorf("required update failed: %w", err)
}
```

### 3. 동시성 처리

```go
type ThreadSafeHandler struct {
    mu     sync.RWMutex
    config *Config
}

func (h *ThreadSafeHandler) OnConfigReload(old, new *UnifiedConfig) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    h.config = extractConfig(new)
    return nil
}

func (h *ThreadSafeHandler) GetConfig() *Config {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    return h.config
}
```

## 디버깅

### 로그 확인

```bash
# 핫 리로드 관련 로그 필터링
tail -f app.log | grep -E "HotReload|Handler|Config"
```

### 상태 확인

```go
// 실행 상태 확인
if hr.IsRunning() {
    log.Println("Hot reload is active")
}

// Viper 모드 확인
if viper := hr.GetViper(); viper != nil {
    log.Println("Running in Viper mode")
}
```

## 트러블슈팅

### 설정이 리로드되지 않는 경우

1. 파일 권한 확인
2. 파일 시스템 이벤트 지원 확인
3. 디바운스 시간이 너무 긴지 확인
4. 핸들러 에러 로그 확인

### 핸들러가 실패하는 경우

1. 핸들러 로그 확인
2. 설정 검증 로직 확인
3. 의존성 상태 확인
4. 타이밍 이슈 확인

### 메모리 누수

1. 핸들러에서 리소스 정리 확인
2. 이벤트 리스너 중복 등록 확인
3. 고루틴 누수 확인

## 예제 프로젝트

전체 예제는 `/examples/hot_reload_example.go`를 참조하세요.