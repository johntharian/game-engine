package engine

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"game-engine/models"
)

// responseBufferSize is the capacity of the ResponseChan buffer.
// It absorbs burst traffic from concurrent API handlers while the engine
// processes responses one-by-one. 1024 provides sufficient headroom
// without over-allocating memory.
const responseBufferSize = 1024

// GameEngine evaluates user responses and determines the winner.
// It uses a channel-driven design for real-time, event-driven processing.
type GameEngine struct {
	// ResponseChan is the buffered channel through which the API server
	// sends incoming user responses for real-time evaluation.
	ResponseChan chan models.UserResponse

	// startTime records when the engine began accepting responses,
	// used to calculate the time-to-winner duration.
	startTime time.Time
	// winnerID stores the user ID of the first correct responder.
	winnerID string
	// winnerTime stores the elapsed duration from engine start to winner declaration.
	winnerTime time.Duration
	// winnerFound is an atomic flag set to true once a winner is declared,
	// allowing the API server to check winner status without blocking.
	winnerFound atomic.Bool
	// totalCorrect tracks the number of correct answers received (atomic, lock-free).
	totalCorrect atomic.Int64
	// totalIncorrect tracks the number of incorrect answers received (atomic, lock-free).
	totalIncorrect atomic.Int64
	// totalReceived tracks the total number of responses processed (atomic, lock-free).
	totalReceived atomic.Int64

	// once ensures the winner declaration logic executes exactly once,
	// even under concurrent access.
	once sync.Once
	// done is closed when a winner is found or all responses are processed,
	// unblocking any goroutine waiting on WaitForWinner().
	done chan struct{}
}

// New creates and initialises a new GameEngine.
//
// It allocates a buffered ResponseChan (capacity 1024) for receiving
// user responses and a done channel used to signal winner detection.
//
// Parameters:
//   - (none)
//
// Returns:
//   - *GameEngine: a ready-to-use engine instance. Call Start() in a
//     goroutine to begin the evaluation loop.
func New() *GameEngine {
	return &GameEngine{
		ResponseChan: make(chan models.UserResponse, responseBufferSize),
		done:         make(chan struct{}),
	}
}

// Start begins the evaluation loop that processes incoming user responses.
//
// It reads from ResponseChan until the channel is closed, evaluating each
// response in real-time. When all responses are processed and no correct
// answer was received, it signals completion via the done channel.
// This method should be called in a separate goroutine.
//
// Parameters:
//   - (none) — operates on the GameEngine receiver.
//
// Returns:
//   - (none) — blocks until ResponseChan is closed.
func (g *GameEngine) Start() {
	g.startTime = time.Now()
	fmt.Println("Game Engine started. Waiting for responses...")

	for resp := range g.ResponseChan {
		g.totalReceived.Add(1)
		g.evaluate(resp)
	}

	// Channel closed — all responses processed.
	// If no winner was found (all answers were wrong), signal done anyway.
	g.once.Do(func() {
		fmt.Println("No correct answers received. No winner.")
		close(g.done)
	})
}

// evaluate processes a single user response and determines if it is the winner.
//
// If the response is correct and no winner has been declared yet, the user is
// declared the winner via sync.Once. Subsequent correct answers are counted
// but ignored for winner selection. Incorrect answers increment the incorrect
// counter.
//
// Parameters:
//   - resp (models.UserResponse): the user response to evaluate.
//
// Returns:
//   - (none)
func (g *GameEngine) evaluate(resp models.UserResponse) {
	if resp.IsCorrect {
		g.totalCorrect.Add(1)

		// Only the very first correct answer wins.
		g.once.Do(func() {
			g.winnerID = resp.UserID
			g.winnerTime = time.Since(g.startTime)
			g.winnerFound.Store(true)

			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Printf("	WINNER: %s\n", g.winnerID)
			fmt.Printf("	Time to find winner: %v\n", g.winnerTime)
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			close(g.done)
		})
	} else {
		g.totalIncorrect.Add(1)
	}
}

// HasWinner checks whether a winner has already been declared.
//
// This is a lock-free atomic check safe for concurrent use from the
// API server's HTTP handler goroutines.
//
// Parameters:
//   - (none) — operates on the GameEngine receiver.
//
// Returns:
//   - bool: true if a winner has been found, false otherwise.
func (g *GameEngine) HasWinner() bool {
	return g.winnerFound.Load()
}

// WaitForWinner blocks the calling goroutine until a winner is declared or
// all responses have been processed without finding a correct answer.
//
// Parameters:
//   - (none) — operates on the GameEngine receiver.
//
// Returns:
//   - (none) — blocks until the internal done channel is closed.
func (g *GameEngine) WaitForWinner() {
	<-g.done
}

// GetResult returns the final game result with winner details and metrics.
//
// This method should only be called after WaitForWinner has returned to
// ensure the winner fields are fully populated.
//
// Parameters:
//   - (none) — operates on the GameEngine receiver.
//
// Returns:
//   - models.Result: contains WinnerID, WinnerTime, TotalCorrect,
//     TotalIncorrect, and TotalReceived.
func (g *GameEngine) GetResult() models.Result {
	return models.Result{
		WinnerID:       g.winnerID,
		WinnerTime:     g.winnerTime,
		TotalCorrect:   g.totalCorrect.Load(),
		TotalIncorrect: g.totalIncorrect.Load(),
		TotalReceived:  g.totalReceived.Load(),
	}
}
