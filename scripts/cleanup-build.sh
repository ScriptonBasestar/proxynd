#!/bin/bash

set -e

echo "🧹 Starting build artifacts cleanup..."

# 빌드 출력물 제거
echo "Removing build outputs..."
rm -f proxynd
rm -f bin/proxynd
rm -f tmp/main
rm -f dist/*

# 로그 파일 정리 (빈 파일들)
echo "Cleaning log files..."
mkdir -p logs
> logs/access.log
> logs/verification-alerts.log
mkdir -p tests/integration/logs
> tests/integration/logs/access.log
> tests/integration/logs/verification-alerts.log

# Go 모듈 캐시 정리 (선택적)
echo "Cleaning Go module cache..."
go clean -modcache 2>/dev/null || true

# 테스트 캐시 정리
echo "Cleaning test cache..."
go clean -testcache

# 빌드 캐시 정리
echo "Cleaning build cache..."
go clean -cache

# 커버리지 파일 정리
echo "Removing coverage files..."
rm -f coverage.out coverage.html
rm -f gosec-report.json
rm -f deps-graph.png

# IDE 설정 파일 정리 (선택적)
echo "Cleaning IDE files..."
rm -rf .vscode/settings.json 2>/dev/null || true
rm -rf .idea/ 2>/dev/null || true

# 임시 스토리지 정리 (개발 환경)
if [ -d "./tmp/storage" ]; then
    echo "Cleaning development storage..."
    rm -rf ./tmp/storage/*
fi

# 빈 디렉토리 정리
echo "Cleaning empty directories..."
find . -type d -empty -not -path "./.git/*" -delete 2>/dev/null || true

echo "✅ Build artifacts cleanup completed!"
