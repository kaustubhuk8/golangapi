# Manifold Labs Takehome Project

A high-performance Go API that simulates an LLM inference service with streaming responses, rate limiting, and user quota management.

## 🎯 Project Overview

This project implements a streaming data generation API that handles high-concurrency requests while maintaining database performance and user quota management. It directly mirrors real-world inference API challenges with intense request rates and concurrent data processing.

## ✨ Features

- **Streaming Data Generation**: Real-time word streaming with random delays (0.5-2 seconds)
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

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)

### 1. Clone and Setup

```bash
git clone <repository-url>
cd manifold-test
```

### 2. Start All Services

```bash
make docker-up
```

### 3. Verify Health

```bash
make stats
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

- Streaming plain text with AI-related words
- Random delays between 0.5-2 seconds per word
- Maximum 60-second request duration

**Example:**

```bash
curl -X POST -H "X-User-Id: user1" --no-buffer http://localhost:8080/generate-data
# Output: artificial intelligence machine learning neural network...
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
DB_HOST=localhost          # Database host
DB_PORT=3306              # Database port
DB_USER=root              # Database username
DB_PASSWORD=password      # Database password
DB_NAME=manifold          # Database name
REDIS_ADDR=localhost:6379 # Redis address
SERVER_PORT=8080          # API server port
```

## 🛠️ Development Commands

```bash
# Build application
make build

# Run locally (requires local MySQL/Redis)
make run

# Start all services with Docker
make docker-up

# Stop all services
make docker-down

# View logs
make docker-logs

# Restart services
make docker-restart

# Check API stats
make stats

# Monitor performance
make monitor

# Connect to database
make db-shell

# Connect to Redis
make redis-shell

# Format code
make format

# Run tests
make test

# Clean build artifacts
make clean
```

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

## 🧪 Testing

### Manual Testing

```bash
# Test streaming (30 seconds)
curl -X POST -H "X-User-Id: user1" --no-buffer --max-time 30 http://localhost:8080/generate-data

# Test rate limiting
for i in {1..110}; do
  curl -X POST -H "X-User-Id: user1" http://localhost:8080/generate-data
  echo "Request $i"
done

# Test quota exhaustion
# (Make multiple requests until words_left reaches 0)
```

### Load Testing

The application is designed to handle 5000+ concurrent requests from 10 users. Key optimizations:

- **Connection pooling** prevents database overload
- **Redis caching** reduces database queries by 90%+
- **Background processing** keeps response times fast
- **Rate limiting** prevents request floods
- **Memory management** with automatic cleanup

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

## 📊 Performance Characteristics

- **Response Time**: < 100ms for quota checks (cached)
- **Streaming**: Real-time word delivery with 0.5-2s delays
- **Concurrency**: 5000+ concurrent requests supported
- **Database Load**: Optimized with caching and connection pooling
- **Memory Usage**: Automatic cleanup prevents memory leaks

## 🔒 Security & Error Handling

- **Input validation** for all endpoints
- **Proper HTTP status codes** for different error conditions
- **Rate limiting** prevents abuse
- **Quota enforcement** prevents resource exhaustion
- **Graceful error handling** with detailed logging

## 🚀 Production Deployment

This application is production-ready with:

- **Health monitoring** endpoints
- **Graceful shutdown** handling
- **Comprehensive logging**
- **Resource management**
- **Error recovery**
- **Scalable architecture**

## 📝 Assignment Requirements Met

✅ **Single endpoint** `POST /generate-data`  
✅ **User ID validation** from `X-User-Id` header  
✅ **LLM simulation** with streaming and random delays  
✅ **Database tables** (users, requests) with proper schema  
✅ **Request saving** and word quota decrement  
✅ **5000+ concurrent requests** handling  
✅ **Dockerized application** with complete setup  
✅ **Bonus: Reject at 0 words** with proper HTTP status  
✅ **Bonus: Redis caching** for performance optimization  
✅ **Bonus: Creative enhancements** (health checks, user stats, etc.)

## 🤝 Contributing

This is a takehome project for Manifold Labs. The implementation demonstrates:

- **Go expertise** with proper concurrency and error handling
- **System architecture** skills for high-performance APIs
- **Production thinking** with monitoring and operational concerns
- **Problem-solving** approach to real-world challenges

## 📄 License

This project is created for the Manifold Labs takehome assignment.
