package interfaces

import (
	"context"
	
	"proxynd/alerts"
	"proxynd/configs"
)

// WebhookBatchManager defines the interface for webhook batch management
type WebhookBatchManager interface {
	// Start begins the batch processing
	Start(ctx context.Context)
	
	// Stop gracefully stops the batch processing
	Stop(ctx context.Context) error
	
	// AddEvent adds an event to the appropriate batch group
	AddEvent(event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig)
	
	// GetStats returns batch processing statistics
	GetStats() WebhookBatchStats
}

// WebhookBatchStats represents webhook batch processing statistics
type WebhookBatchStats struct {
	ActiveGroups   int                    `json:"active_groups"`
	TotalBatches   int64                  `json:"total_batches"`
	TotalEvents    int64                  `json:"total_events"`
	FailedBatches  int64                  `json:"failed_batches"`
	GroupStats     map[string]GroupStats  `json:"group_stats"`
}

// GroupStats represents statistics for a single batch group
type GroupStats struct {
	EventCount     int   `json:"event_count"`
	LastFlush      int64 `json:"last_flush"`
	TotalFlushed   int64 `json:"total_flushed"`
}

// WebhookSender defines the interface for sending webhooks
type WebhookSender interface {
	// Send sends a single webhook
	Send(ctx context.Context, endpoint string, payload interface{}) error
	
	// SendBatch sends multiple webhooks in a batch
	SendBatch(ctx context.Context, endpoint string, payloads []interface{}) error
}