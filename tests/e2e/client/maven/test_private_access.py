"""
Maven Private Repository Access Tests

Spec: Private Maven artifacts require authentication.

Manual Test (unauthorized):
    curl -s -o /dev/null -w "%{http_code}" \
        http://localhost:8080/proxy/maven-private/com/internal/lib/1.0/lib-1.0.pom
    # Expected: 401 or 403

Manual Test (Basic Auth):
    curl -u testuser:testpass \
        http://localhost:8080/proxy/maven-private/com/internal/lib/1.0/lib-1.0.pom
"""

import pytest


class TestMavenPrivateUnauthorized:
    """Private Maven access without authentication."""

    @pytest.mark.private
    @pytest.mark.maven
    def test_unauthorized_returns_401_or_403(self, client):
        """
        Spec: Accessing private repository without auth returns 401/403.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                http://localhost:8080/proxy/maven-private/com/internal/lib/1.0/lib-1.0.pom
        """
        response = client.get(
            "/proxy/maven-private/com/internal/artifact/1.0/artifact-1.0.pom"
        )

        assert response.status_code in [401, 403]


class TestMavenPrivateBasicAuth:
    """Private Maven access with Basic Authentication."""

    @pytest.mark.private
    @pytest.mark.maven
    def test_basic_auth_access(self, basic_auth_client):
        """
        Spec: Private repository accessible with valid Basic Auth.

        Manual:
            curl -u testuser:testpass \
                http://localhost:8080/proxy/maven-private/com/internal/lib/1.0/lib-1.0.pom
        """
        response = basic_auth_client.get(
            "/proxy/maven-private/com/internal/artifact/1.0/artifact-1.0.pom"
        )

        assert response.status_code in [200, 404], \
            f"Expected 200/404, got {response.status_code}"


class TestMavenPrivateBearerToken:
    """Private Maven access with Bearer Token."""

    @pytest.mark.private
    @pytest.mark.maven
    def test_bearer_token_access(self, token_client):
        """
        Spec: Private repository accessible with valid Bearer token.

        Manual:
            curl -H "Authorization: Bearer <jwt-token>" \
                http://localhost:8080/proxy/maven-private/com/internal/lib/1.0/lib-1.0.pom
        """
        response = token_client.get(
            "/proxy/maven-private/com/internal/artifact/1.0/artifact-1.0.pom"
        )

        assert response.status_code in [200, 404], \
            f"Expected 200/404, got {response.status_code}"
