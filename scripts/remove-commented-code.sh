#!/bin/bash

set -e

echo "🧹 Starting commented code cleanup..."

# 백업 생성
BACKUP_DIR="backup/commented-code-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# 제거할 패턴들 정의 - 보수적 접근법
declare -a PATTERNS=(
    "^[[:space:]]*//[[:space:]]*func[[:space:]]"
    "^[[:space:]]*//[[:space:]]*type[[:space:]].*struct"
    "^[[:space:]]*//[[:space:]]*var[[:space:]]"
    "^[[:space:]]*//[[:space:]]*const[[:space:]]"
)

# 위험한 패턴들 (수동 검토 필요)
declare -a DANGEROUS_PATTERNS=(
    "^[[:space:]]*//[[:space:]]*import[[:space:]]"
    "^[[:space:]]*//[[:space:]]*package[[:space:]]"
)

# Go 파일들 처리
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" -not -path "./backup/*" | while read -r file; do
    echo "Processing: $file"
    
    # 백업
    cp "$file" "$BACKUP_DIR/$(basename $file).backup"
    
    # 임시 파일 생성
    temp_file=$(mktemp)
    cp "$file" "$temp_file"
    
    # 안전한 패턴들만 제거
    for pattern in "${PATTERNS[@]}"; do
        grep -v "$pattern" "$temp_file" > "${temp_file}.tmp"
        mv "${temp_file}.tmp" "$temp_file"
    done
    
    # 연속된 빈 줄 정리 (3줄 이상 → 2줄로)
    awk '/^[[:space:]]*$/ {blank++} !/^[[:space:]]*$/ {
        if (blank > 2) print ""
        else for(i=0; i<blank; i++) print ""
        blank=0; print
    } END {if (blank > 2) print ""; else for(i=0; i<blank; i++) print ""}' "$temp_file" > "${temp_file}.final"
    
    # 원본 파일 업데이트
    mv "${temp_file}.final" "$file"
    rm -f "$temp_file" "${temp_file}.tmp"
done

echo "✅ Commented code removal completed!"
echo "📁 Backups saved in: $BACKUP_DIR"