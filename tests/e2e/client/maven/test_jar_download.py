"""
Maven JAR Download Tests

Spec: Maven JAR files can be downloaded.

Manual Test:
    curl -o junit.jar http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar
    file junit.jar  # should show "Zip archive data" (JAR is ZIP format)
"""

import pytest


class TestMavenJarDownload:
    """Maven JAR file download tests."""

    @pytest.mark.public
    @pytest.mark.maven
    @pytest.mark.slow
    def test_download_junit_jar(self, client):
        """
        Spec: GET .../artifact.jar returns JAR file (ZIP format).

        Manual:
            curl -o junit.jar \
                http://localhost:8080/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar
            file junit.jar
            # Output: Zip archive data (JAR format)
        """
        response = client.get("/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar")

        assert response.status_code == 200
        # JAR files are ZIP format (magic bytes: PK = 0x50 0x4b)
        assert response.content[:2] == b"PK", "Response is not a JAR/ZIP file"

    @pytest.mark.public
    @pytest.mark.maven
    @pytest.mark.slow
    def test_download_guava_jar(self, client):
        """
        Spec: Guava JAR is downloadable.

        Manual:
            curl -o guava.jar \
                http://localhost:8080/proxy/maven/com/google/guava/guava/31.1-jre/guava-31.1-jre.jar
        """
        response = client.get(
            "/proxy/maven/com/google/guava/guava/31.1-jre/guava-31.1-jre.jar"
        )

        assert response.status_code == 200
        assert response.content[:2] == b"PK"

    @pytest.mark.public
    @pytest.mark.maven
    def test_jar_not_found(self, client):
        """
        Spec: Non-existent JAR version returns 404.

        Manual:
            curl -s -o /dev/null -w "%{http_code}" \
                http://localhost:8080/proxy/maven/junit/junit/999.999/junit-999.999.jar
        """
        response = client.get("/proxy/maven/junit/junit/999.999.999/junit-999.999.999.jar")

        assert response.status_code == 404
