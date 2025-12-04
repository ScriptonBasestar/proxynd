# P2: Webhook System Enhancement

**Priority**: P2 (Medium)
**Status**: Pending
**Created**: 2025-12-04
**Estimated Time**: 3-4 hours

---

## Overview

Complete webhook reliability features:
- Dead Letter Queue (DLQ) for failed webhooks
- WebhookHistoryManager integration fix
- Enhanced error handling and retry logic

---

## Current State

### ✅ Complete
- Basic webhook worker implemented
- Webhook sender functional
- Test handler exists
- Event delivery working

### ⏳ Pending
- Dead Letter Queue implementation
- WebhookHistoryManager proper integration
- HTTP HEAD/GET test endpoints
- Test history storage/retrieval

---

## TODOs to Address

### 1. Dead Letter Queue
**File**: `internal/webhook/worker.go`
**Line**: 157

**Current Code**:
```go
// TODO: Dead Letter Queue 구현
```

**Issue**: Failed webhooks are not stored for manual retry or debugging

**Implementation Plan**:
```go
type DeadLetterQueue struct {
    mu      sync.RWMutex
    items   []DeadLetterItem
    maxSize int
    storage Storage // Persistent storage
}

type DeadLetterItem struct {
    Event       *Event
    Attempts    int
    LastAttempt time.Time
    Error       string
    Endpoint    string
}

func (dlq *DeadLetterQueue) Add(item DeadLetterItem) error
func (dlq *DeadLetterQueue) Get(limit int) []DeadLetterItem
func (dlq *DeadLetterQueue) Remove(id string) error
func (dlq *DeadLetterQueue) Retry(id string) error
func (dlq *DeadLetterQueue) Purge(before time.Time) error
```

### 2. WebhookHistoryManager Integration
**File**: `internal/webhook/sender.go`
**Line**: 124

**Current Code**:
```go
// TODO: sender.Core에서 실제 WebhookHistoryManager를 반환하도록 수정 필요
```

**Issue**: History manager not properly connected to core system

**Fix**:
- Update sender.Core to return actual WebhookHistoryManager
- Ensure history is persisted correctly
- Add history query methods

### 3. Test Handler Endpoints
**File**: `internal/webhook/test_handler.go`

**TODOs**:
- HTTP HEAD request implementation
- HTTP GET request implementation
- Test history storage
- Test history retrieval

---

## Implementation Steps

### Step 1: Design DLQ System (30 min)

**Storage Options**:
1. **In-Memory** (quick start):
   - Simple slice with max size
   - Lost on restart
   - Good for development

2. **File-based** (recommended):
   - JSON file per failed event
   - Survives restarts
   - Easy to inspect

3. **Database** (production):
   - PostgreSQL/MongoDB
   - Query capabilities
   - Best for high volume

**Initial Implementation**: File-based

### Step 2: Implement DLQ (1.5 hours)

**Create** `internal/webhook/dlq.go`:

