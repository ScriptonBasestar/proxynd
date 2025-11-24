#!/bin/bash
# 스크립트명: analyze-security-alerts.sh
# 용도: GitHub Code Scanning alerts 분석 및 처리
# 사용법: scripts/analyze-security-alerts.sh [--dismiss-false-positives] [--export]
# 예시: scripts/analyze-security-alerts.sh --export

set -e

REPO="ScriptonBasestar/proxynd"
DISMISS_FALSE_POSITIVES=false
EXPORT_REPORT=false

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dismiss-false-positives)
            DISMISS_FALSE_POSITIVES=true
            shift
            ;;
        --export)
            EXPORT_REPORT=true
            shift
            ;;
        *)
            echo "Usage: $0 [--dismiss-false-positives] [--export]"
            echo ""
            echo "Options:"
            echo "  --dismiss-false-positives  Auto-dismiss known false positives"
            echo "  --export                   Export detailed report to file"
            exit 1
            ;;
    esac
done

echo -e "${MAGENTA}🔍 Analyzing GitHub Code Scanning Alerts...${NC}"
echo ""

# Fetch all open alerts
echo -e "${CYAN}Fetching alerts from GitHub...${NC}"
ALERTS=$(gh api 'repos/'"$REPO"'/code-scanning/alerts?per_page=100&state=open' --paginate 2>/dev/null || echo "[]")

TOTAL_COUNT=$(echo "$ALERTS" | jq 'length')

if [ "$TOTAL_COUNT" -eq 0 ]; then
    echo -e "${GREEN}✅ No open Code Scanning alerts!${NC}"
    exit 0
fi

echo -e "${RED}Found ${TOTAL_COUNT} open alerts${NC}"
echo ""

# Group by rule
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Alerts by Rule:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "$ALERTS" | jq -r '.[] | .rule.id' | sort | uniq -c | sort -rn | while read count rule; do
    severity=$(echo "$ALERTS" | jq -r ".[] | select(.rule.id == \"$rule\") | .rule.security_severity_level" | head -1)

    case "$severity" in
        critical|high)
            color="${RED}"
            ;;
        medium)
            color="${YELLOW}"
            ;;
        low)
            color="${GREEN}"
            ;;
        *)
            color="${NC}"
            ;;
    esac

    printf "${color}%-6s %s${NC} [%s]\n" "$count" "$rule" "$severity"
done

echo ""

# Group by severity
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Alerts by Severity:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "$ALERTS" | jq -r '.[] | .rule.security_severity_level // "none"' | sort | uniq -c | sort -rn | while read count severity; do
    case "$severity" in
        critical)
            printf "${RED}%-6s %s${NC}\n" "$count" "🔴 Critical"
            ;;
        high)
            printf "${RED}%-6s %s${NC}\n" "$count" "🟠 High"
            ;;
        medium)
            printf "${YELLOW}%-6s %s${NC}\n" "$count" "🟡 Medium"
            ;;
        low)
            printf "${GREEN}%-6s %s${NC}\n" "$count" "🟢 Low"
            ;;
        none)
            printf "${NC}%-6s %s${NC}\n" "$count" "⚪ None"
            ;;
    esac
done

echo ""

# Top 5 most common issues
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Top 5 Most Common Issues:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

cat << 'RULES'
G104: Unhandled errors (에러 체크 누락)
      해결: err 반환값 체크 또는 _ 명시

G304: File path from external input (경로 주입 취약점)
      해결: filepath.Clean() 사용 + 허용 디렉토리 검증

go/clear-text-logging: Sensitive data in logs
      해결: 비밀번호/토큰 로깅 제거 또는 마스킹

G301: Poor file permissions for Mkdir
      해결: os.MkdirAll(path, 0750) 권한 강화

G306: Poor file permissions for WriteFile
      해결: os.WriteFile(path, data, 0640) 권한 강화
RULES

echo ""

