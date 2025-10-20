# 프록신디 (ProxyND)

Go로 작성된 고성능 패키지 매니저 프록시/미러 서버

## 🚀 빠른 시작

### 1. 개발 환경 설정
```bash
make dev-prepare  # 의존성 설치
make dev-setup    # 설정 디렉토리 준비
make dev-run      # 개발 서버 실행
```

### 2. Docker 실행
```bash
make docker-build
make docker-run
```

## 📦 지원하는 패키지 매니저
- **Maven** (Java/Kotlin/Scala)
- **NPM** (Node.js)
- **APT** (Ubuntu/Debian)
- **Docker Registry**
- **PyPI** (Python)
- **YUM** (RedHat/CentOS)
- **APK** (Alpine Linux)

## 📚 문서
- [📖 전체 문서](docs/INDEX.md)
- [🚀 시작하기](docs/01-getting-started/README.md)
- [⚙️ 설정 가이드](docs/03-configuration/README.md)
- [🔧 개발 가이드](docs/05-development/README.md)

## 🤖 AI 협업 가이드
- [🛡️ AI 협업 가드레일](CLAUDE.md) - **대규모 리팩토링 필수 가이드**
- [🤝 기여 가이드](CONTRIBUTING.md) - 일반 기여 및 AI 지원 개발
- [📋 리팩토링 체크리스트](REFACTORING_CHECKLIST.md) - 단계별 마이그레이션 절차
- [🧪 테스트 전략](TESTING.md) - 4계층 테스트 아키텍처

## 🔨 빌드

### 개발 빌드
```bash
make build                    # 빌드: ./tmp/bin/proxynd
./tmp/bin/proxynd --version  # 버전 확인
```

### 멀티플랫폼 빌드
```bash
make build-all               # 빌드: ./dist/proxynd-{platform}-{arch}
```

### 빌드 출력 위치
- **개발 빌드**: `tmp/bin/proxynd` (gitignored)
- **릴리즈 빌드**: `dist/` (gitignored)
- **Air 개발 서버**: `tmp/bin/proxynd` (hot reload)

모든 빌드 출력은 자동으로 git에서 제외되므로 commit 걱정 없이 개발할 수 있습니다.

## ⚙️ Configuration

1. Copy example configuration:
```bash
cp examples/config.minimal.yaml config.yaml
```

2. Edit `config.yaml` for your needs

3. Build and run ProxyND:
```bash
make build
./tmp/bin/proxynd --config config.yaml
```

### Configuration Files Location Priority:
1. `--config` flag
2. `./config.yaml` (current directory)
3. `~/.config/proxynd/config.yaml` (user config)
4. `/etc/proxynd/config.yaml` (system config)

### Environment Variables

ProxyND requires the following environment variables:

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `CONFIG_DIR` | Configuration directory path | **Yes** | - |
| `STORAGE_DIR` | Cache storage directory path | **Yes** | - |
| `SERVER_PORT` | Server port | No | 8080 |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | No | info |
| `LOG_FORMAT` | Log format (json, text) | No | json |

**Important**: The application will exit with a fatal error if `CONFIG_DIR` or `STORAGE_DIR` are not set.

Example:
```bash
export CONFIG_DIR=/etc/proxynd
export STORAGE_DIR=/var/lib/proxynd
./tmp/bin/proxynd
```

Or use a `.env` file:
```bash
CONFIG_DIR=./config
STORAGE_DIR=./storage
SERVER_PORT=8080
```

## 🏗️ 주요 기능
- 멀티 프록시 타입 지원 (Maven, NPM, APT, Docker, PyPI, YUM, APK)
- 고성능 Fiber v2 웹 프레임워크 기반
- S3 호환 캐시 백엔드 지원
- OAuth2 인증 및 JWT 토큰 지원
- Prometheus 메트릭 수집
- 구조화된 로깅 (JSON)
- 핫 리로드 설정
- Kubernetes/Helm 차트 제공
- 강력한 CLI 관리 도구 (proxyndctl)

## 🏛️ 아키텍처

ProxyND는 **헥사고널 아키텍처 (Ports and Adapters)** 패턴을 채택하여 유지보수성과 테스트 가능성을 극대화했습니다:

