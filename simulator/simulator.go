package simulator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"game-engine/models"
)

const (
	// correctChance is the probability of generating a correct answer (30%).
	correctChance = 0.3
	// defaultMaxConcurrent is the default worker pool size.
	defaultMaxConcurrent = 100
	// minDelayMs and maxDelayMs define the random network lag range (ms).
	minDelayMs = 10
	maxDelayMs = 1000
)

// Config holds the simulator configuration.
type Config struct {
	NumUsers      int
	ServerURL     string
	MaxConcurrent int    // max goroutines in flight (worker pool size)
	LogDir        string // directory to write request JSONL log
}

// Run launches simulated users using a worker pool and sends their responses
// to the API server concurrently, bounded by MaxConcurrent goroutines.
//
// It also writes each outgoing request to a JSONL log file in LogDir.
// It blocks until all users have completed their requests and returns
// the total duration of the simulation.
//
// Parameters:
//   - cfg (Config): simulation configuration containing NumUsers,
//     ServerURL, MaxConcurrent, and LogDir.
//
// Returns:
//   - time.Duration: the total elapsed time from first goroutine launch
//     to last goroutine completion.
//   - string: the absolute path to the JSONL request log file.
func Run(cfg Config) (time.Duration, string) {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = defaultMaxConcurrent // default pool size
	}

	fmt.Printf("Simulating %d users (workers: %d, correct chance: %.0f%%)...\n",
		cfg.NumUsers, cfg.MaxConcurrent, correctChance*100)

	// Set up JSONL request logger
	logFile, err := createLogFile(cfg.LogDir)
	var logPath string
	if err != nil {
		fmt.Printf("Warning: could not create request log: %v\n", err)
	} else {
		logPath = logFile.Name()
		fmt.Printf("Request log: %s\n", logPath)
		defer logFile.Close()
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.MaxConcurrent) // semaphore for worker pool
	start := time.Now()

	for i := 0; i < cfg.NumUsers; i++ {
		wg.Add(1)
		sem <- struct{}{} // acquire slot (blocks if pool is full)
		go func(userID int) {
			defer wg.Done()
			defer func() { <-sem }() // release slot
			simulateUser(userID, cfg, logFile)
		}(i + 1)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("All %d users finished in %v\n", cfg.NumUsers, elapsed)
	return elapsed, logPath
}

// createLogFile creates the JSONL log file inside the specified directory.
//
// If the directory does not exist, it is created recursively.
//
// Parameters:
//   - logDir (string): path to the directory for the log file.
//
// Returns:
//   - *os.File: the opened log file, or nil on error.
//   - error: non-nil if the directory or file could not be created.
func createLogFile(logDir string) (*os.File, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", logDir, err)
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("requests-%s.jsonl", time.Now().Format("20060102-150405")))
	return os.Create(logPath)
}

// simulateUser simulates a single user submitting a game response.
//
// It sleeps for a random duration between 10–1000ms to simulate network
// latency, then sends a POST request with a JSON-encoded UserResponse
// to the configured API server. The request payload is also logged to the
// JSONL log file (if available).
//
// Parameters:
//   - userID (int): the numeric identifier for this user (1-based).
//   - cfg (Config): simulation configuration with ServerURL.
//   - logFile (*os.File): open JSONL log file, may be nil if logging failed.
//
// Returns:
//   - (none)
func simulateUser(userID int, cfg Config, logFile *os.File) {
	delay := randomDelay()
	time.Sleep(delay)

	isCorrect := rand.Float64() < correctChance

	resp := models.UserResponse{
		UserID:    fmt.Sprintf("user-%d", userID),
		Answer:    generateAnswer(isCorrect),
		IsCorrect: isCorrect,
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(resp)

	// Log the request payload to JSONL file.
	if logFile != nil {
		logFile.Write(append(body, '\n'))
	}

	httpResp, err := http.Post(
		cfg.ServerURL+"/submit",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Printf("User %d: request failed: %v\n", userID, err)
		return
	}
	defer httpResp.Body.Close()
}

// generateAnswer produces a mock answer string for a simulated user.
//
// If correct is true, it returns the canonical correct answer ("42").
// Otherwise, it returns a randomly selected wrong answer from a
// predefined pool.
//
// Parameters:
//   - correct (bool): whether to return the correct answer.
//
// Returns:
//   - string: the answer string ("42" if correct, random wrong answer
//     otherwise).
func generateAnswer(correct bool) string {
	if correct {
		return "42" // the correct answer
	}
	// Random wrong answers
	wrong := []string{"7", "13", "99", "0", "256"}
	return wrong[rand.Intn(len(wrong))]
}

// randomDelay returns a random duration between minDelayMs and maxDelayMs
// milliseconds to simulate variable network latency.
//
// Parameters:
//   - (none)
//
// Returns:
//   - time.Duration: a random delay between 10ms and 1000ms.
func randomDelay() time.Duration {
	return time.Duration(minDelayMs+rand.Intn(maxDelayMs-minDelayMs+1)) * time.Millisecond
}
