# Maven Proxy Testing Guide

## Overview

This guide provides comprehensive testing strategies and procedures for the Maven proxy functionality in ProxyND.

## Test Environment Setup

### Prerequisites

```bash
# Install Java and Maven
java -version
mvn -version

# Install curl for manual testing
curl --version

# Start ProxyND server
make dev-run
# Or: CONFIG_DIR=./tmp/config STORAGE_DIR=./tmp/storage SERVER_PORT=8081 go run ./cmd/proxynd/main.go
```

### Configuration

Ensure your Maven proxy configuration is enabled:

```yaml
# config/maven-proxy.yaml
enabled: true
repositories:
  - url: "https://repo1.maven.org/maven2"
    name: "Maven Central"
  - url: "https://repository.apache.org/content/repositories/releases"
    name: "Apache Releases"

cache:
  ttl: 3600s # 1 hour
  max_file_size: 500MB

client_config:
  timeout: 30s
  retry_attempts: 3
```

## Manual Testing

### 1. Basic Connectivity Test

```bash
# Test basic proxy endpoint
curl -I http://localhost:8081/proxy/maven/

# Expected: 404 Not Found (normal for root path)
```

### 2. Artifact Download Tests

```bash
export PROXY_URL="http://localhost:8081/proxy/maven"

# Test popular artifacts
curl -O "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"
curl -O "$PROXY_URL/com/google/guava/guava/31.1-jre/guava-31.1-jre.jar"
curl -O "$PROXY_URL/org/slf4j/slf4j-api/1.7.36/slf4j-api-1.7.36.jar"
curl -O "$PROXY_URL/junit/junit/4.13.2/junit-4.13.2.jar"
curl -O "$PROXY_URL/org/springframework/spring-core/5.3.27/spring-core-5.3.27.jar"

# Verify file sizes and integrity
ls -la *.jar
sha1sum *.jar
```

### 3. POM File Tests

```bash
# Download POM files
curl -o commons-lang3-3.12.0.pom "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.pom"
curl -o guava-31.1-jre.pom "$PROXY_URL/com/google/guava/guava/31.1-jre/guava-31.1-jre.pom"

# Verify POM content
head -20 commons-lang3-3.12.0.pom
xmllint --format guava-31.1-jre.pom | head -30
```

### 4. Checksum Verification Tests

```bash
# Download checksums
curl "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar.sha1"
curl "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar.md5"

# Verify checksums match
sha1sum commons-lang3-3.12.0.jar
md5sum commons-lang3-3.12.0.jar
```

### 5. Metadata Tests

```bash
# Download metadata files
curl "$PROXY_URL/org/apache/commons/commons-lang3/maven-metadata.xml"
curl "$PROXY_URL/com/google/guava/guava/maven-metadata.xml"

# Verify metadata structure
xmllint --format maven-metadata.xml
```

### 6. Cache Behavior Tests

```bash
# First request (cache miss)
time curl -o test1.jar "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"

# Second request (cache hit - should be faster)
time curl -o test2.jar "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"

# Verify cache files
ls -la $STORAGE_DIR/maven/

# Compare download times and verify identical files
diff test1.jar test2.jar
```

### 7. Error Handling Tests

```bash
# Test non-existent artifact
curl -I "$PROXY_URL/non/existent/artifact/1.0/artifact-1.0.jar"
# Expected: 404 Not Found

# Test malformed requests
curl -I "$PROXY_URL/invalid-path"
# Expected: 400 Bad Request

# Test upstream failure simulation
curl -I "$PROXY_URL/org/apache/commons/commons-lang3/999.999.999/commons-lang3-999.999.999.jar"
# Expected: 404 Not Found
```

## Maven Client Integration Tests

### 1. Simple Project Test

Create a test project:

