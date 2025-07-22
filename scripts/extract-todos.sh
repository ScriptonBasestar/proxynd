#!/bin/bash

echo "📋 Extracting TODO/FIXME items..."

# TODO 분석 파일 생성
grep -rn "TODO\|FIXME\|XXX\|HACK" --include="*.go" . > todos_extracted.txt

# 카테고리별 분류
echo "## 즉시 해결 가능" > todos_categorized.md
echo "" >> todos_categorized.md

echo "## GitHub Issues 변환 필요" >> todos_categorized.md
echo "" >> todos_categorized.md

echo "## 삭제 가능 (오래된 것)" >> todos_categorized.md
echo "" >> todos_categorized.md

# TODO 내용 분석 및 분류
while IFS= read -r line; do
    file=$(echo "$line" | cut -d: -f1)
    line_num=$(echo "$line" | cut -d: -f2)
    content=$(echo "$line" | cut -d: -f3-)

    echo "- **$file:$line_num** $content" >> todos_categorized.md
done < todos_extracted.txt

echo "📄 TODO analysis saved to: todos_categorized.md"
