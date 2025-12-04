package webhook

import (
	"encoding/json"
	"net/http"
	"time"

	"proxynd/internal/logging"
)

// DLQHandler HTTP handler for Dead Letter Queue management
type DLQHandler struct {
	dlq    *DeadLetterQueue
	sender *WebhookSender
	logger logging.Logger
}

// NewDLQHandler creates a new DLQ HTTP handler
func NewDLQHandler(sender *WebhookSender) *DLQHandler {
	return &DLQHandler{
		dlq:    sender.dlq,
		sender: sender,
		logger: logging.GetLogger(),
	}
}

// ListDLQItems handles GET /api/webhooks/dlq
// Returns all items in the DLQ
func (h *DLQHandler) ListDLQItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	items, err := h.dlq.List()
	if err != nil {
		h.logger.Error("Failed to list DLQ items", logging.F("error", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"items": items,
		"count": len(items),
		"timestamp": time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", logging.F("error", err))
	}
}

// GetDLQItem handles GET /api/webhooks/dlq/{id}
// Returns a specific DLQ item
func (h *DLQHandler) GetDLQItem(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	item, err := h.dlq.Get(id)
	if err != nil {
		h.logger.Error("Failed to get DLQ item", logging.F("id", id), logging.F("error", err))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(item); err != nil {
		h.logger.Error("Failed to encode response", logging.F("error", err))
	}
}

// RetryDLQItem handles POST /api/webhooks/dlq/{id}/retry
// Retries a failed webhook from DLQ
func (h *DLQHandler) RetryDLQItem(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve item from DLQ
	item, err := h.dlq.Retry(id)
	if err != nil {
		h.logger.Error("Failed to retry DLQ item", logging.F("id", id), logging.F("error", err))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Re-queue the event for sending
	if err := h.sender.SendEvent(item.Event); err != nil {
		h.logger.Error("Failed to re-queue event",
			logging.F("id", id),
			logging.F("event_id", item.Event.ID),
			logging.F("error", err))
		http.Error(w, "Failed to re-queue event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("DLQ item retried successfully",
		logging.F("id", id),
		logging.F("event_id", item.Event.ID),
		logging.F("endpoint", item.Endpoint))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "retried",
		"id":        id,
		"event_id":  item.Event.ID,
		"endpoint":  item.Endpoint,
		"timestamp": time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", logging.F("error", err))
	}
}

// DeleteDLQItem handles DELETE /api/webhooks/dlq/{id}
// Permanently removes an item from DLQ
func (h *DLQHandler) DeleteDLQItem(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.dlq.Remove(id); err != nil {
		h.logger.Error("Failed to delete DLQ item", logging.F("id", id), logging.F("error", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("DLQ item deleted", logging.F("id", id))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "deleted",
		"id":        id,
		"timestamp": time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", logging.F("error", err))
	}
}

// GetDLQStatistics handles GET /api/webhooks/dlq/stats
// Returns DLQ statistics
func (h *DLQHandler) GetDLQStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.dlq.GetStatistics()
	if err != nil {
		h.logger.Error("Failed to get DLQ statistics", logging.F("error", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"statistics": stats,
		"timestamp":  time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", logging.F("error", err))
	}
}

// PurgeDLQItems handles POST /api/webhooks/dlq/purge
// Purges old items from DLQ
func (h *DLQHandler) PurgeDLQItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body for purge duration
	var req struct {
		BeforeHours int `json:"before_hours"` // Purge items older than X hours
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.BeforeHours <= 0 {
		req.BeforeHours = 168 // Default: 7 days
	}

	beforeTime := time.Now().Add(-time.Duration(req.BeforeHours) * time.Hour)
	removed, err := h.dlq.Purge(beforeTime)
	if err != nil {
		h.logger.Error("Failed to purge DLQ", logging.F("error", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("DLQ purged",
		logging.F("removed", removed),
		logging.F("before_hours", req.BeforeHours))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":       "purged",
		"removed":      removed,
		"before_hours": req.BeforeHours,
		"before_time":  beforeTime,
		"timestamp":    time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", logging.F("error", err))
	}
}

// ClearDLQ handles DELETE /api/webhooks/dlq
// Clears all items from DLQ
func (h *DLQHandler) ClearDLQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.dlq.Clear(); err != nil {
		h.logger.Error("Failed to clear DLQ", logging.F("error", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("DLQ cleared")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "cleared",
		"timestamp": time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", logging.F("error", err))
	}
}
