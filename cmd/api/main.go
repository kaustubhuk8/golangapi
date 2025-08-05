package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"manifold-test/internal/config"
	"manifold-test/internal/database"
	"manifold-test/internal/handlers"
	"manifold-test/internal/middleware/ratelimit"
	"manifold-test/internal/services"
)

func main() {
	
	// Load configuration
	cfg := config.Load()

	// Initialize database
	log.Printf("Connecting to database with DSN: %s", cfg.DSN)
	db, err := database.NewConnection(cfg.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := database.NewRedisConnection(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize services
	userService := services.NewUserService(db)
	requestService := services.NewRequestService(db)
	rateLimiter := ratelimit.NewRateLimiter()

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize handlers
	h := handlers.NewHandler(userService, requestService, rateLimiter, redisClient)

	// Routes
	e.GET("/health", h.HealthCheck)
	e.POST("/generate-data", h.GenerateData)
	e.GET("/user/stats", h.GetUserStats)

	// Start server
	go func() {
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
} 