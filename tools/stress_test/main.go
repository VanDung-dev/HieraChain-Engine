package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// StressTestConfig holds configuration for the stress test.
type StressTestConfig struct {
	Address      string
	Concurrency  int
	RequestCount int
	Duration     time.Duration
	AuthToken    string
	AuthEnabled  bool
	ReportFile   string
}

// StressTestResult holds the results of a stress test.
type StressTestResult struct {
	TotalRequests  int64
	SuccessfulReqs int64
	FailedReqs     int64
	TotalDuration  time.Duration
	AvgLatency     time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	RequestsPerSec float64
}

func main() {
	config := parseFlags()

	fmt.Println("=== HieraChain Arrow Server Stress Test ===")
	fmt.Printf("Target: %s\n", config.Address)
	fmt.Printf("Concurrency: %d workers\n", config.Concurrency)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Auth: %v\n", config.AuthEnabled)
	fmt.Println()

	result := runStressTest(config)

	printResults(result)

	if config.ReportFile != "" {
		saveReport(config, result)
	}
}

func parseFlags() StressTestConfig {
	config := StressTestConfig{}

	flag.StringVar(&config.Address, "addr", "127.0.0.1:50051", "Arrow server address")
	flag.IntVar(&config.Concurrency, "c", 10, "Number of concurrent workers")
	flag.IntVar(&config.RequestCount, "n", 0, "Total number of requests (0 = unlimited, use -d instead)")
	flag.DurationVar(&config.Duration, "d", 30*time.Second, "Duration of test")
	flag.StringVar(&config.AuthToken, "token", "", "Authentication token")
	flag.BoolVar(&config.AuthEnabled, "auth", false, "Enable authentication")
	flag.StringVar(&config.ReportFile, "o", "", "Output report file (JSON)")

	flag.Parse()

	return config
}

func runStressTest(config StressTestConfig) StressTestResult {
	var (
		totalReqs    int64
		successReqs  int64
		failedReqs   int64
		totalLatency int64
		minLatency   int64 = 1<<63 - 1
		maxLatency   int64
		wg           sync.WaitGroup
		stopChan     = make(chan struct{})
	)

	startTime := time.Now()

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(workerID, config, stopChan, &totalReqs, &successReqs, &failedReqs, &totalLatency, &minLatency, &maxLatency)
		}(i)
	}

	// Wait for duration
	time.Sleep(config.Duration)
	close(stopChan)
	wg.Wait()

	duration := time.Since(startTime)
	total := atomic.LoadInt64(&totalReqs)
	success := atomic.LoadInt64(&successReqs)
	failed := atomic.LoadInt64(&failedReqs)
	latencySum := atomic.LoadInt64(&totalLatency)
	minLat := atomic.LoadInt64(&minLatency)
	maxLat := atomic.LoadInt64(&maxLatency)

	var avgLatency time.Duration
	if success > 0 {
		avgLatency = time.Duration(latencySum / success)
	}

	return StressTestResult{
		TotalRequests:  total,
		SuccessfulReqs: success,
		FailedReqs:     failed,
		TotalDuration:  duration,
		AvgLatency:     avgLatency,
		MinLatency:     time.Duration(minLat),
		MaxLatency:     time.Duration(maxLat),
		RequestsPerSec: float64(total) / duration.Seconds(),
	}
}

func runWorker(id int, config StressTestConfig, stop chan struct{}, totalReqs, successReqs, failedReqs, totalLatency, minLatency, maxLatency *int64) {
	for {
		select {
		case <-stop:
			return
		default:
			latency, err := sendRequest(config)
			atomic.AddInt64(totalReqs, 1)

			if err != nil {
				atomic.AddInt64(failedReqs, 1)
				// Small sleep on error to avoid hammering
				time.Sleep(10 * time.Millisecond)
			} else {
				atomic.AddInt64(successReqs, 1)
				atomic.AddInt64(totalLatency, int64(latency))

				// Update min/max latency
				lat := int64(latency)
				for {
					old := atomic.LoadInt64(minLatency)
					if lat >= old || atomic.CompareAndSwapInt64(minLatency, old, lat) {
						break
					}
				}
				for {
					old := atomic.LoadInt64(maxLatency)
					if lat <= old || atomic.CompareAndSwapInt64(maxLatency, old, lat) {
						break
					}
				}
			}
		}
	}
}

func sendRequest(config StressTestConfig) (time.Duration, error) {
	conn, err := net.DialTimeout("tcp", config.Address, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Auth handshake if needed
	if config.AuthEnabled {
		authMsg := fmt.Sprintf(`{"type":"auth","token":"%s"}`, config.AuthToken)
		if err := writeMessage(conn, []byte(authMsg)); err != nil {
			return 0, err
		}
		if _, err := readMessage(conn); err != nil {
			return 0, err
		}
	}

	// Send test request (simple JSON that will be processed)
	start := time.Now()

	testPayload := []byte(`[{"entity_id":"stress_test","event":"test","timestamp":1234567890.0}]`)
	if err := writeMessage(conn, testPayload); err != nil {
		return 0, err
	}

	// Read response
	_, err = readMessage(conn)
	latency := time.Since(start)

	return latency, err
}

func writeMessage(conn net.Conn, data []byte) error {
	// Write length prefix (4 bytes big-endian)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(data)))
	if _, err := conn.Write(length); err != nil {
		return err
	}
	_, err := conn.Write(data)
	return err
}

func readMessage(conn net.Conn) ([]byte, error) {
	// Read length prefix
	length := make([]byte, 4)
	if _, err := conn.Read(length); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint32(length)

	// Read message body
	data := make([]byte, msgLen)
	_, err := conn.Read(data)
	return data, err
}

func printResults(result StressTestResult) {
	fmt.Println("=== Results ===")
	fmt.Printf("Duration:        %v\n", result.TotalDuration.Round(time.Millisecond))
	fmt.Printf("Total Requests:  %d\n", result.TotalRequests)
	fmt.Printf("Successful:      %d (%.2f%%)\n", result.SuccessfulReqs, float64(result.SuccessfulReqs)/float64(result.TotalRequests)*100)
	fmt.Printf("Failed:          %d (%.2f%%)\n", result.FailedReqs, float64(result.FailedReqs)/float64(result.TotalRequests)*100)
	fmt.Printf("Requests/sec:    %.2f\n", result.RequestsPerSec)
	fmt.Printf("Avg Latency:     %v\n", result.AvgLatency.Round(time.Microsecond))
	fmt.Printf("Min Latency:     %v\n", result.MinLatency.Round(time.Microsecond))
	fmt.Printf("Max Latency:     %v\n", result.MaxLatency.Round(time.Microsecond))
}

func saveReport(config StressTestConfig, result StressTestResult) {
	report := map[string]interface{}{
		"config": map[string]interface{}{
			"address":     config.Address,
			"concurrency": config.Concurrency,
			"duration":    config.Duration.String(),
		},
		"results": map[string]interface{}{
			"total_requests":   result.TotalRequests,
			"successful":       result.SuccessfulReqs,
			"failed":           result.FailedReqs,
			"requests_per_sec": result.RequestsPerSec,
			"avg_latency_ms":   float64(result.AvgLatency.Microseconds()) / 1000,
			"min_latency_ms":   float64(result.MinLatency.Microseconds()) / 1000,
			"max_latency_ms":   float64(result.MaxLatency.Microseconds()) / 1000,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	data, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(config.ReportFile, data, 0644); err != nil {
		log.Printf("Failed to write report: %v", err)
	} else {
		fmt.Printf("Report saved to: %s\n", config.ReportFile)
	}
}
