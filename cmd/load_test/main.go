package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestResult struct {
	TotalRequests    int64
	SuccessfulRequests int64
	FailedRequests   int64
	TotalDuration    time.Duration
	AverageResponseTime time.Duration
	MinResponseTime   time.Duration
	MaxResponseTime   time.Duration
	RequestsPerSecond float64
}

type RequestResult struct {
	UserID     string
	Success    bool
	Duration   time.Duration
	Error      error
	StatusCode int
}

func main() {
	log.Println("🚀 Starting Load Test for Manifold API")
	log.Println("Testing 5000 concurrent requests from 10 users")

	// Configuration
	baseURL := "http://localhost:8080"
	totalRequests := 5000
	numUsers := 10
	concurrentWorkers := 100 // Number of goroutines to use

	// Check for quick test mode
	if len(os.Args) > 1 && os.Args[1] == "quick" {
		totalRequests = 50
		concurrentWorkers = 10
		log.Println("🔧 QUICK TEST MODE: 50 requests, 10 concurrent workers")
	}

	// Generate user IDs
	userIDs := generateUserIDs(numUsers)
	log.Printf("Generated %d user IDs: %v", len(userIDs), len(userIDs))

	// Run load test
	result := runLoadTest(baseURL, totalRequests, userIDs, concurrentWorkers)

	// Print results
	printResults(result)
}

func generateUserIDs(count int) []string {
	userIDs := make([]string, count)
	for i := 0; i < count; i++ {
		userIDs[i] = fmt.Sprintf("loadtest_user_%d", i+1)
	}
	return userIDs
}

func runLoadTest(baseURL string, totalRequests int, userIDs []string, concurrentWorkers int) LoadTestResult {
	var (
		successfulRequests int64
		failedRequests     int64
		totalDuration      int64
		minResponseTime    int64 = 1<<63 - 1
		maxResponseTime    int64
		mu                 sync.Mutex
	)

	// Create channels for coordination
	requestChan := make(chan string, totalRequests)
	resultChan := make(chan RequestResult, totalRequests)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrentWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for userID := range requestChan {
				result := makeRequest(baseURL, userID)
				resultChan <- result
			}
		}(i)
	}

	// Start result collector
	go func() {
		for result := range resultChan {
			if result.Success {
				atomic.AddInt64(&successfulRequests, 1)
			} else {
				atomic.AddInt64(&failedRequests, 1)
			}

			duration := int64(result.Duration)
			atomic.AddInt64(&totalDuration, duration)

			mu.Lock()
			if duration < minResponseTime {
				minResponseTime = duration
			}
			if duration > maxResponseTime {
				maxResponseTime = duration
			}
			mu.Unlock()
		}
	}()

	// Start the test
	startTime := time.Now()
	log.Printf("Starting %d requests with %d concurrent workers...", totalRequests, concurrentWorkers)

	// Send requests
	for i := 0; i < totalRequests; i++ {
		userID := userIDs[i%len(userIDs)]
		requestChan <- userID
	}

	// Close channels and wait
	close(requestChan)
	wg.Wait()
	close(resultChan)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Calculate results
	successful := atomic.LoadInt64(&successfulRequests)
	failed := atomic.LoadInt64(&failedRequests)
	total := atomic.LoadInt64(&totalDuration)

	mu.Lock()
	minTime := minResponseTime
	maxTime := maxResponseTime
	mu.Unlock()

	avgTime := time.Duration(0)
	if successful > 0 {
		avgTime = time.Duration(total / successful)
	}

	return LoadTestResult{
		TotalRequests:      int64(totalRequests),
		SuccessfulRequests: successful,
		FailedRequests:     failed,
		TotalDuration:      duration,
		AverageResponseTime: avgTime,
		MinResponseTime:    time.Duration(minTime),
		MaxResponseTime:    time.Duration(maxTime),
		RequestsPerSecond:  float64(totalRequests) / duration.Seconds(),
	}
}

func makeRequest(baseURL, userID string) RequestResult {
	startTime := time.Now()

	// Create request
	req, err := http.NewRequest("POST", baseURL+"/generate-data", nil)
	if err != nil {
		return RequestResult{
			UserID:  userID,
			Success: false,
			Error:   err,
		}
	}

	// Add headers
	req.Header.Set("X-User-Id", userID)
	req.Header.Set("Connection", "close")

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req = req.WithContext(ctx)

	// Make request
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return RequestResult{
			UserID:   userID,
			Success:  false,
			Duration: duration,
			Error:    err,
		}
	}
	defer resp.Body.Close()

	// Read response body (for streaming)
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return RequestResult{
			UserID:     userID,
			Success:    false,
			Duration:   duration,
			Error:      err,
			StatusCode: resp.StatusCode,
		}
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	return RequestResult{
		UserID:     userID,
		Success:    success,
		Duration:   duration,
		StatusCode: resp.StatusCode,
	}
}

func printResults(result LoadTestResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 LOAD TEST RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Requests:        %d\n", result.TotalRequests)
	fmt.Printf("Successful Requests:   %d (%.2f%%)\n", result.SuccessfulRequests, 
		float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("Failed Requests:       %d (%.2f%%)\n", result.FailedRequests,
		float64(result.FailedRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("Total Duration:        %v\n", result.TotalDuration)
	fmt.Printf("Requests Per Second:   %.2f\n", result.RequestsPerSecond)
	fmt.Printf("Average Response Time: %v\n", result.AverageResponseTime)
	fmt.Printf("Min Response Time:     %v\n", result.MinResponseTime)
	fmt.Printf("Max Response Time:     %v\n", result.MaxResponseTime)
	fmt.Println(strings.Repeat("=", 60))

	// Assessment criteria check
	fmt.Println("\n🎯 ASSESSMENT CRITERIA CHECK:")
	if result.SuccessfulRequests >= int64(float64(result.TotalRequests)*0.95) {
		fmt.Println("✅ SUCCESS: >95% requests successful")
	} else {
		fmt.Println("❌ FAILED: <95% requests successful")
	}

	if result.RequestsPerSecond >= 50 {
		fmt.Println("✅ SUCCESS: >50 requests/second throughput")
	} else {
		fmt.Println("❌ FAILED: <50 requests/second throughput")
	}

	if result.AverageResponseTime < 30*time.Second {
		fmt.Println("✅ SUCCESS: Average response time < 30 seconds")
	} else {
		fmt.Println("❌ FAILED: Average response time > 30 seconds")
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
} 