```go
package webhook

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type DeadLetterQueue struct {
    mu      sync.RWMutex
    dir     string
    maxSize int
}

type DeadLetterItem struct {
    ID          string    `json:"id"`
    Event       *Event    `json:"event"`
    Attempts    int       `json:"attempts"`
    LastAttempt time.Time `json:"last_attempt"`
    Error       string    `json:"error"`
    Endpoint    string    `json:"endpoint"`
}

func NewDeadLetterQueue(dir string, maxSize int) (*DeadLetterQueue, error) {
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create DLQ directory: %w", err)
    }

    return &DeadLetterQueue{
        dir:     dir,
        maxSize: maxSize,
    }, nil
}

func (dlq *DeadLetterQueue) Add(item DeadLetterItem) error {
    dlq.mu.Lock()
    defer dlq.mu.Unlock()

    // Generate unique ID if not set
    if item.ID == "" {
        item.ID = fmt.Sprintf("dlq-%d", time.Now().UnixNano())
    }

    // Write to file
    filePath := filepath.Join(dlq.dir, item.ID+".json")
    data, err := json.MarshalIndent(item, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal DLQ item: %w", err)
    }

    if err := os.WriteFile(filePath, data, 0644); err != nil {
        return fmt.Errorf("failed to write DLQ item: %w", err)
    }

    return nil
}

func (dlq *DeadLetterQueue) List() ([]DeadLetterItem, error) {
    dlq.mu.RLock()
    defer dlq.mu.RUnlock()

    entries, err := os.ReadDir(dlq.dir)
    if err != nil {
        return nil, fmt.Errorf("failed to read DLQ directory: %w", err)
    }

    var items []DeadLetterItem
    for _, entry := range entries {
        if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
            continue
        }

        filePath := filepath.Join(dlq.dir, entry.Name())
        data, err := os.ReadFile(filePath)
        if err != nil {
            continue // Skip corrupted files
        }

        var item DeadLetterItem
        if err := json.Unmarshal(data, &item); err != nil {
            continue
        }

        items = append(items, item)
    }

    return items, nil
}

func (dlq *DeadLetterQueue) Remove(id string) error {
    dlq.mu.Lock()
    defer dlq.mu.Unlock()

    filePath := filepath.Join(dlq.dir, id+".json")
    if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to remove DLQ item: %w", err)
    }

    return nil
}

func (dlq *DeadLetterQueue) Retry(id string, worker *Worker) error {
    // Load item
    filePath := filepath.Join(dlq.dir, id+".json")
    data, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("failed to read DLQ item: %w", err)
    }

    var item DeadLetterItem
    if err := json.Unmarshal(data, &item); err != nil {
        return fmt.Errorf("failed to unmarshal DLQ item: %w", err)
    }

    // Re-queue event
    worker.Queue(item.Event)

    // Remove from DLQ
    return dlq.Remove(id)
}
```

### Step 3: Integrate DLQ into Worker (30 min)

**Update** `internal/webhook/worker.go`:

```go
type Worker struct {
    // ... existing fields ...
    dlq *DeadLetterQueue
}

func NewWorker(config Config, sender *Sender, dlqDir string) (*Worker, error) {
    dlq, err := NewDeadLetterQueue(dlqDir, 10000)
    if err != nil {
        return nil, fmt.Errorf("failed to create DLQ: %w", err)
    }

    return &Worker{
        // ... existing initialization ...
        dlq: dlq,
    }, nil
}

func (w *Worker) processEvent(event *Event) {
    // ... existing code ...

    // On final failure, add to DLQ
    if attempts >= maxRetries {
        dlqItem := DeadLetterItem{
            Event:       event,
            Attempts:    attempts,
            LastAttempt: time.Now(),
            Error:       lastError.Error(),
            Endpoint:    event.Endpoint,
        }

        if err := w.dlq.Add(dlqItem); err != nil {
            w.logger.Error("Failed to add to DLQ", "error", err)
        }
    }
}
```

### Step 4: Fix WebhookHistoryManager (1 hour)

**Update** `internal/webhook/sender.go`:

```go
type Sender struct {
    core    *Core
    history *WebhookHistoryManager
}

func NewSender(core *Core) *Sender {
    return &Sender{
        core:    core,
        history: core.GetHistoryManager(), // Proper integration
    }
}

func (s *Sender) Send(event *Event) error {
    // ... send logic ...

    // Record history
    if s.history != nil {
        s.history.Record(HistoryEntry{
            EventID:   event.ID,
            Endpoint:  event.Endpoint,
            Status:    status,
            Timestamp: time.Now(),
            Error:     err,
        })
    }

    return err
}
```

**Update** `internal/webhook/core.go`:

```go
type Core struct {
    // ... existing fields ...
    historyManager *WebhookHistoryManager
}

func (c *Core) GetHistoryManager() *WebhookHistoryManager {
    return c.historyManager
}
```

### Step 5: Implement Test Handler (30 min)

**Update** `internal/webhook/test_handler.go`:

