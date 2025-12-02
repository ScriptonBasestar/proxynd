# Batch Job System - CLI Usage Guide

## Overview

The batch job system provides CLI commands to automate complex management tasks through script execution.

## Commands

### `proxyndctl batch run <file>`

Execute a batch script file.

**Usage:**
```bash
proxyndctl batch run maintenance.batch
proxyndctl batch run /path/to/script.batch
```

**Options:**
- `--async` - Run in background and return immediately
- `--output <file>` - Save output to file
- `--timeout <duration>` - Set execution timeout (default: no limit)

**Examples:**
```bash
# Run maintenance script
proxyndctl batch run examples/batch/maintenance.batch

# Run in background
proxyndctl batch run --async cleanup.batch

# Run with timeout
proxyndctl batch run --timeout 30m long-running.batch
```

**Output:**
```
[Step 1/5] echo Starting maintenance...
Starting maintenance...
[Step 2/5] cache clear --older-than 30d --force
Cache cleared: 152 entries removed
[Step 3/5] if last_exit == 0 then
[Step 4/5] echo ✅ Cache cleared successfully
✅ Cache cleared successfully
[Step 5/5] echo Done
Done

Job completed successfully
Exit code: 0
```

---

### `proxyndctl batch validate <file>`

Validate a batch script for syntax errors.

**Usage:**
```bash
proxyndctl batch validate script.batch
```

**Examples:**
```bash
# Validate script
proxyndctl batch validate maintenance.batch

# Validate multiple scripts
for file in *.batch; do
    proxyndctl batch validate "$file"
done
```

**Output (Success):**
```
✅ Script is valid
- 5 commands parsed
- 1 conditional block
- No syntax errors
```

**Output (Error):**
```
❌ Syntax error at line 10:
if last_exit == 0 then
    echo success
# Missing 'fi' for if statement

Error: missing 'fi' for if statement
```

---

### `proxyndctl batch history`

Show batch job execution history.

**Usage:**
```bash
proxyndctl batch history [options]
```

**Options:**
- `--limit <n>` - Show last N jobs (default: 10)
- `--since <duration>` - Show jobs since duration ago (e.g., 24h, 7d)
- `--status <status>` - Filter by status (success, failed, cancelled)
- `--format <format>` - Output format (table, json, text)

**Examples:**
```bash
# Show last 10 jobs
proxyndctl batch history

# Show last 20 jobs
proxyndctl batch history --limit 20

# Show jobs from last 24 hours
proxyndctl batch history --since 24h

# Show only failed jobs
proxyndctl batch history --status failed

# Get JSON output
proxyndctl batch history --format json
```

**Output (Table Format):**
```
JOB ID              STATUS   START TIME          DURATION  EXIT CODE
─────────────────────────────────────────────────────────────────────
job-1638360000123   success  2024-12-02 10:30    2m 15s    0
job-1638359940456   failed   2024-12-02 10:25    1m 45s    1
job-1638359880789   success  2024-12-02 10:20    3m 02s    0
```

**Output (JSON Format):**
```json
[
  {
    "id": "job-1638360000123",
    "status": "success",
    "start_time": "2024-12-02T10:30:00Z",
    "end_time": "2024-12-02T10:32:15Z",
    "exit_code": 0,
    "output": ["Starting...", "Done"]
  }
]
```

---

### `proxyndctl batch cancel <job-id>`

Cancel a running batch job.

**Usage:**
```bash
proxyndctl batch cancel <job-id>
```

**Examples:**
```bash
# Cancel a running job
proxyndctl batch cancel job-1638360000123

# Cancel with confirmation
proxyndctl batch cancel --yes job-1638360000123
```

**Output (Success):**
```
✅ Job cancelled: job-1638360000123
Status changed: running → cancelled
```

**Output (Error):**
```
❌ Cannot cancel job: job-1638360000123
Reason: job is not running (status: success)
```

---

### `proxyndctl batch show <job-id>`

Show detailed information about a job execution.

**Usage:**
```bash
proxyndctl batch show <job-id>
```

**Examples:**
```bash
# Show job details
proxyndctl batch show job-1638360000123

# Show with full output
proxyndctl batch show --full-output job-1638360000123
```

**Output:**
```
Job ID:      job-1638360000123
Status:      success
Start Time:  2024-12-02 10:30:00
End Time:    2024-12-02 10:32:15
Duration:    2m 15s
Exit Code:   0

Script:
─────────────────────────────────────
echo Starting
cache clear --older-than 30d
echo Done
─────────────────────────────────────

Output:
─────────────────────────────────────
[Step 1/3] echo Starting
Starting
[Step 2/3] cache clear --older-than 30d
Cache cleared: 152 entries
[Step 3/3] echo Done
Done
─────────────────────────────────────
```

