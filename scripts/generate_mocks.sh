#!/bin/bash

# Mock 생성 스크립트

set -e

echo "🔧 Installing mockery if not present..."
if ! command -v mockery &> /dev/null; then
    echo "Installing mockery..."
    go install github.com/vektra/mockery/v2@latest
fi

echo "📦 Generating mocks for all interfaces..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Generate mocks using mockery config
echo -e "${YELLOW}Running mockery with config file...${NC}"
mockery --config .mockery.yaml

# Count generated files
MOCK_COUNT=$(find . -path "*/mocks/*.go" -type f | wc -l)

echo -e "${GREEN}✅ Generated ${MOCK_COUNT} mock files${NC}"

# List generated mock files
echo -e "\n${YELLOW}Generated mock files:${NC}"
find . -path "*/mocks/*.go" -type f | sort

echo -e "\n${GREEN}✅ Mock generation complete!${NC}"