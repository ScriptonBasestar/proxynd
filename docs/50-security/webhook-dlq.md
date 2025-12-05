# Webhook Dead Letter Queue (DLQ)

**Version**: 1.0
**Last Updated**: 2025-12-05
**Status**: Production Ready

---

## Overview

The Webhook Dead Letter Queue (DLQ) is a fault-tolerant storage system for webhook events that fail after exhausting all retry attempts. It provides visibility into failed webhook deliveries and allows manual intervention when automatic retries are insufficient.

### Purpose

1. **Prevent Data Loss**: Store failed webhook events for later investigation
2. **Manual Recovery**: Enable manual retry of failed webhooks
3. **Debugging**: Provide detailed failure information (attempts, errors, timestamps)
4. **Monitoring**: Track webhook reliability and failure patterns

---

## Architecture

### Components

```
┌─────────────┐
│   Event     │
│  Generated  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Webhook   │
│   Worker    │
└──────┬──────┘
       │
       ▼ (Retry 1-3 times)
┌─────────────┐
│   Retry     │
│   Logic     │
└──────┬──────┘
       │
       ▼ (Max retries exceeded)
┌─────────────┐
│     DLQ     │
│   Storage   │
└─────────────┘
```

### Storage

**Type**: File-based persistence
**Location**: `{StorageDir}/dlq/`
**Format**: JSON files (one per failed event)
**Filename**: `dlq-{timestamp}-{endpoint}.json`

---

## DLQ Item Structure

```go
type DeadLetterItem struct {
    ID          string             // Unique identifier
    Event       *alerts.AlertEvent // Original event
    Endpoint    string             // Target endpoint name
    Attempts    int                // Number of delivery attempts
    LastAttempt time.Time          // Last delivery attempt timestamp
    FirstFailed time.Time          // First failure timestamp
    Error       string             // Last error message
    Metadata    map[string]string  // Additional context
}
```

### Example DLQ Item

```json
{
  "id": "dlq-1733363924000000000-slack-alerts",
  "event": {
    "id": "evt-20251205-001",
    "type": "cache.full",
    "level": "ERROR",
    "message": "Cache storage is full",
    "source": "cache-manager",
    "timestamp": "2025-12-05T08:30:00Z",
    "metadata": {
      "cache_size": "10GB",
      "usage": "98%"
    }
  },
  "endpoint": "slack-alerts",
  "attempts": 3,
  "last_attempt": "2025-12-05T08:35:00Z",
  "first_failed": "2025-12-05T08:30:00Z",
  "error": "connection timeout: dial tcp 192.168.1.100:443: i/o timeout",
  "metadata": {
    "reason": "max_retries_exceeded",
    "max_retries": "3"
  }
}
```

---

## Configuration

### Webhook Configuration

```yaml
webhook:
  enabled: true
  workers: 10
  retry:
    max_attempts: 3
    initial_delay: 5s
    max_delay: 5m
    backoff_factor: 2.0
  failure_storage:
    storage_dir: /var/lib/proxynd/webhook
    retention_hours: 168  # 7 days
```

### DLQ Limits

| Setting | Default | Description |
|---------|---------|-------------|
| **Max Size** | 10,000 items | Maximum DLQ capacity |
| **Retention** | 7 days | Auto-purge older items |
| **Storage Dir** | `{StorageDir}/dlq` | DLQ file location |

---

## HTTP API Endpoints

### 1. List All DLQ Items

**Endpoint**: `GET /api/webhooks/dlq`

**Response**:
```json
{
  "items": [
    {
      "id": "dlq-1733363924000000000-slack-alerts",
      "event": { ... },
      "endpoint": "slack-alerts",
      "attempts": 3,
      "error": "connection timeout"
    }
  ],
  "count": 15,
  "timestamp": "2025-12-05T09:00:00Z"
}
```

**Example**:
```bash
curl http://localhost:8080/api/webhooks/dlq
```

---

### 2. Get Specific DLQ Item

**Endpoint**: `GET /api/webhooks/dlq/{id}`

**Response**:
```json
{
  "id": "dlq-1733363924000000000-slack-alerts",
  "event": { ... },
  "endpoint": "slack-alerts",
  "attempts": 3,
  "last_attempt": "2025-12-05T08:35:00Z",
  "first_failed": "2025-12-05T08:30:00Z",
  "error": "connection timeout",
  "metadata": {
    "reason": "max_retries_exceeded"
  }
}
```

**Example**:
```bash
curl http://localhost:8080/api/webhooks/dlq/dlq-1733363924000000000-slack-alerts
```

---

### 3. Retry Failed Webhook

**Endpoint**: `POST /api/webhooks/dlq/{id}/retry`

**Description**: Removes item from DLQ and re-queues for delivery

**Response**:
```json
{
  "status": "retried",
  "id": "dlq-1733363924000000000-slack-alerts",
  "event_id": "evt-20251205-001",
  "endpoint": "slack-alerts",
  "timestamp": "2025-12-05T09:00:00Z"
}
```

