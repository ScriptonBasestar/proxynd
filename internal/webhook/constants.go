package webhook

// Common field names used in webhook operations
const (
	fieldError        = "error"
	fieldEndpoint     = "endpoint"
	fieldEventID      = "event_id"
	fieldWorkerID     = "worker_id"
	fieldItemID       = "item_id"
	fieldAttempts     = "attempts"
	fieldFile         = "file"
	fieldNextRetry    = "next_retry"
	fieldResponseTime = "response_time"
)

// Status values
const (
	statusSuccess = "success"
	statusFailed  = "failed"
)

// Webhook types
const (
	webhookTypeSlack = "slack"
)

// HTTP methods
const (
	methodPOST = "POST"
)
