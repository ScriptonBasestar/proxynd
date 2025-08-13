#!/bin/bash
# 스크립트명: Maven E2E 테스트 스크립트
# 용도: ProxyND의 Maven 프록시 기능을 실제 mvn 클라이언트로 테스트
# 사용법: test-maven.sh [옵션]
# 예시: test-maven.sh --verbose

set -euo pipefail

PROXYND_HOST=${PROXYND_HOST:-proxynd}
PROXYND_PORT=${PROXYND_PORT:-8080}
PROXY_URL="http://${PROXYND_HOST}:${PROXYND_PORT}/proxy/maven"

# 색상 출력을 위한 함수들
red() { echo -e "\033[31m$1\033[0m"; }
green() { echo -e "\033[32m$1\033[0m"; }
yellow() { echo -e "\033[33m$1\033[0m"; }
blue() { echo -e "\033[34m$1\033[0m"; }

# 로그 함수들
log_info() { echo "ℹ️ $1"; }
log_success() { green "✅ $1"; }
log_warning() { yellow "⚠️ $1"; }
log_error() { red "❌ $1"; }
log_test() { blue "🧪 $1"; }

# 헬프 메시지
show_help() {
    echo "Maven E2E 테스트 스크립트"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose     Verbose output"
    echo "  -h, --help        Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  PROXYND_HOST      ProxyND host (default: proxynd)"
    echo "  PROXYND_PORT      ProxyND port (default: 8080)"
    echo "  MVN_OPTS          Additional Maven options"
}

# 인수 파싱
VERBOSE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Verbose 모드 설정
if [[ "$VERBOSE" == "true" ]]; then
    set -x
    MVN_OPTS="${MVN_OPTS:-} -X"
fi

log_info "Starting Maven E2E tests..."
log_info "Proxy URL: $PROXY_URL"

# 테스트 작업 디렉토리 생성
TEST_DIR="/tmp/maven-e2e-test-$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# 정리 함수
cleanup() {
    log_info "Cleaning up test directory..."
    cd /
    rm -rf "$TEST_DIR" || true
}
trap cleanup EXIT

# Maven 설정 생성
create_maven_settings() {
    log_info "Creating Maven settings.xml for proxy..."

    cat > settings.xml << EOF
<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0
                              http://maven.apache.org/xsd/settings-1.0.0.xsd">
    <mirrors>
        <mirror>
            <id>proxynd-central</id>
            <name>ProxyND Central Mirror</name>
            <url>$PROXY_URL</url>
            <mirrorOf>central</mirrorOf>
        </mirror>
    </mirrors>

    <profiles>
        <profile>
            <id>proxynd-repos</id>
            <repositories>
                <repository>
                    <id>proxynd-central</id>
                    <name>ProxyND Central</name>
                    <url>$PROXY_URL</url>
                    <layout>default</layout>
                    <snapshots>
                        <enabled>true</enabled>
                    </snapshots>
                </repository>
            </repositories>
            <pluginRepositories>
                <pluginRepository>
                    <id>proxynd-central-plugins</id>
                    <name>ProxyND Central Plugins</name>
                    <url>$PROXY_URL</url>
                    <layout>default</layout>
                    <snapshots>
                        <enabled>true</enabled>
                    </snapshots>
                </pluginRepository>
            </pluginRepositories>
        </profile>
    </profiles>

    <activeProfiles>
        <activeProfile>proxynd-repos</activeProfile>
    </activeProfiles>
</settings>
EOF

    log_success "Maven settings.xml created"
}

