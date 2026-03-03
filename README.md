# Game Engine with User Simulator

A Go backend system that simulates multiple users answering a game question, evaluates responses in real-time, and announces a winner.

## Architecture

```
┌──────────────┐     HTTP POST      ┌──────────────┐    channel     ┌──────────────┐
│  Mock User   │ ──── /submit ────▶ │  API Server  │ ────────────▶  │ Game Engine  │
│   Engine     │    (concurrent)    │  (net/http)  │                │ (evaluator)  │
│ simulator/   │                    │    api/      │                │   engine/    │
└──────────────┘                    └──────────────┘                └──────┬───────┘
                                                                           │
                                                                        Winner
                                                                        Metrics
```

## Components

| Component            | Package          | Description                                               |
|----------------------|------------------|-----------------------------------------------------------|
| **Models**           | `models/`        | Shared types (`UserResponse`, `Result`)                   |
| **Game Engine**      | `engine/`        | Channel-driven evaluator; declares winner via `sync.Once` |
| **API Server**       | `api/`           | HTTP server with `POST /submit` endpoint                  |
| **Mock User Engine** | `simulator/`     | Spawns N goroutines with random delays (10–1000ms)        |

## Concurrency Design

- **Channel**: Buffered channel (1024) passes responses from API → Engine  
- **sync.Once**: Guarantees exactly one winner is declared  
- **sync/atomic**: Lock-free metrics counters (no mutexes in hot path)  
- **sync.WaitGroup**: Waits for all simulated users to finish  
- **Done channel**: Signals winner found  

## Getting Started

### Prerequisites

- Go 1.22+ installed

### Clone

```bash
git clone https://github.com/johntharian/game-engine.git
cd game-engine
```

### Run

```bash
# Default: 1000 users, port 8080
go run main.go

# Custom users and port
go run main.go -n 5000 -port 9090

# With race detector
go run -race main.go -n 1000
```

### Flags

| Flag   | Default | Description |
|--------|---------|-------------|
| `-n`   | 1000    | Number of simulated users |
| `-port`| 8080    | API server port |

