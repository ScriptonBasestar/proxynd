# ADR-006: Implement Structured Logging

## Status

Accepted

## Context

ProxyND needs comprehensive logging for:
- Debugging issues in production
- Monitoring and alerting
- Audit trails for package access
- Performance analysis
- Security events

Current logging issues:
- Mix of fmt.Printf and log.Printf statements
- No consistent format
- Difficult to parse logs programmatically
- No correlation between related log entries
- Missing contextual information
- No log levels or filtering

Requirements:
- Machine-readable log format
- Contextual information (request ID, user, etc.)
- Performance (minimal overhead)
- Different log levels
- Easy integration with log aggregation systems

## Decision

Implement structured logging using zerolog as the logging framework:

1. **Structured Format**:
   - JSON format for machine readability
   - Consistent field names across all logs
   - Contextual fields for correlation

2. **Log Levels**:
   - Debug: Detailed debugging information
   - Info: General informational messages
   - Warn: Warning messages
   - Error: Error messages that don't stop the program
   - Fatal: Critical errors that stop the program

3. **Context Propagation**:
   - Request ID for tracing requests
   - User information for audit
   - Component/package for log source
   - Additional fields as needed

4. **Implementation**:
   ```go
   logger.Info("package downloaded",
       logging.F("package", packageName),
       logging.F("version", version),
       logging.F("size", size),
       logging.F("duration_ms", duration),
   )
   ```

## Consequences

### Positive

- Machine-readable logs for easy parsing and analysis
- Consistent log format across the application
- Better debugging with contextual information
- Easy integration with ELK, Splunk, or other log systems
- High performance with zero allocation
- Ability to filter logs by level or component
- Structured queries in log aggregation systems

### Negative

- Less human-readable in raw format
- Requires discipline to include proper context
- Team needs to learn structured logging patterns
- Migration effort from existing logging

### Neutral

- Need to establish logging conventions
- Requires log aggregation system to realize full benefits
- May need to adjust existing monitoring/alerting

## Alternatives Considered

1. **Standard library log package**
   - Pros: Simple, no dependencies
   - Cons: No structure, limited features
   - Rejected for lack of features

2. **logrus**
   - Pros: Popular, feature-rich
   - Cons: Performance overhead, allocations
   - Rejected for performance reasons

3. **zap**
   - Pros: High performance, structured
   - Cons: More complex API
   - Rejected in favor of zerolog's simpler API

4. **slog (Go 1.21+)**
   - Pros: Standard library, structured
   - Cons: Requires newer Go version
   - Consider migrating when we update Go version

## Logging Guidelines

1. **What to Log**:
   - Request/response summary (INFO)
   - Errors with context (ERROR)
   - Configuration changes (INFO)
   - Authentication events (INFO/WARN)
   - Performance metrics (DEBUG)

2. **What NOT to Log**:
   - Sensitive data (passwords, tokens)
   - Large payloads
   - High-frequency events at INFO level

3. **Context Fields**:
   - Always include: component, request_id (if applicable)
   - Include when relevant: user_id, package, error, duration

## References

- [Structured Logging](https://www.honeycomb.io/blog/structured-logging-101)
- [zerolog](https://github.com/rs/zerolog)
- [Go Logging Best Practices](https://www.datadoghq.com/blog/go-logging/)
- Current implementation in `logging/logger.go`
