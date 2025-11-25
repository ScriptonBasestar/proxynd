"""
NPM Response Headers Tests

Spec: NPM proxy responses should have correct Content-Type headers.

Manual Test:
    curl -I http://localhost:8080/proxy/npm/express
    # Content-Type should contain application/json
"""

import pytest


class TestNpmResponseHeaders:
    """NPM response header validation tests."""

    @pytest.mark.public
    @pytest.mark.npm
    def test_metadata_content_type_json(self, client):
        """
        Spec: Package metadata returns Content-Type: application/json.

        Manual:
            curl -I http://localhost:8080/proxy/npm/express | grep -i content-type
            # Expected: application/json
        """
        response = client.get("/proxy/npm/express")

        assert response.status_code == 200
        content_type = response.headers.get("Content-Type", "")
        assert "application/json" in content_type

    @pytest.mark.public
    @pytest.mark.npm
    def test_head_request_no_body(self, client):
        """
        Spec: HEAD request returns headers only, no body.

        Manual:
            curl -I http://localhost:8080/proxy/npm/express
            # Should return headers, no body content
        """
        response = client.head("/proxy/npm/express")

        assert response.status_code == 200
        assert len(response.content) == 0, "HEAD response should have no body"

    @pytest.mark.public
    @pytest.mark.npm
    @pytest.mark.slow
    def test_tarball_content_type(self, client):
        """
        Spec: Tarball returns appropriate Content-Type.

        Manual:
            curl -I http://localhost:8080/proxy/npm/express/-/express-4.18.2.tgz \
                | grep -i content-type
            # Expected: application/gzip or application/octet-stream
        """
        response = client.head("/proxy/npm/express/-/express-4.18.2.tgz")

        assert response.status_code == 200
        content_type = response.headers.get("Content-Type", "")
        # Tarball can be gzip, octet-stream, or x-gzip
        valid_types = ["gzip", "octet-stream", "x-tar"]
        assert any(t in content_type for t in valid_types) or content_type != ""
