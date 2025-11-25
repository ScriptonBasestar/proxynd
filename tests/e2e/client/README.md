# ProxyND E2E Test Client

Python-based E2E tests for ProxyND proxy server.
Each test includes manual curl commands for easy verification.

## Quick Start

```bash
cd proxynd-core/tests/e2e/client

# Install dependencies
uv sync

# Run all public tests
./run_tests.sh --public

# Run specific package manager
./run_tests.sh --npm
./run_tests.sh --maven
```

## Directory Structure

```
client/
├── pyproject.toml          # Python project config
├── client.py               # HTTP client wrapper
├── conftest.py             # Pytest fixtures
├── run_tests.sh            # Test runner script
├── manual_tests.sh         # Manual curl test commands
│
├── npm/                    # NPM proxy tests
│   ├── test_public_download.py
│   ├── test_tarball_download.py
│   ├── test_private_access.py
│   ├── test_cache_behavior.py
│   └── test_headers.py
│
├── maven/                  # Maven proxy tests
│   ├── test_public_download.py
│   ├── test_jar_download.py
│   ├── test_private_access.py
│   ├── test_gradle_compat.py
│   └── test_headers.py
│
└── auth/                   # Authentication tests
    ├── test_basic_auth.py
    ├── test_bearer_token.py
    └── test_api_key.py
```

## Running Tests

### Using run_tests.sh

```bash
# All public tests (no auth required)
./run_tests.sh --public

# All private tests (auth required)
./run_tests.sh --private

# NPM tests only
./run_tests.sh --npm

# Maven tests only
./run_tests.sh --maven

# Combine options
./run_tests.sh --npm --public -v

# Specific test file
./run_tests.sh npm/test_public_download.py

# With custom server URL
./run_tests.sh -u http://proxynd.local:8080 --public
```

### Using pytest directly

```bash
# All tests
uv run pytest

# By marker
uv run pytest -m public
uv run pytest -m "npm and public"
uv run pytest -m "maven and slow"

# By directory
uv run pytest npm/
uv run pytest maven/
uv run pytest auth/

# Specific test
uv run pytest npm/test_public_download.py::TestNpmPublicDownload::test_get_package_metadata -v
```

## Manual Testing

### Using manual_tests.sh

```bash
# List all available tests
./manual_tests.sh

# Run specific test
./manual_tests.sh npm_public_metadata
./manual_tests.sh maven_pom
./manual_tests.sh auth_basic_valid

# Run all manual tests
./manual_tests.sh all
```

### Direct curl Commands

Each test file includes manual curl commands in docstrings:

```bash
# NPM package metadata
curl http://localhost:8080/proxy/npm/express | jq '.name'

# NPM scoped package
curl http://localhost:8080/proxy/npm/@types/node | jq '.name'

# NPM tarball
curl -o express.tgz http://localhost:8080/proxy/npm/express/-/express-4.18.2.tgz
file express.tgz  # should show "gzip compressed data"

# Maven POM
curl http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom

# Maven JAR
curl -o junit.jar http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar
file junit.jar  # should show "Zip archive data"

# Private access (unauthorized)
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/proxy/npm-private/test
# Expected: 401 or 403

# Private access (Basic Auth)
curl -u testuser:testpass http://localhost:8080/proxy/npm-private/test

# Private access (Bearer Token)
curl -H "Authorization: Bearer <jwt-token>" http://localhost:8080/proxy/npm-private/test
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PROXYND_URL` | Server URL | `http://localhost:8080` |
| `PROXYND_USERNAME` | Basic auth username | `testuser` |
| `PROXYND_PASSWORD` | Basic auth password | `testpass` |
| `PROXYND_TOKEN` | Bearer token (JWT) | - |
| `PROXYND_API_KEY` | API key | - |

## Test Markers

| Marker | Description |
|--------|-------------|
| `@pytest.mark.public` | No authentication required |
| `@pytest.mark.private` | Authentication required |
| `@pytest.mark.slow` | Takes > 5 seconds (downloads) |
| `@pytest.mark.npm` | NPM proxy tests |
| `@pytest.mark.maven` | Maven proxy tests |

## Test Specs by File

### NPM Tests

| File | Spec |
|------|------|
| `test_public_download.py` | Public NPM packages downloadable |
| `test_tarball_download.py` | NPM tarballs (.tgz) downloadable |
| `test_private_access.py` | Private NPM requires authentication |
| `test_cache_behavior.py` | Cache hits faster than misses |
| `test_headers.py` | Correct Content-Type headers |

### Maven Tests

| File | Spec |
|------|------|
| `test_public_download.py` | Public Maven artifacts downloadable |
| `test_jar_download.py` | JAR files downloadable (ZIP format) |
| `test_private_access.py` | Private Maven requires authentication |
| `test_gradle_compat.py` | Works with Gradle dependency resolution |
| `test_headers.py` | Correct Content-Type headers |

### Auth Tests

| File | Spec |
|------|------|
| `test_basic_auth.py` | HTTP Basic Auth works |
| `test_bearer_token.py` | Bearer token (JWT) works |
| `test_api_key.py` | X-API-Key header works |

## Adding New Tests

1. Choose appropriate directory (`npm/`, `maven/`, `auth/`, or create new)
2. Create test file: `test_<feature>.py`
3. Add markers and manual curl commands in docstrings:

```python
"""
Feature Description

Spec: What this tests.

Manual Test:
    curl http://localhost:8080/...
"""

import pytest

class TestFeature:
    @pytest.mark.public  # or @pytest.mark.private
    @pytest.mark.npm     # or @pytest.mark.maven
    def test_something(self, client):
        """
        Spec: Specific behavior being tested.

        Manual:
            curl http://localhost:8080/...
        """
        response = client.get("/proxy/...")
        assert response.status_code == 200
```

## Troubleshooting

### Server not available

```bash
# Check if server is running
curl http://localhost:8080/healthz

# Start development server
cd proxynd-devbox
docker-compose -f docker-compose.dev.yml up proxynd-core -d
```

### Token/API Key tests skipped

```bash
# Set credentials for private tests
export PROXYND_TOKEN="your-jwt-token"
export PROXYND_API_KEY="your-api-key"
./run_tests.sh --private
```

### Cache tests failing

Cache behavior tests may be flaky depending on network conditions.
Run with `-v` for timing information:

```bash
./run_tests.sh -v npm/test_cache_behavior.py
```
