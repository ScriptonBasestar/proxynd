#!/bin/bash

echo "🔍 Finding files with complex comment blocks..."

# 10줄 이상 연속 주석 블록 찾기
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" -not -path "./backup/*" -exec awk '
/^[[:space:]]*\/\// {count++; if (count == 1) start=NR}
!/^[[:space:]]*\/\// {
    if (count >= 10) print FILENAME":"start"-"NR-1" ("count" lines)"
    count=0
}
END {if (count >= 10) print FILENAME":"start"-"NR" ("count" lines)"}
' {} \; > complex_comments.txt

echo "📄 Complex comment blocks saved to: complex_comments.txt"

# 알고리즘 설명 주석 찾기 (유지해야 할 가능성)
grep -rn "algorithm\|implementation\|optimization\|performance\|security" --include="*.go" . | grep "^[[:space:]]*//.*" > important_comments.txt

echo "📄 Potentially important comments saved to: important_comments.txt"

# 라이선스 헤더 확인
echo "📄 License header check:"
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" -not -path "./backup/*" -exec head -5 {} \; | grep -i "license\|copyright" | wc -l
