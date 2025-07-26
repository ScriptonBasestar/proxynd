#!/bin/bash
# scripts/final_validation.sh

echo "🔍 ProxyND 리팩토링 최종 검증 시작"
echo "=================================="

# 로그 파일 생성
LOG_FILE="validation_$(date +%Y%m%d_%H%M%S).log"
exec > >(tee -a "$LOG_FILE")
exec 2>&1

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 검증 결과 추적
pass_count=0
fail_count=0
warn_count=0

echo ""
echo "1️⃣ 코드 품질 검사"
echo "----------------------------------------"
if bash ./scripts/code_quality_check.sh 2>/dev/null; then
    ((pass_count++))
    echo -e "${GREEN}✅ 코드 품질 검사 통과${NC}"
else
    ((fail_count++))
    echo -e "${RED}❌ 코드 품질 검사 실패${NC}"
fi

echo ""
echo "2️⃣ 테스트 커버리지 검증"
echo "----------------------------------------"
echo "테스트 실행 중..."
if go test -coverprofile=coverage.out ./... > /dev/null 2>&1; then
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    echo "전체 커버리지: ${coverage}%"
    
    if [ "${coverage%%.*}" -ge 70 ]; then
        ((pass_count++))
        echo -e "${GREEN}✅ 테스트 커버리지 검증 통과 (${coverage}% >= 70%)${NC}"
    else
        ((fail_count++))
        echo -e "${RED}❌ 테스트 커버리지 부족 (${coverage}% < 70%)${NC}"
    fi
else
    ((fail_count++))
    echo -e "${RED}❌ 테스트 실행 실패${NC}"
fi

echo ""
echo "3️⃣ 빌드 검증"
echo "----------------------------------------"
if make build > /dev/null 2>&1; then
    if [ -f "./bin/proxynd" ]; then
        ((pass_count++))
        echo -e "${GREEN}✅ 빌드 성공${NC}"
    else
        ((fail_count++))
        echo -e "${RED}❌ 빌드 실패: 실행 파일이 생성되지 않음${NC}"
    fi
else
    ((fail_count++))
    echo -e "${RED}❌ 빌드 실패${NC}"
fi

echo ""
echo "4️⃣ 보안 검사"
echo "----------------------------------------"
security_issues=0

# Go 보안 검사
if command -v gosec &> /dev/null; then
    echo "Go 코드 보안 검사 중..."
    if gosec -quiet ./... > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Go 코드 보안 검사 통과${NC}"
    else
        ((security_issues++))
        echo -e "${YELLOW}⚠ Go 코드 보안 이슈 발견${NC}"
    fi
fi

# 의존성 검사
echo "의존성 검증 중..."
if go mod verify > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 의존성 검증 통과${NC}"
else
    ((security_issues++))
    echo -e "${RED}✗ 의존성 검증 실패${NC}"
fi

if [ $security_issues -eq 0 ]; then
    ((pass_count++))
    echo -e "${GREEN}✅ 보안 검사 통과${NC}"
else
    ((warn_count++))
    echo -e "${YELLOW}⚠️ 보안 검사 경고: ${security_issues}개 이슈${NC}"
fi

echo ""
echo "5️⃣ 린트 검사"
echo "----------------------------------------"
if make lint > /dev/null 2>&1; then
    ((pass_count++))
    echo -e "${GREEN}✅ 린트 검사 통과${NC}"
else
    ((warn_count++))
    echo -e "${YELLOW}⚠️ 린트 경고 발견${NC}"
fi

echo ""
echo "=================================="
echo "🏁 최종 검증 결과"
echo "=================================="

echo "총 검사 항목: $((pass_count + fail_count + warn_count))"
echo -e "  ${GREEN}✅ 통과: $pass_count${NC}"
echo -e "  ${YELLOW}⚠️  경고: $warn_count${NC}"
echo -e "  ${RED}❌ 실패: $fail_count${NC}"

echo ""
if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}✅ 모든 필수 검증 단계 통과!${NC}"
    
    if [ $warn_count -gt 0 ]; then
        echo -e "${YELLOW}⚠️  경고: ${warn_count}개의 경고가 있습니다.${NC}"
    fi
    
    echo ""
    echo "📊 프로젝트 현황:"
    echo "- TODO/FIXME: $(grep -r -i "todo\|fixme" --include="*.go" . 2>/dev/null | wc -l)개"
    echo "- 테스트 커버리지: ${coverage}%"
    echo "- 로그 파일: $LOG_FILE"
    
    exit 0
else
    echo -e "${RED}❌ ${fail_count}개의 검증 단계 실패${NC}"
    echo ""
    echo "❗ 실패한 항목을 수정한 후 다시 실행하세요"
    echo "📄 상세 로그: $LOG_FILE"
    
    exit 1
fi