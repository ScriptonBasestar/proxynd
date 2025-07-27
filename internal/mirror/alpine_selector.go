// Package mirror provides mirror selection and management
package mirror

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"proxynd/internal/config"
	"proxynd/logging"
)

// Region constants
const (
	// RegionKorea is a const that region korea
	// RegionJapan is a const that region japan
	// RegionChina is a const that region china
	// RegionAsia is a const that region asia
	// RegionOfficial is a const that region official
	// RegionGlobal is a const that region global
	RegionKorea    = "korea"
	RegionJapan    = "japan"
	RegionChina    = "china"
	RegionAsia     = "asia"
	RegionOfficial = "official"
	RegionGlobal   = "global"
)

// AlpineVersion Alpine 버전 정보
type AlpineVersion struct {
	Major   int    `json:"major"`
	Minor   int    `json:"minor"`
	Version string `json:"version"` // v3.18, v3.19, edge
	IsEdge  bool   `json:"is_edge"`
}

// MirrorHealth 미러 상태 정보
type MirrorHealth struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	ResponseTime time.Duration `json:"response_time"`
	LastCheck    time.Time     `json:"last_check"`
	IsHealthy    bool          `json:"is_healthy"`
	ErrorCount   int           `json:"error_count"`
	Region       string        `json:"region"`
	Priority     int           `json:"priority"`
}

// AlpineMirrorSelector Alpine 미러 선택기
type AlpineMirrorSelector struct {
	logger       logging.Logger
	mirrorHealth map[string]*MirrorHealth
	healthMutex  sync.RWMutex
	lastUpdate   time.Time
	updateTicker *time.Ticker
	stopCh       chan bool
}

// AlpineMirrorConfig 미러 선택 설정
type AlpineMirrorConfig struct {
	HealthCheckInterval time.Duration `yaml:"health_check_interval"` // 헬스체크 간격
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout"`  // 헬스체크 타임아웃
	PreferredRegions    []string      `yaml:"preferred_regions"`     // 선호하는 지역
	FallbackToGlobal    bool          `yaml:"fallback_to_global"`    // 글로벌 미러로 폴백 여부
	MaxErrorCount       int           `yaml:"max_error_count"`       // 최대 에러 허용 횟수
	RegionDetectionMode string        `yaml:"region_detection_mode"` // auto, manual, disabled
}

var (
	alpineVersionRegex = regexp.MustCompile(`/?(v\d+\.\d+|edge)/`)
	selector           *AlpineMirrorSelector
	selectorOnce       sync.Once
)

// NewAlpineMirrorSelector 새로운 Alpine 미러 선택기 생성
func NewAlpineMirrorSelector() *AlpineMirrorSelector {
	selectorOnce.Do(func() {
		selector = &AlpineMirrorSelector{
			logger:       logging.GetLogger(),
			mirrorHealth: make(map[string]*MirrorHealth),
			stopCh:       make(chan bool),
		}
	})
	return selector
}

// GetAlpineMirrorSelector 싱글톤 인스턴스 반환
func GetAlpineMirrorSelector() *AlpineMirrorSelector {
	return NewAlpineMirrorSelector()
}

// Start 미러 헬스체크 시작
func (ams *AlpineMirrorSelector) Start(config AlpineMirrorConfig, proxies []config.ApkProxy) {
	// 기본값 설정
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 5 * time.Minute
	}
	if config.HealthCheckTimeout == 0 {
		config.HealthCheckTimeout = 10 * time.Second
	}
	if config.MaxErrorCount == 0 {
		config.MaxErrorCount = 3
	}

	// 미러 헬스 정보 초기화
	ams.healthMutex.Lock()
	for _, proxy := range proxies {
		region := ams.detectRegion(proxy.URL)
		priority := ams.calculatePriority(region, config.PreferredRegions)

		ams.mirrorHealth[proxy.Name] = &MirrorHealth{
			Name:         proxy.Name,
			URL:          proxy.URL,
			ResponseTime: 0,
			LastCheck:    time.Time{},
			IsHealthy:    true, // 초기에는 healthy로 가정
			ErrorCount:   0,
			Region:       region,
			Priority:     priority,
		}
	}
	ams.healthMutex.Unlock()

	// 초기 헬스체크 수행
	ams.performHealthCheck(config)

	// 주기적 헬스체크 시작
	ams.updateTicker = time.NewTicker(config.HealthCheckInterval)
	go func() {
		for {
			select {
			case <-ams.updateTicker.C:
				ams.performHealthCheck(config)
			case <-ams.stopCh:
				ams.updateTicker.Stop()
				return
			}
		}
	}()

	ams.logger.Info("Alpine 미러 선택기 시작됨",
		logging.F("mirrors", len(proxies)),
		logging.F("check_interval", config.HealthCheckInterval),
		logging.F("preferred_regions", config.PreferredRegions))
}

