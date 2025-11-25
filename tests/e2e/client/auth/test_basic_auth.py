"""
Basic Authentication Tests

Spec: HTTP Basic Authentication works for protected endpoints.

Manual Test:
    # Without auth (should fail)
    curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/user/list
    # Expected: 401

    # With valid auth
    curl -u testuser:testpass http://localhost:8080/api/user/list
    # Expected: 200

    # With invalid auth
    curl -u invalid:invalid http://localhost:8080/api/user/list
    # Expected: 401
"""

import pytest

from client import ProxyNDClient


class TestBasicAuthValid:
    """Valid Basic Authentication tests."""

    @pytest.mark.private
    def test_valid_credentials_accepted(self, base_url):
        """
        Spec: Valid username/password grants access.

        Manual:
            curl -u testuser:testpass http://localhost:8080/api/status
        """
        client = ProxyNDClient(
            base_url=base_url,
            username="testuser",
            password="testpass",
        )
        response = client.get("/api/status")

        # Should get valid response (200 or other non-401)
        assert response.status_code != 401


class TestBasicAuthInvalid:
    """Invalid Basic Authentication tests."""

    @pytest.mark.private
    def test_invalid_credentials_rejected(self, base_url):
        """
        Spec: Invalid username/password returns 401.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                -u invalid:invalid http://localhost:8080/proxy/npm-private/test
        """
        client = ProxyNDClient(
            base_url=base_url,
            username="invalid_user",
            password="invalid_pass",
        )
        response = client.get("/proxy/npm-private/test-package")

        assert response.status_code in [401, 403]

    @pytest.mark.private
    def test_empty_credentials_rejected(self, base_url):
        """
        Spec: Empty credentials return 401.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                -u : http://localhost:8080/proxy/npm-private/test
        """
        client = ProxyNDClient(
            base_url=base_url,
            username="",
            password="",
        )
        response = client.get("/proxy/npm-private/test-package")

        assert response.status_code in [401, 403]
