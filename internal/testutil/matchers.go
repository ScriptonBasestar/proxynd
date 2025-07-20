package testutil

import (
	"strings"
	"time"

	"github.com/stretchr/testify/mock"
)

// Matchers provides custom mock argument matchers
type Matchers struct{}

// NewMatchers creates a new matchers instance
func NewMatchers() *Matchers {
	return &Matchers{}
}

// AnyContext returns a matcher for any context.Context
func (m *Matchers) AnyContext() interface{} {
	return mock.Anything
}

// AnyString returns a matcher for any string
func (m *Matchers) AnyString() interface{} {
	return mock.AnythingOfType("string")
}

// AnyInt returns a matcher for any int
func (m *Matchers) AnyInt() interface{} {
	return mock.AnythingOfType("int")
}

// AnyReader returns a matcher for any io.Reader
func (m *Matchers) AnyReader() interface{} {
	return mock.AnythingOfType("io.Reader")
}

// AnyDuration returns a matcher for any time.Duration
func (m *Matchers) AnyDuration() interface{} {
	return mock.AnythingOfType("time.Duration")
}

// StringContaining returns a matcher for strings containing a substring
func (m *Matchers) StringContaining(substring string) interface{} {
	return mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, substring)
	})
}

// StringHasPrefix returns a matcher for strings with a specific prefix
func (m *Matchers) StringHasPrefix(prefix string) interface{} {
	return mock.MatchedBy(func(s string) bool {
		return strings.HasPrefix(s, prefix)
	})
}

// StringHasSuffix returns a matcher for strings with a specific suffix
func (m *Matchers) StringHasSuffix(suffix string) interface{} {
	return mock.MatchedBy(func(s string) bool {
		return strings.HasSuffix(s, suffix)
	})
}

// DurationBetween returns a matcher for durations within a range
func (m *Matchers) DurationBetween(minDur, maxDur time.Duration) interface{} {
	return mock.MatchedBy(func(d time.Duration) bool {
		return d >= minDur && d <= maxDur
	})
}

// MapContaining returns a matcher for maps containing specific keys
func (m *Matchers) MapContaining(keys ...string) interface{} {
	return mock.MatchedBy(func(m map[string]string) bool {
		for _, key := range keys {
			if _, ok := m[key]; !ok {
				return false
			}
		}
		return true
	})
}

// MapWithValue returns a matcher for maps with a specific key-value pair
func (m *Matchers) MapWithValue(key, value string) interface{} {
	return mock.MatchedBy(func(m map[string]string) bool {
		return m[key] == value
	})
}

// PathStartingWith returns a matcher for paths starting with a prefix
func (m *Matchers) PathStartingWith(prefix string) interface{} {
	return mock.MatchedBy(func(path string) bool {
		return strings.HasPrefix(path, prefix)
	})
}

// URLContaining returns a matcher for URLs containing a substring
func (m *Matchers) URLContaining(substring string) interface{} {
	return mock.MatchedBy(func(url string) bool {
		return strings.Contains(url, substring)
	})
}
