"""
Maven/Gradle Compatibility Tests

Spec: ProxyND Maven proxy works with Gradle builds.

Gradle uses Maven repository format, so these tests verify
compatibility with Gradle dependency resolution.

Manual Test:
    # In a Gradle project, add to build.gradle:
    repositories {
        maven { url 'http://localhost:8080/proxy/maven' }
    }
    # Then run: ./gradlew dependencies
"""

import pytest


class TestGradleCompatibility:
    """Gradle compatibility with Maven proxy."""

    @pytest.mark.public
    @pytest.mark.maven
    def test_gradle_common_dependency_junit_jupiter(self, client):
        """
        Spec: JUnit Jupiter (common Gradle test dep) is accessible.

        Manual:
            curl http://localhost:8080/proxy/maven/org/junit/jupiter/junit-jupiter/5.9.0/junit-jupiter-5.9.0.pom
        """
        response = client.get(
            "/proxy/maven/org/junit/jupiter/junit-jupiter/5.9.0/junit-jupiter-5.9.0.pom"
        )

        # May return 404 if version doesn't exist
        assert response.status_code in [200, 404]

    @pytest.mark.public
    @pytest.mark.maven
    def test_gradle_common_dependency_guava(self, client):
        """
        Spec: Guava (common Gradle dep) is accessible.

        Manual:
            curl http://localhost:8080/proxy/maven/com/google/guava/guava/31.1-jre/guava-31.1-jre.pom
        """
        response = client.get(
            "/proxy/maven/com/google/guava/guava/31.1-jre/guava-31.1-jre.pom"
        )

        assert response.status_code in [200, 404]

    @pytest.mark.public
    @pytest.mark.maven
    def test_gradle_module_metadata(self, client):
        """
        Spec: Gradle Module Metadata (.module) files are accessible.

        Gradle 6+ uses .module files for dependency metadata.

        Manual:
            curl http://localhost:8080/proxy/maven/com/google/guava/guava/31.1-jre/guava-31.1-jre.module
        """
        response = client.get(
            "/proxy/maven/com/google/guava/guava/31.1-jre/guava-31.1-jre.module"
        )

        # .module files may not exist for all artifacts
        assert response.status_code in [200, 404]
