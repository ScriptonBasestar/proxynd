#!/bin/bash

# Setup script for pre-commit hooks

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Setting up pre-commit hooks for ProxyND${NC}"
echo "========================================"

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}❌ Python 3 is required for pre-commit${NC}"
    echo "Please install Python 3 and try again"
    exit 1
fi

# Check if pip is installed
if ! command -v pip3 &> /dev/null; then
    echo -e "${RED}❌ pip3 is required${NC}"
    echo "Please install pip3 and try again"
    exit 1
fi

# Install pre-commit
echo -e "${YELLOW}Installing pre-commit...${NC}"
if command -v pre-commit &> /dev/null; then
    echo -e "${GREEN}✓ pre-commit is already installed${NC}"
    pre-commit --version
else
    pip3 install --user pre-commit
    echo -e "${GREEN}✓ pre-commit installed${NC}"
fi

# Ensure pre-commit is in PATH
if ! command -v pre-commit &> /dev/null; then
    echo -e "${YELLOW}Adding pre-commit to PATH...${NC}"
    export PATH="$HOME/.local/bin:$PATH"
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
    echo -e "${GREEN}✓ Added to PATH${NC}"
fi

# Install the git hook scripts
echo -e "${YELLOW}Installing git hooks...${NC}"
pre-commit install
echo -e "${GREEN}✓ Git hooks installed${NC}"

# Install hooks for commit messages (optional)
echo -e "${YELLOW}Installing commit-msg hooks...${NC}"
pre-commit install --hook-type commit-msg
echo -e "${GREEN}✓ Commit message hooks installed${NC}"

# Run pre-commit on all files (optional, for initial check)
echo -e "${YELLOW}Running initial checks on all files...${NC}"
echo -e "${YELLOW}This may take a few minutes on first run...${NC}"
if pre-commit run --all-files; then
    echo -e "${GREEN}✅ All checks passed!${NC}"
else
    echo -e "${YELLOW}⚠️  Some checks failed. This is normal for initial setup.${NC}"
    echo -e "${YELLOW}   Files may have been auto-fixed. Review and commit the changes.${NC}"
fi

# Show summary
echo -e "\n${GREEN}✅ Pre-commit setup complete!${NC}"
echo -e "\nPre-commit will now run automatically on git commit."
echo -e "To run manually: ${GREEN}pre-commit run --all-files${NC}"
echo -e "To update hooks: ${GREEN}pre-commit autoupdate${NC}"
echo -e "To bypass (emergency only): ${GREEN}git commit --no-verify${NC}"

# Show enabled hooks
echo -e "\n${YELLOW}Enabled hooks:${NC}"
echo "- Go formatting (go fmt)"
echo "- Import organization (goimports)"
echo "- Go vet checks"
echo "- Go mod tidy"
echo "- Unit tests (short mode)"
echo "- Linting (golangci-lint)"
echo "- File checks (trailing spaces, line endings, etc.)"
echo "- Security checks (private keys, etc.)"
echo "- Custom checks (TODOs, fmt.Print, go.mod replace)"

echo -e "\n${GREEN}Happy coding! 🚀${NC}"
