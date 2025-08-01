# 대화형 모드 확장 설계

ProxyND CLI의 대화형 모드는 전체 CLI 기능을 REPL(Read-Eval-Print Loop) 환경에서 사용할 수 있는 고급 인터페이스입니다.

## 📋 개요

### 목적
- CLI 명령어를 대화형으로 실행
- 복잡한 작업 플로우를 단계별로 진행
- 자동완성 및 도움말로 사용성 향상
- 세션 상태 유지로 연속적인 작업 지원

### 핵심 가치
- **생산성**: 반복적인 명령어 입력 시간 90% 단축
- **학습성**: 실시간 도움말과 자동완성으로 학습 지원
- **연속성**: 세션 컨텍스트 유지로 연관된 작업 효율화
- **편의성**: 명령어 히스토리와 별칭으로 편의성 극대화

## 🏗️ 아키텍처 설계

### 시스템 구성도

```
┌─────────────────────────────────────────────────────────────┐
│                 Interactive Shell                          │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Input Engine  │  Command Parser │   Output Formatter      │
│                 │                 │                         │
│ • Prompt        │ • Syntax Check  │ • Colored Output        │
│ • Auto Complete │ • Command Route │ • Progress Bars         │
│ • History       │ • Variable Sub  │ • Table Formatting      │
└─────────────────┴─────────────────┴─────────────────────────┘
            │                │                      │
            ▼                ▼                      ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ Session Manager │ │ Command Engine  │ │ Context Manager │
│                 │ │                 │ │                 │
│ • Variables     │ │ • CLI Commands  │ │ • Working Dir   │
│ • Aliases       │ │ • Builtins      │ │ • Server State  │
│ • History       │ │ • Scripts       │ │ • Environment   │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

### 핵심 컴포넌트

#### 1. Interactive Shell Engine
```go
// InteractiveShell 대화형 쉘 엔진
type InteractiveShell struct {
    prompt      *Prompt
    parser      *CommandParser
    session     *SessionManager
    context     *ContextManager
    history     *HistoryManager
    completer   *AutoCompleter
    formatter   *OutputFormatter
    running     bool
    exitCode    int
}

// ShellConfig 쉘 설정
type ShellConfig struct {
    PromptFormat    string                 `yaml:"prompt_format"`
    HistoryFile     string                 `yaml:"history_file"`
    HistorySize     int                    `yaml:"history_size"`
    AutoComplete    bool                   `yaml:"auto_complete"`
    ColorOutput     bool                   `yaml:"color_output"`
    Aliases         map[string]string      `yaml:"aliases"`
    Variables       map[string]interface{} `yaml:"variables"`
}

func NewInteractiveShell(config *ShellConfig) *InteractiveShell {
    return &InteractiveShell{
        prompt:    NewPrompt(config.PromptFormat),
        parser:    NewCommandParser(),
        session:   NewSessionManager(config),
        context:   NewContextManager(),
        history:   NewHistoryManager(config.HistoryFile, config.HistorySize),
        completer: NewAutoCompleter(),
        formatter: NewOutputFormatter(config.ColorOutput),
        running:   false,
    }
}
```

#### 2. 프롬프트 시스템
```go
// Prompt 프롬프트 관리자
type Prompt struct {
    format     string
    variables  map[string]interface{}
    colors     *ColorScheme
}

// ColorScheme 색상 스키마
type ColorScheme struct {
    Primary   string // 기본 텍스트
    Success   string // 성공 메시지
    Warning   string // 경고 메시지
    Error     string // 에러 메시지
    Highlight string // 강조 텍스트
    Muted     string // 보조 텍스트
}

// 프롬프트 예시
func (p *Prompt) Render(ctx *ContextManager) string {
    status := "✓"
    if ctx.LastCommandFailed() {
        status = "✗"
    }
    
    server := ctx.GetServerURL()
    if server == "" {
        server = "disconnected"
    }
    
    workDir := ctx.GetWorkingDirectory()
    
    return fmt.Sprintf("[%s] %s:%s ProxyND> ", 
        p.colorize(status, p.colors.Primary),
        p.colorize(server, p.colors.Highlight),
        p.colorize(workDir, p.colors.Muted),
    )
}
```

#### 3. 자동완성 시스템
```go
// AutoCompleter 자동완성 엔진
type AutoCompleter struct {
    commands    map[string]*Command
    aliases     map[string]string
    files       *FileCompleter
    context     *ContextManager
}

// CompleteFunc 자동완성 함수
type CompleteFunc func(ctx *CompletionContext) []Suggestion

