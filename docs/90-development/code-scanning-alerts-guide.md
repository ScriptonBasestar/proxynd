# Code Scanning Alerts 처리 가이드

**목적**: GitHub Code Scanning 보안 알림을 효율적으로 처리

---

## 🎯 현재 상황 (306개 alerts)

| Severity | 개수 | 주요 Rule | 처리 방법 |
|----------|------|----------|----------|
| 🟠 High | 38 | go/clear-text-logging | **코드 수정 필수** |
| ⚪ None | 268 | G104, G304, G301, G306 | False positive 검토 |

**주요 문제**:
1. **G104 (131개)**: 에러 처리 누락
2. **G304 (47개)**: 파일 경로 주입 취약점
3. **go/clear-text-logging (38개)**: 민감 정보 로깅
4. **G301/G306 (60개)**: 파일 권한 문제

---

## 📋 우선순위별 처리 계획

### 🔴 Priority 1: High Severity (38개)

**go/clear-text-logging - 민감 정보 로깅**

**문제**: 비밀번호, 토큰, API 키를 로그에 출력

**해결**:
```go
// ❌ 잘못된 예
log.Printf("User logged in: %s with password %s", username, password)

// ✅ 올바른 예
log.Printf("User logged in: %s", username)

// ✅ 마스킹
log.Printf("Token: %s", maskSensitive(token))
```

**자동 검색**:
```bash
# 모든 clear-text-logging 문제 확인
gh api 'repos/ScriptonBasestar/proxynd/code-scanning/alerts?per_page=100&state=open' \
  --jq '.[] | select(.rule.id == "go/clear-text-logging") | {number, file: .most_recent_instance.location.path, line: .most_recent_instance.location.start_line}'
```

---

### 🟡 Priority 2: G304 - 파일 경로 주입 (47개)

**문제**: 외부 입력으로 파일 경로를 받아 사용

**해결**:
```go
// ❌ 잘못된 예
func loadConfig(userPath string) {
    data, _ := os.ReadFile(userPath)  // Path injection!
}

// ✅ 올바른 예
func loadConfig(userPath string) error {
    // 1. 경로 정규화
    cleanPath := filepath.Clean(userPath)

    // 2. 허용 디렉토리 검증
    allowedDir := "/etc/proxynd"
    if !strings.HasPrefix(cleanPath, allowedDir) {
        return fmt.Errorf("invalid path: must be under %s", allowedDir)
    }

    // 3. 심볼릭 링크 방지
    realPath, err := filepath.EvalSymlinks(cleanPath)
    if err != nil {
        return err
    }

    data, err := os.ReadFile(realPath)
    return err
}
```

---

### 🟢 Priority 3: G104 - 에러 처리 (131개)

**대부분 False Positive 가능**

**실제 문제**:
```go
// ❌ 에러 체크 누락 (실제 문제)
file, _ := os.Open("config.yaml")
defer file.Close()  // file이 nil일 수 있음!
```

**의도적 무시** (False Positive):
```go
// ✅ 명시적으로 무시
_, _ = fmt.Fprintf(w, "Hello")  // HTTP 응답 에러는 무시 가능

// ✅ 또는 주석으로 설명
fmt.Fprintf(w, "Hello")  // #nosec G104 - HTTP response error ignored
```

**일괄 검토 필요**:
```bash
# G104 alerts 파일별 그룹화
gh api 'repos/ScriptonBasestar/proxynd/code-scanning/alerts?per_page=100&state=open' \
  --jq '.[] | select(.rule.id == "G104") | .most_recent_instance.location.path' | sort | uniq -c
```

---

### 🔵 Priority 4: 파일 권한 (60개)

**G301: Mkdir 권한 / G306: WriteFile 권한**

**해결**:
```go
// ❌ 너무 관대한 권한
os.MkdirAll("/var/cache", 0777)  // G301
os.WriteFile("config.yaml", data, 0666)  // G306

// ✅ 안전한 권한
os.MkdirAll("/var/cache", 0750)  // rwxr-x---
os.WriteFile("config.yaml", data, 0640)  // rw-r-----
```

**권장 권한**:
- 디렉토리: `0750` (rwxr-x---)
- 설정 파일: `0640` (rw-r-----)
- 실행 파일: `0750` (rwxr-x---)
- 임시 파일: `0600` (rw-------)

