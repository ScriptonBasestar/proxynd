# ProxyND CLI 도구 (proxyndctl) 요구사항

## 개요

`proxyndctl`은 ProxyND 서버를 관리하기 위한 명령줄 도구입니다.
서버 관리자가 ProxyND 인스턴스를 효율적으로 운영할 수 있도록 다양한 관리 기능을 제공합니다.

## 주요 명령어 목록

### 1. 캐시 관리 (cache)

#### `proxyndctl cache list`
- 캐시된 패키지 목록 조회
- 패키지 타입별 필터링 지원
- 크기, 생성 시간, 마지막 접근 시간 표시

#### `proxyndctl cache clear`
- 캐시 정리 기능
- 전체 또는 특정 패키지 타입별 정리
- 확인 프롬프트 제공

#### `proxyndctl cache size`
- 캐시 사용량 통계
- 패키지 타입별 사용량
- 디스크 사용률 및 여유 공간

### 2. 설정 관리 (config)

#### `proxyndctl config validate`
- 설정 파일 문법 검증
- 필수 항목 존재 확인
- 네트워크 연결성 테스트

#### `proxyndctl config show`
- 현재 설정 표시
- 민감한 정보 마스킹
- 설정 소스 파일 경로 표시

### 3. 서버 상태 (status/health/metrics)

#### `proxyndctl status`
- 서버 전반적인 상태 요약
- 활성 연결 수
- 요청 처리 통계

#### `proxyndctl health`
- 헬스체크 수행
- 업스트림 서버 연결 상태
- 디스크/메모리 사용률

#### `proxyndctl metrics`
- Prometheus 메트릭 조회
- 사용자 친화적 포맷팅
- 시간 범위별 통계

### 4. 사용자 관리 (user)

#### `proxyndctl user add <username>`
- 새 사용자 생성
- 권한 설정 (읽기/쓰기/관리자)
- 패스워드 생성 또는 입력

#### `proxyndctl user delete <username>`
- 사용자 삭제
- 확인 프롬프트
- 연관된 세션 정리

#### `proxyndctl user list`
- 사용자 목록 조회
- 권한 레벨 표시
- 마지막 로그인 시간

### 5. 프록시 테스트 (test)

#### `proxyndctl test apt`
- APT 프록시 기능 테스트
- 샘플 패키지 다운로드
- 캐시 동작 확인

#### `proxyndctl test npm`
- NPM 프록시 기능 테스트
- 레지스트리 연결 확인
- 패키지 검색 테스트

## 사용자 시나리오

### 1. 일상적 관리자 워크플로우

```bash
# 서버 상태 확인
proxyndctl status

# 캐시 사용량 확인
proxyndctl cache size

# 설정 검증
proxyndctl config validate

# 필요시 캐시 정리
proxyndctl cache clear --type npm --older-than 7d
```

### 2. 문제 해결 시나리오

```bash
# 헬스체크 수행
proxyndctl health

# 특정 프록시 테스트
proxyndctl test apt

# 상세 메트릭 확인
proxyndctl metrics --detailed

# 로그 조회 (향후 기능)
proxyndctl logs --tail 100 --level error
```

### 3. 사용자 관리 시나리오

```bash
# 새 개발자 계정 생성
proxyndctl user add developer --role readonly

# 사용자 목록 확인
proxyndctl user list

# 임시 계정 정리
proxyndctl user delete temp-user
```

## 기술적 요구사항

### 1. 프레임워크 선택

**Cobra (github.com/spf13/cobra) 권장**
- Go 표준 CLI 프레임워크
- 자동완성 지원
- 계층적 명령어 구조
- 풍부한 문서화

**대안: urfave/cli**
- 더 가벼운 옵션
- 단순한 구조

### 2. 설정 파일

```yaml
# ~/.proxyndctl/config.yaml
server:
  url: "http://localhost:8080"
  timeout: 30s
  
auth:
  type: "basic"  # basic, token, none
  username: "admin"
  # password는 별도 보안 저장

output:
  format: "table"  # table, json, yaml
  color: true
```

### 3. 인증 방식

- Basic Auth (초기 구현)
- API Token (향후)
- 설정 파일 또는 환경 변수
- 안전한 패스워드 저장

### 4. 출력 포맷

- 기본: 테이블 형태
- JSON: 스크립트 연동용
- YAML: 설정 검토용
- 컬러 출력 지원

## 구현 우선순위

### Phase 1 (MVP)
1. 기본 프로젝트 구조 (`cmd/proxyndctl/`)
2. 서버 연결 및 인증
3. `status`, `health` 명령어
4. `cache size` 명령어

### Phase 2 (핵심 기능)
1. `cache list`, `cache clear`
2. `config validate`, `config show`
3. `metrics` 명령어
4. 기본 출력 포맷팅

### Phase 3 (고급 기능)
1. `user` 관리 명령어
2. `test` 명령어들
3. 자동완성 스크립트
4. man page 생성

### Phase 4 (편의 기능)
1. 대화형 모드
2. 설정 마법사
3. 플러그인 시스템
4. 다국어 지원

## 에러 처리

### 1. 네트워크 오류
- 재시도 메커니즘
- 타임아웃 설정
- 명확한 오류 메시지

### 2. 인증 실패
- 자격증명 재입력 프롬프트
- 토큰 갱신 자동화
- 권한 부족 안내

### 3. 설정 오류
- 설정 파일 경로 안내
- 예시 설정 제공
- 단계별 수정 가이드

## 보안 고려사항

### 1. 자격증명 보호
- 패스워드 평문 저장 금지
- 키체인/credential manager 활용
- 세션 토큰 안전한 저장

### 2. 통신 보안
- HTTPS 기본 사용
- TLS 인증서 검증
- API 키 보호

### 3. 감사 로그
- 명령어 실행 이력
- 중요 작업 로깅
- 사용자 추적

## 테스트 전략

### 1. 단위 테스트
- 각 명령어별 로직 테스트
- 모킹된 API 응답 테스트
- 출력 포맷 검증

### 2. 통합 테스트
- 실제 ProxyND 서버 연동
- E2E 시나리오 테스트
- 다양한 환경에서 테스트

### 3. 사용성 테스트
- 명령어 직관성 검증
- 도움말 완성도 확인
- 오류 메시지 명확성

## 배포 전략

### 1. 바이너리 배포
- GitHub Releases 자동화
- 다중 플랫폼 빌드
- 체크섬 및 서명

### 2. 패키지 매니저
- Homebrew (macOS)
- APT/YUM 패키지
- 스냅/플랫팩

### 3. 설치 스크립트
- 원클릭 설치 스크립트
- 버전 업데이트 지원
- 설정 마이그레이션