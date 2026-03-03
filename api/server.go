package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"game-engine/engine"
	"game-engine/models"
)

// Server is the HTTP API server that receives user responses.
type Server struct {
	httpServer *http.Server
	engine     *engine.GameEngine
	port       int
}

// New creates and configures a new API server with its HTTP routes.
//
// It registers the POST /submit endpoint and binds the server to the
// specified port. The server is not started until Start() is called.
//
// Parameters:
//   - port (int): the TCP port number to listen on.
//   - eng (*engine.GameEngine): the game engine instance to forward
//     received responses to via its ResponseChan.
//
// Returns:
//   - *Server: a configured server instance ready to be started.
func New(port int, eng *engine.GameEngine) *Server {
	s := &Server{
		engine: eng,
		port:   port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /submit", s.handleSubmit)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return s
}

// Start binds the server to its configured port and begins accepting
// HTTP connections in a background goroutine.
//
// It blocks until the TCP listener is successfully created, then returns.
// If the listener cannot be created (e.g., port already in use), an error
// is returned immediately.
//
// Parameters:
//   - (none) — operates on the Server receiver.
//
// Returns:
//   - error: non-nil if the TCP listener could not be created.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.httpServer.Addr, err)
	}

	fmt.Printf("API Server listening on http://localhost:%d\n", s.port)

	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the API server, waiting for in-flight
// requests to complete or until the context deadline is reached.
//
// Parameters:
//   - ctx (context.Context): a context with a deadline/timeout controlling
//     how long to wait for active connections to finish.
//
// Returns:
//   - error: non-nil if the shutdown did not complete cleanly.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// handleSubmit processes a POST /submit request containing a user's
// game response in JSON format.
//
// It decodes the request body into a UserResponse and forwards it to the
// game engine's ResponseChan for metrics tracking. If a winner has already
// been declared, it responds with 200 and a game-over status instead of
// forwarding for evaluation. Returns 400 Bad Request if the JSON is malformed.
//
// Parameters:
//   - w (http.ResponseWriter): the HTTP response writer.
//   - r (*http.Request): the incoming HTTP request with JSON body.
//
// Returns:
//   - (none) — writes the HTTP response directly to w.
func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var resp models.UserResponse

	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// Always forward for metrics tracking.
	s.engine.ResponseChan <- resp

	// If a winner was already found, inform the client.
	if s.engine.HasWinner() {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"game_over","correct":%t,"message":"winner already declared"}`, resp.IsCorrect)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, `{"status":"accepted","correct":%t}`, resp.IsCorrect)
}
