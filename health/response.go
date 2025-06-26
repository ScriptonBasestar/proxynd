package health

import (
	"time"
)

// HealthResponse 건강 상태 응답
type HealthResponse struct {
	Status      Status                    `json:"status"`
	Timestamp   time.Time                 `json:"timestamp"`
	Uptime      string                    `json:"uptime"`
	Checks      map[string]*CheckResult   `json:"checks"`
	Version     string                    `json:"version,omitempty"`
	Environment string                    `json:"environment,omitempty"`
	Details     map[string]interface{}    `json:"details,omitempty"`
}

// SimpleHealthResponse 간단한 건강 상태 응답
type SimpleHealthResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// LivenessResponse 라이브니스 응답
type LivenessResponse struct {
	Status string `json:"status"`
}

// ReadinessResponse 레디니스 응답
type ReadinessResponse struct {
	Ready   bool                   `json:"ready"`
	Checks  map[string]bool        `json:"checks"`
	Details map[string]interface{} `json:"details,omitempty"`
}

