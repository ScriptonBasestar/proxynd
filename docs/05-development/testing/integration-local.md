# ProxyND 통합 테스트

ProxyND 프록시 서버의 통합 테스트 스위트입니다. 실제 패키지 요청을 모사하여 전체 시스템의 동작을 검증합니다.

## 테스트 범위

### 지원하는 프록시 타입
- **NPM**: Node.js 패키지 매니저 프록시 테스트
- **PyPI**: Python 패키지 인덱스 프록시 테스트  
- **APT**: Debian/Ubuntu 패키지 매니저 프록시 테스트
- **Docker**: Docker 레지스트리 v2 프록시 테스트

### 테스트 시나리오
1. **헬스체크**: `/healthz` 엔드포인트 정상 동작 검증
2. **프록시 동작**: 각 패키지 타입별 요청/응답 검증
3. **캐시 동작**: 캐시 히트/미스 성능 비교
4. **인증 시나리오**: BasicAuth 인증 동작 검증
5. **보안 시나리오**: SHA256 해시 검증 테스트
6. **부하 테스트**: 동시 요청 처리 능력 검증
7. **컨텍스트 취소**: 타임아웃 및 취소 처리 검증
8. **대용량 파일 처리**: 10MB+ 파일 스트리밍 검증
9. **캐시 무효화**: PUT/POST 요청 시 캐시 무효화 검증

## 실행 방법

### 기본 테스트 실행
```bash
make test
```

### 자세한 출력으로 테스트
```bash
make test-verbose
```

### 레이스 컨디션 체크
```bash
make test-race
```

### 벤치마크 테스트
```bash
make benchmark
```

### 테스트 커버리지
```bash
make coverage
```

### 전체 테스트 스위트
```bash
make full-test
```

## 테스트 환경

### 환경 변수
- `CONFIG_DIR`: 설정 파일 디렉토리 (테스트용: `./`)
- `STORAGE_DIR`: 캐시 저장 디렉토리 (테스트용: `/tmp/proxynd-test-cache`)

### 테스트 포트
- 메인 테스트 서버: `8082`
- 벤치마크 테스트 서버: `8083`

### 설정 파일
- `global.yaml`: 전역 설정 (examples에서 복사)
- `test_config.yaml`: 테스트 전용 설정

## 테스트 구조

```
tests/integration/
├── integration_test.go     # 메인 테스트 스위트
├── proxy_test.go          # 포괄적인 프록시 플로우 테스트
├── test_config.yaml       # 테스트 설정
├── global.yaml           # 전역 설정 (복사본)
├── Makefile              # 테스트 실행 스크립트
└── README.md             # 이 파일
```

## 주요 테스트 케이스

### NPM 프록시 테스트
- 패키지 메타데이터 요청 (`/proxy/npm/express`)
- 스코프 패키지 요청 (`/proxy/npm/@types/node`)
- tarball 다운로드 (`/proxy/npm/express/-/express-4.18.2.tgz`)

### PyPI 프록시 테스트
- 심플 API 요청 (`/proxy/pip/simple/requests/`)
- 패키지 파일 다운로드 (`/proxy/pip/packages/source/r/requests/requests-2.28.2.tar.gz`)

### APT 프록시 테스트
- Release 파일 요청 (`/proxy/apt/ubuntu/dists/jammy/Release`)
- 패키지 목록 요청 (`/proxy/apt/ubuntu/dists/jammy/main/binary-amd64/Packages.gz`)
- .deb 파일 다운로드 (`/proxy/apt/ubuntu/pool/main/a/apt/apt_2.4.8_amd64.deb`)

### Docker 프록시 테스트
- 레지스트리 v2 체크 (`/proxy/docker/v2/`)
- 매니페스트 요청 (`/proxy/docker/v2/library/nginx/manifests/latest`)
- blob 다운로드 (`/proxy/docker/v2/library/nginx/blobs/sha256:abc123`)

## 성능 벤치마크

벤치마크 테스트는 다음을 측정합니다:
- 요청 처리 처리량 (requests/second)
- 평균 응답 시간
- 메모리 사용량
- 동시 연결 처리 능력

## 문제 해결

### 포트 충돌
테스트 실행 시 포트 충돌이 발생하면 다른 포트로 변경하세요.

### 설정 파일 오류
`global.yaml` 파일이 없으면 examples에서 복사하세요:
```bash
cp ../../examples/global.yaml ./global.yaml
```

### 환경 변수
테스트 실행 전 필요한 환경 변수가 설정되어 있는지 확인하세요.

## 기여 가이드

새로운 테스트 시나리오를 추가할 때:
1. `IntegrationTestSuite`에 새 메서드 추가
2. 테스트 이름은 `Test{FeatureName}Scenario` 형식 사용
3. 각 서브테스트는 독립적으로 실행 가능하도록 작성
4. 외부 의존성 없이 실행 가능하도록 모킹 활용
5. 적절한 assertion과 로깅 추가

## 라이선스

이 테스트 코드는 ProxyND 프로젝트와 동일한 라이선스를 따릅니다.
