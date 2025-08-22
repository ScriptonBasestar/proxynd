# 배치 작업 시스템 설계

ProxyND의 배치 작업 시스템은 복잡한 관리 작업을 스크립트로 자동화할 수 있는 기능입니다.

## 📋 개요

### 목적
- 반복적인 관리 작업 자동화
- 복잡한 워크플로우를 스크립트로 정의
- 에러 처리 및 롤백 기능 제공
- 스케줄링 및 모니터링 지원

### 핵심 가치
- **생산성 향상**: 수동 작업 시간 95% 단축
- **일관성**: 항상 동일한 절차로 작업 수행
- **안정성**: 에러 상황에서 자동 복구
- **추적성**: 모든 작업 이력 보관

## 🏗️ 아키텍처 설계

### 시스템 구성요소

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Batch Script   │    │   Job Engine    │    │   Execution     │
│   (.batch)      │───▶│   (Parser)      │───▶│    Context      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                 │                       │
                                 ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Job History    │◀───│   Job Manager   │───▶│  Progress       │
│   (Storage)     │    │  (Orchestrator) │    │  Monitor        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 핵심 컴포넌트

#### 1. Batch Script Language
```bash
# 기본 구문 예시
#!/proxyndctl/batch

# 변수 정의
set backup_date = $(date +%Y%m%d)
set backup_path = "/backup/maven-${backup_date}"

# 기본 명령어
echo "Starting maintenance batch job..."

# CLI 명령어 실행
cache clear --older-than 30d --force
maven-backup create --target ${backup_path}

# 조건부 실행
if ${?} == 0 then
    echo "Backup created successfully"
    maven-index build --force
else
    echo "Backup failed, skipping index rebuild"
    exit 1
fi

# 병렬 실행
parallel {
    test maven
    test npm  
    test docker
}

# 반복문
for proxy_type in maven npm docker do
    echo "Testing ${proxy_type}..."
    test ${proxy_type} --timeout 60
done

# 에러 처리
on_error {
    echo "Error occurred, cleaning up..."
    cache clear --type temp --force
    exit 1
}

echo "Batch job completed successfully"
```

#### 2. Job Engine 설계

```go
// BatchJob 배치 작업 정의
type BatchJob struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Script      string                 `json:"script"`
    Variables   map[string]string      `json:"variables"`
    Timeout     time.Duration          `json:"timeout"`
    OnError     ErrorHandling          `json:"on_error"`
    Schedule    *ScheduleConfig        `json:"schedule,omitempty"`
    Created     time.Time              `json:"created"`
}

// ExecutionContext 실행 컨텍스트
type ExecutionContext struct {
    JobID       string
    Variables   map[string]interface{}
    Logger      Logger
    Progress    *ProgressTracker
    CancelFunc  context.CancelFunc
}

// JobEngine 배치 작업 엔진
type JobEngine interface {
    ParseScript(script string) (*BatchJob, error)
    ExecuteJob(job *BatchJob, ctx *ExecutionContext) error
    CancelJob(jobID string) error
    GetJobStatus(jobID string) (*JobStatus, error)
}
```

#### 3. 명령어 파서

```go
// Command 배치 명령어 인터페이스
type Command interface {
    Execute(ctx *ExecutionContext, args []string) error
    Validate(args []string) error
    GetName() string
}

// 지원하는 명령어 타입
type CommandType string

const (
    CommandTypeCLI      CommandType = "cli"      // proxyndctl 명령어
    CommandTypeShell    CommandType = "shell"    // 쉘 명령어
    CommandTypeBuiltin  CommandType = "builtin"  // 내장 명령어 (echo, set, if)
    CommandTypeControl  CommandType = "control"  // 제어문 (if, for, parallel)
)
```

## 🔧 CLI 인터페이스

### 배치 작업 관리

```bash
# 배치 파일 실행
proxyndctl batch run maintenance.batch
proxyndctl batch run --name "daily-maintenance" maintenance.batch

# 스케줄 실행 등록
proxyndctl batch schedule --cron "0 2 * * *" maintenance.batch

# 배치 파일 검증
proxyndctl batch validate maintenance.batch
proxyndctl batch validate --dry-run maintenance.batch

# 실행 중인 작업 조회
proxyndctl batch list --status running
proxyndctl batch list --limit 10

# 작업 상태 조회
proxyndctl batch status <job-id>
proxyndctl batch status --follow <job-id>  # 실시간 모니터링

# 작업 취소
proxyndctl batch cancel <job-id>
proxyndctl batch cancel --all  # 모든 실행 중인 작업 취소

# 작업 이력 조회
proxyndctl batch history --days 7
proxyndctl batch history --job-name "daily-maintenance"

# 로그 조회
proxyndctl batch logs <job-id>
proxyndctl batch logs --tail 100 <job-id>
```

