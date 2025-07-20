package errors

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorBuilder_CompleteFlow(t *testing.T) {
	originalErr := errors.New("database connection failed")

	domainErr := NewError("DB001", "Database error").
		WithDomain("database").
		WithCause(originalErr).
		WithLevel(ErrorLevelCritical).
		WithDetails(map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"database": "testdb",
		}).
		Build()

	assert.Equal(t, "DB001", domainErr.Code)
	assert.Equal(t, "Database error", domainErr.Message)
	assert.Equal(t, "database", domainErr.Domain)
	assert.Equal(t, ErrorLevelCritical, domainErr.Level)
	assert.Equal(t, originalErr, domainErr.Cause)

	details := domainErr.Details.(map[string]interface{})
	assert.Equal(t, "localhost", details["host"])
	assert.Equal(t, 5432, details["port"])
	assert.Equal(t, "testdb", details["database"])
	assert.NotZero(t, domainErr.Timestamp)
}

func TestErrorBuilder_MinimalBuild(t *testing.T) {
	domainErr := NewError("SIMPLE001", "Simple error").Build()

	assert.Equal(t, "SIMPLE001", domainErr.Code)
	assert.Equal(t, "Simple error", domainErr.Message)
	assert.Equal(t, "", domainErr.Domain)             // Default empty
	assert.Equal(t, ErrorLevelError, domainErr.Level) // Default level
	assert.Nil(t, domainErr.Cause)
	assert.Nil(t, domainErr.Details)
}

func TestErrorBuilder_ChainedErrors(t *testing.T) {
	// Create a chain of errors
	rootCause := errors.New("root cause")

	middleErr := NewError("MID001", "Middle error").
		WithCause(rootCause).
		Build()

	topErr := NewError("TOP001", "Top error").
		WithCause(middleErr).
		WithDomain("application").
		Build()

	// Verify the chain
	assert.Equal(t, "TOP001", topErr.Code)
	assert.Equal(t, middleErr, topErr.Cause)

	assert.Equal(t, "MID001", middleErr.Code)
	assert.Equal(t, rootCause, middleErr.Cause)
}

func TestErrorBuilder_NilHandling(t *testing.T) {
	// Test with nil cause
	domainErr1 := NewError("NIL001", "Nil cause").
		WithCause(nil).
		Build()

	assert.Nil(t, domainErr1.Cause)

	// Test with nil details
	domainErr2 := NewError("NIL002", "Nil details").
		WithDetails(nil).
		Build()

	assert.Nil(t, domainErr2.Details)

	// Test with empty details
	domainErr3 := NewError("NIL003", "Empty details").
		WithDetails(map[string]interface{}{}).
		Build()

	assert.NotNil(t, domainErr3.Details)
	details := domainErr3.Details.(map[string]interface{})
	assert.Empty(t, details)
}

func TestErrorBuilder_OverwriteValues(t *testing.T) {
	// Test overwriting values
	domainErr := NewError("OVER001", "Original message").
		WithDomain("domain1").
		WithLevel(ErrorLevelInfo).
		WithDomain("domain2").         // Overwrite domain
		WithLevel(ErrorLevelCritical). // Overwrite level
		Build()

	assert.Equal(t, "domain2", domainErr.Domain)
	assert.Equal(t, ErrorLevelCritical, domainErr.Level)
}

func TestErrorBuilder_ConcurrentBuilding(t *testing.T) {
	// Test concurrent error building
	var wg sync.WaitGroup
	errors := make([]error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			errors[idx] = NewError("CONCURRENT", "Concurrent error").
				WithDetails(map[string]interface{}{
					"index": idx,
					"time":  time.Now().UnixNano(),
				}).
				Build()
		}(i)
	}

	wg.Wait()

	// Verify all errors were created correctly
	for i, err := range errors {
		domainErr := err.(*DomainError)
		assert.Equal(t, "CONCURRENT", domainErr.Code)
		details := domainErr.Details.(map[string]interface{})
		assert.Equal(t, i, details["index"])
		assert.NotNil(t, details["time"])
	}
}

func TestErrorBuilder_ComplexDetails(t *testing.T) {
	// Test with complex nested details
	domainErr := NewError("COMPLEX001", "Complex error").
		WithDetails(map[string]interface{}{
			"user": map[string]interface{}{
				"id":    12345,
				"name":  "John Doe",
				"roles": []string{"admin", "user"},
			},
			"request": map[string]interface{}{
				"method": "POST",
				"path":   "/api/users",
				"headers": map[string]string{
					"Content-Type": "application/json",
					"User-Agent":   "test-client/1.0",
				},
			},
			"metadata": []interface{}{
				"value1",
				42,
				true,
				nil,
			},
		}).
		Build()

	// Verify nested structure
	details := domainErr.Details.(map[string]interface{})
	user := details["user"].(map[string]interface{})
	assert.Equal(t, 12345, user["id"])
	assert.Equal(t, "John Doe", user["name"])

	roles := user["roles"].([]string)
	assert.Contains(t, roles, "admin")
	assert.Contains(t, roles, "user")

	request := details["request"].(map[string]interface{})
	assert.Equal(t, "POST", request["method"])

	headers := request["headers"].(map[string]string)
	assert.Equal(t, "application/json", headers["Content-Type"])
}

func TestErrorBuilder_AllLevels(t *testing.T) {
	levels := []ErrorLevel{
		ErrorLevelInfo,
		ErrorLevelWarning,
		ErrorLevelError,
		ErrorLevelCritical,
	}

	for _, level := range levels {
		domainErr := NewError("LEVEL001", "Level test").
			WithLevel(level).
			Build()

		assert.Equal(t, level, domainErr.Level)
	}
}

func TestErrorBuilder_TimestampConsistency(t *testing.T) {
	// Create multiple errors in sequence
	before := time.Now()

	domainErr1 := NewError("TIME001", "First error").Build()
	time.Sleep(10 * time.Millisecond)
	domainErr2 := NewError("TIME002", "Second error").Build()

	after := time.Now()

	// Verify timestamps are in correct order
	assert.True(t, domainErr1.Timestamp.After(before))
	assert.True(t, domainErr1.Timestamp.Before(domainErr2.Timestamp))
	assert.True(t, domainErr2.Timestamp.Before(after))
}

func TestErrorBuilder_EmptyCodeAndMessage(t *testing.T) {
	// Test with empty code
	domainErr1 := NewError("", "Empty code").Build()
	assert.Equal(t, "", domainErr1.Code)
	assert.Equal(t, "Empty code", domainErr1.Message)

	// Test with empty message
	domainErr2 := NewError("EMPTY001", "").Build()
	assert.Equal(t, "EMPTY001", domainErr2.Code)
	assert.Equal(t, "", domainErr2.Message)
}

// Benchmark error building
func BenchmarkErrorBuilder_Simple(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewError("BENCH001", "Benchmark error").Build()
		}
	})
}

func BenchmarkErrorBuilder_WithDetails(b *testing.B) {
	details := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewError("BENCH002", "Benchmark error").
				WithDomain("benchmark").
				WithDetails(details).
				Build()
		}
	})
}

func BenchmarkErrorBuilder_Complete(b *testing.B) {
	cause := errors.New("benchmark cause")
	details := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"nested": map[string]interface{}{
			"subkey": "subvalue",
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewError("BENCH003", "Complete benchmark error").
				WithDomain("benchmark").
				WithCause(cause).
				WithLevel(ErrorLevelWarning).
				WithDetails(details).
				Build()
		}
	})
}
