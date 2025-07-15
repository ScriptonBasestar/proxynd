# Quality Targets Guide

## Overview

This guide documents the quality-related Makefile targets added to the ProxyND project. These targets help maintain code quality, security, and consistency across the codebase.

## Quality Assurance Targets

### Core Quality Commands

#### `make quality`
Runs all essential quality checks:
- Code formatting (`fmt`)
- Linting (`lint`)
- Test coverage analysis (`test-coverage`)

**Usage:**
```bash
make quality
```

#### `make quality-fix`
Automatically fixes code quality issues:
- Applies code formatting
- Fixes auto-fixable linting issues

**Usage:**
```bash
make quality-fix
```

#### `make check`
Quick quality check for rapid feedback:
- Runs linting
- Executes unit tests

**Usage:**
```bash
make check
```

#### `make check-all`
Comprehensive quality verification:
- Full linting
- All test suites
- Coverage analysis

**Usage:**
```bash
make check-all
```

## Security Targets

### `make security`
Complete security audit:
- Dependency vulnerability scanning
- Static security analysis

**Usage:**
```bash
make security
```

### `make security-deps`
Scans dependencies for known vulnerabilities using Nancy:
- Analyzes Go module dependencies
- Reports CVEs and security issues

**Usage:**
```bash
make security-deps
```

### `make security-code`
Static security analysis using Gosec:
- Identifies security antipatterns
- Checks for common vulnerabilities
- Generates detailed JSON report

**Usage:**
```bash
make security-code
# Report saved to: gosec-report.json
```

## Code Analysis Targets

### `make analyze`
Comprehensive code analysis:
- Cyclomatic complexity analysis
- Unused code detection

**Usage:**
```bash
make analyze
```

### `make analyze-complexity`
Identifies complex functions:
- Reports functions with cyclomatic complexity > 10
- Helps identify refactoring candidates

**Usage:**
```bash
make analyze-complexity
```

### `make analyze-unused`
Finds unused code:
- Detects unused functions, types, and variables
- Helps reduce code bloat

**Usage:**
```bash
make analyze-unused
```

## Dependency Management

### `make deps`
Manages Go module dependencies:
- Downloads dependencies
- Tidies go.mod and go.sum
- Verifies module integrity

**Usage:**
```bash
make deps
```

### `make deps-update`
Updates all dependencies to latest versions:
- Updates direct and indirect dependencies
- Cleans up go.mod

**Usage:**
```bash
make deps-update
```

### `make deps-graph`
Generates visual dependency graph:
- Creates PNG image of dependency tree
- Requires graphviz installed

**Usage:**
```bash
make deps-graph
# Output: deps-graph.png
```

## Documentation Targets

### `make docs`
Generates and serves documentation:
- Generates godoc documentation
- Starts local documentation server

**Usage:**
```bash
make docs
# View at: http://localhost:6060/pkg/proxynd/
```

## Build Targets

### `make build`
Builds the ProxyND binary:
- Compiles for current platform
- Verbose output

**Usage:**
```bash
make build
# Output: ./proxynd
```

### `make build-all`
Cross-platform compilation:
- Linux (amd64, arm64)
- macOS (amd64, arm64)

**Usage:**
```bash
make build-all
# Output: dist/proxynd-{os}-{arch}
```

## Validation Targets

### `make validate`
Validates project files:
- Configuration file syntax
- Go module integrity

**Usage:**
```bash
make validate
```

## Utility Targets

### `make clean`
Removes all generated files:
- Build artifacts
- Test coverage reports
- Generated documentation
- Mock files

**Usage:**
```bash
make clean
```

### `make version`
Displays current version:
- Based on git tags
- Shows dirty state

**Usage:**
```bash
make version
```

### `make help`
Shows all available commands:
- Organized by category
- Brief descriptions

**Usage:**
```bash
make help
```

## Development Workflows

### Daily Development
```bash
# Start your day
make dev

# Before committing
make check

# Fix issues
make quality-fix
```

### Before Pull Request
```bash
# Run comprehensive checks
make check-all

# Verify security
make security

# Analyze code quality
make analyze
```

### CI/CD Pipeline
```bash
# Typical CI flow
make ci
```

## Tool Requirements

The quality targets automatically install required tools:

- **golangci-lint**: Comprehensive Go linter
- **goimports**: Import formatting
- **nancy**: Dependency vulnerability scanner
- **gosec**: Security analyzer
- **gocyclo**: Complexity analyzer
- **unused**: Dead code detector
- **godepgraph**: Dependency visualizer
- **godoc**: Documentation generator

## Best Practices

1. **Run `make check` before every commit**
   - Catches issues early
   - Maintains code quality

2. **Use `make quality-fix` for automatic fixes**
   - Saves time on formatting
   - Ensures consistency

3. **Run `make security` weekly**
   - Stay on top of vulnerabilities
   - Maintain security posture

4. **Use `make analyze` during refactoring**
   - Identify complex code
   - Find dead code

5. **Run `make deps-update` carefully**
   - Test thoroughly after updates
   - Check for breaking changes

## Integration with Pre-commit

These targets complement pre-commit hooks:
- Pre-commit: Fast, automatic checks
- Make targets: Comprehensive, on-demand analysis

## Troubleshooting

### Tool Installation Issues
If a tool fails to install:
```bash
# Manual installation example
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Performance
For faster feedback during development:
```bash
# Quick checks only
make check

# Skip coverage for speed
make test-unit
```

### CI Integration
Example GitHub Actions workflow:
```yaml
- name: Quality Checks
  run: make ci
```

## Summary

The quality targets provide a comprehensive toolkit for maintaining high code quality in the ProxyND project. They cover:

- ✅ Code formatting and style
- ✅ Static analysis and linting
- ✅ Security vulnerability scanning
- ✅ Test coverage analysis
- ✅ Dependency management
- ✅ Documentation generation
- ✅ Cross-platform builds

Regular use of these targets ensures a healthy, maintainable codebase.