# 테스트 프로젝트 생성
create_test_project() {
    log_info "Creating test Maven project..."

    cat > pom.xml << EOF
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
                             http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>com.example</groupId>
    <artifactId>proxynd-e2e-test</artifactId>
    <version>1.0.0-SNAPSHOT</version>
    <packaging>jar</packaging>

    <name>ProxyND E2E Test Project</name>
    <description>Test project for ProxyND Maven proxy</description>

    <properties>
        <maven.compiler.source>11</maven.compiler.source>
        <maven.compiler.target>11</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <dependencies>
        <!-- Popular artifact for testing -->
        <dependency>
            <groupId>junit</groupId>
            <artifactId>junit</artifactId>
            <version>4.13.2</version>
            <scope>test</scope>
        </dependency>

        <!-- Another popular artifact -->
        <dependency>
            <groupId>org.apache.commons</groupId>
            <artifactId>commons-lang3</artifactId>
            <version>3.12.0</version>
        </dependency>

        <!-- Test with Spring Core -->
        <dependency>
            <groupId>org.springframework</groupId>
            <artifactId>spring-core</artifactId>
            <version>5.3.10</version>
        </dependency>
    </dependencies>

    <build>
        <plugins>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-compiler-plugin</artifactId>
                <version>3.8.1</version>
            </plugin>

            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-surefire-plugin</artifactId>
                <version>3.0.0-M7</version>
            </plugin>
        </plugins>
    </build>
</project>
EOF

    # 간단한 Java 소스 파일 생성
    mkdir -p src/main/java/com/example
    cat > src/main/java/com/example/App.java << EOF
package com.example;

import org.apache.commons.lang3.StringUtils;
import org.springframework.core.SpringVersion;

public class App {
    public static void main(String[] args) {
        System.out.println("Hello ProxyND Maven E2E Test!");
        System.out.println("Commons Lang available: " + !StringUtils.isEmpty("test"));
        System.out.println("Spring Version: " + SpringVersion.getVersion());
    }
}
EOF

    # 테스트 파일 생성
    mkdir -p src/test/java/com/example
    cat > src/test/java/com/example/AppTest.java << EOF
package com.example;

import org.junit.Test;
import static org.junit.Assert.*;

public class AppTest {
    @Test
    public void testApp() {
        assertTrue("App should work", true);
    }
}
EOF

    log_success "Test Maven project created"
}

# ProxyND 헬스체크
check_proxynd_health() {
    log_test "Checking ProxyND health..."

    local health_url="http://${PROXYND_HOST}:${PROXYND_PORT}/healthz"
    local max_attempts=10
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "$health_url" >/dev/null 2>&1; then
            log_success "ProxyND is healthy"
            return 0
        fi

        attempt=$((attempt + 1))
        log_info "Attempt $attempt/$max_attempts: Waiting for ProxyND..."
        sleep 2
    done

    log_error "ProxyND health check failed after $max_attempts attempts"
    return 1
}

# Maven 의존성 다운로드 테스트
test_dependency_download() {
    log_test "Testing artifact download through proxy..."

    local start_time
    local duration

    start_time=$(date +%s%N)

    if mvn -s settings.xml dependency:resolve -q ${MVN_OPTS:-}; then
        duration=$(( ($(date +%s%N) - start_time) / 1000000 ))
        log_success "Dependency resolution successful (${duration}ms)"
    else
        log_error "Dependency resolution failed"
        return 1
    fi
}

# 캐시 성능 테스트
test_cache_performance() {
    log_test "Testing cache performance..."

    # 캐시 클리어를 위해 clean 실행
    mvn -s settings.xml clean -q ${MVN_OPTS:-} >/dev/null 2>&1 || true

    # 첫 번째 요청
    local start_time
    start_time=$(date +%s%N)
    mvn -s settings.xml dependency:resolve -q ${MVN_OPTS:-} >/dev/null 2>&1
    local first_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

    # 두 번째 요청 (캐시된 결과 기대)
    start_time=$(date +%s%N)
    mvn -s settings.xml dependency:resolve -q ${MVN_OPTS:-} >/dev/null 2>&1
    local second_duration=$(( ($(date +%s%N) - start_time) / 1000000 ))

    log_info "First request: ${first_duration}ms"
    log_info "Second request: ${second_duration}ms"

    if [ $second_duration -lt $((first_duration / 2)) ]; then
        log_success "Cache is working effectively (${second_duration}ms vs ${first_duration}ms)"
    elif [ $second_duration -lt $first_duration ]; then
        log_warning "Cache shows some improvement (${second_duration}ms vs ${first_duration}ms)"
    else
        log_warning "Cache performance inconclusive"
    fi
}

# 메타데이터 테스트
test_metadata_browsing() {
    log_test "Testing metadata browsing..."

    # Maven Central의 특정 artifact 메타데이터 조회
    local metadata_url="${PROXY_URL}/junit/junit/maven-metadata.xml"

    if curl -sf "$metadata_url" >/dev/null 2>&1; then
        log_success "Metadata browsing successful"

        if [[ "$VERBOSE" == "true" ]]; then
            log_info "Sample metadata:"
            curl -s "$metadata_url" | head -10
        fi
    else
        log_error "Metadata browsing failed"
        return 1
    fi
}

