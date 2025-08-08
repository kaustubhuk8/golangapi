package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	appmetrics "manifold-test/internal/metrics"
	"manifold-test/internal/middleware/ratelimit"
	"manifold-test/internal/models"
	"manifold-test/internal/services"
)

type Handler struct {
	userService    *services.UserService
	requestService *services.RequestService
	rateLimiter    *ratelimit.RateLimiter
	redisClient    *redis.Client
}

func NewHandler(
	userService *services.UserService,
	requestService *services.RequestService,
	rateLimiter *ratelimit.RateLimiter,
	redisClient *redis.Client,
) *Handler {
	return &Handler{
		userService:    userService,
		requestService: requestService,
		rateLimiter:    rateLimiter,
		redisClient:    redisClient,
	}
}

func (h *Handler) HealthCheck(c echo.Context) error {
	ctx := c.Request().Context()

	// Database check
	dbStatus := "healthy"
	if _, err := h.userService.GetUserStats(ctx, "healthcheck-probe"); err != nil {
		dbStatus = "unhealthy"
	}

	// Redis check
	redisStatus := "healthy"
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "unhealthy"
	}

	response := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Database:  dbStatus,
		Redis:     redisStatus,
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) GenerateData(c echo.Context) error {
	ctx := c.Request().Context()

	// Metrics: count + in-flight
	appmetrics.RequestsTotal.Inc()
	appmetrics.ActiveRequests.Inc()
	defer appmetrics.ActiveRequests.Dec()

	startWall := time.Now()
	wordsGenerated := 0
	defer func() {
		appmetrics.RequestDurationSeconds.Observe(time.Since(startWall).Seconds())
		// Add once at the end to avoid hot counters on tight loops
		appmetrics.WordsGeneratedTotal.Add(float64(wordsGenerated))
	}()

	// Get user ID
	userID := c.Request().Header.Get("X-User-Id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-User-Id header is required")
	}

	// Optional controls
	stopToken := c.Request().Header.Get("X-Stop-Token")
	seedStr := c.Request().Header.Get("X-Seed")
	var streamRand *rand.Rand
	if seedStr != "" {
		var seed int64
		fmt.Sscanf(seedStr, "%d", &seed)
		streamRand = rand.New(rand.NewSource(seed))
	} else {
		streamRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	maxTokens := -1
	if maxTokenStr := c.Request().Header.Get("X-Max-Tokens"); maxTokenStr != "" {
		fmt.Sscanf(maxTokenStr, "%d", &maxTokens)
	}

	// Rate limit
	if !h.rateLimiter.IsAllowed(userID) {
		appmetrics.RateLimitDroppedTotal.Inc()
		return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
	}

	// Get or create user + quota
	user, err := h.userService.GetOrCreateUser(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user")
	}
	if user.WordsLeft <= 0 {
		return echo.NewHTTPError(http.StatusForbidden, "No words left")
	}

	// Streaming response headers
	c.Response().Header().Set("Content-Type", "text/plain")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	// Stream for up to 1 minute
	streamCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var generatedData strings.Builder

	for {
		select {
		case <-streamCtx.Done():
			// timeout or client cancel — we still persist what we have
			goto end
		default:
			// Early stops
			if (maxTokens != -1 && wordsGenerated >= maxTokens) || wordsGenerated >= user.WordsLeft {
				goto end
			}

			word, stopTokenFound := services.GenerateRandomWords(streamRand, 1, stopToken)

			if _, err := fmt.Fprintf(c.Response().Writer, "%s ", word); err != nil {
				return err
			}
			if flusher, ok := c.Response().Writer.(http.Flusher); ok {
				flusher.Flush()
			}

			generatedData.WriteString(word + " ")
			wordsGenerated++

			if stopTokenFound {
				goto end
			}

			// 500–1000ms delay
			time.Sleep(time.Duration(rand.Intn(500)+500) * time.Millisecond)
		}
	}

end:
	// Persist request with measured duration
	duration := time.Since(startWall).Seconds()
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dbCancel()

	dbStart := time.Now()
	if err := h.requestService.SaveRequest(dbCtx, userID, generatedData.String(), duration); err != nil {
		// Observe duration even on failure to reveal slow/failing path
		appmetrics.DBWriteDurationSeconds.Observe(time.Since(dbStart).Seconds())
	} else {
		appmetrics.DBWriteDurationSeconds.Observe(time.Since(dbStart).Seconds())
	}

	// Update user's word count; invalidate cache (best-effort)
	if err := h.userService.UpdateWordsLeft(dbCtx, userID, wordsGenerated); err == nil {
		_ = h.redisClient.Del(dbCtx, "user_stats:"+userID).Err()
	}

	return nil
}

func (h *Handler) GetUserStats(c echo.Context) error {
	ctx := c.Request().Context()

	userID := c.Request().Header.Get("X-User-Id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-User-Id header is required")
	}

	// Try Redis cache first
	cacheKey := "user_stats:" + userID
	if cached, err := h.redisClient.Get(ctx, cacheKey).Result(); err == nil {
		return c.String(http.StatusOK, cached)
	}

	// Fallback to DB
	stats, err := h.userService.GetUserStats(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user stats")
	}

	// Cache for 5 minutes (best-effort)
	statsJSON := fmt.Sprintf(`{"user_id":"%s","words_left":%d,"total_words":%d,"words_used":%d}`,
		stats.UserID, stats.WordsLeft, stats.TotalWords, stats.TotalWords-stats.WordsLeft)
	_ = h.redisClient.Set(ctx, cacheKey, statsJSON, 5*time.Minute).Err()

	return c.JSON(http.StatusOK, stats)
}