**Example**:
```bash
curl -X POST http://localhost:8080/api/webhooks/dlq/dlq-1733363924000000000-slack-alerts/retry
```

**Note**: If retry fails again, the event will be added back to the DLQ after max retries.

---

### 4. Delete DLQ Item

**Endpoint**: `DELETE /api/webhooks/dlq/{id}`

**Description**: Permanently removes item from DLQ

**Response**:
```json
{
  "status": "deleted",
  "id": "dlq-1733363924000000000-slack-alerts",
  "timestamp": "2025-12-05T09:00:00Z"
}
```

**Example**:
```bash
curl -X DELETE http://localhost:8080/api/webhooks/dlq/dlq-1733363924000000000-slack-alerts
```

**Warning**: Deleted items cannot be recovered.

---

### 5. Get DLQ Statistics

**Endpoint**: `GET /api/webhooks/dlq/stats`

**Response**:
```json
{
  "statistics": {
    "total_items": 15,
    "max_size": 10000,
    "usage_pct": 0.15,
    "oldest_item": "2025-12-03T10:00:00Z",
    "newest_item": "2025-12-05T08:35:00Z",
    "by_endpoint": {
      "slack-alerts": 8,
      "discord-notifications": 5,
      "email-alerts": 2
    }
  },
  "timestamp": "2025-12-05T09:00:00Z"
}
```

**Example**:
```bash
curl http://localhost:8080/api/webhooks/dlq/stats
```

---

### 6. Purge Old DLQ Items

**Endpoint**: `POST /api/webhooks/dlq/purge`

**Request Body**:
```json
{
  "before_hours": 168  // Purge items older than 7 days (default)
}
```

**Response**:
```json
{
  "status": "purged",
  "removed": 5,
  "before_hours": 168,
  "before_time": "2025-11-28T09:00:00Z",
  "timestamp": "2025-12-05T09:00:00Z"
}
```

**Example**:
```bash
curl -X POST http://localhost:8080/api/webhooks/dlq/purge \
  -H "Content-Type: application/json" \
  -d '{"before_hours": 72}'
```

---

### 7. Clear All DLQ Items

**Endpoint**: `DELETE /api/webhooks/dlq`

**Description**: Removes all items from DLQ

**Response**:
```json
{
  "status": "cleared",
  "timestamp": "2025-12-05T09:00:00Z"
}
```

**Example**:
```bash
curl -X DELETE http://localhost:8080/api/webhooks/dlq
```

**Warning**: This action cannot be undone.

---

## Operational Guide

### Monitoring DLQ Health

**Check DLQ size regularly**:
```bash
# Get current statistics
curl http://localhost:8080/api/webhooks/dlq/stats | jq '.statistics.total_items'
```

**Alert thresholds**:
- **Warning**: DLQ > 100 items
- **Critical**: DLQ > 1000 items or >10% capacity

### Common Scenarios

#### Scenario 1: Temporary Network Issue

**Symptom**: Multiple DLQ items with "connection timeout" error

**Solution**:
```bash
# 1. Fix network issue
# 2. Retry all failed webhooks
for id in $(curl -s http://localhost:8080/api/webhooks/dlq | jq -r '.items[].id'); do
  curl -X POST "http://localhost:8080/api/webhooks/dlq/${id}/retry"
  sleep 1
done
```

#### Scenario 2: Endpoint Configuration Error

**Symptom**: All items for specific endpoint failing

**Solution**:
1. Fix endpoint configuration in `config.yaml`
2. Reload configuration (hot reload)
3. Retry failed items for that endpoint

```bash
# Filter and retry specific endpoint
curl -s http://localhost:8080/api/webhooks/dlq | \
  jq -r '.items[] | select(.endpoint=="slack-alerts") | .id' | \
  while read id; do
    curl -X POST "http://localhost:8080/api/webhooks/dlq/${id}/retry"
  done
```

#### Scenario 3: Endpoint Permanently Down

**Symptom**: Endpoint no longer exists

**Solution**:
```bash
# Delete all items for removed endpoint
curl -s http://localhost:8080/api/webhooks/dlq | \
  jq -r '.items[] | select(.endpoint=="old-endpoint") | .id' | \
  while read id; do
    curl -X DELETE "http://localhost:8080/api/webhooks/dlq/${id}"
  done
```

#### Scenario 4: DLQ Capacity Approaching Limit

**Symptom**: `usage_pct` approaching 100%

**Solution**:
```bash
# Purge items older than 3 days
curl -X POST http://localhost:8080/api/webhooks/dlq/purge \
  -H "Content-Type: application/json" \
  -d '{"before_hours": 72}'
```

---

## Troubleshooting

### Problem: DLQ Items Not Being Created

**Possible Causes**:
1. Max retries set to 0
2. DLQ directory not writable
3. Disk full

