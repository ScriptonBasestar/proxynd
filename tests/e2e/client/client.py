"""
ProxyND Test Client - Minimal HTTP wrapper for testing.

Usage:
    client = ProxyNDClient("http://localhost:8080")
    response = client.get("/proxy/npm/lodash")
    response = client.get("/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom")

Authentication:
    # Basic Auth
    client = ProxyNDClient(base_url, username="user", password="pass")

    # Bearer Token
    client = ProxyNDClient(base_url, token="jwt-token")

    # API Key
    client = ProxyNDClient(base_url, api_key="my-api-key")
"""

from dataclasses import dataclass
from typing import Optional

import requests


@dataclass
class ProxyNDClient:
    """Simple HTTP client for ProxyND proxy testing."""

    base_url: str = "http://localhost:8080"
    timeout: int = 30

    # Auth options (mutually exclusive for token/api_key, basic auth can combine)
    username: Optional[str] = None
    password: Optional[str] = None
    token: Optional[str] = None
    api_key: Optional[str] = None

    def get(self, path: str, **kwargs) -> requests.Response:
        """GET request."""
        return self._request("GET", path, **kwargs)

    def head(self, path: str, **kwargs) -> requests.Response:
        """HEAD request (metadata only)."""
        return self._request("HEAD", path, **kwargs)

    def post(self, path: str, **kwargs) -> requests.Response:
        """POST request."""
        return self._request("POST", path, **kwargs)

    def put(self, path: str, **kwargs) -> requests.Response:
        """PUT request."""
        return self._request("PUT", path, **kwargs)

    def delete(self, path: str, **kwargs) -> requests.Response:
        """DELETE request."""
        return self._request("DELETE", path, **kwargs)

    def _request(self, method: str, path: str, **kwargs) -> requests.Response:
        """Execute HTTP request with auth headers applied."""
        url = f"{self.base_url.rstrip('/')}{path}"
        headers = kwargs.pop("headers", {}).copy()

        # Apply token-based auth (Bearer or API Key)
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        elif self.api_key:
            headers["X-API-Key"] = self.api_key

        # Basic auth (can be used alongside or separately)
        auth = None
        if self.username and self.password:
            auth = (self.username, self.password)

        return requests.request(
            method,
            url,
            headers=headers,
            auth=auth,
            timeout=self.timeout,
            **kwargs,
        )

    def health_check(self) -> bool:
        """Check if the server is healthy."""
        try:
            response = self.get("/health")
            return response.status_code == 200
        except requests.RequestException:
            return False