### 배치 파일 관리

```bash
# 배치 파일 템플릿 생성
proxyndctl batch init --template maintenance
proxyndctl batch init --template backup

# 배치 파일 등록 (재사용을 위해)
proxyndctl batch register --name daily-maintenance maintenance.batch

# 등록된 배치 작업 조회
proxyndctl batch registry list
proxyndctl batch registry show daily-maintenance

# 배치 작업 실행 (등록된 것)
proxyndctl batch exec daily-maintenance
```

## 📚 배치 스크립트 문법

### 기본 구문

#### 변수
```bash
# 변수 정의
set var_name = "value"
set number = 42
set date_today = $(date +%Y-%m-%d)

# 환경 변수 사용
set storage_dir = ${STORAGE_DIR}
set config_dir = ${CONFIG_DIR:-/etc/proxynd}

# 변수 사용
echo "Today is ${date_today}"
cache clear --older-than ${cleanup_days}d
```

#### 조건문
```bash
# 기본 조건문
if ${?} == 0 then
    echo "Previous command succeeded"
else
    echo "Previous command failed"
    exit 1
fi

# 복합 조건
if ${cache_size} > 1000000000 then  # 1GB
    echo "Cache size is large: ${cache_size} bytes"
    cache clear --older-than 7d --force
fi

# 파일/디렉토리 존재 확인
if exists "/backup/maven" then
    echo "Backup directory exists"
fi
```

#### 반복문
```bash
# 리스트 반복
for proxy_type in maven npm docker apt do
    echo "Testing ${proxy_type}..."
    test ${proxy_type} --timeout 30
    if ${?} != 0 then
        echo "Test failed for ${proxy_type}"
    fi
done

# 숫자 범위 반복
for i in 1..5 do
    echo "Attempt ${i}"
    test all --retry 1
    if ${?} == 0 then
        break
    fi
    sleep 10
done
```

#### 병렬 실행
```bash
# 병렬 블록
parallel {
    test maven --timeout 60
    test npm --timeout 60
    test docker --timeout 60
}

# 병렬 실행 결과 처리
parallel --wait-all {
    cache clear --type maven &
    cache clear --type npm &
    cache clear --type docker &
}

if ${?} == 0 then
    echo "All cache clearing completed"
fi
```

#### 에러 처리
```bash
# 전역 에러 핸들러
on_error {
    echo "Error occurred in batch job"
    echo "Last command exit code: ${?}"

    # 정리 작업
    cache clear --type temp --force

    # 알림 발송 (향후 구현)
    # notify --email admin@company.com --subject "Batch job failed"

    exit 1
}

# 특정 명령어 에러 처리
try {
    maven-backup create --target /backup/maven
} catch {
    echo "Backup failed, trying alternative location"
    maven-backup create --target /tmp/maven-backup
}
```

### 내장 함수

```bash
# 시간 관련
echo $(date)                    # 현재 시간
echo $(date +%Y%m%d)           # 포맷된 날짜
set timestamp = $(timestamp)    # Unix 타임스탬프

# 파일 시스템
if $(exists "/path/to/file") then
    echo "File exists"
fi

set file_size = $(size "/path/to/file")
set dir_count = $(count "/path/to/dir/*")

# 문자열 처리
set upper_name = $(upper "hello")      # HELLO
set lower_name = $(lower "WORLD")      # world
set length = $(len "hello world")      # 11

# ProxyND 특화 함수
set cache_size = $(cache_size "maven")
set server_status = $(server_status)
set proxy_list = $(proxy_types)
```

## 📊 모니터링 및 로깅

### 진행률 추적

```go
// ProgressTracker 진행률 추적기
type ProgressTracker struct {
    JobID       string            `json:"job_id"`
    TotalSteps  int              `json:"total_steps"`
    CurrentStep int              `json:"current_step"`
    StepName    string           `json:"step_name"`
    StartTime   time.Time        `json:"start_time"`
    Progress    float64          `json:"progress"` // 0-100
    ETA         *time.Time       `json:"eta,omitempty"`
    Logs        []LogEntry       `json:"logs"`
}

// 실시간 진행률 출력 예시
[========================================>     ] 80% (4/5)
Current: Running maven backup... (ETA: 2m 15s)
```

### 구조화된 로깅

```json
{
  "timestamp": "2025-01-01T10:30:45Z",
  "job_id": "batch-20250101-103045",
  "job_name": "daily-maintenance",
  "level": "info",
  "step": 3,
  "command": "maven-backup create",
  "message": "Backup completed successfully",
  "duration_ms": 45230,
  "metadata": {
    "backup_size": "2.1GB",
    "files_count": 15432
  }
}
```

## 🔐 보안 고려사항

