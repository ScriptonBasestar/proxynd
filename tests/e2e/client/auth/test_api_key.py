"""
API Key Authentication Tests

Spec: X-API-Key header authentication works for protected endpoints.

Manual Test:
    # Without API key (should fail)
    curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/proxy/npm-private/test
    # Expected: 401

    # With valid API key
    curl -H "X-API-Key: <api-key>" http://localhost:8080/proxy/npm-private/test
    # Expected: 200 or 404 (but not 401)

    # With invalid API key
    curl -H "X-API-Key: invalid-key" http://localhost:8080/proxy/npm-private/test
    # Expected: 401
"""

import pytest

from client import ProxyNDClient


class TestApiKeyValid:
    """Valid API Key tests."""

    @pytest.mark.private
    def test_valid_api_key_accepted(self, api_key_client):
        """
        Spec: Valid API key grants access.

        Manual:
            curl -H "X-API-Key: <valid-key>" http://localhost:8080/api/status
        """
        response = api_key_client.get("/api/status")

        # Should get valid response (not 401)
        assert response.status_code != 401


class TestApiKeyInvalid:
    """Invalid API Key tests."""

    @pytest.mark.private
    def test_invalid_api_key_rejected(self, base_url):
        """
        Spec: Invalid API key returns 401.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                -H "X-API-Key: invalid-key" \
                http://localhost:8080/proxy/npm-private/test
        """
        client = ProxyNDClient(
            base_url=base_url,
            api_key="invalid-api-key",
        )
        response = client.get("/proxy/npm-private/test-package")

        assert response.status_code in [401, 403]

    @pytest.mark.private
    def test_empty_api_key_rejected(self, base_url):
        """
        Spec: Empty API key returns 401.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                -H "X-API-Key: " \
                http://localhost:8080/proxy/npm-private/test
        """
        client = ProxyNDClient(
            base_url=base_url,
            api_key="",
        )
        response = client.get("/proxy/npm-private/test-package")

        assert response.status_code in [401, 403]
