# 🏗️ ProxyND 아키텍처 가이드

ProxyND는 헥사고날 아키텍처를 기반으로 한 고성능 멀티 패키지 매니저 프록시입니다. 포트와 어댑터 패턴으로 외부 의존성을 분리하고, Container 패턴으로 의존성을 관리합니다.

## 📋 아키텍처 개요

ProxyND는 다음과 같은 핵심 아키텍처 패턴을 사용합니다:

- **헥사고날 아키텍처**: 포트와 어댑터 패턴으로 외부 의존성 분리
- **Container 패턴**: Thread-safe 싱글톤으로 의존성 관리
- **멀티 프록시**: 7개 패키지 매니저 동시 지원
- **팩토리 패턴**: 프록시 서비스 동적 생성

## 🔗 주요 문서

### 핵심 아키텍처
- **[헥사고날 아키텍처](hexagonal-architecture.md)** - 전체 구조 상세 (포트와 어댑터)
- **[Container 의존성 주입](container-dependency-injection.md)** - DI 패턴 및 서비스 관리
- **[프록시 서비스 팩토리](proxy-service-factory.md)** - 팩토리 패턴 구현
- **[멀티 프록시 아키텍처](multi-proxy-architecture.md)** - 확장성 설계

### 설계 결정 기록
- **[ADR 문서](adr/README.md)** - 모든 설계 결정의 배경과 근거

## 🚀 빠른 시작

### 개발자를 위한 아키텍처 이해 순서
1. **[헥사고날 아키텍처](hexagonal-architecture.md)** 먼저 읽기 - 전체 구조 파악
2. **[Container 패턴](container-dependency-injection.md)** 으로 DI 이해
3. **[ADR 문서](adr/README.md)** 에서 설계 배경 파악
4. **코드베이스 탐색**: `internal/` 디렉터리 구조와 매핑

### 아키텍처 계층 구조
```
Adapters (HTTP, DB, Cache) → Ports (Interfaces) → Usecase (Business Logic) → Domain (Entities)
```

## 🔍 실제 구현

### 디렉터리 매핑
| 아키텍처 계층 | 코드 위치 | 역할 |
|---------------|-----------|------|
| **Domain** | `internal/domain/` | 비즈니스 엔티티 및 규칙 |
| **Usecase** | `internal/usecase/` | 애플리케이션 로직 |
| **Ports** | `internal/ports/` | 인터페이스 정의 |
| **Adapters** | `internal/adapters/` | 외부 시스템 연동 |
| **Container** | `internal/app/container.go` | 의존성 주입 |

### 실제 코드 예시
- Container 구현: `internal/app/container.go:45`
- 프록시 팩토리: `internal/containerhandlers/`
- 헥사고날 구조: `internal/` 하위 계층별 분리

## 📊 성능 특성

### 아키텍처 장점
- **확장성**: 새로운 패키지 매니저 쉽게 추가
- **테스트 용이성**: 계층별 독립적 테스트 가능
- **유지보수성**: 관심사 분리로 코드 변경 영향 최소화
- **성능**: Container 기반 효율적 인스턴스 관리

### 성능 벤치마크
자세한 성능 정보는 [성능 벤치마크](../80-reference/performance-benchmarks.md)를 참조하세요.

---

**다음 단계**: [헥사고날 아키텍처](hexagonal-architecture.md)를 읽고 전체 구조를 이해하세요.
