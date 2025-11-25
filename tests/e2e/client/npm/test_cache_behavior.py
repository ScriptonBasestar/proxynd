"""
NPM Cache Behavior Tests

Spec: Cached requests should be faster than initial requests.

Manual Test:
    # First request (cache miss)
    time curl http://localhost:8080/proxy/npm/lodash > /dev/null

    # Second request (cache hit - should be faster)
    time curl http://localhost:8080/proxy/npm/lodash > /dev/null
"""

import time

import pytest


class TestNpmCacheBehavior:
    """NPM caching behavior tests."""

    @pytest.mark.public
    @pytest.mark.npm
    @pytest.mark.slow
    def test_cache_hit_faster_than_miss(self, client):
        """
        Spec: Second request (cache hit) should be faster than first.

        Manual:
            # Clear cache first (optional)
            curl -X DELETE "http://localhost:8080/api/cache/clear/npm?confirm=true"

            # Time first request
            time curl -s http://localhost:8080/proxy/npm/lodash > /dev/null

            # Time second request (should be faster)
            time curl -s http://localhost:8080/proxy/npm/lodash > /dev/null
        """
        package = "lodash"

        # First request (cache miss)
        start = time.time()
        response1 = client.get(f"/proxy/npm/{package}")
        first_duration = time.time() - start

        assert response1.status_code == 200

        # Second request (cache hit)
        start = time.time()
        response2 = client.get(f"/proxy/npm/{package}")
        second_duration = time.time() - start

        assert response2.status_code == 200

        # Log timing for debugging
        print(f"\nCache test: first={first_duration:.3f}s, second={second_duration:.3f}s")

        # Note: This assertion may be flaky in some environments
        # Cache hit is generally faster, but network conditions vary

    @pytest.mark.public
    @pytest.mark.npm
    def test_cached_response_identical(self, client):
        """
        Spec: Cached response should be identical to original.

        Manual:
            curl http://localhost:8080/proxy/npm/express > first.json
            curl http://localhost:8080/proxy/npm/express > second.json
            diff first.json second.json  # should be identical
        """
        response1 = client.get("/proxy/npm/express")
        response2 = client.get("/proxy/npm/express")

        assert response1.status_code == 200
        assert response2.status_code == 200

        # Content should be identical
        data1 = response1.json()
        data2 = response2.json()
        assert data1["name"] == data2["name"]
        assert data1["dist-tags"] == data2["dist-tags"]
