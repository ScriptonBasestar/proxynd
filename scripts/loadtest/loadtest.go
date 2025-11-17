// Package main provides a simple load testing tool for ProxyND Enterprise API
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Config holds load test configuration
type Config struct {
	URL         string
	Duration    time.Duration
	Concurrency int
	RampUp      time.Duration
	Token       string
}

// Result holds test results
type Result struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	Latencies       []time.Duration
	StatusCodes     map[int]int64
}

func main() {
	// Parse flags
	url := flag.String("url", "http://localhost:8080/api/v1/enterprise/analytics/overview", "Target URL")
	duration := flag.Duration("duration", 10*time.Second, "Test duration")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	rampUp := flag.Duration("ramp-up", 0, "Ramp-up period")
	token := flag.String("token", "", "Bearer token for authentication")
	flag.Parse()

	config := &Config{
		URL:         *url,
		Duration:    *duration,
		Concurrency: *concurrency,
		RampUp:      *rampUp,
		Token:       *token,
	}

	fmt.Printf("Load Test Configuration\n")
	fmt.Printf("=======================\n")
	fmt.Printf("URL:         %s\n", config.URL)
	fmt.Printf("Duration:    %s\n", config.Duration)
	fmt.Printf("Concurrency: %d\n", config.Concurrency)
	fmt.Printf("Ramp-up:     %s\n", config.RampUp)
	fmt.Printf("\n")

	result := runLoadTest(config)
	printResults(result, config)
}

func runLoadTest(config *Config) *Result {
	result := &Result{
		StatusCodes: make(map[int]int64),
		Latencies:   make([]time.Duration, 0, 10000),
		MinLatency:  time.Hour, // Initialize with high value
	}

	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	latencyChan := make(chan time.Duration, 1000)
	statusChan := make(chan int, 1000)

	// Start time
	startTime := time.Now()
	endTime := startTime.Add(config.Duration)

	// Start workers
	workersPerSecond := 0
	if config.RampUp > 0 {
		workersPerSecond = config.Concurrency / int(config.RampUp.Seconds())
		if workersPerSecond == 0 {
			workersPerSecond = 1
		}
	}

	activeWorkers := 0
	for i := 0; i < config.Concurrency; i++ {
		// Ramp-up delay
		if config.RampUp > 0 && workersPerSecond > 0 {
			if i > 0 && i%workersPerSecond == 0 {
				time.Sleep(1 * time.Second)
			}
		}

		wg.Add(1)
		activeWorkers++
		go worker(config, stopChan, latencyChan, statusChan, &wg)
	}

	// Collector goroutine
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		var mu sync.Mutex
		for {
			select {
			case latency := <-latencyChan:
				atomic.AddInt64(&result.TotalRequests, 1)
				mu.Lock()
				result.Latencies = append(result.Latencies, latency)
				result.TotalLatency += latency
				if latency < result.MinLatency {
					result.MinLatency = latency
				}
				if latency > result.MaxLatency {
					result.MaxLatency = latency
				}
				mu.Unlock()
			case status := <-statusChan:
				mu.Lock()
				result.StatusCodes[status]++
				if status >= 200 && status < 300 {
					atomic.AddInt64(&result.SuccessRequests, 1)
				} else {
					atomic.AddInt64(&result.FailedRequests, 1)
				}
				mu.Unlock()
			case <-time.After(100 * time.Millisecond):
				if time.Now().After(endTime) {
					return
				}
			}
		}
	}()

	// Wait for duration
	time.Sleep(config.Duration)
	close(stopChan)
	wg.Wait()

	// Give collector time to finish
	time.Sleep(500 * time.Millisecond)
	collectorWg.Wait()

	return result
}

func worker(config *Config, stopChan chan struct{}, latencyChan chan time.Duration, statusChan chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for {
		select {
		case <-stopChan:
			return
		default:
			start := time.Now()
			req, err := http.NewRequest("GET", config.URL, nil)
			if err != nil {
				continue
			}

			if config.Token != "" {
				req.Header.Set("Authorization", "Bearer "+config.Token)
			}

			resp, err := client.Do(req)
			latency := time.Since(start)

			if err != nil {
				statusChan <- 0
				latencyChan <- latency
				continue
			}

			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			statusChan <- resp.StatusCode
			latencyChan <- latency
		}
	}
}

func printResults(result *Result, config *Config) {
	fmt.Printf("\nLoad Test Results\n")
	fmt.Printf("=================\n\n")

	// Request statistics
	fmt.Printf("Requests:\n")
	fmt.Printf("  Total:    %d\n", result.TotalRequests)
	fmt.Printf("  Success:  %d (%.2f%%)\n", result.SuccessRequests, float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("  Failed:   %d (%.2f%%)\n", result.FailedRequests, float64(result.FailedRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("\n")

	// Throughput
	rps := float64(result.TotalRequests) / config.Duration.Seconds()
	fmt.Printf("Throughput:\n")
	fmt.Printf("  Requests/sec: %.2f\n", rps)
	fmt.Printf("\n")

	// Latency statistics
	if len(result.Latencies) > 0 {
		avgLatency := result.TotalLatency / time.Duration(len(result.Latencies))
		p50 := percentile(result.Latencies, 50)
		p95 := percentile(result.Latencies, 95)
		p99 := percentile(result.Latencies, 99)

		fmt.Printf("Latency:\n")
		fmt.Printf("  Min:  %s\n", result.MinLatency)
		fmt.Printf("  Avg:  %s\n", avgLatency)
		fmt.Printf("  P50:  %s\n", p50)
		fmt.Printf("  P95:  %s\n", p95)
		fmt.Printf("  P99:  %s\n", p99)
		fmt.Printf("  Max:  %s\n", result.MaxLatency)
		fmt.Printf("\n")
	}

	// Status codes
	fmt.Printf("Status Codes:\n")
	for code, count := range result.StatusCodes {
		fmt.Printf("  %d: %d\n", code, count)
	}
	fmt.Printf("\n")

	// Performance verdict
	fmt.Printf("Performance Verdict:\n")
	if result.SuccessRequests > 0 {
		avgLatency := result.TotalLatency / time.Duration(result.SuccessRequests)
		if avgLatency < 200*time.Millisecond && rps > 100 {
			fmt.Printf("  ✓ EXCELLENT: Low latency (<%s) and high throughput (%.0f rps)\n", avgLatency, rps)
		} else if avgLatency < 500*time.Millisecond && rps > 50 {
			fmt.Printf("  ✓ GOOD: Acceptable latency (%s) and throughput (%.0f rps)\n", avgLatency, rps)
		} else if avgLatency < 1*time.Second {
			fmt.Printf("  ⚠ FAIR: Latency is acceptable (%s) but could be improved\n", avgLatency)
		} else {
			fmt.Printf("  ✗ POOR: High latency (%s) - optimization needed\n", avgLatency)
		}
	}

	// Error rate check
	errorRate := float64(result.FailedRequests) / float64(result.TotalRequests) * 100
	if errorRate > 5 {
		fmt.Printf("  ✗ WARNING: High error rate (%.2f%%) - investigate failures\n", errorRate)
	} else if errorRate > 1 {
		fmt.Printf("  ⚠ NOTICE: Error rate is %.2f%% - monitor closely\n", errorRate)
	} else if result.FailedRequests > 0 {
		fmt.Printf("  ✓ Error rate is low (%.2f%%)\n", errorRate)
	} else {
		fmt.Printf("  ✓ No errors detected\n")
	}
}

func percentile(latencies []time.Duration, p int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Sort latencies (simple bubble sort for small datasets)
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := (len(sorted) * p) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
