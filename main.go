package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"game-engine/api"
	"game-engine/engine"
	"game-engine/simulator"
)

// main is the entry point for the Game Engine with User Simulator application.
//
// It orchestrates the full lifecycle:
//  1. Parses command-line flags (-n for user count, -port for server port).
//  2. Creates and starts the Game Engine evaluation loop in a goroutine.
//  3. Creates and starts the HTTP API server.
//  4. Runs the Mock User Engine to simulate N concurrent users.
//  5. Closes the response channel and waits for the engine to finish.
//  6. Prints final metrics (winner, time-to-winner, correct/incorrect counts).
//  7. Gracefully shuts down the API server.
//
// Parameters:
//   - (none) — reads -n (int, default 1000) and -port (int, default 8080)
//     from command-line flags.
//
// Returns:
//   - (none) — exits with code 0 on success, 1 on startup failure.
func main() {
	// Flags
	numUsers := flag.Int("n", 1000, "Number of simulated users")
	port := flag.Int("port", 8080, "API server port")
	workers := flag.Int("workers", 100, "Max concurrent goroutines for simulation")
	flag.Parse()

	if *numUsers <= 0 {
		fmt.Fprintln(os.Stderr, "Error: -n must be a positive integer")
		os.Exit(1)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Game Engine with User Simulator")
	fmt.Printf("   Users: %d | Workers: %d | Port: %d\n", *numUsers, *workers, *port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Create & start the Game Engine
	eng := engine.New()
	go eng.Start()

	// Create & start the API Server
	srv := api.New(*port, eng)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	// Create temp log directory for request logging
	logDir := filepath.Join(".", "logs")

	// Run the Mock User Engine
	simCfg := simulator.Config{
		NumUsers:      *numUsers,
		ServerURL:     fmt.Sprintf("http://localhost:%d", *port),
		MaxConcurrent: *workers,
		LogDir:        logDir,
	}

	_, logPath := simulator.Run(simCfg)

	// Close the response channel (no more responses)
	close(eng.ResponseChan)

	// Wait for the engine to finish processing
	eng.WaitForWinner()

	// Print final metrics
	result := eng.GetResult()
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("    Final Metrics")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Total Responses:  %d\n", result.TotalReceived)
	fmt.Printf("   Correct Answers:  %d\n", result.TotalCorrect)
	fmt.Printf("   Incorrect Answers: %d\n", result.TotalIncorrect)

	if result.WinnerID != "" {
		fmt.Printf("   Winner:         %s\n", result.WinnerID)
		fmt.Printf("   Winner Found In: %v\n", result.WinnerTime)
	} else {
		fmt.Println("  No winner found.")
	}

	fmt.Printf("   Request Log:    %s\n", logPath)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Graceful server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
