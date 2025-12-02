# Batch Job System Documentation

> Part of ProxyND Core - Automated maintenance task execution

## Overview

The batch job system provides automated execution of complex management tasks through script-based workflows. It supports conditional logic, command execution, and comprehensive job history tracking.

## Quick Links

- **[Usage Guide](usage.md)** - Complete CLI command reference with examples
- **[Design Document](../../../../internal/batch/design.md)** - Architecture and implementation details
- **[Example Scripts](../../../../examples/batch/)** - Pre-built maintenance scripts

## Key Features

- **Script-based Execution**: Execute commands with conditional logic
- **Job Management**: Start, monitor, and cancel long-running jobs
- **History Tracking**: Persistent storage of job execution history
- **Multiple Output Formats**: Table, JSON, and text output
- **Filtering**: Filter history by time range and status
- **Validation**: Pre-execution syntax validation

## Quick Start

### Execute a Batch Script

```bash
proxyndctl batch run maintenance.batch
```

### Validate Script Syntax

```bash
proxyndctl batch validate script.batch
```

### View Job History

```bash
proxyndctl batch history --limit 10
```

## Commands

| Command | Purpose | Documentation |
|---------|---------|--------------|
| `batch run` | Execute batch script | [usage.md#batch-run](usage.md#proxyndctl-batch-run-file) |
| `batch validate` | Validate syntax | [usage.md#batch-validate](usage.md#proxyndctl-batch-validate-file) |
| `batch history` | Show execution history | [usage.md#batch-history](usage.md#proxyndctl-batch-history) |
| `batch cancel` | Cancel running job | [usage.md#batch-cancel](usage.md#proxyndctl-batch-cancel-job-id) |
| `batch show` | Show job details | [usage.md#batch-show](usage.md#proxyndctl-batch-show-job-id) |
| `batch clean` | Clean old history | [usage.md#batch-clean](usage.md#proxyndctl-batch-clean) |

## Script Syntax

### Basic Commands

```bash
# Echo messages
echo "Starting maintenance..."

# Execute CLI commands
cache clear --older-than 30d --force
maven-backup create --target /backup

# Exit with code
exit 0
```

### Conditional Logic

```bash
if last_exit == 0 then
    echo "✅ Success"
else
    echo "❌ Failed"
    exit 1
fi
```

### Variables

- `last_exit` - Exit code of last command (0 = success)
- `last_exit_code` - Alias for `last_exit`

### Operators

- `==` - Equality
- `!=` - Inequality
- `>`, `<`, `>=`, `<=` - Numeric comparisons

## Example Scripts

### Daily Maintenance

```bash
# Daily Maintenance Script
echo "Starting daily maintenance..."

# Clear old cache
cache clear --older-than 30d --force

if last_exit == 0 then
    echo "✅ Cache cleared successfully"
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

echo "✅ Maintenance completed"
```

### Conditional Cleanup

```bash
# Clean cache only if space is low
echo "Checking disk space..."

# This would be a real command in production
# disk-check --threshold 80

if last_exit != 0 then
    echo "Disk space OK, skipping cleanup"
    exit 0
fi

echo "Low disk space detected, cleaning cache..."
cache clear --older-than 7d --force

if last_exit == 0 then
    echo "✅ Cache cleanup successful"
else
    echo "❌ Cache cleanup failed"
    exit 1
fi
```

## Configuration

### History Storage Location

Set the history storage directory via environment variable:

```bash
export PROXYND_BATCH_HISTORY_DIR=/var/lib/proxynd/batch/history
```

**Default locations** (tried in order):
1. `$PROXYND_BATCH_HISTORY_DIR` (if set)
2. `$HOME/.proxynd/batch/history`
3. `/tmp/proxynd/batch/history` (fallback)

## Integration Examples

### Cron

```bash
# Daily at 2 AM
0 2 * * * /usr/local/bin/proxyndctl batch run /etc/proxynd/daily.batch >> /var/log/proxynd-batch.log 2>&1
```

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

## Best Practices

1. **Always validate** scripts before running in production
2. **Use conditionals** to handle errors gracefully
3. **Add comments** to document complex logic
4. **Keep scripts focused** on single tasks
5. **Test with --dry-run** when available
6. **Monitor job history** regularly
7. **Clean old history** to save space

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
proxyndctl batch validate script.batch
```

### History Growing Too Large

```bash
# Check history size
du -sh ~/.proxynd/batch/history

# Clean old entries
proxyndctl batch clean --older-than 7d
```

## Exit Codes

- `0` - Success
- `1` - General failure
- `2` - Syntax error
- `3` - Execution error
- `4` - Timeout (future)
- `5` - Cancelled

## Related Documentation

- **Design**: [internal/batch/design.md](../../../../internal/batch/design.md)
- **Examples**: [examples/batch/](../../../../examples/batch/)
- **Implementation**: [internal/batch/](../../../../internal/batch/)
- **CLI Reference**: [usage.md](usage.md)
- **API Integration**: Coming in Phase 3

---

**Version**: Phase 2 (CLI Integration Complete)
**Status**: ✅ Production Ready
**Last Updated**: 2025-12-02
