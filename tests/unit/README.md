# ProxyND 단위 테스트

ProxyND 프로젝트의 개별 컴포넌트들에 대한 단위 테스트 모음입니다.

## 테스트 구조

### 파일 구성
```
tests/unit/
├── cache_test.go       # 캐시 시스템 테스트
├── middleware_test.go  # 미들웨어 테스트
├── router_test.go      # 라우터 테스트
├── helpers_test.go     # 헬퍼 함수 테스트
├── Makefile           # 테스트 실행 스크립트
└── README.md          # 이 파일
```

## 테스트 범위

### 1. 캐시 시스템 테스트 (`cache_test.go`)
- **FileSystemBackend**: 파일시스템 캐시 백엔드
  - Put/Get 동작
  - 존재 여부 확인 (Exists)
  - 삭제 (Delete)
  - TTL 만료 처리
  - 캐시 크기 관리
  - 전체 삭제 (Clear)
  
- **CacheManager**: 캐시 매니저
  - 캐시 CRUD 동작
  - 통계 수집 (히트/미스)
  - 캐시 경로 생성
  - 캐시 정리

- **CacheEviction**: 캐시 제거 정책
  - LRU 정책 테스트
  - TTL 기반 제거
  - 크기 기반 제거
  - 제거 후보 선택

### 2. 미들웨어 테스트 (`middleware_test.go`)
- **IPFilterMiddleware**: IP 필터링
  - 특정 IP 허용/차단
  - CIDR 범위 기반 필터링
  - 기본 허용/차단 모드

- **BasicAuthMiddleware**: 기본 인증
  - 올바른 인증 정보 처리
  - 잘못된 인증 정보 처리
  - 인증 헤더 없는 요청
  - 다양한 인증 형식

- **PermissionMiddleware**: 권한 관리
  - 읽기/쓰기/삭제 권한
  - 사용자별 권한 설정
  - 기본 권한 적용

- **PackageFilterMiddleware**: 패키지 필터링
  - 허용된 패키지 목록
  - 와일드카드 패턴 매칭
  - 패키지 타입별 필터링

- **SecurityMiddleware**: 보안 검증
  - SHA256 해시 검증
  - 다양한 해시 헤더 형식
  - 해시 불일치 처리

### 3. 라우터 테스트 (`router_test.go`)
- **HealthRouter**: 헬스체크
  - 정상 상태 응답
  - 실패 상태 응답
  - 환경 변수 검증

- **ProxyRouter**: 프록시 라우팅
  - 다양한 프록시 타입 (npm, pip, apt, docker)
  - URL 파라미터 추출
  - 스코프 패키지 처리
  - 깊은 경로 처리

- **ParameterExtraction**: 파라미터 추출
  - URL 인코딩 처리
  - 쿼리 파라미터
  - 특수 문자 처리

- **MiddlewareChain**: 미들웨어 체인
  - 미들웨어 실행 순서
  - 로컬 변수 전달

### 4. 헬퍼 함수 테스트 (`helpers_test.go`)
- **YAMLHelper**: YAML 처리
  - YAML 읽기/쓰기
  - 복잡한 데이터 구조
  - 오류 처리

- **URLHelper**: URL 처리
  - URL 경로 결합
  - 경로 정리
  - 패키지 이름 파싱
  - URL 유효성 검증
  - URL 인코딩

## 실행 방법

### 기본 테스트 실행
```bash
make test
```

### 개별 테스트 모듈 실행
```bash
make test-cache       # 캐시 테스트만
make test-middleware  # 미들웨어 테스트만
make test-router      # 라우터 테스트만
make test-helpers     # 헬퍼 테스트만
```

### 특정 테스트 함수 실행
```bash
make test-specific TEST=TestCacheManager
```

### 커버리지 측정
```bash
make coverage         # HTML 리포트 생성
make coverage-detail  # 함수별 상세 커버리지
```

### 성능 테스트
```bash
make benchmark        # 벤치마크 실행
make profile-cpu      # CPU 프로파일링
make profile-mem      # 메모리 프로파일링
```

### CI/CD 테스트
```bash
make ci-test          # 전체 검증 (레이스 컨디션 + 커버리지)
```

## 테스트 작성 가이드라인

### 1. 테스트 함수 명명 규칙
```go
func TestComponentName_FunctionName(t *testing.T) {
    // 기본 테스트
}

func TestComponentName_FunctionName_SpecificCase(t *testing.T) {
    // 특정 케이스 테스트
}
```

### 2. 서브테스트 활용
```go
func TestComponent(t *testing.T) {
    t.Run("Success Case", func(t *testing.T) {
        // 성공 케이스
    })
    
    t.Run("Error Case", func(t *testing.T) {
        // 오류 케이스
    })
}
```

### 3. 테스트 데이터 관리
- 임시 디렉토리 사용 (`os.MkdirTemp`)
- 테스트 완료 후 정리 (`defer os.RemoveAll`)
- 격리된 환경 구성

### 4. 어설션 활용
```go
// 기본 검증
assert.Equal(t, expected, actual)
assert.True(t, condition)
assert.NoError(t, err)

// 필수 검증 (실패 시 즉시 중단)
require.NoError(t, err)
require.NotNil(t, object)
```

### 5. 모킹과 의존성 주입
- 외부 의존성 최소화
- 인터페이스 기반 모킹
- 테스트용 설정 사용

## 성능 벤치마크

벤치마크 테스트는 다음을 측정합니다:
- 캐시 조회 성능
- 미들웨어 처리 시간
- URL 파싱 성능
- YAML 처리 성능

### 벤치마크 작성 예시
```go
func BenchmarkCacheGet(b *testing.B) {
    // 설정...
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // 측정할 코드
    }
}
```

## 커버리지 목표

- **전체 커버리지**: 80% 이상
- **핵심 컴포넌트**: 90% 이상
- **미들웨어**: 85% 이상
- **헬퍼 함수**: 80% 이상

## 문제 해결

### 테스트 실패 시
1. 테스트 로그 확인
2. 환경 변수 설정 확인
3. 임시 파일/디렉토리 권한 확인
4. 포트 충돌 확인

### 성능 문제 시
1. 벤치마크 결과 비교
2. CPU/메모리 프로파일 분석
3. 병목 지점 식별
4. 최적화 적용

### 커버리지 향상
1. 미테스트 경로 식별
2. 엣지 케이스 추가
3. 오류 상황 테스트
4. 통합 시나리오 보강

## 기여 가이드

새로운 테스트 추가 시:
1. 적절한 파일에 테스트 추가
2. 명명 규칙 준수
3. 문서화 업데이트
4. 커버리지 확인
5. CI 테스트 통과 확인

## 라이선스

이 테스트 코드는 ProxyND 프로젝트와 동일한 라이선스를 따릅니다.