// CompletionContext 자동완성 컨텍스트
type CompletionContext struct {
    Line        string   // 현재 입력 라인
    Words       []string // 분리된 단어들
    WordIndex   int      // 현재 단어 인덱스
    CursorPos   int      // 커서 위치
}

// Suggestion 자동완성 제안
type Suggestion struct {
    Text        string `json:"text"`        // 완성될 텍스트
    Display     string `json:"display"`     // 표시될 텍스트
    Description string `json:"description"` // 설명
    Type        string `json:"type"`        // 타입 (command, option, file, etc.)
}

// 자동완성 구현 예시
func (ac *AutoCompleter) Complete(ctx *CompletionContext) []Suggestion {
    if ctx.WordIndex == 0 {
        // 첫 번째 단어: 명령어 완성
        return ac.completeCommands(ctx)
    }
    
    command := ctx.Words[0]
    switch command {
    case "cache":
        return ac.completeCacheCommand(ctx)
    case "test":
        return ac.completeTestCommand(ctx)
    case "config":
        return ac.completeConfigCommand(ctx)
    default:
        return ac.completeGeneral(ctx)
    }
}

func (ac *AutoCompleter) completeCacheCommand(ctx *CompletionContext) []Suggestion {
    subcommands := []Suggestion{
        {Text: "list", Description: "캐시 목록 조회", Type: "subcommand"},
        {Text: "clear", Description: "캐시 삭제", Type: "subcommand"},
        {Text: "size", Description: "캐시 크기 조회", Type: "subcommand"},
    }
    
    if ctx.WordIndex == 1 {
        return subcommands
    }
    
    // 옵션 완성
    if strings.HasPrefix(ctx.Words[ctx.WordIndex], "--") {
        return []Suggestion{
            {Text: "--type", Description: "프록시 타입 지정", Type: "option"},
            {Text: "--pattern", Description: "패턴 지정", Type: "option"},
            {Text: "--older-than", Description: "시간 기준 필터", Type: "option"},
        }
    }
    
    return nil
}
```

## 💻 사용자 인터페이스

### 기본 사용법

#### 1. 대화형 모드 시작
```bash
# 기본 대화형 모드
proxyndctl interactive

# 설정 파일 지정
proxyndctl interactive --config ~/.proxyndctl-interactive.yaml

# 특정 서버 연결
proxyndctl interactive --server http://prod-proxy:8080
```

#### 2. 기본 세션 예시
```bash
$ proxyndctl interactive
ProxyND Interactive Shell v1.0.0
Type 'help' for help, 'exit' to quit.

[✓] localhost:8080:/ ProxyND> help
Available commands:
  cache     - 캐시 관리 명령어
  config    - 설정 관리 명령어  
  user      - 사용자 관리 명령어
  test      - 프록시 테스트 명령어
  status    - 서버 상태 조회
  batch     - 배치 작업 관리 (Phase 3)
  
Built-in commands:
  help      - 도움말 표시
  history   - 명령어 히스토리
  alias     - 별칭 관리
  set       - 변수 설정
  cd        - 작업 디렉토리 변경
  clear     - 화면 정리
  exit      - 대화형 모드 종료

[✓] localhost:8080:/ ProxyND> cache list --type maven
┌─────────────────────────────────────┬──────────┬────────────────────┐
│ PATH                                │ SIZE     │ LAST MODIFIED      │
├─────────────────────────────────────┼──────────┼────────────────────┤
│ org/springframework/spring-core/... │ 1.2 MB   │ 2025-01-01 10:30   │
│ com/fasterxml/jackson/jackson-co... │ 856 KB   │ 2025-01-01 09:15   │
└─────────────────────────────────────┴──────────┴────────────────────┘

[✓] localhost:8080:/ ProxyND> set cleanup_days = 30

[✓] localhost:8080:/ ProxyND> cache clear --older-than ${cleanup_days}d --force
Cache cleared successfully. 1.2 GB freed.

[✓] localhost:8080:/ ProxyND> test parallel --concurrency 3
🚀 병렬 프록시 테스트 시작
대상: [maven, npm, docker, apt, pip, yum, apk]
동시 실행 수: 3

[1/7] ✅ maven - Test passed (1.2s)
[2/7] ✅ npm - Test passed (0.8s)
[3/7] ✅ docker - Test passed (2.1s)
[4/7] ✅ apt - Test passed (1.5s)
[5/7] ❌ pip - Connection timeout (30.0s)
[6/7] ✅ yum - Test passed (1.8s)
[7/7] ✅ apk - Test passed (1.3s)

🏁 병렬 테스트 완료
총 소요 시간: 31.2s

