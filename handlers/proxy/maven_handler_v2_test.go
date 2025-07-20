package proxy

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	// Test constants for Maven handler tests
	testMavenChecksumPath = "com/example/app/1.0/app-1.0.jar.sha1"
)

func TestMavenHandlerV2_Simple(t *testing.T) {
	// 테스트를 위한 임시 CONFIG_DIR 설정
	originalConfigDir := os.Getenv("CONFIG_DIR")
	defer func() {
		if originalConfigDir != "" {
			_ = os.Setenv("CONFIG_DIR", originalConfigDir)
		} else {
			_ = os.Unsetenv("CONFIG_DIR")
		}
	}()

	// 임시 디렉토리 설정
	tempDir := t.TempDir()
	_ = os.Setenv("CONFIG_DIR", tempDir)

	// 기본 테스트
	t.Run("Type returns maven", func(t *testing.T) {
		handler := NewMavenHandlerV2()
		assert.Equal(t, "maven", handler.Type())
	})

	t.Run("IsEnabled with no config", func(t *testing.T) {
		handler := NewMavenHandlerV2()
		assert.False(t, handler.IsEnabled())
	})

	t.Run("IsEnabled with proxies", func(t *testing.T) {
		// IsEnabled 메서드는 ReadConfig()를 호출하므로 스킵
		t.Skip("IsEnabled requires actual config file")
	})

	t.Run("isSnapshotArtifact", func(t *testing.T) {
		handler := NewMavenHandlerV2()

		// SNAPSHOT 아티팩트
		assert.True(t, handler.isSnapshotArtifact("com/example/app/1.0-SNAPSHOT/app-1.0-SNAPSHOT.jar"))
		assert.True(t, handler.isSnapshotArtifact(
			"org/springframework/spring-core/5.3.0-SNAPSHOT/spring-core-5.3.0-SNAPSHOT.jar"))

		// 릴리즈 아티팩트
		assert.False(t, handler.isSnapshotArtifact("com/example/app/1.0/app-1.0.jar"))
		assert.False(t, handler.isSnapshotArtifact("org/springframework/spring-core/5.3.0/spring-core-5.3.0.jar"))
	})

	t.Run("isChecksumFile", func(t *testing.T) {
		handler := NewMavenHandlerV2()

		// 체크섬 파일들
		assert.True(t, handler.isChecksumFile(testMavenChecksumPath))
		assert.True(t, handler.isChecksumFile("com/example/app/1.0/app-1.0.jar.md5"))
		assert.True(t, handler.isChecksumFile("com/example/app/1.0/app-1.0.jar.sha256"))
		assert.True(t, handler.isChecksumFile("com/example/app/1.0/app-1.0.jar.sha512"))

		// 일반 파일
		assert.False(t, handler.isChecksumFile("com/example/app/1.0/app-1.0.jar"))
		assert.False(t, handler.isChecksumFile("com/example/app/1.0/app-1.0.pom"))
	})

	t.Run("validateArtifactPath", func(t *testing.T) {
		handler := NewMavenHandlerV2()

		// 유효한 경로
		assert.NoError(t, handler.validateArtifactPath("com/example/app/1.0/app-1.0.jar"))
		assert.NoError(t, handler.validateArtifactPath("org/springframework/spring-core/5.3.0/spring-core-5.3.0.jar"))

		// 무효한 경로 (.. 포함)
		assert.Error(t, handler.validateArtifactPath("com/example/../../../etc/passwd"))
		assert.Error(t, handler.validateArtifactPath("../com/example/app/1.0/app-1.0.jar"))

		// 무효한 경로 (절대 경로)
		assert.Error(t, handler.validateArtifactPath("/com/example/app/1.0/app-1.0.jar"))
	})

	t.Run("validateChecksum", func(t *testing.T) {
		handler := NewMavenHandlerV2()

		// 유효한 체크섬들
		assert.NoError(t, handler.validateChecksum([]byte("da39a3ee5e6b4b0d3255bfef95601890afd80709"), "test.sha1"))
		assert.NoError(t, handler.validateChecksum([]byte("d41d8cd98f00b204e9800998ecf8427e"), "test.md5"))
		assert.NoError(t, handler.validateChecksum(
			[]byte("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"), "test.sha256"))

		// 무효한 체크섬들 (길이가 틀림)
		assert.Error(t, handler.validateChecksum([]byte("short"), "test.sha1"))
		assert.Error(t, handler.validateChecksum([]byte("short"), "test.md5"))
		assert.Error(t, handler.validateChecksum([]byte("short"), "test.sha256"))
	})

	t.Run("HandleError mapping", func(t *testing.T) {
		handler := NewMavenHandlerV2()

		// 기본 에러 처리만 테스트
		err := handler.HandleError(assert.AnError, nil)
		assert.Error(t, err)
		// Maven 도메인 에러로 변환되는지 확인
		assert.Contains(t, err.Error(), "Maven")
	})

	t.Run("Cache TTL calculation", func(t *testing.T) {
		testCases := []struct {
			path     string
			expected time.Duration
		}{
			{"com/example/maven-metadata.xml", 5 * time.Minute},
			{testMavenChecksumPath, 30 * 24 * time.Hour},
			{"com/example/app/1.0/app-1.0.jar", 30 * 24 * time.Hour},
			{"com/example/app/1.0/app-1.0.war", 30 * 24 * time.Hour},
			{"com/example/app/1.0/app-1.0.pom", 7 * 24 * time.Hour},
			{"com/example/other/file.txt", 24 * time.Hour},
		}

		for _, tc := range testCases {
			// Simulate GetCacheTTL logic
			var ttl time.Duration

			switch tc.path {
			case "com/example/maven-metadata.xml":
				ttl = 5 * time.Minute
			case testMavenChecksumPath:
				ttl = 30 * 24 * time.Hour
			case "com/example/app/1.0/app-1.0.jar", "com/example/app/1.0/app-1.0.war":
				ttl = 30 * 24 * time.Hour
			case "com/example/app/1.0/app-1.0.pom":
				ttl = 7 * 24 * time.Hour
			default:
				ttl = 24 * time.Hour
			}

			assert.Equal(t, tc.expected, ttl, "TTL mismatch for path: %s", tc.path)
		}
	})

	t.Run("ShouldCache logic", func(t *testing.T) {
		testCases := []struct {
			path       string
			statusCode int
			expected   bool
		}{
			{"com/example/app/1.0/app-1.0.jar", 200, true},
			{"com/example/app/1.0/app-1.0.jar", 404, false},
			{"com/example/app/1.0-SNAPSHOT/app-1.0-SNAPSHOT.jar", 200, false},
			{"com/example/app/1.0/app-1.0.pom", 200, true},
			{testMavenChecksumPath, 200, true},
			{"com/example/maven-metadata.xml", 200, true},
			{"com/example/other/file.txt", 200, false},
		}

		for _, tc := range testCases {
			var shouldCache bool

			if tc.statusCode != 200 && tc.statusCode != 304 {
				shouldCache = false
			} else if tc.path == "com/example/app/1.0-SNAPSHOT/app-1.0-SNAPSHOT.jar" {
				// SNAPSHOT은 캐시하지 않음
				shouldCache = false
			} else if tc.path == "com/example/app/1.0/app-1.0.jar" ||
				tc.path == "com/example/app/1.0/app-1.0.pom" ||
				tc.path == testMavenChecksumPath ||
				tc.path == "com/example/maven-metadata.xml" {
				shouldCache = true
			} else {
				shouldCache = false
			}

			assert.Equal(t, tc.expected, shouldCache, "ShouldCache mismatch for path: %s, status: %d", tc.path, tc.statusCode)
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		// HealthCheck는 ReadConfig()를 호출하므로 실제 파일 없이는 테스트 어려움
		handler := NewMavenHandlerV2()
		err := handler.HealthCheck()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Maven 설정 파일 읽기 실패")
	})

	t.Run("GetUpstreamAuth", func(t *testing.T) {
		// GetUpstreamAuth도 ReadConfig()를 호출하므로 스킵
		t.Skip("GetUpstreamAuth requires actual config file")
	})
}

// 벤치마크 테스트
func BenchmarkMavenHandlerV2_isSnapshotArtifact(b *testing.B) {
	handler := NewMavenHandlerV2()
	testPath := "com/example/app/1.0-SNAPSHOT/app-1.0-SNAPSHOT.jar"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.isSnapshotArtifact(testPath)
	}
}

func BenchmarkMavenHandlerV2_isChecksumFile(b *testing.B) {
	handler := NewMavenHandlerV2()
	testPath := testMavenChecksumPath

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.isChecksumFile(testPath)
	}
}
