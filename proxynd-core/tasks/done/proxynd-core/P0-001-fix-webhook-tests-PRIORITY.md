# Fix Webhook Sender Tests - PRIORITY 0

> **Status**: Active - Ready for Implementation  
> **Priority**: P0 (CRITICAL - Blocks PRs)  
> **Repository**: proxynd-core  
> **Investigation Completed**: 2025-11-30 23:15 KST  
> **Estimated Fix Time**: 15-30 minutes

---

## 🚨 Why Priority 0

- **Blocks**: All pull request merges
- **Impact**: CI/CD pipeline failing
- **Complexity**: Low - map initialization issue
- **Quick Win**: Can be fixed in single session
- **Ready**: Detailed analysis and fix guide provided below

---

## 📋 Description

Webhook sender tests are failing with assertion errors.

## Failing Tests

1. `TestWebhookSenderGetMetricsWithBatch` (line 343)
2. `TestWebhookSenderBatchingDisabled` (line 357)

## Error Message

```go
sender_test.go:343 map[] should not be equal <nil>
sender_test.go:357 map[] should not be equal <nil>
```

---

## 🔍 Root Cause

The tests expect `metrics.BatchStats` to be a **non-nil map**, but the
`GetMetrics()` method is returning an **empty map** instead.

The `GetMetrics()` method is **not initializing** the `BatchStats` field.

---

## 🛠️ Recommended Fix

Add initialization in `sender.go`:

```go
func (s *WebhookSender) GetMetrics() *Metrics {
    metrics := &Metrics{
        BatchStats: make(map[string]interface{}),  // ← Add this
    }

    // Populate batch statistics
    metrics.BatchStats["enabled"] = s.config.Batching.Enabled
    if s.batchManager != nil {
        metrics.BatchStats["batch_size"] = s.batchManager.GetBatchSize()
        metrics.BatchStats["pending_batches"] = s.batchManager.GetPendingCount()
    }

    return metrics
}
```

---

## 📋 Step-by-Step Fix

### Step 1: Locate Method

```bash
cd proxynd-core
grep -n "func.*GetMetrics" internal/webhook/*.go
```

### Step 2: Add Initialization

Ensure `BatchStats` map is initialized before use.

### Step 3: Run Tests

```bash
go test -v ./internal/webhook/... -run "Batch"
```

### Step 4: Verify

Expected:

```text
=== RUN   TestWebhookSenderGetMetricsWithBatch
--- PASS: TestWebhookSenderGetMetricsWithBatch (0.00s)
=== RUN   TestWebhookSenderBatchingDisabled  
--- PASS: TestWebhookSenderBatchingDisabled (0.00s)
```

---

## ✅ Acceptance Criteria

- [ ] `GetMetrics()` initializes `BatchStats` map
- [ ] `BatchStats["enabled"]` reflects batching configuration
- [ ] `TestWebhookSenderGetMetricsWithBatch` passes
- [ ] `TestWebhookSenderBatchingDisabled` passes
- [ ] No regression in other webhook tests
- [ ] `make test` passes for webhook module

---

**Investigation Completed**: 2025-11-30 23:15 KST  
**Ready for Implementation**: Yes  
**Last Updated**: 2025-11-30 23:30 KST