[✗] localhost:8080:/ ProxyND> history
1. help
2. cache list --type maven
3. set cleanup_days = 30
4. cache clear --older-than ${cleanup_days}d --force
5. test parallel --concurrency 3

[✗] localhost:8080:/ ProxyND> !2
cache list --type maven
[재실행: cache list --type maven...]

[✓] localhost:8080:/ ProxyND> exit
Goodbye!
```

### 고급 기능

#### 1. 변수 시스템
```bash
# 변수 설정
ProxyND> set server_url = "http://prod-proxy:8080"
ProxyND> set backup_path = "/backup/$(date +%Y%m%d)"
ProxyND> set proxy_types = ["maven", "npm", "docker"]

# 변수 사용
ProxyND> connect ${server_url}
Connected to http://prod-proxy:8080

ProxyND> maven-backup create --target ${backup_path}
Creating backup at /backup/20250101...

# 배열 변수 반복
ProxyND> for proxy in ${proxy_types} do test ${proxy}; done
Testing maven... ✅
Testing npm... ✅  
Testing docker... ✅

# 변수 조회
ProxyND> vars
server_url    = "http://prod-proxy:8080"
backup_path   = "/backup/20250101"
proxy_types   = ["maven", "npm", "docker"]
cleanup_days  = 30
```

#### 2. 별칭 시스템
```bash
# 별칭 생성
ProxyND> alias ll = "cache list --format table"
ProxyND> alias clean = "cache clear --older-than 7d --force"
ProxyND> alias daily = "test parallel --concurrency 5"

# 별칭 사용
ProxyND> ll --type maven
[cache list --format table --type maven 실행...]

ProxyND> clean
[cache clear --older-than 7d --force 실행...]

# 별칭 조회
ProxyND> aliases
ll     = "cache list --format table"
clean  = "cache clear --older-than 7d --force"  
daily  = "test parallel --concurrency 5"

# 별칭 삭제
ProxyND> unalias clean
```

#### 3. 스크립트 실행
```bash
# 인라인 스크립트
ProxyND> { cache size; status; test maven; }
Cache size: 2.1 GB
Server status: Healthy
Maven test: ✅ Passed

# 멀티라인 스크립트
ProxyND> script {
... echo "Starting maintenance"
... cache clear --older-than 30d --force
... maven-backup create --target /backup/$(date +%Y%m%d)
... test parallel --concurrency 3
... echo "Maintenance completed"
... }
Starting maintenance
Cache cleared: 1.5 GB freed
Backup created: /backup/20250101
🚀 병렬 테스트 시작...
Maintenance completed

# 스크립트 파일 실행
ProxyND> source maintenance.script
[스크립트 파일 실행...]
```

#### 4. 파이프 및 리다이렉션
```bash
# 명령어 결과를 다른 명령어로 전달
ProxyND> cache list --type maven --format json | grep "spring"
[JSON 결과에서 "spring" 포함 항목 필터링...]

# 결과를 파일로 저장
ProxyND> status --format json > server-status.json
ProxyND> test all > test-results.txt

# 결과를 변수로 저장
ProxyND> cache_size = $(cache size --format json | jq '.total_size')
ProxyND> echo "Total cache size: ${cache_size} bytes"
```

## 🎨 사용자 경험 개선

### 1. 시각적 개선사항

#### 색상 및 테마
```go
// 색상 테마 설정
type Theme struct {
    Name        string
    Colors      ColorScheme
    Prompt      string
    Separators  map[string]string
}

var DefaultTheme = Theme{
    Name: "default",
    Colors: ColorScheme{
        Primary:   "\033[0m",    // 기본 (흰색)
        Success:   "\033[32m",   // 녹색
        Warning:   "\033[33m",   // 노란색
        Error:     "\033[31m",   // 빨간색
        Highlight: "\033[36m",   // 청록색
        Muted:     "\033[37m",   // 회색
    },
    Prompt: "[{status}] {server}:{workdir} ProxyND> ",
    Separators: map[string]string{
        "table": "─",
        "section": "═",
    },
}
```

#### 진행률 표시
```bash
# 길어지는 작업의 진행률 표시
ProxyND> maven-backup create --target /backup/maven
Creating backup...
[████████████████████████████████████████] 100% (2547/2547 files)
Backup created successfully! (Duration: 2m 15s)

