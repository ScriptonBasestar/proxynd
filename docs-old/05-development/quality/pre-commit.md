# Pre-commit Hooks Guide

## Overview

Pre-commit hooks help maintain code quality by automatically running checks before each commit. This ensures that common issues are caught early and fixed before they enter the codebase.

## Installation

### Quick Setup

```bash
# One-time setup
make pre-commit-install
```

### Manual Installation

1. Install pre-commit:
```bash
pip install pre-commit
# or
brew install pre-commit  # macOS
```

2. Install the git hooks:
```bash
pre-commit install
pre-commit install --hook-type commit-msg  # Optional: for commit message checks
```

## Usage

### Automatic Execution

Once installed, pre-commit runs automatically on `git commit`. If any check fails:
1. The commit will be aborted
2. Some issues may be auto-fixed
3. Review the changes and retry the commit

### Manual Execution

```bash
# Run on all files
make pre-commit-run
# or
pre-commit run --all-files

# Run on specific files
pre-commit run --files path/to/file.go

# Run specific hook
pre-commit run go-fmt --all-files
```

### Updating Hooks

```bash
# Update hook versions
make pre-commit-update
# or
pre-commit autoupdate
```

## Configured Hooks

### File Checks
- **trailing-whitespace**: Removes trailing whitespace
- **end-of-file-fixer**: Ensures files end with a newline
- **check-yaml**: Validates YAML syntax
- **check-json**: Validates JSON syntax
- **check-added-large-files**: Prevents large files (>1MB)
- **check-merge-conflict**: Detects merge conflict markers
- **detect-private-key**: Prevents committing private keys
- **mixed-line-ending**: Ensures consistent line endings (LF)

### Go-specific Checks
- **go-fmt**: Formats Go code
- **go-imports**: Organizes imports (with local prefix `proxynd`)
- **go-vet**: Runs Go vet for common issues
- **go-mod-tidy**: Ensures go.mod is tidy
- **go-unit-tests**: Runs short unit tests
- **golangci-lint**: Runs comprehensive linting

### Custom Checks
- **go-no-replacement**: Prevents `replace` directives in go.mod
- **go-generate**: Ensures generated files are committed
- **check-fmt-print**: Prevents fmt.Print usage (use structured logging)
- **todo-format**: Ensures TODOs have issue references

## Bypassing Hooks

### Emergency Bypass

⚠️ **Use only when necessary!**

```bash
# Skip all hooks
git commit --no-verify -m "Emergency fix"

# Skip specific check
SKIP=go-unit-tests git commit -m "Fix without tests"
```

### Temporary Disable

```bash
# Disable for current shell session
export SKIP=golangci-lint,go-unit-tests
```

## Troubleshooting

### Hook Installation Issues

```bash
# Reinstall hooks
pre-commit uninstall
pre-commit install

# Clean cache
pre-commit clean
```

### Slow Performance

1. **First run is slow**: Pre-commit caches environments
2. **Skip expensive checks during development**:
   ```bash
   SKIP=go-unit-tests,golangci-lint git commit
   ```
3. **Run expensive checks separately**:
   ```bash
   git commit --no-verify
   make lint
   make test
   ```

### Auto-fix Not Working

Some hooks auto-fix issues. After fixing:
```bash
# Stage the fixes
git add -u

# Retry commit
git commit
```

### Python/pip Issues

```bash
# Check Python version (3.6+ required)
python3 --version

# Install in user directory
pip3 install --user pre-commit

# Add to PATH if needed
export PATH="$HOME/.local/bin:$PATH"
```

## Configuration

The `.pre-commit-config.yaml` file controls which hooks run.

### Adding New Hooks

```yaml
- repo: https://github.com/example/hooks
  rev: v1.0.0
  hooks:
    - id: new-hook
      args: [--some-arg]
```

### Excluding Files

```yaml
- id: some-hook
  exclude: ^(vendor/|testdata/|.*\.pb\.go)$
```

### Hook Arguments

```yaml
- id: golangci-lint
  args: ['--fix', '--new-from-rev=HEAD~']
```

## Best Practices

1. **Commit often**: Smaller commits = faster checks
2. **Fix immediately**: Don't accumulate linting debt
3. **Review auto-fixes**: Ensure they make sense
4. **Keep hooks updated**: Run `pre-commit autoupdate` periodically
5. **Team alignment**: Ensure all team members use pre-commit

## Integration with CI/CD

Pre-commit can also run in CI:

```yaml
# .github/workflows/pre-commit.yml
name: pre-commit
on: [pull_request]
jobs:
  pre-commit:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-python@v4
    - uses: pre-commit/action@v3.0.0
```

## Hook Details

### go-fmt
- **Purpose**: Format Go code
- **Auto-fix**: Yes
- **Command**: `go fmt`

### go-imports
- **Purpose**: Organize imports
- **Auto-fix**: Yes
- **Command**: `goimports -w -local proxynd`

### go-vet
- **Purpose**: Find suspicious constructs
- **Auto-fix**: No
- **Command**: `go vet`

### go-mod-tidy
- **Purpose**: Clean up go.mod/go.sum
- **Auto-fix**: Yes
- **Command**: `go mod tidy`

### golangci-lint
- **Purpose**: Comprehensive linting
- **Auto-fix**: Yes (some issues)
- **Command**: `golangci-lint run --fix`

### go-unit-tests
- **Purpose**: Run tests before commit
- **Auto-fix**: No
- **Command**: `go test -short`

## Advanced Usage

### Custom Hooks

Add to `.pre-commit-config.yaml`:

```yaml
- repo: local
  hooks:
    - id: my-custom-check
      name: My custom check
      entry: ./scripts/my-check.sh
      language: script
      files: \.go$
```

### Stage-specific Hooks

```yaml
- id: commitizen
  stages: [commit-msg]  # Only run on commit message
```

### Language-specific Config

```yaml
default_language_version:
  golang: 1.24
  python: python3.12
```

## Summary

Pre-commit hooks are a powerful tool for maintaining code quality. They:
- Catch issues before they're committed
- Enforce consistent code style
- Run automatically (no need to remember)
- Can fix many issues automatically
- Are configurable and extensible

Start with the default configuration and adjust based on your team's needs.
