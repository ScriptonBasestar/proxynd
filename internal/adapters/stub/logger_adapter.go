package stub

import (
	"context"

	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// LoggerAdapter adapts logging.Logger to ports.Logger interface
type LoggerAdapter struct {
	logger logging.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger logging.Logger) ports.Logger {
	return &LoggerAdapter{logger: logger}
}

// Debug logs debug level message
func (l *LoggerAdapter) Debug(ctx context.Context, msg string, fields ...ports.Field) {
	logFields := convertFields(fields)
	l.logger.Debug(msg, logFields...)
}

// Info logs info level message
func (l *LoggerAdapter) Info(ctx context.Context, msg string, fields ...ports.Field) {
	logFields := convertFields(fields)
	l.logger.Info(msg, logFields...)
}

// Warn logs warning level message
func (l *LoggerAdapter) Warn(ctx context.Context, msg string, fields ...ports.Field) {
	logFields := convertFields(fields)
	l.logger.Warn(msg, logFields...)
}

// Error logs error level message
func (l *LoggerAdapter) Error(ctx context.Context, msg string, fields ...ports.Field) {
	logFields := convertFields(fields)
	l.logger.Error(msg, logFields...)
}

// Fatal logs fatal level message and exits
func (l *LoggerAdapter) Fatal(ctx context.Context, msg string, fields ...ports.Field) {
	logFields := convertFields(fields)
	l.logger.Fatal(msg, logFields...)
}

// With creates logger with additional fields
func (l *LoggerAdapter) With(fields ...ports.Field) ports.Logger {
	// Note: logging.Logger doesn't support With() method
	// Return same logger for now
	return l
}

// WithContext creates logger with context
func (l *LoggerAdapter) WithContext(ctx context.Context) ports.Logger {
	// Note: logging.Logger doesn't support WithContext() method
	// Return same logger for now
	return l
}

// convertFields converts ports.Field to logging.Field
func convertFields(fields []ports.Field) []logging.Field {
	if len(fields) == 0 {
		return nil
	}

	logFields := make([]logging.Field, len(fields))
	for i, f := range fields {
		logFields[i] = logging.F(f.Key(), f.Value())
	}
	return logFields
}