```bash
mkdir maven-proxy-test
cd maven-proxy-test

# Create pom.xml
cat > pom.xml << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    
    <groupId>com.example</groupId>
    <artifactId>maven-proxy-test</artifactId>
    <version>1.0.0</version>
    
    <properties>
        <maven.compiler.source>11</maven.compiler.source>
        <maven.compiler.target>11</maven.compiler.target>
    </properties>
    
    <dependencies>
        <dependency>
            <groupId>org.apache.commons</groupId>
            <artifactId>commons-lang3</artifactId>
            <version>3.12.0</version>
        </dependency>
        <dependency>
            <groupId>com.google.guava</groupId>
            <artifactId>guava</artifactId>
            <version>31.1-jre</version>
        </dependency>
        <dependency>
            <groupId>junit</groupId>
            <artifactId>junit</artifactId>
            <version>4.13.2</version>
            <scope>test</scope>
        </dependency>
    </dependencies>
    
    <repositories>
        <repository>
            <id>proxynd-maven</id>
            <url>http://localhost:8081/proxy/maven</url>
        </repository>
    </repositories>
</project>
EOF
```

### 2. Maven Settings Configuration

Create or update `~/.m2/settings.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 
          http://maven.apache.org/xsd/settings-1.0.0.xsd">
    
    <mirrors>
        <mirror>
            <id>proxynd-maven-mirror</id>
            <name>ProxyND Maven Mirror</name>
            <url>http://localhost:8081/proxy/maven</url>
            <mirrorOf>central</mirrorOf>
        </mirror>
    </mirrors>
    
    <profiles>
        <profile>
            <id>proxynd</id>
            <repositories>
                <repository>
                    <id>proxynd-maven</id>
                    <url>http://localhost:8081/proxy/maven</url>
                    <releases>
                        <enabled>true</enabled>
                    </releases>
                    <snapshots>
                        <enabled>false</enabled>
                    </snapshots>
                </repository>
            </repositories>
        </profile>
    </profiles>
    
    <activeProfiles>
        <activeProfile>proxynd</activeProfile>
    </activeProfiles>
</settings>
```

### 3. Run Maven Tests

```bash
# Clean local repository cache (optional)
rm -rf ~/.m2/repository/org/apache/commons/commons-lang3
rm -rf ~/.m2/repository/com/google/guava

# Run Maven commands
mvn clean compile
mvn dependency:tree
mvn test
mvn package

# Verify dependencies were downloaded through proxy
ls -la ~/.m2/repository/org/apache/commons/commons-lang3/3.12.0/
ls -la ~/.m2/repository/com/google/guava/guava/31.1-jre/
```

### 4. Verify Proxy Cache

```bash
# Check cache directory
ls -la $STORAGE_DIR/maven/org/apache/commons/commons-lang3/3.12.0/
ls -la $STORAGE_DIR/maven/com/google/guava/guava/31.1-jre/

# Verify cache statistics via health endpoint
curl http://localhost:8081/health/cache | jq .
```

## Automated Testing

### 1. Integration Test Script

```bash
#!/bin/bash
# maven-proxy-integration-test.sh

set -e

PROXY_URL="http://localhost:8081/proxy/maven"
TEST_DIR="$(mktemp -d)"
cd "$TEST_DIR"

echo "Starting Maven proxy integration tests..."

# Test 1: Basic artifact download
echo "Test 1: Basic artifact download"
curl -f -o commons-lang3.jar "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"
[ -f commons-lang3.jar ] && echo "✓ Artifact downloaded successfully"

# Test 2: POM file download
echo "Test 2: POM file download"
curl -f -o commons-lang3.pom "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.pom"
[ -f commons-lang3.pom ] && echo "✓ POM downloaded successfully"

# Test 3: Checksum verification
echo "Test 3: Checksum verification"
curl -f "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar.sha1" > expected.sha1
echo "$(sha1sum commons-lang3.jar | cut -d' ' -f1)  commons-lang3.jar" > actual.sha1
if diff expected.sha1 actual.sha1; then
    echo "✓ Checksum verification passed"
else
    echo "✗ Checksum verification failed"
    exit 1
fi

# Test 4: Cache hit performance
echo "Test 4: Cache performance test"
time1=$(curl -w "%{time_total}" -s -o /dev/null "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar")
time2=$(curl -w "%{time_total}" -s -o /dev/null "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar")

if (( $(echo "$time2 < $time1" | bc -l) )); then
    echo "✓ Cache hit is faster than initial download"
else
    echo "⚠ Cache performance may need investigation"
fi

# Test 5: Error handling
echo "Test 5: Error handling"
if curl -f "$PROXY_URL/non/existent/artifact/1.0/artifact-1.0.jar" 2>/dev/null; then
    echo "✗ Should have returned 404 for non-existent artifact"
    exit 1
else
    echo "✓ Correctly returned error for non-existent artifact"
fi

echo "All tests passed! ✓"
cd - > /dev/null
rm -rf "$TEST_DIR"
```

