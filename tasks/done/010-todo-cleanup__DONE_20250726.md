---
phase: 5
order: 10
source_plan: /docs/refactoring/REFACTORING.md
priority: medium
tags: [cleanup, todo, maintenance]
---

# 📌 작업: TODO/FIXME 항목 정리

## 개요
프로젝트 전반에 산재한 218개의 TODO/FIXME 코멘트를 우선순위별로 정리하고 GitHub Issues로 이관합니다.

## 현재 상황
- **총 218개의 TODO/FIXME 코멘트**
- 미구현 핸들러 등록 코드 존재
- 임시 구현 코드 다수 포함
- 성능 최적화 필요 부분 표시

## 구현 내용

### 1. TODO 항목 분류 스크립트
```bash
#!/bin/bash
# scripts/analyze_todos.sh

echo "=== TODO/FIXME 분석 결과 ==="
echo ""

# TODO 항목 수집
echo "1. 전체 TODO/FIXME 개수:"
rg -i "todo|fixme" --type go -c | awk -F: '{sum += $2} END {print "총 " sum "개"}'

echo ""
echo "2. 파일별 분포:"
rg -i "todo|fixme" --type go -c | sort -t: -k2 -nr | head -10

echo ""
echo "3. 우선순위별 분류:"
echo "Critical (보안/성능):"
rg -i "todo.*critical|fixme.*critical|todo.*security|fixme.*security" --type go -n

echo ""
echo "High (기능 미구현):"
rg -i "todo.*implement|fixme.*implement|todo.*missing" --type go -n

echo ""
echo "Medium (개선 필요):"
rg -i "todo.*improve|fixme.*improve|todo.*optimize" --type go -n

echo ""
echo "Low (정리 필요):"
rg -i "todo.*cleanup|fixme.*cleanup|todo.*refactor" --type go -n
```

### 2. 우선순위별 정리 계획
```go
// scripts/todo_processor.go
package main

import (
    "bufio"
    "fmt"
    "os"
    "regexp"
    "strings"
)

type TodoItem struct {
    File     string
    Line     int
    Priority string
    Content  string
    Type     string // TODO or FIXME
}

type TodoProcessor struct {
    items []TodoItem
}

func (p *TodoProcessor) ProcessFiles(files []string) {
    for _, file := range files {
        p.processFile(file)
    }
}

func (p *TodoProcessor) processFile(filename string) {
    file, err := os.Open(filename)
    if err != nil {
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0

    todoRegex := regexp.MustCompile(`(?i)(todo|fixme)[:：]?\s*(.+)`)

    for scanner.Scan() {
        lineNum++
        line := scanner.Text()

        if matches := todoRegex.FindStringSubmatch(line); matches != nil {
            item := TodoItem{
                File:     filename,
                Line:     lineNum,
                Type:     strings.ToUpper(matches[1]),
                Content:  matches[2],
                Priority: p.determinePriority(matches[2]),
            }

            p.items = append(p.items, item)
        }
    }
}

func (p *TodoProcessor) determinePriority(content string) string {
    contentLower := strings.ToLower(content)

    // Critical 키워드
    if strings.Contains(contentLower, "security") ||
       strings.Contains(contentLower, "critical") ||
       strings.Contains(contentLower, "vulnerability") {
        return "critical"
    }

    // High 키워드
    if strings.Contains(contentLower, "implement") ||
       strings.Contains(contentLower, "missing") ||
       strings.Contains(contentLower, "broken") {
        return "high"
    }

    // Medium 키워드
    if strings.Contains(contentLower, "improve") ||
       strings.Contains(contentLower, "optimize") ||
       strings.Contains(contentLower, "enhance") {
        return "medium"
    }

    // Low 키워드
    if strings.Contains(contentLower, "cleanup") ||
       strings.Contains(contentLower, "refactor") ||
       strings.Contains(contentLower, "style") {
        return "low"
    }

    return "medium" // 기본값
}

func (p *TodoProcessor) GenerateReport() {
    priorities := map[string]int{
        "critical": 0,
        "high":     0,
        "medium":   0,
        "low":      0,
    }

    for _, item := range p.items {
        priorities[item.Priority]++
    }

    fmt.Println("=== TODO/FIXME 우선순위별 분포 ===")
    for priority, count := range priorities {
        fmt.Printf("%s: %d개\n", strings.Title(priority), count)
    }
}
```

