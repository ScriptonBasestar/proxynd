#!/bin/bash
# 스크립트명: merge-dependabot-updates.sh
# 용도: Dependabot PR을 자동으로 rebase merge
# 사용법: scripts/merge-dependabot-updates.sh [--dry-run] [--squash|--rebase|--merge]
# 예시: scripts/merge-dependabot-updates.sh --rebase

set -e

REPO="ScriptonBasestar/proxynd"
DRY_RUN=false
MERGE_METHOD="rebase"  # Default: rebase for linear history

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --squash)
            MERGE_METHOD="squash"
            shift
            ;;
        --rebase)
            MERGE_METHOD="rebase"
            shift
            ;;
        --merge)
            MERGE_METHOD="merge"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--dry-run] [--squash|--rebase|--merge]"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}🔍 Fetching Dependabot PRs...${NC}"
echo -e "${YELLOW}Merge method: ${MERGE_METHOD}${NC}"
echo ""

# Get all open Dependabot PRs
DEPENDABOT_PRS=$(gh pr list \
    --repo "$REPO" \
    --author "app/dependabot" \
    --state open \
    --json number,title,headRefName,mergeable,updatedAt \
    --jq 'sort_by(.updatedAt) | .[] | select(.mergeable == "MERGEABLE") | .number')

if [ -z "$DEPENDABOT_PRS" ]; then
    echo -e "${GREEN}✅ No mergeable Dependabot PRs found${NC}"
    exit 0
fi

# Count total PRs
TOTAL_PRS=$(echo "$DEPENDABOT_PRS" | wc -l)
echo -e "${BLUE}Found ${TOTAL_PRS} mergeable Dependabot PR(s):${NC}"

# Show all PRs first
for PR_NUMBER in $DEPENDABOT_PRS; do
    PR_TITLE=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json title --jq '.title')
    echo -e "  ${YELLOW}#${PR_NUMBER}${NC}: $PR_TITLE"
done
echo ""

# Ask for confirmation unless dry-run
if [ "$DRY_RUN" = false ]; then
    read -p "Proceed with ${MERGE_METHOD} merge? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled."
        exit 0
    fi
    echo ""
fi

# Merge each PR
SUCCESS_COUNT=0
FAILED_COUNT=0

for PR_NUMBER in $DEPENDABOT_PRS; do
    echo -e "${BLUE}🔄 Processing PR #$PR_NUMBER...${NC}"

    # Get PR details
    PR_TITLE=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json title --jq '.title')
    echo -e "  ${YELLOW}Title:${NC} $PR_TITLE"

    if [ "$DRY_RUN" = true ]; then
        echo -e "  ${GREEN}[DRY RUN]${NC} Would approve and ${MERGE_METHOD} merge PR #$PR_NUMBER"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        # Approve the PR
        echo -e "  ${YELLOW}→${NC} Approving PR..."
        if gh pr review "$PR_NUMBER" --repo "$REPO" --approve --body "Auto-approved by merge script" 2>/dev/null; then
            echo -e "  ${GREEN}✓${NC} Approved"
        else
            echo -e "  ${YELLOW}⚠${NC} Already approved or approval failed"
        fi

        # Merge with specified method
        echo -e "  ${YELLOW}→${NC} Merging with ${MERGE_METHOD}..."
        if gh pr merge "$PR_NUMBER" --repo "$REPO" --${MERGE_METHOD} 2>/dev/null; then
            echo -e "  ${GREEN}✅ Successfully merged PR #$PR_NUMBER${NC}"
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        else
            echo -e "  ${RED}❌ Failed to merge PR #$PR_NUMBER${NC}"
            FAILED_COUNT=$((FAILED_COUNT + 1))
        fi
    fi
    echo ""
done

# Summary
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Summary:${NC}"
echo -e "  ${GREEN}✅ Success: ${SUCCESS_COUNT}${NC}"
if [ $FAILED_COUNT -gt 0 ]; then
    echo -e "  ${RED}❌ Failed:  ${FAILED_COUNT}${NC}"
fi
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ $FAILED_COUNT -gt 0 ]; then
    exit 1
fi
