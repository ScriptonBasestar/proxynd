---
phase: 1
order: 1
source_plan: /docs/refactoring/01-dependency-injection.md
priority: high
tags: [container, config, di]
---

# 📌 작업: Container 설정 관리 기능 확장

## 개요
기존 Container 구조체에 설정 관리, 캐싱, 핫 리로드 기능을 추가하여 매 요청마다 발생하는 ReadConfig() 호출을 제거합니다.

## 현재 문제점
- 30+ 곳에서 ReadConfig() 반복 호출
- 매 요청마다 파일 I/O 발생
- 설정 변경 시 즉시 반영 안됨
- 테스트 시 설정 모킹 어려움

## 구현 내용

### 1. Container 구조체 확장
```go
// internal/app/container.go
type Container struct {
    mu            sync.RWMutex
    services      map[string]interface{}
    constructors  map[string]func() (interface{}, error)
    
    // 설정 관련 추가
    config        *configs.Config
    configMu      sync.RWMutex
    configWatcher *fsnotify.Watcher
    
    // 서비스 레지스트리
    handlers      map[string]fiber.Handler
    middlewares   []fiber.Handler
}
```

### 2. 설정 관련 메서드 추가
```go
func (c *Container) Config() *configs.Config {
    c.configMu.RLock()
    defer c.configMu.RUnlock()
    return c.config
}

func (c *Container) ReloadConfig() error {
    c.configMu.Lock()
    defer c.configMu.Unlock()
    
    newConfig, err := configs.ReadConfig()
    if err != nil {
        return err
    }
    
    // 검증
    if err := newConfig.Validate(); err != nil {
        return err
    }
    
    c.config = newConfig
    c.notifyConfigChange()
    return nil
}
```

### 3. 설정 변경 감시 구현
```go
func (c *Container) WatchConfig() {
    go func() {
        for {
            select {
            case event := <-c.configWatcher.Events:
                if event.Op&fsnotify.Write == fsnotify.Write {
                    log.Println("설정 파일 변경 감지")
                    if err := c.ReloadConfig(); err != nil {
                        log.Printf("설정 리로드 실패: %v", err)
                    }
                }
            case err := <-c.configWatcher.Errors:
                log.Printf("설정 감시 에러: %v", err)
            }
        }
    }()
}
```

## 실행 명령어
```bash
# 기존 Container 코드 백업
cp internal/app/container.go internal/app/container.go.bak

# 의존성 추가
go mod tidy

# 빌드 테스트
make dev-run-direct
```

## 검증 방법
1. 설정 파일 변경 시 즉시 반영 확인
2. 메모리 사용량 모니터링
3. 응답 시간 측정 (10ms 감소 목표)

## 완료 조건
- [x] Container 구조체 확장 완료
- [x] 설정 캐싱 기능 동작
- [x] 핫 리로드 기능 동작
- [x] 기존 기능 정상 동작
- [ ] 단위 테스트 작성

## 진행 상황
- Container에 설정 캐싱 및 핫 리로드 기능이 이미 구현되어 있음
- ConfigLoader, ConfigChangeNotifier가 이미 구현됨
- GetUnifiedConfig() 메서드로 캐싱된 설정 접근 가능
- startConfigWatcher()로 파일 변경 감시 중

## 추가 작업 필요
- ReadConfig() 호출하는 기존 코드들을 Container.GetUnifiedConfig() 사용하도록 변경 필요
- 하지만 이는 phase-2의 핸들러 마이그레이션 작업에서 진행 예정