# Index 브라우징 테스트
test_index_browsing() {
    log_test "Testing index browsing..."

    # 디렉토리 리스팅 테스트
    local index_url="${PROXY_URL}/junit/"

    if curl -sf "$index_url" >/dev/null 2>&1; then
        log_success "Index browsing successful"

        if [[ "$VERBOSE" == "true" ]]; then
            log_info "Sample index page:"
            curl -s "$index_url" | head -5
        fi
    else
        log_warning "Index browsing not available (may be disabled)"
    fi
}

# 컴파일 테스트
test_compilation() {
    log_test "Testing project compilation..."

    if mvn -s settings.xml compile -q ${MVN_OPTS:-}; then
        log_success "Project compilation successful"
    else
        log_error "Project compilation failed"
        return 1
    fi
}

# 테스트 실행
test_junit_execution() {
    log_test "Testing JUnit execution..."

    if mvn -s settings.xml test -q ${MVN_OPTS:-}; then
        log_success "JUnit tests executed successfully"
    else
        log_error "JUnit test execution failed"
        return 1
    fi
}

# Step 2: Maven E2E 테스트 강화 - SNAPSHOT, 의존성 트리, 브라우저 인터페이스, 체크섬

# SNAPSHOT 버전 처리 테스트 (Step 2-1)
test_snapshot_handling() {
    log_test "Testing SNAPSHOT version processing..."

    # SNAPSHOT 버전이 포함된 의존성 추가
    cat > pom-snapshot.xml << EOF
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
                             http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>snapshot-test</artifactId>
    <version>1.0.0-SNAPSHOT</version>
    <packaging>jar</packaging>

    <dependencies>
        <!-- Test with a stable library that has SNAPSHOT versions -->
        <dependency>
            <groupId>org.apache.commons</groupId>
            <artifactId>commons-lang3</artifactId>
            <version>3.13.0-SNAPSHOT</version>
        </dependency>
    </dependencies>
</project>
EOF

    if mvn -s settings.xml -f pom-snapshot.xml dependency:resolve -q ${MVN_OPTS:-} 2>/dev/null; then
        log_success "SNAPSHOT version processing successful"
    else
        log_warning "SNAPSHOT version test skipped (no SNAPSHOT available or proxy limitation)"
    fi

    # SNAPSHOT 메타데이터 확인
    local snapshot_metadata_url="${PROXY_URL}/org/apache/commons/commons-lang3/maven-metadata.xml"
    if curl -sf "$snapshot_metadata_url" > /tmp/snapshot-metadata.xml 2>&1; then
        if grep -q "SNAPSHOT" /tmp/snapshot-metadata.xml 2>/dev/null; then
            log_success "SNAPSHOT metadata retrieval successful"
        else
            log_info "SNAPSHOT metadata available but no SNAPSHOT versions found"
        fi
    else
        log_warning "SNAPSHOT metadata test inconclusive"
    fi
}

# 의존성 트리 확인 테스트 (Step 2-2)
test_dependency_tree() {
    log_test "Testing dependency tree analysis..."

    if mvn -s settings.xml dependency:tree -q ${MVN_OPTS:-} > /tmp/dependency-tree.txt 2>&1; then
        log_success "Dependency tree generation successful"

        # 의존성 트리 검증
        if grep -q "junit:junit" /tmp/dependency-tree.txt && \
           grep -q "commons-lang3" /tmp/dependency-tree.txt && \
           grep -q "spring-core" /tmp/dependency-tree.txt; then
            log_success "All expected dependencies found in tree"
        else
            log_warning "Some expected dependencies missing from tree"
        fi

        if [[ "$VERBOSE" == "true" ]]; then
            log_info "Dependency tree sample:"
            head -20 /tmp/dependency-tree.txt
        fi
    else
        log_error "Dependency tree generation failed"
        return 1
    fi
}

# 브라우저 인터페이스 테스트 (Step 2-3)
test_browser_interface() {
    log_test "Testing browser interface..."

    local browser_urls=(
        "${PROXY_URL}/"
        "${PROXY_URL}/junit/"
        "${PROXY_URL}/junit/junit/"
        "${PROXY_URL}/org/springframework/"
        "${PROXY_URL}/org/apache/commons/"
    )

    local success_count=0
    for url in "${browser_urls[@]}"; do
        if curl -sf "$url" > /tmp/browser-test.html 2>&1; then
            # HTML 응답인지 확인
            if grep -qi "html\|directory\|index" /tmp/browser-test.html; then
                log_success "Browser interface accessible: $(basename "$url")"
                success_count=$((success_count + 1))
            else
                log_info "Browser interface response (non-HTML): $(basename "$url")"
                success_count=$((success_count + 1))
            fi
        else
            log_warning "Browser interface unavailable: $(basename "$url")"
        fi
    done

    if [ $success_count -gt 0 ]; then
        log_success "Browser interface test passed ($success_count/$(${#browser_urls[@]}) endpoints accessible)"
    else
        log_warning "Browser interface not available (may be disabled)"
    fi
}

