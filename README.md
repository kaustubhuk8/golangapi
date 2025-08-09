# Manifold API — Take-Home Project

A Go API that simulates a high-concurrency LLM streaming service with per-user quotas, rate limiting, and lightweight monitoring.

## Overview

The service streams random words with small delays to mimic an LLM response. It tracks each user's remaining word quota, rejects requests when quotas are exceeded, and enforces basic per-user rate limits.  
Prometheus metrics and a simple Grafana dashboard are included for quick operational insight.

---

## Prerequisites

- Docker & Docker Compose
- Go 1.21+ (only if building locally)

---

## Running the Service

**Start everything** (API, MySQL, Redis, Prometheus, Grafana):

```bash
make docker-up
```

**Fresh start** (wipe DB & Redis, rebuild images):

```bash
make fresh-start
```

**Stop services:**

```bash
make docker-down
```

---

## API Endpoints

### Start Streaming

```bash
curl -X POST -H "X-User-Id: test_user" --no-buffer \
  http://localhost:8080/generate-data
```

### With Stop Token

```bash
curl -X POST -H "X-User-Id: test_user" -H "X-Stop-Token: day" --no-buffer \
  http://localhost:8080/generate-data
```

### With Deterministic Output

```bash
curl -X POST -H "X-User-Id: test_user" -H "X-Seed: 42" --no-buffer \
  http://localhost:8080/generate-data
```

### User Quota Stats

```bash
curl -H "X-User-Id: test_user" http://localhost:8080/user/stats
```

### Health Check

```bash
curl http://localhost:8080/health
```

### Metrics (Prometheus format)

```bash
curl http://localhost:8080/metrics
```

---

## Monitoring

- **Grafana** → http://localhost:3000 (admin/admin)
- **Prometheus** → http://localhost:9090
- **API** → http://localhost:8080

The default dashboard shows:

- Request volume and concurrency
- Latency percentiles (p50 / p95 / p99)
- Words generated over time
- DB write times
- Rate-limit rejections

---

## Load Testing

**Quick load test** (50 requests, 10 workers):

```bash
make load-test-quick
```

**Full load test** (5000 requests, 100 workers):

```bash
make load-test-full
```

---

## Tech Stack

- **Go 1.21** with Echo framework
- **MySQL 8** for persistent storage
- **Redis** for caching and session management
- **Prometheus** for metrics collection
- **Grafana** for monitoring dashboards
- **Docker Compose** for orchestration
