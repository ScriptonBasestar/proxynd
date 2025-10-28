package plugins

import (
	"fmt"

	"proxynd/internal/logging"
)

// NewLoggerAdapter wraps the core logging.Logger with the lightweight
// plugins.Logger interface to avoid leaking internal details to plugin authors.
func NewLoggerAdapter(l logging.Logger) Logger {
	return &loggerAdapter{logger: l}
}

type loggerAdapter struct {
	logger logging.Logger
}

func (l *loggerAdapter) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, toFields(args...)...)
}

func (l *loggerAdapter) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, toFields(args...)...)
}

func (l *loggerAdapter) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, toFields(args...)...)
}

func (l *loggerAdapter) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, toFields(args...)...)
}

func toFields(args ...interface{}) []logging.Field {
	if len(args) == 0 {
		return nil
	}

	fields := make([]logging.Field, 0, len(args)/2+1)
	for i := 0; i < len(args); i += 2 {
		key := fmt.Sprintf("arg_%d", i)
		if k, ok := args[i].(string); ok && k != "" {
			key = k
		}

		var value interface{}
		if i+1 < len(args) {
			value = args[i+1]
		}

		fields = append(fields, logging.F(key, value))
	}

	return fields
}
