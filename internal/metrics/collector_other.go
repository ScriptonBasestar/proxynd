//go:build !linux
// +build !linux

package metrics

// getFileDescriptorCounts 파일 디스크립터 수 반환 (non-Linux)
func getFileDescriptorCounts() (open, maxVal int) {
	// Linux가 아닌 시스템에서는 기본값 반환
	return 0, 1024
}

// GetSystemMetrics 시스템 메트릭 수집 (non-Linux)
func GetSystemMetrics() map[string]float64 {
	// Linux가 아닌 시스템에서는 빈 맵 반환
	return make(map[string]float64)
}

// GetNetworkMetrics 네트워크 메트릭 수집 (non-Linux)
func GetNetworkMetrics() map[string]map[string]float64 {
	// Linux가 아닌 시스템에서는 빈 맵 반환
	return make(map[string]map[string]float64)
}

// GetDiskMetrics 디스크 메트릭 수집 (non-Linux)
func GetDiskMetrics() map[string]map[string]float64 {
	// Linux가 아닌 시스템에서는 빈 맵 반환
	return make(map[string]map[string]float64)
}