### 2. Performance Test Script

```bash
#!/bin/bash
# maven-proxy-performance-test.sh

PROXY_URL="http://localhost:8081/proxy/maven"
ARTIFACTS=(
    "org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"
    "com/google/guava/guava/31.1-jre/guava-31.1-jre.jar"
    "org/slf4j/slf4j-api/1.7.36/slf4j-api-1.7.36.jar"
    "junit/junit/4.13.2/junit-4.13.2.jar"
    "org/springframework/spring-core/5.3.27/spring-core-5.3.27.jar"
)

echo "Maven Proxy Performance Test"
echo "============================"

for artifact in "${ARTIFACTS[@]}"; do
    echo "Testing: $artifact"
    
    # Clear cache entry if exists
    rm -f "$STORAGE_DIR/maven/$artifact" 2>/dev/null || true
    
    # First request (cache miss)
    time1=$(curl -w "%{time_total}" -s -o /dev/null "$PROXY_URL/$artifact")
    
    # Second request (cache hit)
    time2=$(curl -w "%{time_total}" -s -o /dev/null "$PROXY_URL/$artifact")
    
    printf "  Cache miss: %.3fs, Cache hit: %.3fs, Speedup: %.1fx\n" \
        "$time1" "$time2" "$(echo "scale=1; $time1 / $time2" | bc -l)"
done
```

## Go Test Integration

### 1. Test Configuration

```go
// tests/integration/maven_proxy_test.go
func TestMavenProxy_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Maven proxy integration test in short mode")
    }
    
    server := startTestServer(t)
    defer server.Close()
    
    tests := []struct {
        name         string
        path         string
        expectedType string
        validateFunc func(t *testing.T, body []byte)
    }{
        {
            name:         "download JAR artifact",
            path:         "/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar",
            expectedType: "application/java-archive",
            validateFunc: func(t *testing.T, body []byte) {
                // JAR files should start with PK (ZIP signature)
                assert.Equal(t, []byte{0x50, 0x4B}, body[:2])
                assert.Greater(t, len(body), 1000) // Should be reasonably large
            },
        },
        {
            name:         "download POM file",
            path:         "/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.pom",
            expectedType: "application/xml",
            validateFunc: func(t *testing.T, body []byte) {
                assert.Contains(t, string(body), "<?xml")
                assert.Contains(t, string(body), "<project>")
                assert.Contains(t, string(body), "commons-lang3")
            },
        },
        {
            name:         "download checksum",
            path:         "/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar.sha1",
            expectedType: "text/plain",
            validateFunc: func(t *testing.T, body []byte) {
                checksum := strings.TrimSpace(string(body))
                assert.Len(t, checksum, 40) // SHA1 is 40 characters
                assert.Regexp(t, "^[a-f0-9]+$", checksum)
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp, err := http.Get(server.URL + tt.path)
            require.NoError(t, err)
            defer resp.Body.Close()
            
            assert.Equal(t, http.StatusOK, resp.StatusCode)
            assert.Equal(t, tt.expectedType, resp.Header.Get("Content-Type"))
            
            body, err := io.ReadAll(resp.Body)
            require.NoError(t, err)
            
            if tt.validateFunc != nil {
                tt.validateFunc(t, body)
            }
        })
    }
}
```

