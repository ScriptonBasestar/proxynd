package app

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/config"
)

func TestConfigChangeNotifier_Basic(t *testing.T) {
	// Given
	notifier := NewConfigChangeNotifier()
	assert.NotNil(t, notifier)
	assert.Equal(t, 0, notifier.GetListenerCount())

	// When
	callCount := 0
	listener := func(interface{}) {
		callCount++
	}
	notifier.AddListener(listener)

	// Then
	assert.Equal(t, 1, notifier.GetListenerCount())

	// When
	config := &config.UnifiedConfig{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	notifier.NotifyChange(config)

	// Then (with timeout to avoid hanging)
	timeout := time.After(100 * time.Millisecond)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			assert.Equal(t, 1, callCount)
			return
		case <-ticker.C:
			if callCount == 1 {
				assert.Equal(t, 1, callCount)
				return
			}
		}
	}
}

func TestConfigChangeNotifier_MultipleListeners(t *testing.T) {
	// Given
	notifier := NewConfigChangeNotifier()

	var wg sync.WaitGroup
	callCounts := make([]int, 3)

	// Add multiple listeners
	for i := 0; i < 3; i++ {
		index := i
		wg.Add(1)
		listener := func(interface{}) {
			callCounts[index]++
			wg.Done()
		}
		notifier.AddListener(listener)
	}

	assert.Equal(t, 3, notifier.GetListenerCount())

	// When
	config := &config.UnifiedConfig{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	notifier.NotifyChange(config)

	// Then
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		for i, count := range callCounts {
			assert.Equal(t, 1, count, "Listener %d should be called once", i)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for listeners to be called")
	}
}

func TestConfigChangeNotifier_ClearListeners(t *testing.T) {
	// Given
	notifier := NewConfigChangeNotifier()

	// Add some listeners
	listener1 := func(interface{}) {}
	listener2 := func(interface{}) {}
	notifier.AddListener(listener1)
	notifier.AddListener(listener2)

	require.Equal(t, 2, notifier.GetListenerCount())

	// When
	notifier.ClearListeners()

	// Then
	assert.Equal(t, 0, notifier.GetListenerCount())
}

func TestConfigChangeNotifier_PanicHandling(t *testing.T) {
	// Given
	notifier := NewConfigChangeNotifier()

	var wg sync.WaitGroup
	panicListenerCalled := false
	normalListenerCalled := false

	// Add panic listener
	wg.Add(1)
	panicListener := func(interface{}) {
		panicListenerCalled = true
		wg.Done()
		panic("test panic")
	}
	notifier.AddListener(panicListener)

	// Add normal listener
	wg.Add(1)
	normalListener := func(interface{}) {
		normalListenerCalled = true
		wg.Done()
	}
	notifier.AddListener(normalListener)

	// When
	config := &config.UnifiedConfig{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	notifier.NotifyChange(config)

	// Then
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		assert.True(t, panicListenerCalled, "Panic listener should be called")
		assert.True(t, normalListenerCalled, "Normal listener should be called despite panic")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for listeners to be called")
	}
}

func TestConfigChangeNotifier_ConcurrentAccess(t *testing.T) {
	// Given
	notifier := NewConfigChangeNotifier()

	var wg sync.WaitGroup
	numGoroutines := 10
	numListenersPerGoroutine := 5

	// Add listeners concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < numListenersPerGoroutine; j++ {
				listener := func(interface{}) {}
				notifier.AddListener(listener)
			}
		}(i)
	}

	wg.Wait()

	// Then
	expectedCount := numGoroutines * numListenersPerGoroutine
	assert.Equal(t, expectedCount, notifier.GetListenerCount())

	// Test concurrent notification
	config := &config.UnifiedConfig{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}

	// This should not panic or cause race conditions
	for i := 0; i < 10; i++ {
		go notifier.NotifyChange(config)
	}

	// Give some time for notifications to process
	time.Sleep(100 * time.Millisecond)
}

func TestConfigChangeNotifier_EmptyNotification(t *testing.T) {
	// Given
	notifier := NewConfigChangeNotifier()

	// When (no listeners)
	config := &config.UnifiedConfig{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}

	// Then (should not panic)
	assert.NotPanics(t, func() {
		notifier.NotifyChange(config)
	})

	assert.Equal(t, 0, notifier.GetListenerCount())
}
