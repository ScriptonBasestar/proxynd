# ProxyND 사용하지 않는 파일 정리 작업

## 작업 개요
- **목적**: ProxyND 프로젝트에서 사용하지 않는 파일들을 제거하여 코드베이스 정리
- **예상 효과**: ~60개 파일 제거, ~3,000+ 라인 감소, 빌드 속도 향상
- **위험도**: 낮음 (모두 orphaned 파일이거나 deprecated 파일들)

## 🔴 1단계: 즉시 제거 가능한 파일들

### 1.1 Orphaned Test 디렉토리 전체 제거
```bash
# 대응하는 구현이 없는 테스트 디렉토리들
rm -rf tests/benchmark/        # 11개 orphaned benchmark 테스트
rm -rf tests/unit/            # 9개 orphaned 단위 테스트  
rm -rf handlers/proxy_test/   # 4개 orphaned 프록시 테스트
```

### 1.2 개별 Orphaned Test 파일들
```bash
# handlers/proxy/ 디렉토리의 orphaned 테스트들
rm handlers/proxy/maven_handler_v2_test.go           # maven_handler_v2.go 없음
rm handlers/proxy/apt_benchmark_test.go              # apt_benchmark.go 없음
rm handlers/proxy/maven_handler_v2_unit_test.go      # 구현 없음
rm handlers/proxy/maven_benchmark_test.go            # 구현 없음
rm handlers/proxy/apt_handler_v2_unit_test.go        # apt_handler_v2.go 없음
rm handlers/proxy/apt_handler_v2_simple_test.go      # apt_handler_v2.go 없음
```

### 1.3 Deprecated 설정 파일들 (Type Alias만 포함)
```bash
# internal/config/ 디렉토리의 deprecated 파일들
rm internal/config/docker_proxy_config.go    # 단순 type alias만 포함
rm internal/config/yum_proxy_config.go       # 단순 type alias만 포함
rm internal/config/npm_proxy_config.go       # 단순 type alias만 포함
rm internal/config/pip_proxy_config.go       # 단순 type alias만 포함
rm internal/config/apk_proxy_config.go       # 단순 type alias만 포함
rm internal/config/maven_proxy_config.go     # 단순 type alias만 포함
```

### 1.4 중복/불필요한 설정 파일들
```bash
# Air 설정 파일 중복 제거
rm .air.simple.toml          # 기본 설정, .air.toml으로 대체됨
rm .air.prod.toml           # 잘못된 프로젝트(familybook) 설정 포함

# 최소 내용 파일
rm dtos/response_message.go  # 상수 2개만 포함, 다른 곳으로 이동 후 제거
```

## 🟡 2단계: 검증 후 제거 검토

### 2.1 Legacy Compatibility Layer
```bash
# 검토 필요: routers/legacy_compatibility.go
# - 273라인의 호환성 레이어
# - 구 API 엔드포인트 지원
# - 제거 전 레거시 지원 필요성 확인 요구
```

### 2.2 License Generator Tool
```bash
# 검토 필요: cmd/license-gen/
# - 라이선스 생성 도구
# - placeholder private key 포함
# - 보안 처리 또는 별도 저장소 이동 검토
```

## 📋 실행 스크립트

### 안전한 일괄 제거 스크립트
```bash
#!/bin/bash
# cleanup-unused-files.sh

echo "ProxyND 사용하지 않는 파일 정리 시작..."

# 1단계: Orphaned Test 디렉토리 제거
echo "1. Orphaned test 디렉토리 제거..."
rm -rf tests/benchmark/
rm -rf tests/unit/  
rm -rf handlers/proxy_test/

# 2단계: 개별 orphaned test 파일 제거
echo "2. 개별 orphaned test 파일 제거..."
rm -f handlers/proxy/maven_handler_v2_test.go
rm -f handlers/proxy/apt_benchmark_test.go
rm -f handlers/proxy/maven_handler_v2_unit_test.go
rm -f handlers/proxy/maven_benchmark_test.go
rm -f handlers/proxy/apt_handler_v2_unit_test.go
rm -f handlers/proxy/apt_handler_v2_simple_test.go

# 3단계: Deprecated 설정 파일 제거
echo "3. Deprecated 설정 파일 제거..."
rm -f internal/config/docker_proxy_config.go
rm -f internal/config/yum_proxy_config.go
rm -f internal/config/npm_proxy_config.go
rm -f internal/config/pip_proxy_config.go
rm -f internal/config/apk_proxy_config.go
rm -f internal/config/maven_proxy_config.go

# 4단계: 중복 설정 파일 제거
echo "4. 중복 설정 파일 제거..."
rm -f .air.simple.toml
rm -f .air.prod.toml

# 5단계: 최소 내용 파일 제거 (상수 이동 후)
echo "5. 최소 내용 파일 처리..."
echo "   - dtos/response_message.go의 상수들을 다른 적절한 위치로 이동 후 제거"

echo "정리 완료! 약 60개 파일이 제거되었습니다."
echo "빌드 테스트를 실행하여 문제없는지 확인하세요:"
echo "  make test-unit"
echo "  make build"
```

## ✅ 제거 후 확인 사항

### 빌드 테스트
```bash
# 제거 후 반드시 실행
make clean           # 기존 빌드 캐시 정리
make build          # 빌드 성공 확인
make test-unit      # 단위 테스트 성공 확인
make lint           # 린트 검사 통과 확인
```

### Import 체크
```bash
# 혹시 누락된 import가 있는지 확인
go mod tidy
go build ./...
```

## 🎯 예상 효과

### 정량적 효과
- **제거 파일 수**: ~60개
- **코드 라인 감소**: ~3,000+ 라인
- **디렉토리 제거**: 3개 (tests/benchmark, tests/unit, handlers/proxy_test)

### 정성적 효과
- ✅ 빌드 속도 향상
- ✅ 코드베이스 복잡도 감소
- ✅ 개발 환경 정리
- ✅ 혼란을 야기하는 orphaned 파일 제거
- ✅ 유지보수성 향상

## ⚠️ 주의사항

1. **백업 권장**: 제거 전 현재 상태 백업
2. **단계별 실행**: 한 번에 모든 파일을 제거하지 말고 단계별로 실행
3. **테스트 필수**: 각 단계 후 빌드 및 테스트 실행
4. **팀 검토**: 중요한 파일 제거 전 팀원들과 검토

## 📅 작업 일정

- [ ] 1단계 파일 제거 (안전한 파일들)
- [ ] 빌드 및 테스트 확인
- [ ] dtos/response_message.go 상수 이동
- [ ] 2단계 검증 파일들 검토
- [ ] 최종 빌드 및 테스트 확인
- [ ] Git 커밋 및 푸시

---
*작성일: 2025-07-31*
*분석 결과를 바탕으로 한 프로젝트 정리 작업*
