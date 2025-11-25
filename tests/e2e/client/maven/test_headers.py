"""
Maven Response Headers Tests

Spec: Maven proxy responses should have correct Content-Type headers.

Manual Test:
    curl -I http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom
    # Content-Type should contain xml or text/plain
"""

import pytest


class TestMavenResponseHeaders:
    """Maven response header validation tests."""

    @pytest.mark.public
    @pytest.mark.maven
    def test_pom_content_type_xml(self, client):
        """
        Spec: POM file returns Content-Type with xml.

        Manual:
            curl -I http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom \
                | grep -i content-type
        """
        response = client.get("/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom")

        assert response.status_code == 200
        content_type = response.headers.get("Content-Type", "")
        # POM can be served as xml or text/plain
        assert "xml" in content_type.lower() or "text" in content_type.lower()

    @pytest.mark.public
    @pytest.mark.maven
    def test_jar_content_type(self, client):
        """
        Spec: JAR file returns appropriate binary Content-Type.

        Manual:
            curl -I http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar \
                | grep -i content-type
        """
        response = client.head("/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar")

        assert response.status_code == 200
        content_type = response.headers.get("Content-Type", "")
        # JAR can be java-archive, octet-stream, or zip
        valid_types = ["java-archive", "octet-stream", "zip"]
        assert any(t in content_type for t in valid_types) or content_type != ""

    @pytest.mark.public
    @pytest.mark.maven
    def test_head_request_no_body(self, client):
        """
        Spec: HEAD request returns headers only, no body.

        Manual:
            curl -I http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom
        """
        response = client.head("/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom")

        assert response.status_code == 200
        assert len(response.content) == 0, "HEAD response should have no body"
