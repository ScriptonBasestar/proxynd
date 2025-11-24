#!/bin/bash
# 스크립트명: merge-security-updates.sh
# 용도: Dependabot 보안 업데이트 PR만 자동으로 rebase merge
# 사용법: scripts/merge-security-updates.sh [--dry-run] [--all]
# 예시: scripts/merge-security-updates.sh

set -e

REPO="ScriptonBasestar/proxynd"
DRY_RUN=false
MERGE_ALL=false

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --all)
            MERGE_ALL=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--dry-run] [--all]"
            echo ""
            echo "Options:"
            echo "  --dry-run  Show what would be done without making changes"
            echo "  --all      Include all Dependabot PRs (not just security)"
            exit 1
            ;;
    esac
done

echo -e "${MAGENTA}🔒 Fetching Security Updates...${NC}"
echo ""

# Get all open Dependabot PRs with labels
ALL_PRS=$(gh pr list \
    --repo "$REPO" \
    --author "app/dependabot" \
    --state open \
    --json number,title,labels,headRefName,mergeable,updatedAt \
    --jq 'sort_by(.updatedAt)')

if [ -z "$ALL_PRS" ] || [ "$ALL_PRS" = "[]" ]; then
    echo -e "${GREEN}✅ No Dependabot PRs found${NC}"
    exit 0
fi

# Filter security PRs
if [ "$MERGE_ALL" = false ]; then
    SECURITY_PRS=$(echo "$ALL_PRS" | jq -r '.[] | select(
        (.labels[]?.name == "security") or
        (.title | test("(?i)(security|cve|vulnerability|exploit)"))
    ) | select(.mergeable == "MERGEABLE") | .number')
else
    SECURITY_PRS=$(echo "$ALL_PRS" | jq -r '.[] | select(.mergeable == "MERGEABLE") | .number')
fi

if [ -z "$SECURITY_PRS" ]; then
    echo -e "${YELLOW}⚠️ No mergeable security updates found${NC}"
    echo ""
    echo "All Dependabot PRs:"
    echo "$ALL_PRS" | jq -r '.[] | "  #\(.number): \(.title) [\(.mergeable)]"'
    exit 0
fi

# Count PRs
TOTAL_PRS=$(echo "$SECURITY_PRS" | wc -l)

if [ "$MERGE_ALL" = false ]; then
    echo -e "${RED}🔒 Found ${TOTAL_PRS} SECURITY update(s):${NC}"
else
    echo -e "${BLUE}Found ${TOTAL_PRS} Dependabot PR(s):${NC}"
fi
echo ""

# Show all PRs
for PR_NUMBER in $SECURITY_PRS; do
    PR_DATA=$(echo "$ALL_PRS" | jq -r ".[] | select(.number == $PR_NUMBER)")
    PR_TITLE=$(echo "$PR_DATA" | jq -r '.title')
    PR_LABELS=$(echo "$PR_DATA" | jq -r '.labels[]?.name' | tr '\n' ', ' | sed 's/,$//')

    if echo "$PR_LABELS" | grep -q "security"; then
        echo -e "  ${RED}🔒 #${PR_NUMBER}${NC}: $PR_TITLE"
    else
        echo -e "  ${YELLOW}#${PR_NUMBER}${NC}: $PR_TITLE"
    fi
    if [ -n "$PR_LABELS" ]; then
        echo -e "     ${BLUE}Labels:${NC} $PR_LABELS"
    fi
done
echo ""

# Ask for confirmation unless dry-run
if [ "$DRY_RUN" = false ]; then
    read -p "Proceed with REBASE merge? (y/n) " -n 1 -r
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

for PR_NUMBER in $SECURITY_PRS; do
    PR_DATA=$(echo "$ALL_PRS" | jq -r ".[] | select(.number == $PR_NUMBER)")
    PR_TITLE=$(echo "$PR_DATA" | jq -r '.title')
    IS_SECURITY=$(echo "$PR_DATA" | jq -r '.labels[]?.name' | grep -q "security" && echo "true" || echo "false")

    if [ "$IS_SECURITY" = "true" ]; then
        echo -e "${RED}🔒 Processing SECURITY PR #$PR_NUMBER...${NC}"
    else
        echo -e "${BLUE}🔄 Processing PR #$PR_NUMBER...${NC}"
    fi
    echo -e "  ${YELLOW}Title:${NC} $PR_TITLE"

    if [ "$DRY_RUN" = true ]; then
        echo -e "  ${GREEN}[DRY RUN]${NC} Would approve and rebase merge PR #$PR_NUMBER"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        # Approve the PR with security note
        echo -e "  ${YELLOW}→${NC} Approving PR..."
        APPROVE_BODY="Auto-approved by security merge script"
        if [ "$IS_SECURITY" = "true" ]; then
            APPROVE_BODY="🔒 **SECURITY UPDATE** - Auto-approved and fast-tracked for merge"
        fi

        if gh pr review "$PR_NUMBER" --repo "$REPO" --approve --body "$APPROVE_BODY" 2>/dev/null; then
            echo -e "  ${GREEN}✓${NC} Approved"
        else
            echo -e "  ${YELLOW}⚠${NC} Already approved or approval failed"
        fi

        # Merge with rebase
        echo -e "  ${YELLOW}→${NC} Merging with rebase..."
        if gh pr merge "$PR_NUMBER" --repo "$REPO" --rebase 2>/dev/null; then
            if [ "$IS_SECURITY" = "true" ]; then
                echo -e "  ${RED}🔒 Successfully merged SECURITY PR #$PR_NUMBER${NC}"
            else
                echo -e "  ${GREEN}✅ Successfully merged PR #$PR_NUMBER${NC}"
            fi
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
if [ "$MERGE_ALL" = false ]; then
    echo -e "${RED}🔒 Security Updates Summary:${NC}"
else
    echo -e "${BLUE}Summary:${NC}"
fi
echo -e "  ${GREEN}✅ Success: ${SUCCESS_COUNT}${NC}"
if [ $FAILED_COUNT -gt 0 ]; then
    echo -e "  ${RED}❌ Failed:  ${FAILED_COUNT}${NC}"
fi
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ $FAILED_COUNT -gt 0 ]; then
    exit 1
fi
