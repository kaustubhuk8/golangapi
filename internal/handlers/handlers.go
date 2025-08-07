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
	
	// Check database
	dbStatus := "healthy"
	if _, err := h.userService.GetUserStats(ctx, "test"); err != nil {
		dbStatus = "unhealthy"
	}
	
	// Check Redis
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
	
	// Get user ID from header
	userID := c.Request().Header.Get("X-User-Id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-User-Id header is required")
	}
	
	// Get stop token from header (optional)
	stopToken := c.Request().Header.Get("X-Stop-Token")
	seedStr := c.Request().Header.Get("X-Seed")
	var streamRand *rand.Rand

	if seedStr != "" {
		// Deterministic mode
		seed := int64(0)
		fmt.Sscanf(seedStr, "%d", &seed)
		streamRand = rand.New(rand.NewSource(seed))
	} else {
		// Random mode
		streamRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	
	// Check rate limit
	if !h.rateLimiter.IsAllowed(userID) {
		return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
	}
	
	// Get or create user
	user, err := h.userService.GetOrCreateUser(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user")
	}
	
	// Check if user has words left
	if user.WordsLeft <= 0 {
		return echo.NewHTTPError(http.StatusForbidden, "No words left")
	}
	
	// Set up streaming response
	c.Response().Header().Set("Content-Type", "text/plain")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	
	// Start streaming
	startTime := time.Now()
	var generatedData strings.Builder
	wordsGenerated := 0
	
	// Stream for up to 1 minute
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	
			for {
			select {
			case <-ctx.Done():
				goto end
			default:
				// Check if we have words left
				if wordsGenerated >= user.WordsLeft {
					goto end
				}
				
				// Generate single random word with optional stop token support
				word, stopTokenFound := services.GenerateRandomWords(streamRand,1, stopToken)
				
				// Check if stop token was generated
				if stopTokenFound {
					// Send the stop token and end
					if _, err := fmt.Fprintf(c.Response().Writer, "%s ", word); err != nil {
						return err
					}
					
					// Flush immediately for streaming
					if flusher, ok := c.Response().Writer.(http.Flusher); ok {
						flusher.Flush()
					}
					
					generatedData.WriteString(word)
					generatedData.WriteString(" ")
					wordsGenerated += 1
					goto end
				}
				
				// Send single word
				if _, err := fmt.Fprintf(c.Response().Writer, "%s ", word); err != nil {
					return err
				}
				
				// Flush immediately for streaming
				if flusher, ok := c.Response().Writer.(http.Flusher); ok {
					flusher.Flush()
				}
				
				generatedData.WriteString(word)
				generatedData.WriteString(" ")
				wordsGenerated += 1
				
				// Random delay (0.5-1 second)
				delay := time.Duration(rand.Intn(500)+500) * time.Millisecond
				time.Sleep(delay)
			}
		}
	
end:
	duration := time.Since(startTime).Seconds()
	
	// Save request to database with a separate context
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dbCancel()
	
	if err := h.requestService.SaveRequest(dbCtx, userID, generatedData.String(), duration); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to save request: %v\n", err)
	}
	
	// Update user's word count
	if err := h.userService.UpdateWordsLeft(dbCtx, userID, wordsGenerated); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to update words left: %v\n", err)
	} else {
		// Invalidate Redis cache when word count changes
		cacheKey := "user_stats:" + userID
		if err := h.redisClient.Del(dbCtx, cacheKey).Err(); err != nil {
			fmt.Printf("Failed to invalidate cache: %v\n", err)
		} else {
			fmt.Printf("Updated words left for user %s: -%d words (cache invalidated)\n", userID, wordsGenerated)
		}
	}
	
	return nil
}

func (h *Handler) GetUserStats(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Get user ID from header
	userID := c.Request().Header.Get("X-User-Id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-User-Id header is required")
	}
	
	// Try to get from cache first
	cacheKey := "user_stats:" + userID
	cachedStats, err := h.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Return cached stats
		return c.String(http.StatusOK, cachedStats)
	}
	
	// Get from database
	stats, err := h.userService.GetUserStats(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user stats")
	}
	
	// Cache for 5 minutes
	statsJSON := fmt.Sprintf(`{"user_id":"%s","words_left":%d,"total_words":%d,"words_used":%d}`,
		stats.UserID, stats.WordsLeft, stats.TotalWords, stats.WordsUsed)
	
	h.redisClient.Set(ctx, cacheKey, statsJSON, 5*time.Minute)
	
	return c.JSON(http.StatusOK, stats)
} 