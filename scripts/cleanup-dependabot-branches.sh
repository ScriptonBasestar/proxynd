#!/bin/bash

# Dependabot 브랜치 정리 스크립트
# 이 스크립트는 병합되었거나 오래된 dependabot 브랜치들을 정리합니다.

set -e

echo "🤖 Dependabot 브랜치 정리 시작..."

# 현재 브랜치 확인
current_branch=$(git rev-parse --abbrev-ref HEAD)
echo "현재 브랜치: $current_branch"

# 최신 정보 가져오기
echo "📡 원격 브랜치 정보 업데이트 중..."
git fetch --all --prune

# dependabot 브랜치 목록 가져오기
echo "🔍 Dependabot 브랜치 검색 중..."
dependabot_branches=$(git branch -r | grep "origin/dependabot" | sed 's/origin\///' | tr -d ' ')

if [ -z "$dependabot_branches" ]; then
    echo "✅ 정리할 dependabot 브랜치가 없습니다."
    exit 0
fi

echo "발견된 dependabot 브랜치:"
echo "$dependabot_branches"
echo ""

# 각 브랜치 상태 확인 및 정리
cleaned_count=0
total_count=$(echo "$dependabot_branches" | wc -l)

for branch in $dependabot_branches; do
    echo "🔍 분석 중: $branch"

    # 브랜치가 메인 브랜치에 병합되었는지 확인
    merged_into_main=$(git merge-base --is-ancestor origin/$branch origin/main 2>/dev/null && echo "yes" || echo "no")
    merged_into_master=$(git merge-base --is-ancestor origin/$branch origin/master 2>/dev/null && echo "yes" || echo "no")
    merged_into_develop=$(git merge-base --is-ancestor origin/$branch origin/develop 2>/dev/null && echo "yes" || echo "no")

    # 브랜치가 30일 이상 오래되었는지 확인
    last_commit_date=$(git log -1 --format="%at" origin/$branch 2>/dev/null || echo "0")
    current_date=$(date +%s)
    days_old=$(( (current_date - last_commit_date) / 86400 ))

    should_delete=false
    reason=""

    # 삭제 조건 확인
    if [ "$merged_into_main" = "yes" ] || [ "$merged_into_master" = "yes" ] || [ "$merged_into_develop" = "yes" ]; then
        should_delete=true
        reason="병합됨"
    elif [ $days_old -gt 30 ]; then
        should_delete=true
        reason="30일 이상 오래됨 ($days_old일)"
    fi

    if [ "$should_delete" = true ]; then
        echo "❌ 삭제: $branch ($reason)"

        # GitHub CLI가 있으면 PR도 확인
        if command -v gh &> /dev/null; then
            pr_number=$(gh pr list --head $branch --state all --json number --jq '.[0].number' 2>/dev/null || echo "")
            if [ -n "$pr_number" ] && [ "$pr_number" != "null" ]; then
                pr_state=$(gh pr view $pr_number --json state --jq '.state' 2>/dev/null || echo "")
                echo "  📝 관련 PR #$pr_number ($pr_state)"
            fi
        fi

        # 원격 브랜치 삭제
        if git push origin --delete $branch 2>/dev/null; then
            echo "  ✅ 원격 브랜치 삭제 성공"
            cleaned_count=$((cleaned_count + 1))
        else
            echo "  ⚠️ 원격 브랜치 삭제 실패 (이미 삭제되었을 수 있음)"
        fi

        # 로컬 원격 추적 브랜치 정리
        git branch -dr origin/$branch 2>/dev/null || true
    else
        echo "✅ 유지: $branch (최근 활동: $days_old일 전)"
    fi

    echo ""
done

echo "🎉 정리 완료!"
echo "📊 통계:"
echo "  - 총 브랜치: $total_count"
echo "  - 삭제된 브랜치: $cleaned_count"
echo "  - 유지된 브랜치: $((total_count - cleaned_count))"

# 최종 정리
echo "🧹 로컬 참조 정리 중..."
git remote prune origin

echo "✨ Dependabot 브랜치 정리 완료!"