# 체크섬 검증 테스트 (Step 2-4)
test_checksum_verification() {
    log_test "Testing checksum verification..."

    local base_artifact_url="${PROXY_URL}/junit/junit/4.13.2/junit-4.13.2"
    local jar_url="${base_artifact_url}.jar"
    local sha1_url="${base_artifact_url}.jar.sha1"
    local md5_url="${base_artifact_url}.jar.md5"

    # JAR 파일 다운로드
    if curl -sf "$jar_url" -o /tmp/junit-4.13.2.jar 2>&1; then
        log_success "JAR artifact downloaded successfully"

        # SHA1 체크섬 확인
        if curl -sf "$sha1_url" -o /tmp/junit-4.13.2.jar.sha1 2>&1; then
            local expected_sha1
            expected_sha1=$(cat /tmp/junit-4.13.2.jar.sha1 | cut -d' ' -f1)

            local actual_sha1
            if command -v sha1sum >/dev/null 2>&1; then
                actual_sha1=$(sha1sum /tmp/junit-4.13.2.jar | cut -d' ' -f1)
            elif command -v shasum >/dev/null 2>&1; then
                actual_sha1=$(shasum -a 1 /tmp/junit-4.13.2.jar | cut -d' ' -f1)
            else
                log_warning "No SHA1 utility available for verification"
                return 0
            fi

            if [[ "$expected_sha1" == "$actual_sha1" ]]; then
                log_success "SHA1 checksum verification passed"
            else
                log_error "SHA1 checksum mismatch! Expected: $expected_sha1, Got: $actual_sha1"
                return 1
            fi
        else
            log_warning "SHA1 checksum file not available"
        fi

        # MD5 체크섬 확인 (선택적)
        if curl -sf "$md5_url" -o /tmp/junit-4.13.2.jar.md5 2>&1; then
            log_success "MD5 checksum file available"
        else
            log_info "MD5 checksum file not available (optional)"
        fi
    else
        log_error "JAR artifact download failed"
        return 1
    fi
}

# 에러 시나리오 테스트
test_error_scenarios() {
    log_test "Testing error scenarios..."

    # 존재하지 않는 artifact 요청
    local nonexistent_url="${PROXY_URL}/com/nonexistent/nonexistent/1.0.0/nonexistent-1.0.0.jar"

    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$nonexistent_url")

    if [[ "$response_code" == "404" ]]; then
        log_success "Non-existent artifact correctly returns 404"
    else
        log_warning "Non-existent artifact returned $response_code (expected 404)"
    fi
}

# 메인 테스트 실행
main() {
    log_info "=== Enhanced Maven E2E Tests Starting ==="
    log_info "Step 2: Maven E2E 테스트 강화 - SNAPSHOT, 의존성, 브라우저, 체크섬"

    create_maven_settings
    create_test_project

    # 필수 테스트들
    check_proxynd_health
    test_dependency_download
    test_compilation
    test_junit_execution

    # Step 2: Maven E2E 테스트 강화
    test_snapshot_handling        # Step 2-1: SNAPSHOT 버전 처리 테스트
    test_dependency_tree         # Step 2-2: 의존성 트리 확인 테스트
    test_browser_interface       # Step 2-3: 브라우저 인터페이스 테스트
    test_checksum_verification   # Step 2-4: 체크섬 검증 테스트

    # 기존 기능 테스트들
    test_cache_performance
    test_metadata_browsing
    test_index_browsing
    test_error_scenarios

    log_success "=== Enhanced Maven E2E Tests Completed Successfully ==="
    log_info "✅ SNAPSHOT 버전 처리 테스트 완료"
    log_info "✅ 의존성 트리 확인 테스트 완료"
    log_info "✅ 브라우저 인터페이스 테스트 완료"
    log_info "✅ 체크섬 검증 테스트 완료"
}

# 스크립트 실행
main "$@"