### 실행 권한
- 배치 파일은 현재 사용자 권한으로 실행
- 민감한 명령어 실행 시 추가 확인 프롬프트
- 시스템 명령어 실행 제한 (화이트리스트 방식)

### 변수 보안
```bash
# 민감한 정보는 환경 변수로 전달
set api_key = ${API_KEY}  # OK
set password = "secret123"  # 권장하지 않음

# 스크립트 파일 권한 확인
chmod 600 maintenance.batch  # 소유자만 읽기/쓰기
```

### 감사 로그
- 모든 배치 작업 실행 기록
- 실행한 사용자 및 시간 기록
- 변경된 시스템 상태 추적

## 📈 성능 최적화

### 병렬 처리
- 독립적인 작업들의 병렬 실행
- CPU 및 I/O 바운드 작업 구분
- 적응적 동시성 제어

### 리소스 관리
```go
// ResourceLimits 리소스 제한
type ResourceLimits struct {
    MaxMemory      int64         `json:"max_memory"`      // 최대 메모리 (bytes)
    MaxDuration    time.Duration `json:"max_duration"`    // 최대 실행 시간
    MaxFileSize    int64         `json:"max_file_size"`   // 최대 파일 크기
    MaxConcurrency int           `json:"max_concurrency"` // 최대 동시 실행
}
```

### 캐싱
- 스크립트 파싱 결과 캐싱
- 자주 사용되는 명령어 결과 캐싱
- 메타데이터 캐싱으로 성능 향상

## 🧪 테스트 전략

### 단위 테스트
```go
func TestBatchParser(t *testing.T) {
    script := `
    set name = "test"
    echo ${name}
    `

    parser := NewBatchParser()
    job, err := parser.Parse(script)

    assert.NoError(t, err)
    assert.Equal(t, 2, len(job.Commands))
}
```

### 통합 테스트
- 실제 ProxyND 서버와 연동 테스트
- 다양한 시나리오별 배치 스크립트 테스트
- 에러 상황 및 복구 테스트

### 성능 테스트
- 대용량 배치 작업 성능 측정
- 병렬 처리 효율성 검증
- 메모리 사용량 모니터링

## 🚀 구현 로드맵

### Phase 1: 기본 엔진 (2주)
- [ ] 스크립트 파서 구현
- [ ] 기본 명령어 실행 엔진
- [ ] 변수 시스템
- [ ] 기본 CLI 인터페이스

### Phase 2: 고급 기능 (2주)
- [ ] 조건문 및 반복문
- [ ] 병렬 실행
- [ ] 에러 처리
- [ ] 진행률 추적

### Phase 3: 운영 기능 (1주)
- [ ] 스케줄링 지원
- [ ] 작업 이력 관리
- [ ] 모니터링 대시보드
- [ ] 성능 최적화

## 📋 사용 예시

### 일일 유지보수 스크립트
```bash
#!/proxyndctl/batch
# daily-maintenance.batch

echo "=== Daily Maintenance Started ==="
set start_time = $(timestamp)
set backup_date = $(date +%Y%m%d)

# 1. 오래된 캐시 정리
echo "Step 1: Cleaning old cache..."
cache clear --older-than 7d --force

# 2. 백업 생성
echo "Step 2: Creating backup..."
set backup_path = "/backup/maven-${backup_date}"
maven-backup create --target ${backup_path} --workers 8

if ${?} != 0 then
    echo "Backup failed, aborting maintenance"
    exit 1
fi

# 3. 인덱스 리빌드
echo "Step 3: Rebuilding search index..."
maven-index build --force

# 4. 시스템 테스트
echo "Step 4: Running system tests..."
parallel {
    test maven --timeout 60
    test npm --timeout 60
    test docker --timeout 60
}

# 5. 보고서 생성
echo "Step 5: Generating report..."
set end_time = $(timestamp)
set duration = $((${end_time} - ${start_time}))

echo "=== Maintenance Completed ==="
echo "Duration: ${duration} seconds"
echo "Backup location: ${backup_path}"
```

### 긴급 복구 스크립트
```bash
#!/proxyndctl/batch
# emergency-recovery.batch

echo "=== Emergency Recovery Started ==="

# 모든 캐시 정리
cache clear --force --all

# 최신 백업에서 복구
set latest_backup = $(ls -t /backup/maven-* | head -1)
echo "Restoring from: ${latest_backup}"

maven-backup restore --source ${latest_backup} --target /storage/cache/maven --overwrite

# 인덱스 재생성
maven-index build --force

# 서비스 재시작 (향후 구현)
# service restart proxynd

echo "=== Recovery Completed ==="
```

---

**📅 작성일**: 2025-01-01  
**📝 버전**: v1.0  
**🎯 구현 우선순위**: 높음
