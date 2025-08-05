# Manifold Labs Takehome Project

A high-performance Go API that simulates an LLM inference service with streaming responses, rate limiting, and user quota management.

## 🎯 Project Overview

This project implements a streaming data generation API that handles high-concurrency requests while maintaining database performance and user quota management. It directly mirrors real-world inference API challenges with intense request rates and concurrent data processing.

## ✨ Features

- **Streaming Data Generation**: Real-time word streaming with random delays (0.5-1 seconds)
- **Rate Limiting**: 100 requests per minute per user with automatic cleanup
- **User Quota Management**: 1M words per user with automatic decrement
- **High Concurrency**: Handles 5000+ concurrent requests from multiple users
- **Database Optimization**: Connection pooling, Redis caching, and background processing
- **Production Ready**: Health checks, graceful shutdown, and comprehensive error handling
- **Dockerized**: Complete containerized setup with MySQL and Redis

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │───▶│   Go API        │───▶│   MySQL DB      │
│                 │    │   (Port 8080)   │    │   (Port 3307)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   Redis Cache   │
                       │   (Port 6380)   │
                       └─────────────────┘
```

## 📁 Project Structure

```
manifold-test/
├── cmd/
│   ├── api/              # Main API server
│   └── load_test/        # Load testing tool
├── internal/
│   ├── config/           # Configuration management
│   ├── database/         # Database connections
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # Rate limiting middleware
│   ├── models/           # Data structures
│   └── services/         # Business logic
├── bin/                  # Compiled binaries
├── docker-compose.yml    # Service orchestration
├── Dockerfile           # Application container
├── Makefile             # Build and deployment commands
├── init.sql             # Database initialization
└── README.md            # This file
```

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)

### 1. Clone and Setup

```bash
git clone <repository-url>
cd manifold-test
```

### 2. Complete Fresh Start (Recommended)

```bash
make fresh-start
```

This command will:

- Stop all existing services
- Clean up Docker resources
- Build fresh application
- Start all services
- Verify everything is working
- Clear Redis cache

### 3. Alternative: Quick Start

```bash
make quick-start
```

### 4. Test the API

```bash
# Test streaming endpoint
curl -X POST -H "X-User-Id: user1" --no-buffer http://localhost:8080/generate-data

# Check user stats
curl -H "X-User-Id: user1" http://localhost:8080/user/stats

# Health check
curl http://localhost:8080/health
```

## 📋 API Endpoints

### POST /generate-data

Generates streaming text data with random delays.

**Headers:**

- `X-User-Id` (required): User identifier

**Response:**

- Streaming plain text with english words
- Random delays between 0.5-1 seconds per word
- Maximum 60-second request duration

**Example:**

```bash
curl -X POST -H "X-User-Id: user1" --no-buffer http://localhost:8080/generate-data
# Output: out your want for any people...
```

### GET /user/stats

Returns user's current word quota and usage.

**Headers:**

- `X-User-Id` (required): User identifier

**Response:**

```json
{
  "user_id": "user1",
  "words_left": 999500,
  "total_words": 1000000
}
```

### GET /health

Returns service health status.

**Response:**

```json
{
  "status": "healthy",
  "database": "healthy",
  "redis": "healthy",
  "timestamp": "2025-07-31T22:04:49Z"
}
```

## 🗄️ Database Schema

### Users Table

```sql
CREATE TABLE users (
    user_id VARCHAR(255) PRIMARY KEY,
    words_left INT NOT NULL DEFAULT 1000000,
    total_words INT NOT NULL DEFAULT 1000000,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_words_left (words_left)
) ENGINE=InnoDB;
```

### Requests Table

```sql
CREATE TABLE requests (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    data TEXT,
    duration INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
) ENGINE=InnoDB;
```

## ⚙️ Configuration

Environment variables (with defaults):

```bash
DSN=manifold:manifoldpassword@tcp(localhost:3306)/manifold?parseTime=true
REDIS_URL=redis://localhost:6379
```

The application uses:

- **DSN**: MySQL connection string with username, password, host, port, database, and parseTime option
- **REDIS_URL**: Redis connection URL

These are automatically configured in the Docker environment via `docker-compose.yml`.

## 🔧 Technical Implementation

### Rate Limiting

- **In-memory rate limiter** with automatic cleanup
- **100 requests per minute** per user
- **Sliding window** implementation
- **Background cleanup** every minute

### Caching Strategy

- **Redis caching** for user word quotas
- **5-minute cache expiration**
- **Database fallback** for cache misses
- **Cache invalidation** on quota updates

### Database Optimization

- **Connection pooling** (200 max connections)
- **Background processing** for database writes
- **Database transactions** for quota updates
- **Proper indexing** for performance

### Concurrency Handling

- **Goroutines** for background processing
- **Mutexes** for thread-safe operations
- **Context cancellation** for timeouts
- **Graceful shutdown** handling

## 🐳 Docker Setup

### Services

- **app**: Go API server (Port 8080)
- **mysql**: MySQL database (Port 3307)
- **redis**: Redis cache (Port 6380)

### Volumes

- `mysql_data`: Persistent MySQL data
- `redis_data`: Persistent Redis data

### Health Checks

All services include health checks to ensure proper startup order.

## 🔒 Security & Error Handling

- **Input validation** for all endpoints
- **Proper HTTP status codes** for different error conditions
- **Rate limiting** prevents abuse
- **Quota enforcement** prevents resource exhaustion
- **Graceful error handling** with detailed logging

## 🧪 Load Testing

### Quick Test (5 minutes)

```bash
make load-test-quick
```

Tests 50 requests with 10 concurrent workers.

### Full Assessment Test (60-90 minutes)

```bash
make load-test-full
```

Tests 5000 requests with 100 concurrent workers.

### Performance Criteria

- **Success Rate**: >95% requests successful
- **Throughput**: >50 requests/second
- **Response Time**: Average <30 seconds

### Manual Testing

```bash
# Test streaming endpoint
curl -X POST -H "X-User-Id: user1" --no-buffer http://localhost:8080/generate-data

# Check user stats
curl -H "X-User-Id: user1" http://localhost:8080/user/stats

# Monitor system health
make stats
```

## 🛠️ Development

### Available Make Commands

```bash
make help                    # Show all available commands
make fresh-start            # Complete reset and fresh start
make quick-start            # Quick start everything
make docker-up              # Start all services
make docker-down            # Stop all services
make docker-logs            # View service logs
make stats                  # Show API stats
make monitor                # Monitor API performance
make load-test-quick        # Quick load test (5 minutes)
make load-test-full         # Full load test (60-90 minutes)
make dev                    # Run with hot reload (requires air)
```

### Development Workflow

1. **Fresh Start**: `make fresh-start`
2. **Quick Test**: `make load-test-quick`
3. **Full Test**: `make load-test-full`
4. **Monitor**: `make stats` or `make monitor`
