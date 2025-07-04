#!/bin/bash

# Script to run all service tests with coverage

set -e

echo "Running service tests with coverage..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to run tests for a package
run_package_tests() {
    local package=$1
    echo -e "\n${GREEN}Testing ${package}${NC}"
    go test -v -cover -coverprofile=/tmp/coverage_$(echo $package | tr '/' '_').out $package
}

# Run tests for each service package
packages=(
    "./internal/services/config"
    "./internal/services/proxy"
    "./internal/services/adapters"
)

for pkg in "${packages[@]}"; do
    run_package_tests $pkg
done

# Merge coverage reports
echo -e "\n${GREEN}Merging coverage reports...${NC}"
echo "mode: set" > /tmp/coverage_merged.out
for pkg in "${packages[@]}"; do
    coverage_file="/tmp/coverage_$(echo $pkg | tr '/' '_').out"
    if [ -f "$coverage_file" ]; then
        tail -n +2 "$coverage_file" >> /tmp/coverage_merged.out
    fi
done

# Generate coverage report
echo -e "\n${GREEN}Coverage Report:${NC}"
go tool cover -func=/tmp/coverage_merged.out | tail -n 1

# Generate HTML coverage report
go tool cover -html=/tmp/coverage_merged.out -o /tmp/coverage_services.html
echo -e "\n${GREEN}HTML coverage report generated at: /tmp/coverage_services.html${NC}"

# Run race detection tests
echo -e "\n${GREEN}Running race detection tests...${NC}"
for pkg in "${packages[@]}"; do
    echo -e "Race testing ${pkg}"
    go test -race $pkg
done

echo -e "\n${GREEN}All service tests completed!${NC}"