# Export report if requested
if [ "$EXPORT_REPORT" = true ]; then
    REPORT_FILE="security-alerts-$(date +%Y%m%d-%H%M%S).json"
    echo "$ALERTS" > "$REPORT_FILE"
    echo -e "${GREEN}✅ Exported detailed report to: ${REPORT_FILE}${NC}"
    echo ""

    # Generate markdown summary
    MD_FILE="security-alerts-$(date +%Y%m%d-%H%M%S).md"
    cat > "$MD_FILE" << EOF
# Code Scanning Alerts Report

**Generated**: $(date)
**Repository**: $REPO
**Total Alerts**: $TOTAL_COUNT

## Alerts by Severity

$(echo "$ALERTS" | jq -r '.[] | .rule.security_severity_level // "none"' | sort | uniq -c | sort -rn | awk '{printf "- **%s**: %d\n", $2, $1}')

## Alerts by Rule

$(echo "$ALERTS" | jq -r '.[] | .rule.id' | sort | uniq -c | sort -rn | awk '{printf "- **%s**: %d\n", $2, $1}')

## Top Issues

### G104: Unhandled errors ($(echo "$ALERTS" | jq -r '.[] | select(.rule.id == "G104") | .number' | wc -l) occurrences)
에러 반환값을 체크하지 않는 코드. \`_ = func()\` 또는 에러 처리 추가 필요.

### G304: File path injection ($(echo "$ALERTS" | jq -r '.[] | select(.rule.id == "G304") | .number' | wc -l) occurrences)
외부 입력으로부터 파일 경로를 받는 경우. \`filepath.Clean()\` 및 경로 검증 필요.

### go/clear-text-logging: Sensitive data ($(echo "$ALERTS" | jq -r '.[] | select(.rule.id == "go/clear-text-logging") | .number' | wc -l) occurrences)
비밀번호, 토큰 등 민감 정보를 로그에 출력. 마스킹 또는 제거 필요.

## Recommendations

1. **Prioritize Critical/High severity** issues first
2. **G104**: Add explicit error handling or use \`_ =\` for intentionally ignored errors
3. **G304**: Use \`filepath.Clean()\` and validate paths against allowed directories
4. **Logging**: Remove or mask sensitive data (passwords, tokens, API keys)
5. **File Permissions**: Use 0750 for directories, 0640 for files

## Actions

\`\`\`bash
# Review all G104 alerts
gh api 'repos/$REPO/code-scanning/alerts?per_page=100&state=open' --jq '.[] | select(.rule.id == "G104") | {number, location: .most_recent_instance.location.path}'

# Dismiss false positives (if applicable)
# gh api repos/$REPO/code-scanning/alerts/{alert_number} -X PATCH -f state=dismissed -f dismissed_reason=false_positive
\`\`\`
EOF
    echo -e "${GREEN}✅ Generated markdown report: ${MD_FILE}${NC}"
    echo ""
fi

# Recommendations
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}📋 Recommended Actions:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

cat << 'ACTIONS'
1. 🔴 Critical/High severity 우선 처리

2. ✅ 빠른 수정 가능 (자동화 가능):
   - G104: 에러 체크 추가
   - go/useless-assignment: Dead code 제거
   - go/redundant-operation: 중복 코드 제거

3. ⚠️ 코드 리뷰 필요 (보안 중요):
   - G304: 파일 경로 검증
   - go/clear-text-logging: 민감 정보 마스킹
   - G301/G306: 파일 권한 강화

4. 📝 False positive 처리:
   - 의도적으로 무시한 에러
   - 테스트 코드의 weak crypto
   - 샘플/예제 코드

다음 단계:
./scripts/fix-security-alerts.sh        # 자동 수정 도구 (추후 제공)
./scripts/dismiss-false-positives.sh    # False positive 일괄 처리
ACTIONS

echo ""
echo -e "${CYAN}💡 Tip: Use --export to save detailed report${NC}"
echo -e "${CYAN}💡 Tip: Review alerts at: https://github.com/$REPO/security/code-scanning${NC}"
