"""
NPM Tarball Download Tests

Spec: NPM package tarballs (.tgz) can be downloaded.

Manual Test:
    curl -o express.tgz http://localhost:8080/proxy/npm/express/-/express-4.18.2.tgz
    file express.tgz  # should show "gzip compressed data"
"""

import pytest


class TestNpmTarballDownload:
    """NPM tarball (.tgz) download tests."""

    @pytest.mark.public
    @pytest.mark.npm
    @pytest.mark.slow
    def test_download_tarball(self, client):
        """
        Spec: GET /proxy/npm/<pkg>/-/<pkg>-<ver>.tgz returns gzip tarball.

        Manual:
            curl -o express.tgz http://localhost:8080/proxy/npm/express/-/express-4.18.2.tgz
            file express.tgz
            # Output: express.tgz: gzip compressed data
        """
        response = client.get("/proxy/npm/express/-/express-4.18.2.tgz")

        assert response.status_code == 200
        # Verify gzip magic bytes (1f 8b)
        assert response.content[:2] == b"\x1f\x8b", "Response is not gzip compressed"

    @pytest.mark.public
    @pytest.mark.npm
    @pytest.mark.slow
    def test_download_lodash_tarball(self, client):
        """
        Spec: Lodash tarball is downloadable.

        Manual:
            curl -o lodash.tgz http://localhost:8080/proxy/npm/lodash/-/lodash-4.17.21.tgz
        """
        response = client.get("/proxy/npm/lodash/-/lodash-4.17.21.tgz")

        assert response.status_code == 200
        assert response.content[:2] == b"\x1f\x8b"

    @pytest.mark.public
    @pytest.mark.npm
    def test_tarball_not_found(self, client):
        """
        Spec: Non-existent tarball version returns 404.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                http://localhost:8080/proxy/npm/express/-/express-999.999.999.tgz
        """
        response = client.get("/proxy/npm/express/-/express-999.999.999.tgz")

        assert response.status_code == 404
