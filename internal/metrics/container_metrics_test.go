package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"proxynd/internal/metrics"
)

func TestNewContainerMetrics(t *testing.T) {
	containerMetrics := metrics.NewContainerMetrics()

	assert.NotNil(t, containerMetrics)
	assert.NotNil(t, containerMetrics.ContainerHandlerInitializations)
	assert.NotNil(t, containerMetrics.ContainerHandlerRequests)
	assert.NotNil(t, containerMetrics.ContainerHandlerDuration)
	assert.NotNil(t, containerMetrics.ContainerHandlerErrors)
	assert.NotNil(t, containerMetrics.ConfigLoadOperations)
	assert.NotNil(t, containerMetrics.ConfigLoadDuration)
	assert.NotNil(t, containerMetrics.ConfigCacheHits)
	assert.NotNil(t, containerMetrics.ConfigCacheMisses)
	assert.NotNil(t, containerMetrics.ContainerProviderCalls)
	assert.NotNil(t, containerMetrics.HandlerFactoryOperations)
	assert.NotNil(t, containerMetrics.HandlerInstancesActive)
	assert.NotNil(t, containerMetrics.HandlerHealthChecks)
	assert.NotNil(t, containerMetrics.HandlerConfigReloads)
	assert.NotNil(t, containerMetrics.HandlerUpstreamBuilds)
	assert.NotNil(t, containerMetrics.CacheKeyGenerations)
	assert.NotNil(t, containerMetrics.CacheKeyGenerationTime)
}

func TestGetContainerMetrics(t *testing.T) {
	// Reset global instance for clean test
	metrics.ResetContainerMetrics()

	containerMetrics1 := metrics.GetContainerMetrics()
	containerMetrics2 := metrics.GetContainerMetrics()

	assert.NotNil(t, containerMetrics1)
	assert.NotNil(t, containerMetrics2)
	assert.Equal(t, containerMetrics1, containerMetrics2) // Same instance
}

func TestInitContainerMetrics(t *testing.T) {
	metrics.InitContainerMetrics()

	containerMetrics := metrics.GetContainerMetrics()
	assert.NotNil(t, containerMetrics)
}

func TestResetContainerMetrics(t *testing.T) {
	// Get initial instance
	metrics1 := metrics.GetContainerMetrics()

	// Reset metrics
	metrics.ResetContainerMetrics()

	// Get new instance
	metrics2 := metrics.GetContainerMetrics()

	assert.NotNil(t, metrics1)
	assert.NotNil(t, metrics2)
	// They should be different instances
	assert.NotEqual(t, metrics1, metrics2)
}

func TestContainerMetricsRecordMethods(t *testing.T) {
	// Use separate registry for this test
	metrics.ResetContainerMetrics()
	containerMetrics := metrics.GetContainerMetrics()

	// Test RecordHandlerInitialization
	containerMetrics.RecordHandlerInitialization("apt", true)
	containerMetrics.RecordHandlerInitialization("maven", false)

	// Test RecordHandlerRequest
	containerMetrics.RecordHandlerRequest("apt", "GET", 200)
	containerMetrics.RecordHandlerRequest("maven", "POST", 404)

	// Test RecordHandlerDuration
	containerMetrics.RecordHandlerDuration("apt", "GET", 0.5)
	containerMetrics.RecordHandlerDuration("maven", "POST", 1.2)

	// Test RecordHandlerError
	containerMetrics.RecordHandlerError("apt", "client_error")
	containerMetrics.RecordHandlerError("maven", "server_error")

	// Test RecordConfigLoad
	containerMetrics.RecordConfigLoad("apt", "initial", true, 0.1)
	containerMetrics.RecordConfigLoad("maven", "reload", false, 0.2)

	// Test RecordConfigCacheHit/Miss
	containerMetrics.RecordConfigCacheHit("apt", "apt-config")
	containerMetrics.RecordConfigCacheMiss("maven", "maven-config")

	// Test RecordContainerProviderCall
	containerMetrics.RecordContainerProviderCall("GetAptProxyConfig", "apt", true)
	containerMetrics.RecordContainerProviderCall("GetMavenProxyConfig", "maven", false)

	// Test RecordHandlerFactoryOperation
	containerMetrics.RecordHandlerFactoryOperation("create", "apt", true)
	containerMetrics.RecordHandlerFactoryOperation("register", "maven", false)

	// Test SetActiveHandlerInstances
	containerMetrics.SetActiveHandlerInstances("apt", 3)
	containerMetrics.SetActiveHandlerInstances("maven", 2)

	// Test RecordHealthCheck
	containerMetrics.RecordHealthCheck("apt", true)
	containerMetrics.RecordHealthCheck("maven", false)

	// Test RecordConfigReload
	containerMetrics.RecordConfigReload("apt", true)
	containerMetrics.RecordConfigReload("maven", false)

	// Test RecordUpstreamBuild
	containerMetrics.RecordUpstreamBuild("apt", true)
	containerMetrics.RecordUpstreamBuild("maven", false)

	// Test RecordCacheKeyGeneration
	containerMetrics.RecordCacheKeyGeneration("apt", 0.001)
	containerMetrics.RecordCacheKeyGeneration("maven", 0.002)

	// No errors should occur - this test verifies the methods don't panic
}

func TestContainerMetricsMultipleHandlerTypes(t *testing.T) {
	// Use separate registry to avoid conflicts
	metrics.ResetContainerMetrics()
	containerMetrics := metrics.GetContainerMetrics()

	handlerTypes := []string{"apt", "maven", "npm", "docker", "pip", "yum", "apk"}

	for _, handlerType := range handlerTypes {
		// Test all metrics for each handler type
		containerMetrics.RecordHandlerInitialization(handlerType, true)
		containerMetrics.RecordHandlerRequest(handlerType, "GET", 200)
		containerMetrics.RecordHandlerDuration(handlerType, "GET", 0.5)
		containerMetrics.RecordConfigLoad(handlerType, "initial", true, 0.1)
		containerMetrics.RecordConfigCacheHit(handlerType, handlerType+"-config")
		containerMetrics.RecordContainerProviderCall("Get"+handlerType+"ProxyConfig", handlerType, true)
		containerMetrics.RecordHandlerFactoryOperation("create", handlerType, true)
		containerMetrics.SetActiveHandlerInstances(handlerType, 1)
		containerMetrics.RecordHealthCheck(handlerType, true)
		containerMetrics.RecordConfigReload(handlerType, true)
		containerMetrics.RecordUpstreamBuild(handlerType, true)
		containerMetrics.RecordCacheKeyGeneration(handlerType, 0.001)
	}

	// Should complete without errors
}

func TestContainerMetricsEdgeCases(t *testing.T) {
	metrics.ResetContainerMetrics()
	containerMetrics := metrics.GetContainerMetrics()

	// Test with empty strings
	containerMetrics.RecordHandlerInitialization("", true)
	containerMetrics.RecordHandlerRequest("", "", 0)
	containerMetrics.RecordConfigLoad("", "", false, 0)

	// Test with extreme values
	containerMetrics.RecordHandlerDuration("test", "GET", 999999.0)
	containerMetrics.SetActiveHandlerInstances("test", -1)     // Negative values
	containerMetrics.SetActiveHandlerInstances("test", 999999) // Large values

	// Should not panic
}
