#!/bin/bash
# Format Go files passed as arguments or via CLAUDE_FILES environment variable

set -e

# Color codes
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'

# Get files to format
if [ $# -gt 0 ]; then
    # Use command line arguments
    FILES="$@"
elif [ -n "$CLAUDE_FILES" ]; then
    # Use CLAUDE_FILES environment variable
    FILES="$CLAUDE_FILES"
else
    echo -e "${RED}❌ Error: No files specified${RESET}"
    echo -e "${YELLOW}Usage: $0 file1.go file2.go ...${RESET}"
    echo -e "${YELLOW}Or set CLAUDE_FILES environment variable${RESET}"
    exit 1
fi

echo -e "${CYAN}🔄 Formatting Go files...${RESET}"

# Check if tools are installed
if ! command -v gofumpt &> /dev/null; then
    echo -e "${YELLOW}Installing gofumpt...${RESET}"
    go install mvdan.cc/gofumpt@latest
fi

if ! command -v goimports &> /dev/null; then
    echo -e "${YELLOW}Installing goimports...${RESET}"
    go install golang.org/x/tools/cmd/goimports@latest
fi

# Format each file
for file in $FILES; do
    if [ -n "$file" ]; then
        if [ ! -f "$file" ]; then
            echo -e "${RED}❌ Error: File '$file' does not exist${RESET}"
            continue
        fi
        
        if ! echo "$file" | grep -q "\.go$"; then
            echo -e "${YELLOW}⚠️  Warning: File '$file' is not a Go file (.go extension), skipping${RESET}"
            continue
        fi
        
        echo -e "${CYAN}📝 Formatting file: $file${RESET}"
        echo "  1. Running gofumpt..."
        if gofumpt -w "$file"; then
            echo -e "${GREEN}  ✓ gofumpt completed${RESET}"
        else
            echo -e "${RED}  ✗ gofumpt failed${RESET}"
        fi
        
        echo "  2. Running goimports..."
        if goimports -w -local proxynd "$file"; then
            echo -e "${GREEN}  ✓ goimports completed${RESET}"
        else
            echo -e "${RED}  ✗ goimports failed${RESET}"
        fi
        
        echo -e "${GREEN}✅ File '$file' formatted successfully!${RESET}"
    fi
done

echo -e "${GREEN}🎉 All files processed!${RESET}"