---

### `proxyndctl batch clean`

Clean up old job history.

**Usage:**
```bash
proxyndctl batch clean [options]
```

**Options:**
- `--older-than <duration>` - Remove jobs older than duration (default: 30d)
- `--status <status>` - Only remove jobs with specific status
- `--dry-run` - Show what would be removed without actually removing

**Examples:**
```bash
# Clean jobs older than 30 days
proxyndctl batch clean

# Clean jobs older than 7 days
proxyndctl batch clean --older-than 7d

# Clean only failed jobs
proxyndctl batch clean --status failed

# Preview what would be cleaned
proxyndctl batch clean --dry-run
```

**Output:**
```
Cleaning job history...
- Removed 15 jobs older than 30 days
- Total jobs remaining: 45
```

---

## Script Syntax

### Commands

**Echo:**
```bash
echo "message"
echo message without quotes
```

**CLI Commands:**
```bash
cache clear --type maven
maven-backup create --target /backup
status
```

**Exit:**
```bash
exit          # Exit with code 0
exit 1        # Exit with code 1
```

**Conditionals:**
```bash
if last_exit == 0 then
    echo success
else
    echo failure
fi
```

### Variables

- `last_exit` - Exit code of last command (0 = success)
- `last_exit_code` - Alias for last_exit

### Operators

- `==` - Equality
- `!=` - Inequality
- `>` - Greater than (numeric)
- `<` - Less than (numeric)
- `>=` - Greater than or equal (numeric)
- `<=` - Less than or equal (numeric)

### Comments

```bash
# This is a comment
echo "This runs"  # Inline comments not supported
```

---

## Complete Example

**File: `daily-maintenance.batch`**
```bash
# Daily Maintenance Script
echo "Starting daily maintenance at $(date)"

# Clear old cache
cache clear --older-than 30d --force

if last_exit == 0 then
    echo "✅ Cache cleanup successful"
else
    echo "❌ Cache cleanup failed"
    exit 1
fi

# Create backup
maven-backup create --target /backup/daily

if last_exit == 0 then
    echo "✅ Backup created"
else
    echo "⚠️ Backup failed, continuing..."
fi

# Rebuild indexes
maven-index build --force

if last_exit != 0 then
    echo "❌ Index rebuild failed"
    exit 1
fi

echo "✅ Daily maintenance completed successfully"
```

**Execute:**
```bash
proxyndctl batch run daily-maintenance.batch
```

---

## Error Handling

### Exit Codes

- `0` - Success
- `1` - General failure
- `2` - Syntax error
- `3` - Execution error
- `4` - Timeout
- `5` - Cancelled

### Common Errors

**Syntax Error:**
```
Error: missing 'fi' for if statement
Line: 10
```

**Command Not Allowed:**
```
Error: command not allowed: rm
Allowed commands: cache, maven-backup, maven-index, test, status, health, version
```

**Timeout:**
```
Error: job timeout after 30m
Job ID: job-1638360000123
Status: cancelled
```

---

## Best Practices

1. **Always validate** scripts before running in production
2. **Use conditionals** to handle errors gracefully
3. **Add comments** to document complex logic
4. **Keep scripts focused** on single tasks
5. **Test with --dry-run** when available
6. **Monitor job history** regularly
7. **Clean old history** to save space

---

## Integration with CI/CD

### GitHub Actions

```yaml
- name: Run maintenance
  run: |
    ./proxyndctl batch run scripts/maintenance.batch
```

### Jenkins

```groovy
stage('Maintenance') {
    steps {
        sh './proxyndctl batch run maintenance.batch'
    }
}
```

### Cron

```bash
# Daily at 2 AM
0 2 * * * /usr/local/bin/proxyndctl batch run /etc/proxynd/daily.batch >> /var/log/proxynd-batch.log 2>&1
```

---

## Troubleshooting

### Job Stuck in Running State

```bash
# List running jobs
proxyndctl batch history --status running

# Cancel if needed
proxyndctl batch cancel <job-id>
```

### Script Validation Fails

```bash
# Validate with verbose output
proxyndctl batch validate --verbose script.batch

# Check syntax at specific line
head -n 10 script.batch | tail -n 1
```

### History Growing Too Large

```bash
# Check history size
du -sh ~/.proxynd/batch/history

# Clean old entries
proxyndctl batch clean --older-than 7d
```

---

**For more information**, see:
- Design document: `internal/batch/design.md`
- Example scripts: `examples/batch/`
- API documentation: `internal/batch/README.md`
