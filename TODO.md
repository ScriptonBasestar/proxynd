# ✅ TODO.md

ProxyND 프로젝트의 **최초 릴리스(MVP)** 에 반드시 포함되어야 할 기능들을 정리한 목록입니다.  
이 문서는 Artifactory, Nexus3 등 상용 레지스트리의 핵심 기능들을 참고하였으며,  
**경량 서버 구조 + 실사용 최적화**를 기준으로 작성되었습니다.

---

## 🧭 핵심 구조

- [x] 프록시 요청 라우터 구성 (`/proxy/:type/*path`)
  - `type`: `apt`, `pip`, `npm`, `docker` 등 레지스트리 유형별로 라우팅
  - `path`: 원본 패키지 경로 그대로 유지
- [x] 프록시 정책 처리 미들웨어
  - 캐시 hit/miss 판단
  - 인증/허가 체크
  - 요청 허용/차단 정책 적용

---

## 📦 지원 레지스트리 타입 (1차 릴리스)

- [x] `apt` (Ubuntu, Debian 등 APT 프록시 처리)
- [x] `pip` (PyPI)
- [x] `npm` (npmjs)
- [x] `docker` (Docker registry v2)

> 단일 프레임워크 안에서 멀티 포맷 프록시 지원

---

## ⚙️ 캐시 관리

- [x] 캐시 스토리지 인터페이스 정의 (`CacheBackend`)
  - 공통 read/write/delete/isHit API 제공
- [ ] 로컬 파일 시스템 캐시 드라이버
- [ ] S3 캐시 드라이버 (버킷 설정 가능)
- [ ] TTL/만료 정책: LRU, max size 기반 정리 기능

---

## 🔐 인증/인가/접근 제어

- [ ] IP 화이트리스트 + CIDR 기반 필터
- [ ] BasicAuth (설정파일 기반)
- [ ] 사용자별 권한 정책 (읽기/쓰기/삭제 허용)
- [ ] 허용된 패키지 목록 기반 필터 (AllowList)

---

## 🛡️ 보안 및 검증

- [ ] SHA256 해시 검증 (패키지 메타와 비교)
- [ ] 요청 응답 기록 로깅 (`access.log`)
- [ ] 패키지 검증 실패 시 차단/알림

---

## 🧪 테스트 프레임워크

- [ ] 통합 테스트: 실제 패키지 요청을 모사하는 시나리오 테스트
- [ ] 단위 테스트: 라우팅, 캐시 hit, 인증 등
- [ ] 로컬 e2e 테스트용 Docker Compose 환경 제공

---

## 📚 설정 및 구성 관리

- [ ] `config.yaml` 구조 정의
  - 서버 (포트, TLS)
  - 지원 레지스트리 (활성화 여부, upstream 주소)
  - 인증/접근 정책
  - 캐시 설정
- [ ] 환경 변수 오버라이드 지원
- [ ] 핫리로드 지원 (`SIGHUP`)

---

## 🧰 운영 및 관측

- [ ] `/metrics` 엔드포인트 (Prometheus 포맷)
  - 요청 수, 캐시 hit/miss, 인증 실패 수
- [ ] `/healthz` 상태 체크 API
- [ ] structured logging (JSON + leveled log)

---

## 🐳 실행 및 배포

- [ ] 다중 아키텍처 Docker 이미지 빌드 (`amd64`, `arm64`)
- [ ] CI 파이프라인에서 자동 테스트 & 빌드
- [ ] Helm chart or systemd 서비스 템플릿 제공

---

## 📄 문서화

- [ ] README.md (기능, 설치법, 아키텍처 설명)
- [ ] `conf/config.example.yaml`
- [ ] REF.md: APT/PIP/NPM 패키지 흐름 및 헤더 구조 설명
- [ ] 패키지별 설정 예시 (APT sources.list 등)

---

> 이 목록은 **모듈화**를 전제로 하며, 각 기능은 독립적으로 설계되어야 유지보수가 용이합니다.
> 모든 항목은 커밋 단위 또는 PR 단위로 나누어 개발하고, `CHANGELOG.md`에 기록하세요.