# 실시간 상태 업데이트
ProxyND> test parallel --progress
🚀 병렬 프록시 테스트 시작
[⏳] maven - Testing... (1.2s)
[✅] npm - Test passed (0.8s)  
[⏳] docker - Testing... (2.1s)
[✅] maven - Test passed (1.8s)
[✅] docker - Test passed (2.3s)
```

### 2. 자동완성 고도화

#### 컨텍스트 인식 자동완성
```bash
# 현재 상태에 따른 스마트 제안
ProxyND> cache clear --type <TAB>
maven    npm      docker   apt      pip      yum      apk

ProxyND> test <TAB>
all           parallel      connectivity  types
maven         npm           docker        apt

# 파일 경로 자동완성
ProxyND> config validate --file <TAB>
./maven-proxy.yaml    ./npm-proxy.yaml    ./global.yaml

# 히스토리 기반 제안
ProxyND> test ma<TAB>
maven (recently used)
```

#### 도움말 통합
```bash
# 실시간 도움말
ProxyND> cache clear --help
Usage: cache clear [OPTIONS]

Clear cached packages

Options:
  --type string       프록시 타입 지정 (maven, npm, docker, ...)
  --pattern string    파일 패턴 지정 (*.jar, spring-*, ...)
  --older-than string 시간 기준 (7d, 24h, 30m)
  --size-limit string 크기 기준 (1GB, 500MB)
  --force            확인 없이 강제 삭제
  
Examples:
  cache clear --type maven --older-than 30d
  cache clear --pattern "*.tmp" --force
  cache clear --size-limit 1GB

# 명령어 입력 중 실시간 힌트
ProxyND> cache clear --older-than 
Hint: 시간 형식 예시 - 7d (7일), 24h (24시간), 30m (30분)
```

## 🔧 설정 및 커스터마이징

### 설정 파일 구조
```yaml
# ~/.proxyndctl-interactive.yaml
interactive:
  # 기본 설정
  prompt_format: "[{status}] {server}:{workdir} ProxyND> "
  history_file: "~/.proxyndctl_history"
  history_size: 1000
  auto_complete: true
  color_output: true
  
  # 테마 설정
  theme: "default"
  custom_colors:
    primary: "\033[0m"
    success: "\033[32m"
    warning: "\033[33m"
    error: "\033[31m"
    highlight: "\033[36m"
    muted: "\033[37m"
  
  # 별칭 설정
  aliases:
    ll: "cache list --format table"
    st: "status --format table"
    clean: "cache clear --older-than 7d --force"
    backup: "maven-backup create --target /backup/$(date +%Y%m%d)"
    daily: "test parallel --concurrency 5"
  
  # 변수 설정
  variables:
    cleanup_days: 7
    backup_path: "/backup"
    default_concurrency: 3
    server_timeout: "30s"
  
  # 자동완성 설정
  completion:
    fuzzy_matching: true
    max_suggestions: 10
    show_descriptions: true
    cache_completions: true
  
  # 출력 설정
  output:
    page_size: 20
    table_style: "rounded"
    progress_bar_width: 40
    timestamp_format: "15:04:05"
  
  # 플러그인 설정 (Phase 3 연동)
  plugins:
    - name: "git-integration"
      enabled: true
      config:
        auto_commit_history: true
    - name: "slack-notifications"  
      enabled: false
      config:
        webhook_url: "https://hooks.slack.com/..."
```

### 세션 상태 관리
```go
// SessionState 세션 상태
type SessionState struct {
    Variables    map[string]interface{} `json:"variables"`
    Aliases      map[string]string      `json:"aliases"`
    History      []string               `json:"history"`
    WorkingDir   string                 `json:"working_dir"`
    ServerURL    string                 `json:"server_url"`
    LastError    string                 `json:"last_error,omitempty"`
    StartTime    time.Time              `json:"start_time"`
    CommandCount int                    `json:"command_count"`
}

// 세션 상태 저장/복원
func (s *SessionManager) SaveState(filename string) error {
    data, err := json.MarshalIndent(s.state, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(filename, data, 0644)
}

func (s *SessionManager) LoadState(filename string) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, &s.state)
}
```

## 🧪 테스트 및 품질 보증

### 단위 테스트
```go
func TestAutoCompleter(t *testing.T) {
    completer := NewAutoCompleter()
    
    ctx := &CompletionContext{
        Line:      "cache ",
        Words:     []string{"cache", ""},
        WordIndex: 1,
        CursorPos: 6,
    }
    
    suggestions := completer.Complete(ctx)
    
    expected := []string{"list", "clear", "size"}
    actual := make([]string, len(suggestions))
    for i, s := range suggestions {
        actual[i] = s.Text
    }
    
    assert.ElementsMatch(t, expected, actual)
}

