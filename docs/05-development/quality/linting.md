# ProxyND Linting Guide

## Overview

ProxyND uses [golangci-lint](https://golangci-lint.run/) for comprehensive code quality checks. The configuration is designed to enforce consistent code style and catch common bugs while being practical for development.

## Quick Start

### Run Linting

```bash
# Run all linters
make lint

# Auto-fix issues (where possible)
make lint-fix

# Check only new code (changes since last commit)
make lint-new

# Run for CI environment
make lint-ci
```

### Install golangci-lint

```bash
# Automatic installation via Makefile
make install-golangci-lint

# Manual installation
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Via brew (macOS)
brew install golangci-lint
```

## Configuration

The `.golangci.yml` file in the project root contains all linting rules. Key configurations:

### Enabled Linters

#### Default Go Linters
- `errcheck` - Check for unchecked errors
- `gosimple` - Simplify code
- `govet` - Go vet on steroids
- `ineffassign` - Detect ineffectual assignments
- `staticcheck` - Advanced static analysis
- `typecheck` - Type checking
- `unused` - Find unused code

#### Code Quality Linters
- `bodyclose` - Check HTTP response body is closed
- `dupl` - Code duplication detection
- `goconst` - Find repeated strings that could be constants
- `gocritic` - Opinionated linter with many checks
- `gocyclo` - Cyclomatic complexity checker
- `gofmt` - Check code formatting
- `goimports` - Check import formatting
- `gosec` - Security issues detection
- `misspell` - Spell checker
- `revive` - Fast, configurable linter (golint replacement)

#### Best Practices
- `exhaustive` - Check exhaustiveness of enum switches
- `exportloopref` - Check loop variable capture issues
- `noctx` - Find HTTP requests without context
- `prealloc` - Find slice pre-allocation opportunities
- `sqlclosecheck` - Check sql.Rows and sql.Stmt are closed

### Linter Settings

#### Line Length
```yaml
lll:
  line-length: 120
```

#### Cyclomatic Complexity
```yaml
gocyclo:
  min-complexity: 20
```

#### Duplicate Code
```yaml
dupl:
  threshold: 150
```

#### Security
```yaml
gosec:
  severity: medium
  confidence: medium
```

## Common Issues and Fixes

### 1. Unchecked Errors

**Issue**: `errcheck: Error return value is not checked`

**Fix**:
```go
// Bad
someFunction()

// Good
if err := someFunction(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Or explicitly ignore
_ = someFunction()
```

### 2. Ineffectual Assignment

**Issue**: `ineffassign: ineffectual assignment to err`

**Fix**:
```go
// Bad
err := doSomething()
err = doSomethingElse() // Previous err value never used

// Good
if err := doSomething(); err != nil {
    return err
}
if err := doSomethingElse(); err != nil {
    return err
}
```

### 3. Code Duplication

**Issue**: `dupl: duplicate of code`

**Fix**: Extract common code into a function:
```go
// Bad
func processA() {
    // 20 lines of code
}
func processB() {
    // Same 20 lines of code
}

// Good
func processCommon() {
    // 20 lines of code
}
func processA() {
    processCommon()
}
func processB() {
    processCommon()
}
```

### 4. Unused Variables

**Issue**: `unused: variable 'x' is unused`

**Fix**:
```go
// Bad
x := getValue()
// x never used

// Good
_ = getValue() // If you need the side effect
// Or remove the line entirely
```

### 5. Long Lines

**Issue**: `lll: line is 121 characters`

**Fix**:
```go
// Bad
func veryLongFunctionNameWithManyParametersAndReturnValues(param1 string, param2 int, param3 bool) (string, error) {

// Good
func veryLongFunctionNameWithManyParametersAndReturnValues(
    param1 string,
    param2 int,
    param3 bool,
) (string, error) {
```

## Excluding Issues

### Inline Exclusion

```go
//nolint:errcheck // We don't care about this error
someFunction()

//nolint:gosec // This is safe in our context
/* #nosec G104 */
exec.Command("ls", userInput)
```

### File-level Exclusion

```go
//nolint:gocyclo,gocritic // This file has complex but necessary logic
package complex
```

### Configuration Exclusion

In `.golangci.yml`:
```yaml
issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - dupl
        - gosec
```

## Integration with CI/CD

### GitHub Actions

The project includes `.github/workflows/lint.yml` for automatic linting on:
- Push to main/develop branches
- Pull requests

### Pre-commit Hook

To run linting before each commit:

```bash
# Create pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/sh
make lint-new
EOF

chmod +x .git/hooks/pre-commit
```

## IDE Integration

### VS Code

Install the Go extension and add to `settings.json`:
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": [
    "--fast"
  ]
}
```

### GoLand/IntelliJ

1. Go to Settings → Go → Linters
2. Enable golangci-lint
3. Set path to `.golangci.yml`

## Performance Tips

### Fast Mode

For quick checks during development:
```bash
golangci-lint run --fast
```

### Specific Linters

Run only specific linters:
```bash
golangci-lint run --disable-all -E errcheck,govet
```

### Specific Files

Check specific files or directories:
```bash
golangci-lint run ./internal/services/...
```

## Troubleshooting

### Linter Not Found

```bash
# Ensure golangci-lint is in PATH
which golangci-lint

# If not found, reinstall
make install-golangci-lint
```

### Out of Memory

For large codebases, increase memory:
```bash
GOGC=20 golangci-lint run
```

### Slow Performance

1. Use `--fast` flag for development
2. Exclude vendor and generated files
3. Run on specific packages

### False Positives

1. Check if it's really a false positive
2. Use inline `//nolint` with explanation
3. Update `.golangci.yml` exclusion rules
4. Report to golangci-lint if it's a bug

## Best Practices

1. **Run before commit**: Use pre-commit hooks or run manually
2. **Fix incrementally**: Use `--new-from-rev` for existing codebases
3. **Don't disable linters globally**: Use targeted exclusions
4. **Document exclusions**: Always explain why you're disabling a check
5. **Keep config updated**: Regularly update golangci-lint and review settings
6. **Team agreement**: Ensure the team agrees on linting rules

## Additional Resources

- [golangci-lint documentation](https://golangci-lint.run/)
- [Linter descriptions](https://golangci-lint.run/usage/linters/)
- [Configuration options](https://golangci-lint.run/usage/configuration/)
- [CI/CD integration](https://golangci-lint.run/usage/integrations/)