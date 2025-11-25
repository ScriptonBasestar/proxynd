"""
Global pytest fixtures - environment-driven configuration.

Environment Variables:
    PROXYND_URL       - Base URL (default: http://localhost:8080)
    PROXYND_USERNAME  - Basic auth username (default: testuser)
    PROXYND_PASSWORD  - Basic auth password (default: testpass)
    PROXYND_TOKEN     - Bearer token (JWT)
    PROXYND_API_KEY   - API key
"""

import os

import pytest

from client import ProxyNDClient


# =============================================================================
# Environment Configuration
# =============================================================================


@pytest.fixture(scope="session")
def base_url() -> str:
    """ProxyND server URL from environment."""
    return os.getenv("PROXYND_URL", "http://localhost:8080")


# =============================================================================
# Client Fixtures
# =============================================================================


@pytest.fixture
def client(base_url) -> ProxyNDClient:
    """Anonymous client (public access, no authentication)."""
    return ProxyNDClient(base_url=base_url)


@pytest.fixture
def basic_auth_client(base_url) -> ProxyNDClient:
    """Client with Basic Authentication."""
    return ProxyNDClient(
        base_url=base_url,
        username=os.getenv("PROXYND_USERNAME", "testuser"),
        password=os.getenv("PROXYND_PASSWORD", "testpass"),
    )


@pytest.fixture
def token_client(base_url) -> ProxyNDClient:
    """Client with Bearer Token (JWT) authentication."""
    token = os.getenv("PROXYND_TOKEN")
    if not token:
        pytest.skip("PROXYND_TOKEN not set")
    return ProxyNDClient(base_url=base_url, token=token)


@pytest.fixture
def api_key_client(base_url) -> ProxyNDClient:
    """Client with API Key authentication."""
    api_key = os.getenv("PROXYND_API_KEY")
    if not api_key:
        pytest.skip("PROXYND_API_KEY not set")
    return ProxyNDClient(base_url=base_url, api_key=api_key)


# =============================================================================
# Test Markers Configuration
# =============================================================================


def pytest_configure(config):
    """Register custom markers."""
    config.addinivalue_line("markers", "public: tests for public repositories (no auth)")
    config.addinivalue_line("markers", "private: tests requiring authentication")
    config.addinivalue_line("markers", "slow: tests that take > 5 seconds")
    config.addinivalue_line("markers", "npm: NPM registry tests")
    config.addinivalue_line("markers", "maven: Maven repository tests")
    config.addinivalue_line("markers", "pypi: PyPI repository tests")
    config.addinivalue_line("markers", "docker: Docker registry tests")


# =============================================================================
# Skip Conditions
# =============================================================================


@pytest.fixture(scope="session")
def server_available(base_url) -> bool:
    """Check if ProxyND server is running."""
    client = ProxyNDClient(base_url=base_url)
    return client.health_check()


@pytest.fixture(autouse=True)
def skip_if_server_unavailable(request, server_available):
    """Skip tests if server is not available."""
    if not server_available:
        pytest.skip(f"ProxyND server not available")
