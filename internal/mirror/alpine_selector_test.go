//nolint:govet // 테스트 파일에서 unusedwrite 경고 무시
package mirror

import (
	"testing"
	"time"

	"github.com/go-playground/assert/v2"

	"proxynd/internal/config"
	"proxynd/internal/logging"
)

// TestNewAlpineMirrorSelector 미러 선택기 생성 테스트
func TestNewAlpineMirrorSelector(t *testing.T) {
	selector := NewAlpineMirrorSelector()

	assert.NotEqual(t, selector, nil)
	assert.Equal(t, len(selector.mirrorHealth), 0)
}

// TestExtractAlpineVersion Alpine 버전 추출 테스트
func TestExtractAlpineVersion(t *testing.T) {
	selector := NewAlpineMirrorSelector()

	// edge 버전 테스트
	version := selector.extractAlpineVersion("/proxy/apk/edge/main/x86_64/package.apk")
	assert.Equal(t, version.Version, "edge")
	assert.Equal(t, version.IsEdge, true)

	// 일반 버전 테스트
	version = selector.extractAlpineVersion("/proxy/apk/v3.18/main/x86_64/package.apk")
	assert.Equal(t, version.Version, "v3.18")
	assert.Equal(t, version.Major, 3)
	assert.Equal(t, version.Minor, 18)
	assert.Equal(t, version.IsEdge, false)

	// 다른 버전 테스트
	version = selector.extractAlpineVersion("/proxy/apk/v3.19/community/aarch64/package.apk")
	assert.Equal(t, version.Version, "v3.19")
	assert.Equal(t, version.Major, 3)
	assert.Equal(t, version.Minor, 19)

	// 버전이 없는 경우 기본값 테스트
	version = selector.extractAlpineVersion("/proxy/apk/some/path/package.apk")
	assert.Equal(t, version.Version, "v3.19")
	assert.Equal(t, version.Major, 3)
	assert.Equal(t, version.Minor, 19)
}

// TestDetectRegion 지역 감지 테스트
func TestDetectRegion(t *testing.T) {
	selector := NewAlpineMirrorSelector()

	// 한국 미러 테스트
	region := selector.detectRegion("https://mirror.kakao.com/alpine")
	assert.Equal(t, region, "korea")

	region = selector.detectRegion("https://mirror.navercorp.com/alpine")
	assert.Equal(t, region, "korea")

	// 일본 미러 테스트
	region = selector.detectRegion("https://ftp.riken.jp/alpine")
	assert.Equal(t, region, "japan")

	// 중국 미러 테스트
	region = selector.detectRegion("https://mirrors.tuna.tsinghua.edu.cn/alpine")
	assert.Equal(t, region, "china")

	// 유럽 미러 테스트
	region = selector.detectRegion("https://mirrors.dotsrc.org/alpine")
	assert.Equal(t, region, "europe")

	// 공식 미러 테스트
	region = selector.detectRegion("https://dl-cdn.alpinelinux.org/alpine")
	assert.Equal(t, region, "official")

	// 북미 미러 테스트
	region = selector.detectRegion("https://alpine.global.ssl.fastly.net/alpine")
	assert.Equal(t, region, "north_america")

	// 기타 미러 테스트
	region = selector.detectRegion("https://unknown.mirror.com/alpine")
	assert.Equal(t, region, "global")
}

// TestCalculatePriority 우선순위 계산 테스트
func TestCalculatePriority(t *testing.T) {
	selector := NewAlpineMirrorSelector()

	// 선호 지역 설정
	preferredRegions := []string{"korea", "japan", "official"}

	// 선호 지역 우선순위 테스트
	priority := selector.calculatePriority("korea", preferredRegions)
	assert.Equal(t, priority, 1)

	priority = selector.calculatePriority("japan", preferredRegions)
	assert.Equal(t, priority, 2)

	priority = selector.calculatePriority("official", preferredRegions)
	assert.Equal(t, priority, 3)

	// 기본 우선순위 테스트
	priority = selector.calculatePriority("china", preferredRegions)
	assert.Equal(t, priority, 25)

	priority = selector.calculatePriority("global", preferredRegions)
	assert.Equal(t, priority, 100)
}

// TestCalculateRegionScore 지역 점수 계산 테스트
func TestCalculateRegionScore(t *testing.T) {
	selector := NewAlpineMirrorSelector()

	// 같은 지역 점수
	score := selector.calculateRegionScore("korea", "korea")
	assert.Equal(t, score, 30.0)

	// 인접 지역 점수
	score = selector.calculateRegionScore("japan", "korea")
	assert.Equal(t, score, 25.0)

	score = selector.calculateRegionScore("asia", "korea")
	assert.Equal(t, score, 25.0)

	// 중국 점수
	score = selector.calculateRegionScore("china", "korea")
	assert.Equal(t, score, 20.0)

	// 공식 미러 점수
	score = selector.calculateRegionScore("official", "korea")
	assert.Equal(t, score, 15.0)

	// 글로벌 점수
	score = selector.calculateRegionScore("global", "korea")
	assert.Equal(t, score, 10.0)

	// 기타 지역 점수
	score = selector.calculateRegionScore("europe", "korea")
	assert.Equal(t, score, 5.0)
}

