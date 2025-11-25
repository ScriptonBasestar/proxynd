"""
Maven Public Download Tests

Spec: Public Maven artifacts can be downloaded without authentication.

Maven Repository Structure:
    /proxy/maven/<groupId>/<artifactId>/<version>/<artifactId>-<version>.<ext>

Example:
    /proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom
    /proxy/maven/org/springframework/spring-core/5.3.20/spring-core-5.3.20.jar

Manual Test:
    curl http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom
"""

import pytest


class TestMavenPublicPom:
    """Public Maven POM file download tests."""

    @pytest.mark.public
    @pytest.mark.maven
    def test_get_junit_pom(self, client):
        """
        Spec: GET /proxy/maven/.../artifact.pom returns POM XML.

        Manual:
            curl http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom
        """
        response = client.get("/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom")

        assert response.status_code == 200
        content = response.text
        assert "<?xml" in content or "<project" in content
        assert "junit" in content.lower()

    @pytest.mark.public
    @pytest.mark.maven
    def test_get_spring_core_pom(self, client):
        """
        Spec: Spring Framework POM is accessible.

        Manual:
            curl http://localhost:8080/proxy/maven/org/springframework/spring-core/5.3.20/spring-core-5.3.20.pom
        """
        response = client.get(
            "/proxy/maven/org/springframework/spring-core/5.3.20/spring-core-5.3.20.pom"
        )

        assert response.status_code == 200
        assert "spring" in response.text.lower()

    @pytest.mark.public
    @pytest.mark.maven
    def test_pom_not_found_returns_404(self, client):
        """
        Spec: Non-existent artifact returns 404.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                http://localhost:8080/proxy/maven/nonexistent/artifact/1.0/artifact-1.0.pom
        """
        response = client.get(
            "/proxy/maven/nonexistent/nonexistent/1.0.0/nonexistent-1.0.0.pom"
        )

        assert response.status_code == 404


class TestMavenPublicMetadata:
    """Maven metadata.xml download tests."""

    @pytest.mark.public
    @pytest.mark.maven
    def test_get_maven_metadata(self, client):
        """
        Spec: GET .../maven-metadata.xml returns version metadata.

        Manual:
            curl http://localhost:8080/proxy/maven/junit/junit/maven-metadata.xml
        """
        response = client.get("/proxy/maven/junit/junit/maven-metadata.xml")

        assert response.status_code == 200
        content = response.text
        assert "<metadata>" in content or "metadata" in content.lower()

    @pytest.mark.public
    @pytest.mark.maven
    def test_get_spring_metadata(self, client):
        """
        Spec: Spring Framework metadata is accessible.

        Manual:
            curl http://localhost:8080/proxy/maven/org/springframework/spring-core/maven-metadata.xml
        """
        response = client.get(
            "/proxy/maven/org/springframework/spring-core/maven-metadata.xml"
        )

        assert response.status_code == 200
