---
phase: 1
order: 2
source_plan: /docs/refactoring/01-dependency-injection.md
priority: medium
tags: [config, validation, error-handling]
---

# 📌 작업: 설정 검증 로직 구현

## 개요
Container에서 설정 리로드 시 안전성을 보장하기 위한 검증 로직을 구현합니다.

## 구현 내용

### 1. 설정 검증 인터페이스
```go
// configs/validator.go
type Validator interface {
    Validate() error
}

type ValidationError struct {
    Field   string
    Message string
    Value   interface{}
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field '%s': %s (value: %v)", 
        e.Field, e.Message, e.Value)
}
```

### 2. 각 설정 구조체에 Validate 메서드 추가
```go
// configs/apt_proxy_config.go
func (c *APTProxyConfig) Validate() error {
    if c.Enabled && len(c.Mirrors) == 0 {
        return &ValidationError{
            Field:   "mirrors",
            Message: "APT 프록시가 활성화되어 있으나 미러가 설정되지 않음",
            Value:   c.Mirrors,
        }
    }
    
    for _, mirror := range c.Mirrors {
        if _, err := url.Parse(mirror); err != nil {
            return &ValidationError{
                Field:   "mirrors",
                Message: "잘못된 미러 URL",
                Value:   mirror,
            }
        }
    }
    
    return nil
}
```

### 3. 전체 설정 검증
```go
// configs/config.go
func (c *Config) Validate() error {
    validators := []Validator{
        &c.APTProxy,
        &c.MavenProxy,
        &c.NPMProxy,
    }
    
    for _, validator := range validators {
        if err := validator.Validate(); err != nil {
            return err
        }
    }
    
    return nil
}
```

### 4. 설정 변경 알림 시스템
```go
// internal/app/config_notifier.go
type ConfigChangeNotifier struct {
    listeners []func(*configs.Config)
    mu        sync.RWMutex
}

func (n *ConfigChangeNotifier) AddListener(listener func(*configs.Config)) {
    n.mu.Lock()
    defer n.mu.Unlock()
    n.listeners = append(n.listeners, listener)
}

func (n *ConfigChangeNotifier) NotifyChange(config *configs.Config) {
    n.mu.RLock()
    defer n.mu.RUnlock()
    
    for _, listener := range n.listeners {
        go listener(config)
    }
}
```

## 실행 명령어
```bash
# 검증 로직 테스트
go test -v ./configs/...

# 잘못된 설정으로 테스트
echo 'invalid yaml content' > configs/test-invalid.yaml
go run main.go # 검증 실패 확인

# 정상 설정으로 복원
git checkout configs/
```

## 검증 방법
1. 잘못된 설정 파일로 시작 시 적절한 에러 메시지 출력
2. 런타임 중 잘못된 설정 리로드 시 기존 설정 유지
3. 설정 변경 시 관련 서비스에 알림 전달

## 완료 조건
- [x] 각 프록시 설정에 Validate 메서드 구현
- [x] 전체 설정 검증 로직 구현
- [x] 설정 변경 알림 시스템 구현
- [x] 검증 실패 시 적절한 에러 처리
- [x] 단위 테스트 작성 (커버리지 80%+)

## 진행 상황
- ValidationError 타입과 Validatable 인터페이스가 이미 validator.go에 구현됨
- APT, Maven, NPM 프록시 설정에 Validate 메서드가 이미 구현됨
- UnifiedConfig에 레지스트리 설정 검증 추가
- ConfigChangeNotifier가 이미 구현되어 Container에 통합됨
- 모든 검증 테스트 통과