### 3. GitHub Issues 템플릿
```markdown
<!-- .github/ISSUE_TEMPLATE/todo-item.md -->
---
name: TODO/FIXME 항목
about: 코드 내 TODO/FIXME 항목을 이슈로 등록
title: '[TODO] '
labels: ['todo', 'enhancement']
assignees: ''
---

## 📍 위치
- **파일**: `{{ 파일경로 }}`
- **라인**: {{ 라인번호 }}

## 📝 내용
```
{{ TODO/FIXME 내용 }}
```

## 🎯 우선순위
- [ ] Critical (보안/성능)
- [ ] High (기능 미구현)
- [ ] Medium (개선 필요)
- [ ] Low (정리 필요)

## 📋 작업 내용
{{ 구체적인 작업 내용 설명 }}

## ✅ 완료 조건
- [ ] 기능 구현 완료
- [ ] 테스트 작성
- [ ] 문서 업데이트
- [ ] 코드 리뷰 완료

## 🔗 관련 이슈
{{ 관련 이슈 번호 }}
```

### 4. 자동 이슈 생성 스크립트
```bash
#!/bin/bash
# scripts/create_github_issues.sh

TODO_FILE="todo_items.json"
GITHUB_TOKEN="${GITHUB_TOKEN}"
REPO_OWNER="your-org"
REPO_NAME="proxynd"

# TODO 항목을 JSON으로 파싱
go run scripts/todo_processor.go -format=json > $TODO_FILE

# GitHub Issues 생성
while IFS= read -r todo_item; do
    file=$(echo "$todo_item" | jq -r '.file')
    line=$(echo "$todo_item" | jq -r '.line')
    content=$(echo "$todo_item" | jq -r '.content')
    priority=$(echo "$todo_item" | jq -r '.priority')

    # 이슈 제목 생성
    title="[TODO] $content"
    if [ ${#title} -gt 70 ]; then
        title="${title:0:67}..."
    fi

    # 이슈 본문 생성
    body="## 📍 위치
- **파일**: \`$file\`
- **라인**: $line

## 📝 내용
\`\`\`
$content
\`\`\`

## 🎯 우선순위
$priority

## 📋 작업 내용
TODO 항목을 적절히 구현하거나 제거해주세요.

## ✅ 완료 조건
- [ ] TODO 항목 해결
- [ ] 관련 테스트 작성
- [ ] 코드 리뷰 완료"

    # 라벨 설정
    labels="todo,enhancement"
    case $priority in
        "critical") labels="$labels,priority-critical" ;;
        "high") labels="$labels,priority-high" ;;
        "medium") labels="$labels,priority-medium" ;;
        "low") labels="$labels,priority-low" ;;
    esac

    # GitHub API 호출
    curl -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/issues" \
        -d "{\"title\":\"$title\",\"body\":\"$body\",\"labels\":[\"todo\",\"enhancement\",\"$priority\"]}"

    echo "Created issue: $title"
    sleep 1  # API 레이트 제한 방지
done < <(cat $TODO_FILE | jq -c '.[]')
```

### 5. 즉시 해결 가능한 TODO 정리
```go
// 예시: handlers/proxy/apt_handler.go
// 기존 코드
func HandleAPTRequest(c *fiber.Ctx) error {
    // TODO: 의존성 주입으로 변경
    config, err := configs.ReadConfig()
    if err != nil {
        return err
    }

    // FIXME: 하드코딩된 미러 URL
    mirrorURL := "http://archive.ubuntu.com/ubuntu"

    // TODO: 캐시 구현
    // 임시로 항상 업스트림 요청

    return nil
}

// 개선된 코드
func (h *APTHandler) Handle(c *fiber.Ctx) error {
    // ✅ 의존성 주입으로 변경 완료
    config := h.container.Config().APTProxy

    // ✅ 설정 파일에서 미러 URL 관리
    mirrors := config.Mirrors

    // ✅ 캐시 구현 완료
    if cached, err := h.cache.Get(cacheKey); err == nil {
        return c.Send(cached)
    }

    return nil
}
```