```go
// HandleTestWebhook handles HEAD request for webhook endpoint testing
func (h *TestHandler) HandleTestWebhook(c *fiber.Ctx) error {
    if c.Method() == "HEAD" {
        // Return 200 OK with no body
        return c.SendStatus(fiber.StatusOK)
    }

    if c.Method() == "GET" {
        // Return webhook test information
        return c.JSON(fiber.Map{
            "status": "ok",
            "endpoint": c.OriginalURL(),
            "timestamp": time.Now(),
        })
    }

    return c.SendStatus(fiber.StatusMethodNotAllowed)
}

// GetTestHistory retrieves test webhook history
func (h *TestHandler) GetTestHistory(c *fiber.Ctx) error {
    history, err := h.historyManager.List(100)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.JSON(fiber.Map{
        "history": history,
        "count": len(history),
    })
}
```

### Step 6: Add DLQ API Endpoints (30 min)

**Create** `internal/adapters/http/fiber/handlers/webhook/dlq_handler.go`:

```go
package webhook

type DLQHandler struct {
    dlq    *webhook.DeadLetterQueue
    worker *webhook.Worker
}

func (h *DLQHandler) ListDLQ(c *fiber.Ctx) error {
    items, err := h.dlq.List()
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(fiber.Map{
        "items": items,
        "count": len(items),
    })
}

func (h *DLQHandler) RetryDLQ(c *fiber.Ctx) error {
    id := c.Params("id")

    if err := h.dlq.Retry(id, h.worker); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(fiber.Map{"status": "retried", "id": id})
}

func (h *DLQHandler) DeleteDLQ(c *fiber.Ctx) error {
    id := c.Params("id")

    if err := h.dlq.Remove(id); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(fiber.Map{"status": "deleted", "id": id})
}
```

---

## Testing

### Unit Tests (30 min)

**Create** `internal/webhook/dlq_test.go`:

```go
func TestDLQ_Add(t *testing.T) { /* ... */ }
func TestDLQ_List(t *testing.T) { /* ... */ }
func TestDLQ_Remove(t *testing.T) { /* ... */ }
func TestDLQ_Retry(t *testing.T) { /* ... */ }
```

### Integration Tests (30 min)

```bash
# Test DLQ flow
go test ./internal/webhook/... -v -run TestDLQ

# Test webhook history
go test ./internal/webhook/... -v -run TestHistory

# Test endpoints
curl http://localhost:8080/api/webhooks/dlq
curl -X POST http://localhost:8080/api/webhooks/dlq/{id}/retry
curl -X DELETE http://localhost:8080/api/webhooks/dlq/{id}
```

---

## Acceptance Criteria

- [ ] DLQ implemented with file-based storage
- [ ] Failed webhooks automatically added to DLQ
- [ ] DLQ API endpoints working (list, retry, delete)
- [ ] WebhookHistoryManager properly integrated
- [ ] Test handler supports HEAD and GET
- [ ] Test history storage/retrieval working
- [ ] Unit tests passing (>90% coverage)
- [ ] Integration tests passing
- [ ] Documentation updated
- [ ] No TODO comments remain

---

## Configuration

Add to config:

```yaml
webhook:
  enabled: true
  workers: 10
  retry_max: 3
  retry_interval: 5s
  dlq:
    enabled: true
    directory: /var/lib/proxynd/webhook/dlq
    max_size: 10000
  history:
    enabled: true
    retention_days: 30
```

---

## Documentation to Update

- [ ] `docs/WEBHOOK_SYSTEM.md` - Add DLQ section
- [ ] API documentation - Add DLQ endpoints
- [ ] Configuration reference - Add webhook config
- [ ] Troubleshooting guide - Add DLQ usage

---

## Related Files

- `internal/webhook/worker.go` - Main worker
- `internal/webhook/sender.go` - Sender with history
- `internal/webhook/test_handler.go` - Test endpoints
- `internal/webhook/core.go` - Core system
- `internal/adapters/http/fiber/handlers/webhook/` - HTTP handlers

---

**Dependencies**: None

**Blocks**: Production webhook reliability

**Related Tasks**: None

---

**Last Updated**: 2025-12-04