### 2. Cache Behavior Tests

```go
func TestMavenProxy_CacheBehavior(t *testing.T) {
    server := startTestServer(t)
    defer server.Close()
    
    artifactPath := "/proxy/maven/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"
    
    // First request (cache miss)
    start1 := time.Now()
    resp1, err := http.Get(server.URL + artifactPath)
    require.NoError(t, err)
    duration1 := time.Since(start1)
    body1, err := io.ReadAll(resp1.Body)
    require.NoError(t, err)
    resp1.Body.Close()
    
    // Second request (cache hit)
    start2 := time.Now()
    resp2, err := http.Get(server.URL + artifactPath)
    require.NoError(t, err)
    duration2 := time.Since(start2)
    body2, err := io.ReadAll(resp2.Body)
    require.NoError(t, err)
    resp2.Body.Close()
    
    // Verify responses are identical
    assert.Equal(t, body1, body2)
    assert.Equal(t, resp1.StatusCode, resp2.StatusCode)
    
    // Cache hit should be faster (with some tolerance)
    assert.Less(t, duration2, duration1+50*time.Millisecond)
    
    t.Logf("Cache miss: %v, Cache hit: %v, Speedup: %.1fx", 
        duration1, duration2, float64(duration1)/float64(duration2))
}
```

## Monitoring and Metrics

### 1. Health Check Verification

```bash
# Check Maven proxy health
curl http://localhost:8081/health/proxy/maven | jq .

# Expected response:
# {
#   "status": "healthy",
#   "upstream_status": "healthy",
#   "response_time": "45ms",
#   "cache_hit_ratio": 0.75,
#   "total_requests": 150
# }
```

### 2. Metrics Collection

```bash
# Check Prometheus metrics
curl http://localhost:8081/metrics | grep maven

# Look for metrics like:
# proxynd_proxy_requests_total{proxy_type="maven",status="success"} 145
# proxynd_proxy_cache_hits_total{proxy_type="maven"} 108
# proxynd_proxy_request_duration_seconds{proxy_type="maven",quantile="0.5"} 0.023
```

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```bash
   # Check if server is running
   curl -I http://localhost:8081/healthz
   
   # Check logs for errors
   tail -f logs/proxynd.log
   ```

2. **Slow Downloads**
   ```bash
   # Check upstream connectivity
   curl -I https://repo1.maven.org/maven2/
   
   # Monitor cache statistics
   curl http://localhost:8081/health/cache
   ```

3. **Authentication Issues**
   ```bash
   # Check proxy configuration
   curl http://localhost:8081/api/v1/config/maven-proxy
   
   # Verify repository access
   curl -I https://repo1.maven.org/maven2/org/apache/commons/commons-lang3/maven-metadata.xml
   ```

### Debug Mode

```bash
# Start server with debug logging
LOG_LEVEL=debug CONFIG_DIR=./tmp/config STORAGE_DIR=./tmp/storage go run ./cmd/proxynd/main.go

# Enable trace logging for specific requests
curl -H "X-Trace-Level: debug" "$PROXY_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar"
```

## Best Practices

### 1. Test Data Management

- Use popular, stable artifacts for testing
- Include various file types (JAR, POM, checksum files)
- Test both small and large artifacts
- Include artifacts with complex dependency trees

### 2. Performance Testing

- Measure cache hit/miss performance
- Test concurrent download scenarios
- Monitor memory usage during large downloads
- Verify cleanup of temporary files

### 3. Error Scenarios

- Test network timeouts
- Test upstream service failures
- Test malformed requests
- Test cache corruption scenarios

### 4. Security Testing

- Verify path traversal protection
- Test checksum validation
- Verify artifact integrity
- Test authentication if configured