---

## 🔧 자동화 도구

### 1. 분석 도구

```bash
# 전체 분석
./scripts/analyze-security-alerts.sh

# 상세 리포트 생성
./scripts/analyze-security-alerts.sh --export
```

### 2. False Positive 처리

많은 alerts가 false positive일 수 있습니다:

**자동 dismiss (주의해서 사용)**:
```bash
# 특정 rule의 모든 alerts dismiss
./scripts/dismiss-alerts-by-rule.sh G104 "Intentionally ignored errors"

# 특정 파일의 alerts dismiss
./scripts/dismiss-alerts-by-file.sh "test/*" "Test files"
```

**수동 dismiss**:
```bash
# 단일 alert dismiss
gh api repos/ScriptonBasestar/proxynd/code-scanning/alerts/123 \
  -X PATCH \
  -f state=dismissed \
  -f dismissed_reason=false_positive \
  -f dismissed_comment="Intentionally ignored error in logging"
```

---

## 📊 처리 워크플로우

### Phase 1: High Priority (1-2일)

```bash
# 1. High severity 확인
./scripts/analyze-security-alerts.sh --export

# 2. clear-text-logging 수정
# - 민감 정보 로깅 제거
# - 마스킹 함수 추가

# 3. 수정 후 재스캔 트리거
git commit -am "fix(security): remove sensitive data from logs"
git push origin master
```

### Phase 2: 파일 경로 검증 (2-3일)

```bash
# 1. G304 alerts 확인
gh api 'repos/ScriptonBasestar/proxynd/code-scanning/alerts' \
  --jq '.[] | select(.rule.id == "G304")'

# 2. 파일별 검토 및 수정
# - filepath.Clean() 추가
# - 경로 검증 로직 추가

# 3. 커밋
git commit -am "fix(security): add file path validation"
```

### Phase 3: False Positive 정리 (1일)

```bash
# 1. G104 에러 처리 검토
# - 실제 문제: 에러 체크 추가
# - False positive: #nosec 또는 dismiss

# 2. 파일 권한 수정
# - 0777 → 0750
# - 0666 → 0640

# 3. 정리
git commit -am "fix(security): improve error handling and file permissions"
```

---

## 🚫 Dismiss 기준

### ✅ Dismiss 가능

- **G104**: HTTP 응답 에러 (`fmt.Fprintf(w, ...)`)
- **G104**: 테스트 코드의 에러
- **G404**: 테스트/예제의 weak random
- **G401**: 비보안 용도 해시 (checksum)
- **go/useless-assignment**: Dead code (이후 삭제 예정)

### ❌ Dismiss 불가

- **go/clear-text-logging**: 실제 민감 정보
- **G304**: 실제 외부 입력 경로
- **G101**: 실제 하드코딩된 credentials
- **G505/G401**: 보안 용도 weak crypto

---

## 📈 진행 상황 추적

### 목표

| Phase | 목표 | 기한 |
|-------|------|------|
| Phase 1 | High severity → 0 | 2일 |
| Phase 2 | G304 → < 10 | 3일 |
| Phase 3 | Total → < 50 | 1일 |

### 측정

```bash
# 현재 상태
./scripts/analyze-security-alerts.sh

# High severity만
gh api 'repos/ScriptonBasestar/proxynd/code-scanning/alerts' \
  --jq '.[] | select(.rule.security_severity_level == "high") | length'
```

---

## 🔗 관련 리소스

- **분석 도구**: `scripts/analyze-security-alerts.sh`
- **GitHub Security**: https://github.com/ScriptonBasestar/proxynd/security/code-scanning
- **gosec 룰**: https://github.com/securego/gosec#available-rules
- **CodeQL Go 쿼리**: https://codeql.github.com/codeql-query-help/go/

---

## 💡 팁

1. **우선순위**: High severity → Medium → Low
2. **배치 처리**: 같은 종류의 문제를 한번에
3. **False Positive**: 의심스러우면 코드 리뷰
4. **자동화**: 반복 패턴은 스크립트로
5. **재스캔**: 수정 후 CI에서 자동 재스캔

---

**마지막 업데이트**: 2025-11-24
