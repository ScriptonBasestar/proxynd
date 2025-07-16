---
phase: 1
order: 3
source_plan: /docs/refactoring/01-dependency-injection.md
priority: high
tags: [interfaces, handlers, architecture]
---

# 📌 작업: Handler 인터페이스 정의

## 개요
일관된 핸들러 구조를 위한 인터페이스를 정의하고 팩토리 패턴을 구현합니다.

## 구현 내용

### 1. Handler 인터페이스 정의
```go
// internal/handlers/interfaces.go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "proxynd/internal/app"
)

type Handler interface {
    Handle(c *fiber.Ctx) error
    Name() string
    Type() string
}

type HandlerFactory interface {
    Create(container *app.Container) Handler
}

type Configurable interface {
    Configure(config interface{}) error
}

type Testable interface {
    Test() error
}
```

### 2. 기본 Handler 구현체
```go
// internal/handlers/base_handler.go
package handlers

import (
    "proxynd/internal/app"
)

type BaseHandler struct {
    container *app.Container
    name      string
    proxyType string
}

func NewBaseHandler(container *app.Container, name, proxyType string) *BaseHandler {
    return &BaseHandler{
        container: container,
        name:      name,
        proxyType: proxyType,
    }
}

func (h *BaseHandler) Name() string {
    return h.name
}

func (h *BaseHandler) Type() string {
    return h.proxyType
}

func (h *BaseHandler) GetContainer() *app.Container {
    return h.container
}

func (h *BaseHandler) GetConfig() *configs.Config {
    return h.container.Config()
}
```

### 3. Handler 팩토리 구현
```go
// internal/handlers/factory.go
package handlers

import (
    "fmt"
    "proxynd/internal/app"
)

type HandlerFactory struct {
    container *app.Container
    creators  map[string]func(*app.Container) Handler
}

func NewHandlerFactory(container *app.Container) *HandlerFactory {
    return &HandlerFactory{
        container: container,
        creators:  make(map[string]func(*app.Container) Handler),
    }
}

func (f *HandlerFactory) Register(handlerType string, creator func(*app.Container) Handler) {
    f.creators[handlerType] = creator
}

func (f *HandlerFactory) Create(handlerType string) (Handler, error) {
    creator, exists := f.creators[handlerType]
    if !exists {
        return nil, fmt.Errorf("unknown handler type: %s", handlerType)
    }

    return creator(f.container), nil
}

func (f *HandlerFactory) GetSupportedTypes() []string {
    types := make([]string, 0, len(f.creators))
    for handlerType := range f.creators {
        types = append(types, handlerType)
    }
    return types
}
```

### 4. Handler 레지스트리
```go
// internal/handlers/registry.go
package handlers

import (
    "sync"
    "proxynd/internal/app"
)

type HandlerRegistry struct {
    handlers map[string]Handler
    factory  *HandlerFactory
    mu       sync.RWMutex
}

func NewHandlerRegistry(container *app.Container) *HandlerRegistry {
    return &HandlerRegistry{
        handlers: make(map[string]Handler),
        factory:  NewHandlerFactory(container),
    }
}

func (r *HandlerRegistry) Register(handlerType string, creator func(*app.Container) Handler) {
    r.factory.Register(handlerType, creator)
}

func (r *HandlerRegistry) Get(handlerType string) (Handler, error) {
    r.mu.RLock()
    if handler, exists := r.handlers[handlerType]; exists {
        r.mu.RUnlock()
        return handler, nil
    }
    r.mu.RUnlock()

    r.mu.Lock()
    defer r.mu.Unlock()

    // 더블 체크 락킹
    if handler, exists := r.handlers[handlerType]; exists {
        return handler, nil
    }

    handler, err := r.factory.Create(handlerType)
    if err != nil {
        return nil, err
    }

    r.handlers[handlerType] = handler
    return handler, nil
}
```

## 실행 명령어
```bash
# 인터페이스 파일 생성
mkdir -p internal/handlers
touch internal/handlers/interfaces.go
touch internal/handlers/base_handler.go
touch internal/handlers/factory.go
touch internal/handlers/registry.go

# 빌드 테스트
go build -o /tmp/proxynd ./cmd/proxynd

# 타입 검증
go vet ./internal/handlers/...
```

## 검증 방법
1. 인터페이스 구현 검증
2. 팩토리 패턴 동작 테스트
3. 핸들러 레지스트리 성능 테스트

## 완료 조건
- [ ] Handler 인터페이스 정의 완료
- [ ] BaseHandler 구현 완료
- [ ] HandlerFactory 구현 완료
- [ ] HandlerRegistry 구현 완료
- [ ] 인터페이스 호환성 검증
- [ ] 단위 테스트 작성
