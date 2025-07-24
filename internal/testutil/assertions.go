// Package testutil provides testing utilities and assertions
package testutil

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Assertions provides custom assertion helpers for tests
type Assertions struct {
	t *testing.T
}

// NewAssertions creates a new assertions helper
func NewAssertions(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

// AssertReaderContent asserts that a reader contains expected content
func (a *Assertions) AssertReaderContent(reader io.Reader, expected string) {
	content, err := io.ReadAll(reader)
	require.NoError(a.t, err)
	assert.Equal(a.t, expected, string(content))
}

// AssertReaderContains asserts that a reader contains a substring
func (a *Assertions) AssertReaderContains(reader io.Reader, substring string) {
	content, err := io.ReadAll(reader)
	require.NoError(a.t, err)
	assert.Contains(a.t, string(content), substring)
}

// AssertMapContains asserts that a map contains expected key-value pairs
func (a *Assertions) AssertMapContains(actual, expected map[string]string) {
	for k, v := range expected {
		assert.Equal(a.t, v, actual[k], "Key %s", k)
	}
}

// AssertHeadersEqual asserts that two header maps are equal
func (a *Assertions) AssertHeadersEqual(expected, actual map[string]string) {
	assert.Equal(a.t, len(expected), len(actual), "Header count mismatch")
	for k, v := range expected {
		assert.Equal(a.t, v, actual[k], "Header %s", k)
	}
}

// AssertCacheHit asserts that a response was served from cache
func (a *Assertions) AssertCacheHit(cached bool) {
	assert.True(a.t, cached, "Expected cache hit")
}

// AssertCacheMiss asserts that a response was not served from cache
func (a *Assertions) AssertCacheMiss(cached bool) {
	assert.False(a.t, cached, "Expected cache miss")
}

// AssertStatusCode asserts response status code
func (a *Assertions) AssertStatusCode(expected, actual int) {
	assert.Equal(a.t, expected, actual, "Status code mismatch")
}

// AssertNoError asserts that no error occurred
func (a *Assertions) AssertNoError(err error) {
	assert.NoError(a.t, err)
}

// AssertError asserts that an error occurred
func (a *Assertions) AssertError(err error) {
	assert.Error(a.t, err)
}

// AssertErrorContains asserts that an error contains a specific message
func (a *Assertions) AssertErrorContains(err error, message string) {
	require.Error(a.t, err)
	assert.Contains(a.t, err.Error(), message)
}
