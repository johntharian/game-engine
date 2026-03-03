package models

import "time"

// UserResponse represents a single user's answer submission.
type UserResponse struct {
	// UserID is the unique identifier for the user (e.g., "user-42").
	UserID string `json:"user_id"`
	// Answer is the user's submitted answer string.
	Answer string `json:"answer"`
	// IsCorrect indicates whether the submitted answer is correct.
	IsCorrect bool `json:"is_correct"`
	// Timestamp records when the user generated the response.
	Timestamp time.Time `json:"timestamp"`
}

// Result represents the final outcome of the game.
type Result struct {
	// WinnerID is the user ID of the first correct responder, empty if no winner.
	WinnerID string
	// WinnerTime is the elapsed duration from engine start to winner declaration.
	WinnerTime time.Duration
	// TotalCorrect is the count of correct answers received.
	TotalCorrect int64
	// TotalIncorrect is the count of incorrect answers received.
	TotalIncorrect int64
	// TotalReceived is the total number of responses processed by the engine.
	TotalReceived int64
}