// TestCalculateMirrorScore 미러 점수 계산 테스트
func TestCalculateMirrorScore(t *testing.T) {
	selector := NewAlpineMirrorSelector()

	// 테스트 미러 헬스 정보 설정
	selector.mirrorHealth["test-mirror"] = &MirrorHealth{
		Name:         "test-mirror",
		URL:          "https://test.mirror.com/alpine",
		ResponseTime: 100 * time.Millisecond,
		IsHealthy:    true,
		ErrorCount:   0,
		Region:       "korea",
		Priority:     1,
	}

	proxy := config.ApkProxy{
		Name: "test-mirror",
		URL:  "https://test.mirror.com/alpine",
	}

	version := AlpineVersion{
		Major:   3,
		Minor:   18,
		Version: "v3.18",
		IsEdge:  false,
	}

	score := selector.calculateMirrorScore(proxy, version, "korea")

	// 점수가 합리적인 범위에 있는지 확인
	assert.Equal(t, score > 100.0, true) // 건강한 미러는 100점 이상
	assert.Equal(t, score < 200.0, true) // 최대 점수 확인
}

// TestSelectBestMirror 최적 미러 선택 테스트
func TestSelectBestMirror(t *testing.T) {
	selector := NewAlpineMirrorSelector()

	// 테스트 프록시 목록
	proxies := []config.ApkProxy{
		{Name: "korea-mirror", URL: "https://mirror.kakao.com/alpine"},
		{Name: "japan-mirror", URL: "https://ftp.riken.jp/alpine"},
		{Name: "official-mirror", URL: "https://dl-cdn.alpinelinux.org/alpine"},
	}

	// 테스트 미러 헬스 정보 설정
	selector.mirrorHealth["korea-mirror"] = &MirrorHealth{
		Name:         "korea-mirror",
		IsHealthy:    true,
		ResponseTime: 50 * time.Millisecond,
		Region:       "korea",
		Priority:     1,
		ErrorCount:   0,
	}

	selector.mirrorHealth["japan-mirror"] = &MirrorHealth{
		Name:         "japan-mirror",
		IsHealthy:    true,
		ResponseTime: 80 * time.Millisecond,
		Region:       "japan",
		Priority:     2,
		ErrorCount:   0,
	}

	selector.mirrorHealth["official-mirror"] = &MirrorHealth{
		Name:         "official-mirror",
		IsHealthy:    true,
		ResponseTime: 120 * time.Millisecond,
		Region:       "official",
		Priority:     3,
		ErrorCount:   0,
	}

	// 미러 선택 테스트
	selected := selector.SelectBestMirror("/proxy/apk/v3.18/main/x86_64/package.apk", proxies)

	// 선택된 미러가 있는지 확인
	assert.Equal(t, len(selected) > 0, true)

	// 첫 번째 미러가 한국 미러인지 확인 (가장 높은 점수)
	assert.Equal(t, selected[0].Name, "korea-mirror")
}

// TestMirrorHealthStats 미러 헬스 통계 테스트
func TestMirrorHealthStats(t *testing.T) {
	// 새로운 선택기 인스턴스 생성 (싱글톤 우회)
	selector := &AlpineMirrorSelector{
		logger:       logging.GetLogger(),
		mirrorHealth: make(map[string]*MirrorHealth),
		stopCh:       make(chan bool),
	}

	// 테스트 미러 헬스 정보 설정
	selector.mirrorHealth["healthy-mirror"] = &MirrorHealth{
		Name:         "healthy-mirror",
		IsHealthy:    true,
		ResponseTime: 100 * time.Millisecond,
	}

	selector.mirrorHealth["unhealthy-mirror"] = &MirrorHealth{
		Name:         "unhealthy-mirror",
		IsHealthy:    false,
		ResponseTime: 0,
	}

	stats := selector.GetHealthStats()

	assert.Equal(t, stats["total_mirrors"], 2)
	assert.Equal(t, stats["healthy_mirrors"], 1)
	assert.Equal(t, stats["unhealthy_mirrors"], 1)
	assert.Equal(t, stats["health_rate"], 50.0)
}

// TestAlpineVersionStruct Alpine 버전 구조체 테스트
func TestAlpineVersionStruct(t *testing.T) {
	// 일반 버전
	version := AlpineVersion{
		Major:   3,
		Minor:   18,
		Version: "v3.18",
		IsEdge:  false,
	}

	assert.Equal(t, version.Major, 3)
	assert.Equal(t, version.Minor, 18)
	assert.Equal(t, version.Version, "v3.18")
	assert.Equal(t, version.IsEdge, false)

	// Edge 버전
	edgeVersion := AlpineVersion{
		Major:   99,
		Minor:   99,
		Version: "edge",
		IsEdge:  true,
	}

	assert.Equal(t, edgeVersion.Version, "edge")
	assert.Equal(t, edgeVersion.IsEdge, true)
}

// TestMirrorHealthStruct 미러 헬스 구조체 테스트
func TestMirrorHealthStruct(t *testing.T) {
	health := &MirrorHealth{
		Name:         "test-mirror",
		URL:          "https://test.mirror.com",
		ResponseTime: 100 * time.Millisecond,
		LastCheck:    time.Now(),
		IsHealthy:    true,
		ErrorCount:   0,
		Region:       "korea",
		Priority:     1,
	}

	assert.Equal(t, health.Name, "test-mirror")
	assert.Equal(t, health.URL, "https://test.mirror.com")
	assert.Equal(t, health.IsHealthy, true)
	assert.Equal(t, health.ErrorCount, 0)
	assert.Equal(t, health.Region, "korea")
	assert.Equal(t, health.Priority, 1)
}
