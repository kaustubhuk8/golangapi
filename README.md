# Manifold API — Take-Home Project

A Go API that simulates a high-concurrency LLM streaming service with per-user quotas, rate limiting, and lightweight monitoring.

## Overview

The service streams random words with small delays to mimic an LLM response. It tracks each user's remaining word quota, rejects requests when quotas are exceeded, and enforces basic per-user rate limits.  
Prometheus metrics and a simple Grafana dashboard are included for quick operational insight.

---

## Live EC2 Deployment

The service is deployed at:

```
http://3.138.235.69:8080
```

### Start Streaming

```bash
curl -X POST -H "X-User-Id: test_user" --no-buffer http://3.138.235.69:8080/generate-data
```

### With Deterministic Output

```bash
curl -X POST -H "X-User-Id: test_user" -H "X-Seed: 42" --no-buffer http://3.138.235.69:8080/generate-data
```


### With Stop Token

```bash
curl -X POST -H "X-User-Id: test_user" -H "X-Seed: 42" -H "X-Stop-Token: by" --no-buffer http://3.138.235.69:8080/generate-data
```

### User Quota Stats

```bash
curl -H "X-User-Id: test_user" http://3.138.235.69:8080/user/stats
```

### Health Check

```bash
curl http://3.138.235.69:8080/health
```

### Metrics (Prometheus format)

```bash
curl http://3.138.235.69:8080/metrics
```

---

## Running Locally (If EC2 is Unavailable)

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (only if building locally)

### Start All Services

```bash
make docker-up
```

### Fresh Start (wipe DB & Redis, rebuild images)

```bash
make fresh-start
```

### Stop Services

```bash
make docker-down
```

Once started locally, the services will be available at:

- **API** → http://localhost:8080
- **Prometheus** → http://localhost:9090
- **Grafana** → http://localhost:3000 (username: `admin`, password: `admin`)

---

## Monitoring

Grafana dashboards are preloaded.  
To view the main dashboard:

1. Open **http://localhost:3000/dashboards**
2. Log in with **username:** `admin` and **password:** `admin`
3. Select **Manifold Demo**

This dashboard shows:

- Request volume and concurrency
- Latency percentiles (p50 / p95 / p99)
- Words generated over time
- DB write times
- Rate-limit rejections

---

## Load Testing (You will need to clone the repo for this to work)

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
