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

## 🏗️ 주요 기능
- 멀티 프록시 타입 지원 (Maven, NPM, APT, Docker, PyPI, YUM, APK)
- 고성능 Fiber v2 웹 프레임워크 기반
- S3 호환 캐시 백엔드 지원
- OAuth2 인증 및 JWT 토큰 지원
- Prometheus 메트릭 수집
- 구조화된 로깅 (JSON)
- 핫 리로드 설정
- Kubernetes/Helm 차트 제공
- CLI 관리 도구 (proxyndctl)

## 🛠️ 기술 스택
- **언어**: Go 1.23+
- **웹 프레임워크**: Fiber v2
- **설정**: YAML/TOML
- **캐시**: 파일시스템/S3
- **모니터링**: Prometheus, 구조화 로깅
- **배포**: Docker, Kubernetes, Systemd

## 🤝 기여하기
[개발 가이드](docs/05-development/README.md)를 참조하세요.

## 📄 라이선스
ProxyND는 듀얼 라이선스로 제공됩니다:
- **오픈소스**: [AGPL-3.0](LICENSE) - 소스 공개 의무가 있는 무료 라이선스
- **상용**: [Commercial License](LICENSE-COMMERCIAL.md) - 독점 사용 및 엔터프라이즈 기능

자세한 내용은 [LICENSING.md](LICENSING.md)를 참조하세요.

---

> ℹ️ 상세한 설정 및 사용법은 [문서](docs/INDEX.md)를 참조하세요.