// Stop 미러 헬스체크 중지
func (ams *AlpineMirrorSelector) Stop() {
	if ams.updateTicker != nil {
		ams.updateTicker.Stop()
	}
	select {
	case ams.stopCh <- true:
	default:
	}
	ams.logger.Info("Alpine 미러 선택기 중지됨")
}

// SelectBestMirror 요청 경로에 따른 최적 미러 선택
func (ams *AlpineMirrorSelector) SelectBestMirror(requestPath string, proxies []config.ApkProxy) []config.ApkProxy {
	version := ams.extractAlpineVersion(requestPath)
	clientRegion := ams.detectClientRegion() // 클라이언트 지역 감지

	ams.logger.Debug("미러 선택 요청",
		logging.F("path", requestPath),
		logging.F("version", version.Version),
		logging.F("client_region", clientRegion))

	// 건강한 미러들만 필터링
	healthyMirrors := ams.getHealthyMirrors(proxies)
	if len(healthyMirrors) == 0 {
		ams.logger.Warn("건강한 미러가 없음, 모든 미러 사용")
		return proxies
	}

	// 미러 순서 결정
	selectedMirrors := ams.rankMirrors(healthyMirrors, version, clientRegion)

	ams.logger.Info("미러 선택 완료",
		logging.F("selected_count", len(selectedMirrors)),
		logging.F("primary_mirror", selectedMirrors[0].Name))

	return selectedMirrors
}

// extractAlpineVersion 요청 경로에서 Alpine 버전 추출
func (ams *AlpineMirrorSelector) extractAlpineVersion(requestPath string) AlpineVersion {
	matches := alpineVersionRegex.FindStringSubmatch(requestPath)
	if len(matches) > 1 {
		versionStr := matches[1]
		if versionStr == "edge" {
			return AlpineVersion{
				Major:   99,
				Minor:   99,
				Version: "edge",
				IsEdge:  true,
			}
		}

		// v3.18 형식 파싱
		if strings.HasPrefix(versionStr, "v") {
			parts := strings.Split(versionStr[1:], ".")
			if len(parts) >= 2 {
				var major, minor int
				// 버전 파싱 오류는 무시하고 기본값 0 사용
				//nolint:errcheck
				_, _ = fmt.Sscanf(parts[0], "%d", &major)
				//nolint:errcheck
				_, _ = fmt.Sscanf(parts[1], "%d", &minor)
				return AlpineVersion{
					Major:   major,
					Minor:   minor,
					Version: versionStr,
					IsEdge:  false,
				}
			}
		}
	}

	// 기본값: 최신 안정 버전
	return AlpineVersion{
		Major:   3,
		Minor:   19,
		Version: "v3.19",
		IsEdge:  false,
	}
}

// detectRegion URL에서 지역 감지
func (ams *AlpineMirrorSelector) detectRegion(url string) string {
	// URL 기반 지역 감지 로직
	lowerURL := strings.ToLower(url)

	// 한국
	if strings.Contains(lowerURL, "kakao.com") || strings.Contains(lowerURL, "naver") ||
		strings.Contains(lowerURL, "korea") || strings.Contains(lowerURL, ".kr") {
		return RegionKorea
	}

	// 일본
	if strings.Contains(lowerURL, "japan") || strings.Contains(lowerURL, ".jp") ||
		strings.Contains(lowerURL, "riken") {
		return RegionJapan
	}

	// 중국
	if strings.Contains(lowerURL, "china") || strings.Contains(lowerURL, ".cn") ||
		strings.Contains(lowerURL, "tsinghua") || strings.Contains(lowerURL, "ustc") {
		return RegionChina
	}

	// 아시아
	if strings.Contains(lowerURL, "asia") || strings.Contains(lowerURL, "singapore") ||
		strings.Contains(lowerURL, ".sg") {
		return RegionAsia
	}

	// 유럽
	if strings.Contains(lowerURL, "europe") || strings.Contains(lowerURL, "dotsrc") ||
		strings.Contains(lowerURL, ".dk") || strings.Contains(lowerURL, ".de") ||
		strings.Contains(lowerURL, ".uk") {
		return "europe"
	}

	// 북미
	if strings.Contains(lowerURL, "fastly") || strings.Contains(lowerURL, "america") ||
		strings.Contains(lowerURL, ".us") {
		return "north_america"
	}

	// 공식/글로벌
	if strings.Contains(lowerURL, "alpinelinux.org") || strings.Contains(lowerURL, "dl-cdn") {
		return RegionOfficial
	}

	return RegionGlobal
}