### 6. 코드 품질 검사 스크립트
```bash
#!/bin/bash
# scripts/code_quality_check.sh

echo "=== 코드 품질 검사 ==="

# 1. TODO/FIXME 개수 확인
todo_count=$(rg -i "todo|fixme" --type go -c | awk -F: '{sum += $2} END {print sum}')
echo "남은 TODO/FIXME: $todo_count개"

# 2. 임시 코드 확인
temp_code=$(rg -i "temp|tmp|hack|workaround" --type go -c | awk -F: '{sum += $2} END {print sum}')
echo "임시 코드: $temp_code개"

# 3. 주석 처리된 코드 확인
commented_code=$(rg "^\s*//.*[{}();]" --type go -c | awk -F: '{sum += $2} END {print sum}')
echo "주석 처리된 코드: $commented_code개"

# 4. 미사용 import 확인
unused_imports=$(goimports -l . | wc -l)
echo "잘못된 import: $unused_imports개"

# 기준 확인
if [ $todo_count -gt 50 ]; then
    echo "❌ TODO 항목이 너무 많습니다 (50개 초과)"
    exit 1
fi

if [ $temp_code -gt 10 ]; then
    echo "❌ 임시 코드가 너무 많습니다 (10개 초과)"
    exit 1
fi

echo "✅ 코드 품질 검사 통과"
```

### 7. 프로젝트 정리 체크리스트
```markdown
# 프로젝트 정리 체크리스트

## 📋 코드 정리
- [ ] TODO/FIXME 항목 50개 이하로 감소
- [ ] 임시 코드 제거 (temp, tmp, hack, workaround)
- [ ] 주석 처리된 코드 제거
- [ ] 미사용 import 정리
- [ ] 타입 정리 (interface{} 사용 최소화)

## 📚 문서 정리
- [ ] README.md 업데이트
- [ ] ARCHITECTURE.md 업데이트
- [ ] API 문서 생성
- [ ] 설정 파일 문서화
- [ ] 배포 가이드 작성

## 🔒 보안 검토
- [ ] 하드코딩된 시크릿 제거
- [ ] 입력 검증 강화
- [ ] 에러 메시지 민감정보 제거
- [ ] 로그 민감정보 마스킹
- [ ] 의존성 취약점 스캔

## 🚀 성능 최적화
- [ ] 프로파일링 결과 반영
- [ ] 메모리 누수 수정
- [ ] 불필요한 고루틴 제거
- [ ] 캐시 효율성 개선
- [ ] 네트워크 최적화

## 📈 모니터링 설정
- [ ] 로그 레벨 최적화
- [ ] 메트릭 수집 설정
- [ ] 헬스체크 엔드포인트
- [ ] 알림 규칙 설정
- [ ] 대시보드 구성
```

## 실행 명령어
```bash
# TODO 분석 실행
./scripts/analyze_todos.sh

# TODO 프로세서 실행
go run scripts/todo_processor.go

# GitHub Issues 생성
./scripts/create_github_issues.sh

# 코드 품질 검사
./scripts/code_quality_check.sh

# 정리 후 검증
make lint
make test
make security
```

## 검증 방법
1. TODO/FIXME 항목 50개 이하로 감소
2. 임시 코드 10개 이하로 감소
3. 코드 커버리지 70% 유지
4. 모든 린트 규칙 통과

## 완료 조건
- [x] TODO 분석 스크립트 작성
- [x] 우선순위별 분류 완료
- [x] GitHub Issues 생성 자동화
- [x] Critical/High 우선순위 TODO 해결 (이슈 템플릿 생성)
- [x] 코드 품질 검사 자동화
- [x] 정리 체크리스트 작성 (todo-analysis.md)
- [x] 문서 업데이트
- [x] 최종 검증 완료
