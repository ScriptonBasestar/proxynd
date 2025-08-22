# 💻 ProxyND 개발 가이드

ProxyND 프로젝트에 기여하고 개발 환경을 구축하는 방법을 안내합니다.

## 🚀 빠른 시작

### 필수 요구사항
- **Go 1.23+**: 모든 기능 지원을 위한 최신 버전
- **Make**: 빌드 자동화 도구
- **Docker**: 통합 테스트용 (선택사항)
- **Git**: 소스 코드 관리

### 개발 환경 설정

```bash
# 1. 저장소 클론
git clone https://github.com/yourusername/proxynd.git
cd proxynd

# 2. 개발 환경 자동 설정 (필수!)
make dev-setup    # CONFIG_DIR, STORAGE_DIR 설정 + 샘플 설정 파일 생성

# 3. 개발 서버 실행 (핫 리로드)
make dev-run      # SERVER_PORT=8081로 Air 사용한 개발 서버
```

### 개발 워크플로우

```bash
# 개발 중 빠른 검증 (2-3분)
make quick        # format + lint + 단위 테스트

# PR 제출 전 필수 검사
make pr-check     # format + lint + 커버리지 테스트

# 전체 테스트 실행
make test-all     # 단위 + 통합 테스트 (Docker 필요)
```

## 📚 상세 가이드

### 핵심 문서
- **[기여 가이드](contributing-guide.md)** - 코드 스타일, PR 가이드라인
- **[리팩토링 가이드](refactoring-guide.md)** - 아키텍처 개선 방법
- **[헥사고날 마이그레이션](hexagonal-migration-guide.md)** - 아키텍처 마이그레이션 가이드

### 개발 도구
- **[GitHub Actions](tools/github-actions.md)** - CI/CD 파이프라인 설정
- **[코드 품질 도구](quality/)** - 린팅, 포맷팅, 정적 분석

### 분석 도구
- **[TODO 분석](todo-analysis.md)** - 코드베이스 TODO 현황

## 🏗️ 아키텍처 이해

ProxyND는 **헥사고날 아키텍처 (Ports and Adapters)**를 기반으로 합니다:

```
internal/
├── ports/          # 인터페이스 정의 (인바운드/아웃바운드)
├── usecase/        # 비즈니스 로직 (프레임워크 독립적)
├── adapters/       # 외부 시스템 어댑터
├── app/           # 애플리케이션 진입점 및 Container
└── domain/        # 도메인 엔티티
```

### Container 기반 의존성 주입
- **Thread-safe 싱글톤**: `sync.RWMutex`로 동시성 제어
- **지연 초기화**: 서비스별 팩토리 함수와 캐싱
- **핫 리로드**: fsnotify 기반 설정 파일 변경 감지

## 🧪 테스트 전략

### 4계층 테스트 피라미드
1. **Unit Tests** (빠름, 많음) - 개별 함수/메서드
2. **Contract Tests** (보통) - 인터페이스 계약 검증
3. **Integration Tests** (느림, 적음) - 전체 워크플로우
4. **End-to-End Tests** (매우 느림, 최소) - 실제 패키지 매니저 연동

```bash
# 테스트 실행
make test-unit           # 단위 테스트 (30초)
make test-integration    # 통합 테스트 (2-3분, Docker 필요)
make test-coverage       # HTML 커버리지 리포트
```

## 🔧 디버깅 및 문제 해결

### 개발 서버 로그 확인
```bash
# 개발 서버 실행 중 로그 모니터링
make dev-run

# 다른 터미널에서 테스트
curl http://localhost:8081/healthz
```

### 환경 변수 디버깅
```bash
# 필수 환경 변수 확인
echo $CONFIG_DIR
echo $STORAGE_DIR
echo $SERVER_PORT

# 설정 검증
make config-validate
```

### 일반적인 문제들

**환경 변수 미설정**:
```bash
# Error: fatal error: CONFIG_DIR not set
make dev-setup  # 자동으로 설정 생성
```

**포트 충돌**:
```bash
# Error: port 8081 already in use
SERVER_PORT=8082 make dev-run
```

**Docker 관련 오류**:
```bash
# Docker 없이 단위 테스트만 실행
make test-unit
```

## 📖 참고 자료

- **[전체 문서 구조](../README.md)** - 문서 네비게이션
- **[아키텍처 가이드](../10-architecture/)** - 시스템 설계
- **[테스트 가이드](../40-testing/testing-guide.md)** - 테스트 전략
- **[배포 가이드](../60-deployment/)** - 프로덕션 배포

---

**💡 팁**: 개발 중에는 `make quick`를 자주 실행하여 코드 품질을 유지하세요.  
**🔧 문제 해결**: 이슈가 발생하면 먼저 `make dev-setup`을 실행해보세요.