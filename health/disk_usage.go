package health

import (
	"syscall"
)

// DiskStats 디스크 통계
type DiskStats struct {
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	FreePercent float64 `json:"free_percent"`
	UsedPercent float64 `json:"used_percent"`
}

// getDiskUsage 디스크 사용량 조회
func getDiskUsage(path string) (*DiskStats, error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}

	// 블록 크기와 블록 수를 곱하여 바이트 단위로 변환
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	// 퍼센트 계산
	var freePercent, usedPercent float64
	if total > 0 {
		freePercent = float64(free) / float64(total) * 100
		usedPercent = float64(used) / float64(total) * 100
	}

	return &DiskStats{
		Total:       total,
		Free:        free,
		Used:        used,
		FreePercent: freePercent,
		UsedPercent: usedPercent,
	}, nil
}