**Check**:
```bash
# Verify webhook configuration
curl http://localhost:8080/api/config | jq '.webhook.retry.max_attempts'

# Check disk space
df -h /var/lib/proxynd/webhook/dlq

# Check directory permissions
ls -ld /var/lib/proxynd/webhook/dlq
```

### Problem: DLQ Growing Too Large

**Possible Causes**:
1. Endpoint permanently down
2. Configuration error
3. Network connectivity issues

**Investigation**:
```bash
# Check endpoint distribution
curl http://localhost:8080/api/webhooks/dlq/stats | jq '.statistics.by_endpoint'

# Review recent errors
curl http://localhost:8080/api/webhooks/dlq | jq '.items[] | {endpoint, error}' | head -20
```

### Problem: Retry Not Working

**Possible Causes**:
1. Item ID incorrect
2. Event already removed
3. Webhook worker not running

**Check**:
```bash
# Verify item exists
curl http://localhost:8080/api/webhooks/dlq/{id}

# Check webhook worker status
curl http://localhost:8080/health | jq '.components.webhook'
```

---

## Best Practices

### 1. Regular Monitoring

Set up automated monitoring:
```bash
#!/bin/bash
# Monitor DLQ size
dlq_size=$(curl -s http://localhost:8080/api/webhooks/dlq/stats | jq '.statistics.total_items')

if [ "$dlq_size" -gt 100 ]; then
  echo "WARNING: DLQ has $dlq_size items"
  # Send alert
fi
```

### 2. Automated Purging

Schedule regular purges:
```bash
# Cron job: Purge items older than 7 days (daily at 2 AM)
0 2 * * * curl -X POST http://localhost:8080/api/webhooks/dlq/purge -H "Content-Type: application/json" -d '{"before_hours": 168}'
```

### 3. Endpoint Health Checks

Before retrying, verify endpoint is healthy:
```bash
# Test endpoint connectivity
curl -I https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# If healthy, retry
curl -X POST http://localhost:8080/api/webhooks/dlq/{id}/retry
```

### 4. Backup Critical Events

For critical events, export before deleting:
```bash
# Export DLQ items to file
curl http://localhost:8080/api/webhooks/dlq > dlq-backup-$(date +%Y%m%d).json

# Archive
gzip dlq-backup-$(date +%Y%m%d).json
```

---

## Integration Examples

### Prometheus Metrics

```yaml
# Alert on high DLQ size
groups:
  - name: webhook_dlq
    rules:
      - alert: WebhookDLQHigh
        expr: proxynd_webhook_dlq_size > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Webhook DLQ size is high"
          description: "DLQ has {{ $value }} items"
```

### Grafana Dashboard

Query DLQ statistics:
```bash
# Prometheus query
proxynd_webhook_dlq_size
proxynd_webhook_dlq_usage_percent
proxynd_webhook_dlq_items_by_endpoint
```

### Automated Recovery Script

```python
import requests
import time

def retry_all_dlq_items(base_url):
    """Retry all DLQ items with exponential backoff"""
    resp = requests.get(f"{base_url}/api/webhooks/dlq")
    items = resp.json()["items"]

    for item in items:
        item_id = item["id"]
        print(f"Retrying {item_id}...")

        retry_resp = requests.post(
            f"{base_url}/api/webhooks/dlq/{item_id}/retry"
        )

        if retry_resp.status_code == 200:
            print(f"✓ Successfully retried {item_id}")
        else:
            print(f"✗ Failed to retry {item_id}: {retry_resp.text}")

        time.sleep(1)  # Rate limiting

if __name__ == "__main__":
    retry_all_dlq_items("http://localhost:8080")
```

---

## Security Considerations

### Access Control

DLQ endpoints should be protected:
```yaml
security:
  authentication:
    basic_auth:
      enabled: true
      users:
        admin: $2a$10$... # bcrypt hash
```

### Sensitive Data

DLQ items may contain sensitive information:
- Review before sharing
- Sanitize logs
- Consider encryption at rest

### Audit Logging

Enable audit logs for DLQ operations:
```yaml
logging:
  audit:
    enabled: true
    events:
      - webhook.dlq.retry
      - webhook.dlq.delete
      - webhook.dlq.purge
```

---

## Performance Considerations

### Disk I/O

- Each DLQ item is a separate file
- Large DLQ (>1000 items) may impact list performance
- Consider SSD storage for high-volume systems

### Memory Usage

- DLQ list operations load all items into memory
- With max_size=10,000, expect ~50-100MB memory usage

### Optimization Tips

1. **Purge regularly**: Keep DLQ size under 1000 items
2. **Monitor endpoints**: Fix failing endpoints promptly
3. **Use pagination**: For large DLQ, implement client-side pagination

---

## Related Documentation

- [Webhook Events Guide](webhook-events.md)
- [Webhook Configuration](../20-configuration/webhook-config.md)
- [API Reference](../04-api-reference/webhook-api.md)
- [Troubleshooting Guide](../70-operations/troubleshooting.md)

---

**Changelog**:
- 2025-12-05: Initial documentation for DLQ feature