### 계층 구조
```
internal/
├── ports/          # 인바운드/아웃바운드 포트 정의
│   ├── http.go     # HTTP 서버 인터페이스
│   ├── pm.go       # 패키지 매니저 인터페이스
│   ├── cache.go    # 캐시 인터페이스
│   ├── auth.go     # 인증 인터페이스
│   └── observability.go # 모니터링/로깅 인터페이스
├── usecase/        # 비즈니스 로직 (프레임워크 독립적)
│   ├── proxy_service.go    # 프록시 핵심 로직
│   ├── cache_strategy.go   # 캐시 전략 관리
│   └── health.go          # 헬스체크 로직
└── adapters/       # 외부 시스템 어댑터
    └── http/fiber/ # Fiber 웹 프레임워크 어댑터
```

### 설계 원칙
- **단방향 의존성**: `adapters → ports → usecase → domain`
- **프레임워크 독립성**: 비즈니스 로직이 웹 프레임워크에 의존하지 않음
- **테스트 용이성**: 모든 외부 의존성을 인터페이스로 추상화
- **확장 가능성**: 새로운 패키지 매니저나 캐시 백엔드 쉽게 추가

## 🛠️ 기술 스택

상세한 기술 스택 정보는 [TECH_STACK.md](TECH_STACK.md)를 참조하세요.

- **언어**: Go 1.23+ (toolchain 1.24.4)
- **아키텍처**: Hexagonal Architecture (Ports and Adapters)
- **웹 프레임워크**: Fiber v2.52.9
- **설정 관리**: Viper + YAML
- **캐시 백엔드**: 파일시스템/S3 (멀티티어)
- **인증**: OAuth2 (GitHub/GitLab/Google) + JWT + MFA
- **모니터링**: Prometheus + Grafana + 구조화 로깅
- **배포**: Docker (멀티아키텍처) + Kubernetes/Helm + Terraform

## 🔧 CLI 관리 도구

ProxyND는 강력한 CLI 관리 도구 `proxyndctl`을 제공합니다. 모든 관리 작업을 명령줄에서 수행할 수 있습니다.

### CLI 기능

- **캐시 관리**: 캐시 목록 조회, 정리, 통계 확인
- **설정 관리**: 설정 검증, 표시, 리로드
- **사용자 관리**: 사용자 추가/삭제/목록 조회
- **프록시 테스트**: 연결성 테스트 및 기능 검증
- **서버 상태**: 실시간 상태 모니터링
- **Maven 확장**: 인덱스 관리 및 증분 백업

### 사용 예시

```bash
# 캐시 관리
proxyndctl cache list
proxyndctl cache clear --type npm
proxyndctl cache size

# 설정 관리
proxyndctl config validate
proxyndctl config show

# 사용자 관리
proxyndctl user list
proxyndctl user add testuser

# 프록시 테스트
proxyndctl test all
proxyndctl test --proxy maven

# 서버 상태
proxyndctl status

# Maven 확장 기능
proxyndctl maven-index build    # 검색 인덱스 생성
proxyndctl maven-backup create  # 증분 백업
```

### API 엔드포인트

모든 CLI 기능은 REST API로도 제공됩니다:

- **캐시 API**: `/api/cache/*`
- **설정 API**: `/api/config/*`
- **사용자 API**: `/api/user/*`
- **테스트 API**: `/api/test/*`
- **상태 API**: `/api/status/*`

자세한 내용은 [API 문서](docs/API_ENDPOINTS.md)를 참조하세요.

### API 테스트

API 엔드포인트의 정상 작동을 확인할 수 있습니다:

```bash
# 모든 API 엔드포인트 검증
make verify-api

# JSON 형식으로 상세 결과 출력
make verify-api-json

# 수동 검증 스크립트 실행
./scripts/verify-api-endpoints.sh http://localhost:8080 --verbose
```

## 🤝 기여하기
[개발 가이드](docs/05-development/README.md)를 참조하세요.

## 📄 라이선스
ProxyND는 듀얼 라이선스로 제공됩니다:
- **오픈소스**: [AGPL-3.0](LICENSE) - 소스 공개 의무가 있는 무료 라이선스
- **상용**: [Commercial License](LICENSE-COMMERCIAL.md) - 독점 사용 및 엔터프라이즈 기능

자세한 내용은 [LICENSING.md](LICENSING.md)를 참조하세요.

---

> ℹ️ 상세한 설정 및 사용법은 [문서](docs/INDEX.md)를 참조하세요.
