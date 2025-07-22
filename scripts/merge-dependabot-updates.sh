#!/bin/bash

# Dependabot 업데이트 병합 스크립트

echo "🤖 Dependabot 업데이트 처리 시작..."

# 색상 정의
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 현재 브랜치 저장
CURRENT_BRANCH=$(git branch --show-current)

# Dependabot 브랜치 목록
BRANCHES=(
    "dependabot-github_actions-actions-setup-go-5"
    "dependabot-github_actions-azure-setup-helm-4"
    "dependabot-github_actions-peter-evans-create-pull-request-7"
    "dependabot-go_modules-github.com-aws-aws-sdk-go-v2-service-s3-1.83.0"
    "dependabot-go_modules-github.com-valyala-fasthttp-1.62.0"
    "dependabot-go_modules-github.com-valyala-fasthttp-1.63.0"
)

# 업데이트 요약
echo ""
echo "📋 처리할 업데이트:"
echo "- GitHub Actions: setup-go v5"
echo "- GitHub Actions: azure/setup-helm v4"
echo "- GitHub Actions: peter-evans/create-pull-request v7"
echo "- Go Module: aws-sdk-go-v2/service/s3 v1.83.0"
echo "- Go Module: valyala/fasthttp v1.62.0 -> v1.63.0"
echo ""

# 원격 업데이트
git fetch origin

# 각 브랜치 병합
for branch in "${BRANCHES[@]}"; do
    echo -e "${YELLOW}처리 중: $branch${NC}"

    # 브랜치가 존재하는지 확인
    if git ls-remote --heads origin | grep -q "$branch"; then
        # Cherry-pick 방식으로 변경사항 가져오기
        COMMIT=$(git ls-remote origin "refs/heads/$branch" | cut -f1)

        if [ ! -z "$COMMIT" ]; then
            echo "커밋 $COMMIT 적용 중..."

            # Cherry-pick 시도
            if git cherry-pick "$COMMIT" 2>/dev/null; then
                echo -e "${GREEN}✅ 성공적으로 적용됨${NC}"
            else
                # 충돌 발생 시
                if [ -n "$(git status --porcelain)" ]; then
                    echo -e "${RED}⚠️  충돌 발생. 자동 해결 시도...${NC}"

                    # 파일별 처리
                    if [[ "$branch" == *"fasthttp"* ]]; then
                        # fasthttp의 경우 최신 버전 사용
                        if [[ "$branch" == *"1.63.0"* ]]; then
                            # 1.63.0 버전 수락
                            git add go.mod go.sum 2>/dev/null
                            git cherry-pick --continue --no-edit 2>/dev/null
                        else
                            # 이전 버전은 스킵
                            git cherry-pick --skip 2>/dev/null
                            echo "이전 버전 스킵 (최신 버전 사용)"
                        fi
                    else
                        # 다른 업데이트는 그대로 적용
                        git add -A
                        git cherry-pick --continue --no-edit 2>/dev/null
                    fi
                fi
            fi
        fi
    else
        echo -e "${RED}❌ 브랜치를 찾을 수 없음${NC}"
    fi

    echo ""
done

# 의존성 정리
echo "🧹 의존성 정리 중..."
go mod tidy

# 변경사항 확인
if [ -n "$(git status --porcelain)" ]; then
    echo ""
    echo "📝 변경된 파일:"
    git status --porcelain

    # 스테이징
    git add go.mod go.sum .github/workflows/*.yml 2>/dev/null

    # 커밋
    echo ""
    echo "💾 변경사항 커밋 중..."
    git commit -m "chore(deps): merge dependabot updates

- GitHub Actions:
  - actions/setup-go: v2 → v5
  - azure/setup-helm: → v4
  - peter-evans/create-pull-request: → v7

- Go Modules:
  - aws-sdk-go-v2/service/s3: v1.82.0 → v1.83.0
  - valyala/fasthttp: v1.51.0 → v1.63.0

모든 의존성을 최신 버전으로 업데이트"
fi

echo ""
echo -e "${GREEN}✅ Dependabot 업데이트 처리 완료!${NC}"
echo ""
echo "다음 단계:"
echo "1. 변경사항 확인: git log --oneline -5"
echo "2. 테스트 실행: make test"
echo "3. 원격 푸시: git push origin $CURRENT_BRANCH"
