//go:build linux

package metrics

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// getFileDescriptorCounts 파일 디스크립터 수 반환 (Linux)
func getFileDescriptorCounts() (open, max int) {
	// 현재 프로세스의 열린 파일 디스크립터 수
	pid := os.Getpid()
	fdPath := filepath.Join("/proc", strconv.Itoa(pid), "fd")

	if entries, err := os.ReadDir(fdPath); err == nil {
		open = len(entries)
	}

	// 최대 파일 디스크립터 수
	limitsPath := filepath.Join("/proc", strconv.Itoa(pid), "limits")
	if data, err := os.ReadFile(limitsPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Max open files") {
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					if n, err := strconv.Atoi(fields[3]); err == nil {
						max = n
					}
				}
				break
			}
		}
	}

	// 기본값
	if max == 0 {
		max = 1024
	}

	return open, max
}

// GetSystemMetrics 시스템 메트릭 수집 (Linux 전용)
func GetSystemMetrics() map[string]float64 {
	metrics := make(map[string]float64)

	// CPU 사용률 (간단한 버전)
	if stat, err := os.ReadFile("/proc/stat"); err == nil {
		lines := strings.Split(string(stat), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "cpu ") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					user, err := strconv.ParseFloat(fields[1], 64)
					if err != nil {
						user = 0
					}
					system, err := strconv.ParseFloat(fields[3], 64)
					if err != nil {
						system = 0
					}
					idle, err := strconv.ParseFloat(fields[4], 64)
					if err != nil {
						idle = 0
					}
					total := user + system + idle
					if total > 0 {
						metrics["cpu_usage"] = (user + system) / total
					}
				}
				break
			}
		}
	}

	// 메모리 사용률
	if meminfo, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(meminfo), "\n")
		var memTotal, memAvailable float64

		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				switch fields[0] {
				case "MemTotal:":
					if val, err := strconv.ParseFloat(fields[1], 64); err == nil {
						memTotal = val
					}
				case "MemAvailable:":
					if val, err := strconv.ParseFloat(fields[1], 64); err == nil {
						memAvailable = val
					}
				}
			}
		}

		if memTotal > 0 {
			metrics["memory_total_kb"] = memTotal
			metrics["memory_available_kb"] = memAvailable
			metrics["memory_usage"] = 1.0 - (memAvailable / memTotal)
		}
	}

	// 로드 평균
	if loadavg, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(loadavg))
		if len(fields) >= 3 {
			load1, err := strconv.ParseFloat(fields[0], 64)
			if err == nil {
				metrics["load_1m"] = load1
			}
			load5, err := strconv.ParseFloat(fields[1], 64)
			if err == nil {
				metrics["load_5m"] = load5
			}
			load15, err := strconv.ParseFloat(fields[2], 64)
			if err == nil {
				metrics["load_15m"] = load15
			}
		}
	}

	return metrics
}

// GetNetworkMetrics 네트워크 메트릭 수집 (Linux 전용)
func GetNetworkMetrics() map[string]map[string]float64 {
	metrics := make(map[string]map[string]float64)

	if netdev, err := os.ReadFile("/proc/net/dev"); err == nil {
		lines := strings.Split(string(netdev), "\n")

		for i, line := range lines {
			if i < 2 { // 헤더 건너뛰기
				continue
			}

			fields := strings.Fields(line)
			if len(fields) >= 17 {
				iface := strings.TrimSuffix(fields[0], ":")
				if iface == "lo" { // 루프백 인터페이스 제외
					continue
				}

				// ParseFloat 오류는 의도적으로 무시하고 기본값 0.0 사용
				//nolint:errcheck
				rxBytes, _ := strconv.ParseFloat(fields[1], 64)
				//nolint:errcheck
				rxPackets, _ := strconv.ParseFloat(fields[2], 64)
				//nolint:errcheck
				rxErrors, _ := strconv.ParseFloat(fields[3], 64)
				//nolint:errcheck
				rxDropped, _ := strconv.ParseFloat(fields[4], 64)

				//nolint:errcheck
				txBytes, _ := strconv.ParseFloat(fields[9], 64)
				//nolint:errcheck
				txPackets, _ := strconv.ParseFloat(fields[10], 64)
				//nolint:errcheck
				txErrors, _ := strconv.ParseFloat(fields[11], 64)
				//nolint:errcheck
				txDropped, _ := strconv.ParseFloat(fields[12], 64)

				// Note: ParseFloat errors are ignored for network metrics as they default to 0

				metrics[iface] = map[string]float64{
					"rx_bytes":   rxBytes,
					"rx_packets": rxPackets,
					"rx_errors":  rxErrors,
					"rx_dropped": rxDropped,
					"tx_bytes":   txBytes,
					"tx_packets": txPackets,
					"tx_errors":  txErrors,
					"tx_dropped": txDropped,
				}
			}
		}
	}

	return metrics
}

// GetDiskMetrics 디스크 메트릭 수집 (Linux 전용)
func GetDiskMetrics() map[string]map[string]float64 {
	metrics := make(map[string]map[string]float64)

	// df 명령어 대신 /proc/mounts와 statfs 시스템 콜 사용
	// 간단한 구현을 위해 여기서는 생략

	return metrics
}