func TestVariableSubstitution(t *testing.T) {
    session := NewSessionManager(nil)
    session.SetVariable("test_type", "maven")
    session.SetVariable("timeout", 60)
    
    result := session.SubstituteVariables("test ${test_type} --timeout ${timeout}s")
    expected := "test maven --timeout 60s"
    
    assert.Equal(t, expected, result)
}
```

### 통합 테스트
```go
func TestInteractiveSession(t *testing.T) {
    shell := NewInteractiveShell(nil)
    
    // 명령어 실행 테스트
    output, err := shell.ExecuteCommand("cache list --type maven")
    assert.NoError(t, err)
    assert.Contains(t, output, "maven")
    
    // 변수 설정 테스트
    shell.ExecuteCommand("set test_var = hello")
    output, _ = shell.ExecuteCommand("echo ${test_var}")
    assert.Contains(t, output, "hello")
    
    // 별칭 테스트
    shell.ExecuteCommand("alias test_alias = 'status'")
    output, _ = shell.ExecuteCommand("test_alias")
    assert.Contains(t, output, "status")
}
```

## 📊 성능 최적화

### 메모리 최적화
```go
// 효율적인 히스토리 관리
type CircularHistory struct {
    items    []string
    capacity int
    start    int
    end      int
    size     int
}

func (h *CircularHistory) Add(item string) {
    if h.size < h.capacity {
        h.items[h.end] = item
        h.end = (h.end + 1) % h.capacity
        h.size++
    } else {
        h.items[h.start] = item
        h.start = (h.start + 1) % h.capacity
        h.end = (h.end + 1) % h.capacity
    }
}
```

### 자동완성 캐싱
```go
// 자동완성 결과 캐싱
type CompletionCache struct {
    cache map[string][]Suggestion
    mutex sync.RWMutex
    ttl   time.Duration
}

func (c *CompletionCache) Get(key string) ([]Suggestion, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    suggestions, exists := c.cache[key]
    return suggestions, exists
}

func (c *CompletionCache) Set(key string, suggestions []Suggestion) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    c.cache[key] = suggestions
    
    // TTL 기반 정리 (백그라운드에서)
    go c.cleanupAfter(key, c.ttl)
}
```

## 🚀 구현 로드맵

### Phase 1: 기본 REPL (1주)
- [ ] 기본 입력/출력 처리
- [ ] 명령어 파싱 및 실행
- [ ] 기본 자동완성
- [ ] 명령어 히스토리

### Phase 2: 고급 기능 (1주)
- [ ] 변수 시스템
- [ ] 별칭 시스템
- [ ] 멀티라인 입력
- [ ] 스크립트 실행

### Phase 3: UX 개선 (3일)
- [ ] 색상 및 테마
- [ ] 진행률 표시
- [ ] 컨텍스트 인식 자동완성
- [ ] 실시간 도움말

### Phase 4: 고급 기능 (2일)
- [ ] 파이프 및 리다이렉션
- [ ] 세션 상태 저장/복원
- [ ] 플러그인 통합 (Phase 3 연동)
- [ ] 성능 최적화

## 📋 사용 시나리오

### 일일 운영 작업
```bash
$ proxyndctl interactive

ProxyND> # 일일 시스템 체크
ProxyND> alias daily_check = "{ status; cache size; test parallel; }"
ProxyND> daily_check

ProxyND> # 캐시 정리 (조건부)
ProxyND> cache_size = $(cache size --format json | jq '.total_size')
ProxyND> if [ ${cache_size} -gt 5000000000 ]; then cache clear --older-than 7d --force; fi

ProxyND> # 백업 생성
ProxyND> maven-backup create --target /backup/$(date +%Y%m%d) --workers 8

ProxyND> # 모든 작업을 스크립트로 저장
ProxyND> history --save daily-ops.script
```

### 문제 해결 세션
```bash
ProxyND> # 문제 상황 진단
ProxyND> status --detailed
ProxyND> test all --timeout 10

ProxyND> # 특정 프록시 집중 분석
ProxyND> set problem_proxy = "maven"
ProxyND> test ${problem_proxy} --verbose
ProxyND> cache list --type ${problem_proxy} --limit 5

ProxyND> # 복구 작업
ProxyND> cache clear --type ${problem_proxy} --force
ProxyND> maven-index build --force
ProxyND> test ${problem_proxy}

ProxyND> # 결과 확인 및 보고서
ProxyND> { echo "Recovery completed at $(date)"; status; } > recovery-report.txt
```

---

**📅 작성일**: 2025-01-01  
**📝 버전**: v1.0  
**🎯 구현 우선순위**: 중간 (단기 계획)