// calculatePriority 지역과 선호도에 따른 우선순위 계산
func (ams *AlpineMirrorSelector) calculatePriority(region string, preferredRegions []string) int {
	for i, preferred := range preferredRegions {
		if region == preferred {
			return i + 1 // 1이 가장 높은 우선순위
		}
	}

	// 기본 우선순위
	switch region {
	case RegionKorea:
		return 10
	case RegionJapan, RegionAsia:
		return 20
	case RegionChina:
		return 25
	case RegionOfficial:
		return 30
	case "europe":
		return 40
	case "north_america":
		return 50
	default:
		return 100
	}
}

// detectClientRegion 클라이언트 지역 감지 (향후 확장용)
func (ams *AlpineMirrorSelector) detectClientRegion() string {
	// 현재는 기본값으로 korea 반환
	// 향후 GeoIP나 다른 방법으로 확장 가능
	return RegionKorea
}

// performHealthCheck 모든 미러에 대한 헬스체크 수행
func (ams *AlpineMirrorSelector) performHealthCheck(config AlpineMirrorConfig) {
	ams.healthMutex.RLock()
	mirrors := make([]*MirrorHealth, 0, len(ams.mirrorHealth))
	for _, mirror := range ams.mirrorHealth {
		mirrors = append(mirrors, mirror)
	}
	ams.healthMutex.RUnlock()

	var wg sync.WaitGroup
	for _, mirror := range mirrors {
		wg.Add(1)
		go func(m *MirrorHealth) {
			defer wg.Done()
			ams.checkMirrorHealth(m, config)
		}(mirror)
	}
	wg.Wait()

	ams.lastUpdate = time.Now()
	ams.logger.Debug("미러 헬스체크 완료", logging.F("mirrors", len(mirrors)))
}

