"""
NPM Public Download Tests

Spec: Public NPM packages can be downloaded without authentication.

Manual Test:
    curl http://localhost:8080/proxy/npm/express
    curl http://localhost:8080/proxy/npm/@types/node
"""

import pytest


class TestNpmPublicDownload:
    """Public NPM package download - no auth required."""

    @pytest.mark.public
    @pytest.mark.npm
    def test_get_package_metadata(self, client):
        """
        Spec: GET /proxy/npm/<package> returns package metadata.

        Manual:
            curl http://localhost:8080/proxy/npm/express | jq '.name'
        """
        response = client.get("/proxy/npm/express")

        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "express"
        assert "versions" in data
        assert "dist-tags" in data

    @pytest.mark.public
    @pytest.mark.npm
    def test_get_scoped_package(self, client):
        """
        Spec: GET /proxy/npm/@scope/name returns scoped package.

        Manual:
            curl http://localhost:8080/proxy/npm/@types/node | jq '.name'
        """
        response = client.get("/proxy/npm/@types/node")

        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "@types/node"

    @pytest.mark.public
    @pytest.mark.npm
    def test_get_lodash(self, client):
        """
        Spec: Popular package lodash is accessible.

        Manual:
            curl http://localhost:8080/proxy/npm/lodash | jq '.name'
        """
        response = client.get("/proxy/npm/lodash")

        assert response.status_code == 200
        assert response.json()["name"] == "lodash"

    @pytest.mark.public
    @pytest.mark.npm
    def test_get_react(self, client):
        """
        Spec: Popular package react is accessible.

        Manual:
            curl http://localhost:8080/proxy/npm/react | jq '.name'
        """
        response = client.get("/proxy/npm/react")

        assert response.status_code == 200
        assert response.json()["name"] == "react"

    @pytest.mark.public
    @pytest.mark.npm
    def test_package_not_found_returns_404(self, client):
        """
        Spec: Non-existent package returns 404.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                http://localhost:8080/proxy/npm/nonexistent-pkg-12345
        """
        response = client.get("/proxy/npm/nonexistent-package-definitely-not-exist-12345")

        assert response.status_code == 404
