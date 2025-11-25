"""
NPM Private Registry Access Tests

Spec: Private NPM packages require authentication.

Manual Test (unauthorized):
    curl -s -o /dev/null -w "%{http_code}" \
        http://localhost:8080/proxy/npm-private/internal-pkg
    # Expected: 401 or 403

Manual Test (Basic Auth):
    curl -u testuser:testpass \
        http://localhost:8080/proxy/npm-private/internal-pkg

Manual Test (Bearer Token):
    curl -H "Authorization: Bearer <token>" \
        http://localhost:8080/proxy/npm-private/internal-pkg
"""

import pytest


class TestNpmPrivateUnauthorized:
    """Private NPM access without authentication."""

    @pytest.mark.private
    @pytest.mark.npm
    def test_unauthorized_returns_401_or_403(self, client):
        """
        Spec: Accessing private registry without auth returns 401/403.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                http://localhost:8080/proxy/npm-private/internal-pkg
        """
        response = client.get("/proxy/npm-private/internal-package")

        assert response.status_code in [401, 403]


class TestNpmPrivateBasicAuth:
    """Private NPM access with Basic Authentication."""

    @pytest.mark.private
    @pytest.mark.npm
    def test_basic_auth_access(self, basic_auth_client):
        """
        Spec: Private registry accessible with valid Basic Auth.

        Manual:
            curl -u testuser:testpass \
                http://localhost:8080/proxy/npm-private/internal-pkg
            # Expected: 200 (exists) or 404 (not found), NOT 401
        """
        response = basic_auth_client.get("/proxy/npm-private/internal-package")

        # Should NOT be 401/403 if credentials are valid
        assert response.status_code in [200, 404], \
            f"Expected 200/404, got {response.status_code}"


class TestNpmPrivateBearerToken:
    """Private NPM access with Bearer Token (JWT)."""

    @pytest.mark.private
    @pytest.mark.npm
    def test_bearer_token_access(self, token_client):
        """
        Spec: Private registry accessible with valid Bearer token.

        Manual:
            curl -H "Authorization: Bearer <jwt-token>" \
                http://localhost:8080/proxy/npm-private/internal-pkg
        """
        response = token_client.get("/proxy/npm-private/internal-package")

        assert response.status_code in [200, 404], \
            f"Expected 200/404, got {response.status_code}"


class TestNpmPrivateApiKey:
    """Private NPM access with API Key."""

    @pytest.mark.private
    @pytest.mark.npm
    def test_api_key_access(self, api_key_client):
        """
        Spec: Private registry accessible with valid API Key.

        Manual:
            curl -H "X-API-Key: <api-key>" \
                http://localhost:8080/proxy/npm-private/internal-pkg
        """
        response = api_key_client.get("/proxy/npm-private/internal-package")

        assert response.status_code in [200, 404], \
            f"Expected 200/404, got {response.status_code}"
