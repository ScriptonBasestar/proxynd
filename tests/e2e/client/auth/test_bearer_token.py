"""
Bearer Token (JWT) Authentication Tests

Spec: Bearer token authentication works for protected endpoints.

Manual Test:
    # Without token (should fail)
    curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/proxy/npm-private/test
    # Expected: 401

    # With valid token
    curl -H "Authorization: Bearer <jwt-token>" http://localhost:8080/proxy/npm-private/test
    # Expected: 200 or 404 (but not 401)

    # With invalid token
    curl -H "Authorization: Bearer invalid-token" http://localhost:8080/proxy/npm-private/test
    # Expected: 401
"""

import pytest

from client import ProxyNDClient


class TestBearerTokenValid:
    """Valid Bearer Token tests."""

    @pytest.mark.private
    def test_valid_token_accepted(self, token_client):
        """
        Spec: Valid JWT token grants access.

        Manual:
            curl -H "Authorization: Bearer <valid-jwt>" \
                http://localhost:8080/api/status
        """
        response = token_client.get("/api/status")

        # Should get valid response (not 401)
        assert response.status_code != 401


class TestBearerTokenInvalid:
    """Invalid Bearer Token tests."""

    @pytest.mark.private
    def test_invalid_token_rejected(self, base_url):
        """
        Spec: Invalid JWT token returns 401.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer invalid-token" \
                http://localhost:8080/proxy/npm-private/test
        """
        client = ProxyNDClient(
            base_url=base_url,
            token="invalid-jwt-token",
        )
        response = client.get("/proxy/npm-private/test-package")

        assert response.status_code in [401, 403]

    @pytest.mark.private
    def test_malformed_token_rejected(self, base_url):
        """
        Spec: Malformed token returns 401.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer not.a.jwt" \
                http://localhost:8080/proxy/npm-private/test
        """
        client = ProxyNDClient(
            base_url=base_url,
            token="not.a.valid.jwt.token",
        )
        response = client.get("/proxy/npm-private/test-package")

        assert response.status_code in [401, 403]

    @pytest.mark.private
    def test_empty_token_rejected(self, base_url):
        """
        Spec: Empty token returns 401.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer " \
                http://localhost:8080/proxy/npm-private/test
        """
        client = ProxyNDClient(
            base_url=base_url,
            token="",
        )
        response = client.get("/proxy/npm-private/test-package")

        assert response.status_code in [401, 403]