// checkMirrorHealth 개별 미러 헬스체크
func (ams *AlpineMirrorSelector) checkMirrorHealth(mirror *MirrorHealth, config AlpineMirrorConfig) {
	start := time.Now()

	// 헬스체크 URL 생성 (가벼운 파일 요청)
	healthURL := strings.TrimSuffix(mirror.URL, "/") + "/v3.19/main/x86_64/APKINDEX.tar.gz"

	ctx, cancel := context.WithTimeout(context.Background(), config.HealthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", healthURL, nil)
	if err != nil {
		ams.recordMirrorError(mirror)
		return
	}

	client := &http.Client{
		Timeout: config.HealthCheckTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		ams.recordMirrorError(mirror)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	responseTime := time.Since(start)

	ams.healthMutex.Lock()
	mirror.ResponseTime = responseTime
	mirror.LastCheck = time.Now()

	if resp.StatusCode == http.StatusOK {
		mirror.IsHealthy = true
		mirror.ErrorCount = 0
	} else {
		mirror.ErrorCount++
		if mirror.ErrorCount >= config.MaxErrorCount {
			mirror.IsHealthy = false
		}
	}
	ams.healthMutex.Unlock()

	ams.logger.Debug("미러 헬스체크",
		logging.F("mirror", mirror.Name),
		logging.F("healthy", mirror.IsHealthy),
		logging.F("response_time", responseTime),
		logging.F("status_code", resp.StatusCode))
}

// recordMirrorError 미러 에러 기록
func (ams *AlpineMirrorSelector) recordMirrorError(mirror *MirrorHealth) {
	ams.healthMutex.Lock()
	defer ams.healthMutex.Unlock()

	mirror.ErrorCount++
	mirror.LastCheck = time.Now()
	mirror.ResponseTime = 0

	if mirror.ErrorCount >= 3 { // 하드코딩된 값, 설정으로 이동 가능
		mirror.IsHealthy = false
	}

	ams.logger.Warn("미러 에러 발생",
		logging.F("mirror", mirror.Name),
		logging.F("error_count", mirror.ErrorCount),
		logging.F("healthy", mirror.IsHealthy))
}

// getHealthyMirrors 건강한 미러들만 필터링
func (ams *AlpineMirrorSelector) getHealthyMirrors(proxies []config.ApkProxy) []config.ApkProxy {
	ams.healthMutex.RLock()
	defer ams.healthMutex.RUnlock()

	var healthy []config.ApkProxy
	for _, proxy := range proxies {
		if health, exists := ams.mirrorHealth[proxy.Name]; exists && health.IsHealthy {
			healthy = append(healthy, proxy)
		} else if !exists {
			// 헬스 정보가 없는 경우 포함 (새로 추가된 미러)
			healthy = append(healthy, proxy)
		}
	}
	return healthy
}

// rankMirrors 미러들을 우선순위에 따라 정렬
func (ams *AlpineMirrorSelector) rankMirrors(
	proxies []config.ApkProxy,
	version AlpineVersion,
	clientRegion string,
) []config.ApkProxy {
	ams.healthMutex.RLock()
	defer ams.healthMutex.RUnlock()

	// 미러 점수 계산을 위한 구조체
	type mirrorScore struct {
		proxy config.ApkProxy
		score float64
	}

	var scored []mirrorScore
	for _, proxy := range proxies {
		score := ams.calculateMirrorScore(proxy, version, clientRegion)
		scored = append(scored, mirrorScore{proxy: proxy, score: score})
	}

	// 점수에 따라 정렬 (높은 점수가 우선)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 정렬된 프록시 목록 반환
	result := make([]config.ApkProxy, len(scored))
	for i, item := range scored {
		result[i] = item.proxy
	}

	return result
}

// calculateMirrorScore 미러 점수 계산
func (ams *AlpineMirrorSelector) calculateMirrorScore(
	proxy config.ApkProxy, _ AlpineVersion, clientRegion string,
) float64 {
	health, exists := ams.mirrorHealth[proxy.Name]
	if !exists {
		return 50.0 // 기본 점수
	}

	score := 100.0

	// 건강 상태 점수
	if !health.IsHealthy {
		score -= 50.0
	}

	// 응답 시간 점수 (빠를수록 높은 점수)
	if health.ResponseTime > 0 {
		responseScore := 30.0 - (float64(health.ResponseTime.Milliseconds()) / 100.0)
		if responseScore < 0 {
			responseScore = 0
		}
		score += responseScore
	}

	// 지역 점수
	regionScore := ams.calculateRegionScore(health.Region, clientRegion)
	score += regionScore

	// 우선순위 점수 (낮은 우선순위 값이 높은 점수)
	priorityScore := 20.0 - float64(health.Priority)
	if priorityScore < 0 {
		priorityScore = 0
	}
	score += priorityScore

	// 에러 점수 차감
	score -= float64(health.ErrorCount) * 5.0

	if score < 0 {
		score = 0
	}

	return score
}

// calculateRegionScore 지역에 따른 점수 계산
func (ams *AlpineMirrorSelector) calculateRegionScore(mirrorRegion, clientRegion string) float64 {
	if mirrorRegion == clientRegion {
		return 30.0 // 같은 지역
	}

	// 지역별 근접도 점수
	switch clientRegion {
	case RegionKorea:
		switch mirrorRegion {
		case RegionJapan, RegionAsia:
			return 25.0
		case RegionChina:
			return 20.0
		case RegionOfficial:
			return 15.0
		case RegionGlobal:
			return 10.0
		default:
			return 5.0
		}
	default:
		switch mirrorRegion {
		case RegionOfficial:
			return 20.0
		case RegionGlobal:
			return 15.0
		default:
			return 10.0
		}
	}
}

// GetMirrorHealth 미러 상태 정보 반환
func (ams *AlpineMirrorSelector) GetMirrorHealth() map[string]*MirrorHealth {
	ams.healthMutex.RLock()
	defer ams.healthMutex.RUnlock()

	result := make(map[string]*MirrorHealth)
	for name, health := range ams.mirrorHealth {
		// 복사본 생성
		healthCopy := *health
		result[name] = &healthCopy
	}
	return result
}

// GetHealthStats 헬스체크 통계 반환
func (ams *AlpineMirrorSelector) GetHealthStats() map[string]interface{} {
	ams.healthMutex.RLock()
	defer ams.healthMutex.RUnlock()

	totalMirrors := len(ams.mirrorHealth)
	healthyMirrors := 0
	var totalResponseTime time.Duration
	responseCount := 0

	for _, health := range ams.mirrorHealth {
		if health.IsHealthy {
			healthyMirrors++
		}
		if health.ResponseTime > 0 {
			totalResponseTime += health.ResponseTime
			responseCount++
		}
	}

	avgResponseTime := time.Duration(0)
	if responseCount > 0 {
		avgResponseTime = totalResponseTime / time.Duration(responseCount)
	}

	return map[string]interface{}{
		"total_mirrors":     totalMirrors,
		"healthy_mirrors":   healthyMirrors,
		"unhealthy_mirrors": totalMirrors - healthyMirrors,
		"health_rate":       float64(healthyMirrors) / float64(totalMirrors) * 100,
		"avg_response_time": avgResponseTime,
		"last_update":       ams.lastUpdate,
	}
}
