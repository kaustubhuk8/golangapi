# Manifold Labs — Takehome Project

A high-concurrency Go API that simulates a real-time LLM inference service with streaming responses, per-user rate limits, and word quota enforcement.

## Overview

This service streams random words with natural delays to simulate LLM output. It handles 1000s of concurrent requests while tracking user quota and enforcing request limits. Designed for production-style load testing with Docker, Redis, and MySQL integration.

## Features

- **Streaming Words**: Random word stream (0.5–1s delay per word)
- **Stop Token**: Ends the stream early if a generated word matches a given token
- **User Quota**: 1,000,000-word allowance per user (stored in MySQL, cached in Redis)
- **Rate Limiting**: 100 requests/minute per user (in-memory sliding window)
- **Concurrent Safe**: Context handling, goroutines, background DB writes
- **Dockerized**: One-step orchestration with Redis and MySQL
- **Optional Seeded Output**: Predictable word stream via `X-Seed` header

## API Endpoints

#### `POST /generate-data`

- Headers:

  - `X-User-Id`: (required)
  - `X-Stop-Token`: (optional)
  - `X-Seed`: (optional) deterministic stream if provided

- Behavior:
  - Streams English words line by line (up to 60s)
  - Stops if `stop-token` is encountered

#### `GET /user/stats`

Returns current word quota for the user.

#### `GET /health`

Liveness check for API, DB, and Redis.

## Setup

### Prerequisites

- Docker + Docker Compose
- Go 1.21+ (only if building locally)

### Quick Start

```bash
make fresh-start
```

Starts a clean environment: rebuilds images, starts services, clears Redis, and resets DB state.

To just run the app:

```bash
make docker-up
```

### Example Requests

```bash
# Start streaming
curl -X POST -H "X-User-Id: test_user" --no-buffer http://localhost:8080/generate-data

# With stop token
curl -X POST -H "X-User-Id: test_user" -H "X-Stop-Token: day" --no-buffer http://localhost:8080/generate-data

# With deterministic seed
curl -X POST -H "X-User-Id: test_user" -H "X-Seed: 42" --no-buffer http://localhost:8080/generate-data
```

## Load Testing

- `make load-test-quick` — 50 requests / 10 workers (~5 mins)
- `make load-test-full` — 5000 requests / 100 workers (~60–90 mins)

## Development

```bash
make help          # List commands
make dev           # Run with hot reload (requires 'air')
make docker-logs   # View logs
make stats         # View user + health stats
make monitor       # Stream system status
```

## Tech Stack

- **Go 1.21**
- **MySQL 8** (Docker, port `3307`)
- **Redis** (Docker, port `6380`)
- **Docker Compose** for orchestration
- **Makefile** for repeatable commands

## Schema Summary

```sql
CREATE TABLE users (
  user_id VARCHAR(255) PRIMARY KEY,
  words_left INT DEFAULT 1000000,
  total_words INT DEFAULT 1000000,
  ...
);

CREATE TABLE requests (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id VARCHAR(255),
  data TEXT,
  duration INT,
  ...
);
```
