package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	RedisAddr  string
	ServerPort string
}

type Server struct {
	db          *sql.DB
	redis       *redis.Client
	config      *Config
	rateLimiter *RateLimiter
}

type Request struct {
	ID       int64     `json:"id"`
	UserID   string    `json:"user_id"`
	Data     string    `json:"data"`
	Duration int       `json:"duration"`
	Created  time.Time `json:"created"`
}

type User struct {
	UserID     string `json:"user_id"`
	WordsLeft  int    `json:"words_left"`
	TotalWords int    `json:"total_words"`
}

type RateLimiter struct {
	mu       sync.RWMutex
	counters map[string]*UserCounter
}

type UserCounter struct {
	count     int
	resetTime time.Time
	mu        sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		counters: make(map[string]*UserCounter),
	}
	
	
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	
	return rl
}

func (rl *RateLimiter) Allow(userID string, limit int) bool {
	rl.mu.RLock()
	counter, exists := rl.counters[userID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		counter = &UserCounter{
			count:     0,
			resetTime: time.Now().Add(time.Minute),
		}
		rl.counters[userID] = counter
		rl.mu.Unlock()
	}

	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()
	if now.After(counter.resetTime) {
		counter.count = 0
		counter.resetTime = now.Add(time.Minute)
	}

	if counter.count >= limit {
		return false
	}

	counter.count++
	return true
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for userID, counter := range rl.counters {
		counter.mu.RLock()
		if now.After(counter.resetTime.Add(time.Minute)) {
			delete(rl.counters, userID)
		}
		counter.mu.RUnlock()
	}
}

func loadConfig() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "manifold"),
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewServer(config *Config) (*Server, error) {
    
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&interpolateParams=true",
        config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    
    db.SetMaxOpenConns(200)  
    db.SetMaxIdleConns(50)   
    db.SetConnMaxLifetime(30 * time.Minute)  
    db.SetConnMaxIdleTime(5 * time.Minute)   


    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    
    rdb := redis.NewClient(&redis.Options{
        Addr:         config.RedisAddr,
        PoolSize:     100,  
        MinIdleConns: 20,   
        MaxRetries:   3,
        PoolTimeout:  4 * time.Second,
    })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	return &Server{
		db:          db,
		redis:       rdb,
		config:      config,
		rateLimiter: NewRateLimiter(),
	}, nil
}

func (s *Server) initDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			user_id VARCHAR(255) PRIMARY KEY,
			words_left INT NOT NULL DEFAULT 1000000,
			total_words INT NOT NULL DEFAULT 1000000,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_words_left (words_left)
		) ENGINE=InnoDB`,
		
		`CREATE TABLE IF NOT EXISTS requests (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			data TEXT,
			duration INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_user_id (user_id),
			INDEX idx_created_at (created_at),
			FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
		) ENGINE=InnoDB`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

func (s *Server) generateDataHandler(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()
    
    
    userID := r.Header.Get("X-User-Id")
    if userID == "" {
        http.Error(w, "X-User-Id header is required", http.StatusBadRequest)
        return
    }

    
    if !s.rateLimiter.Allow(userID, 100) {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    
    wordsLeft, err := s.getUserWordsLeft(userID)
    if err != nil {
        log.Printf("Error checking words left for user %s: %v", userID, err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if wordsLeft <= 0 {
        http.Error(w, "No words left", http.StatusForbidden)
        return
    }

    
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("X-Accel-Buffering", "no")
    w.Header().Set("Transfer-Encoding", "chunked")

    
    w.WriteHeader(http.StatusOK)

    
    ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
    defer cancel()

    
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
        return
    }

    
    words := []string{
        "artificial", "intelligence", "machine", "learning", "neural", "network",
        "algorithm", "data", "science", "computer", "vision", "natural", "language",
        "processing", "deep", "learning", "transformer", "model", "training",
        "inference", "prediction", "classification", "regression", "clustering",
    }

    var generatedData []string
    wordCount := 0
    maxWords := 100 + rand.Intn(400) 

    for {
        select {
        case <-ctx.Done():
            
            goto finish
        default:
            if wordCount >= maxWords {
                goto finish
            }

            word := words[rand.Intn(len(words))]
            generatedData = append(generatedData, word)
            
            
            if wordCount > 0 {
                fmt.Fprintf(w, " ") 
            }
            fmt.Fprintf(w, "%s", word)
            flusher.Flush() 

            wordCount++

            
            delay := time.Duration(500 + rand.Intn(500)) * time.Millisecond
            time.Sleep(delay)
        }
    }

finish:
    duration := int(time.Since(startTime).Seconds())
    
    
    go func() {
        
        s.saveRequest(userID, strings.Join(generatedData, " "), duration)
        
        
        s.decrementWordsByCount(userID, wordCount)
    }()
}


func (s *Server) getUserWordsLeft(userID string) (int, error) {
    
    cacheKey := fmt.Sprintf("user_words:%s", userID)
    
    ctx := context.Background()
    cachedWords, err := s.redis.Get(ctx, cacheKey).Int()
    if err == nil {
        return cachedWords, nil
    }

    
    var wordsLeft int
    err = s.db.QueryRow(`
        SELECT words_left FROM users WHERE user_id = ?`, userID).Scan(&wordsLeft)
    
    if err == sql.ErrNoRows {
        
        _, err = s.db.Exec(`
            INSERT INTO users (user_id, words_left, total_words) 
            VALUES (?, 1000000, 1000000)`, userID)
        if err != nil {
            return 0, err
        }
        return 1000000, nil
    } else if err != nil {
        return 0, err
    }

    
    s.redis.Set(ctx, cacheKey, wordsLeft, 5*time.Minute)
    return wordsLeft, nil
}


func (s *Server) decrementWordsByCount(userID string, wordCount int) error {
    tx, err := s.db.Begin()
    if err != nil {
        log.Printf("Error starting transaction for user %s: %v", userID, err)
        return err
    }
    defer tx.Rollback()

    
    var wordsLeft int
    err = tx.QueryRow(`
        SELECT words_left FROM users WHERE user_id = ? FOR UPDATE`, userID).Scan(&wordsLeft)
    if err != nil {
        log.Printf("Error getting words_left for user %s: %v", userID, err)
        return err
    }

    newWordsLeft := wordsLeft - wordCount  
    if newWordsLeft < 0 {
        newWordsLeft = 0
    }

    _, err = tx.Exec(`
        UPDATE users SET words_left = ?, updated_at = CURRENT_TIMESTAMP 
        WHERE user_id = ?`, newWordsLeft, userID)
    if err != nil {
        log.Printf("Error updating words_left for user %s: %v", userID, err)
        return err
    }

    if err = tx.Commit(); err != nil {
        log.Printf("Error committing transaction for user %s: %v", userID, err)
        return err
    }

    
    ctx := context.Background()
    cacheKey := fmt.Sprintf("user_words:%s", userID)
    s.redis.Set(ctx, cacheKey, newWordsLeft, 5*time.Minute)

    log.Printf("User %s: decremented %d words, %d words left", userID, wordCount, newWordsLeft)
    return nil
}

func (s *Server) saveRequest(userID, data string, duration int) {
	_, err := s.db.Exec(`
		INSERT INTO requests (user_id, data, duration) 
		VALUES (?, ?, ?)`, userID, data, duration)
	if err != nil {
		log.Printf("Error saving request for user %s: %v", userID, err)
	}
}

func (s *Server) getUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, "X-User-Id header is required", http.StatusBadRequest)
		return
	}

	var user User
	err := s.db.QueryRow(`
		SELECT user_id, words_left, total_words 
		FROM users WHERE user_id = ?`, userID).Scan(
		&user.UserID, &user.WordsLeft, &user.TotalWords)
	
	if err == sql.ErrNoRows {
		
		user = User{
			UserID:     userID,
			WordsLeft:  1000000,
			TotalWords: 1000000,
		}
		_, err = s.db.Exec(`
			INSERT INTO users (user_id, words_left, total_words) 
			VALUES (?, ?, ?)`, user.UserID, user.WordsLeft, user.TotalWords)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	
	if err := s.db.Ping(); err != nil {
		status["database"] = "unhealthy"
		status["status"] = "degraded"
	} else {
		status["database"] = "healthy"
	}

	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.redis.Ping(ctx).Err(); err != nil {
		status["redis"] = "unhealthy"
	} else {
		status["redis"] = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) setupRoutes() *mux.Router {
	r := mux.NewRouter()
	
	r.HandleFunc("/generate-data", s.generateDataHandler).Methods("POST")
	r.HandleFunc("/user/stats", s.getUserStatsHandler).Methods("GET")
	r.HandleFunc("/health", s.healthHandler).Methods("GET")
	
	return r
}

func main() {
	config := loadConfig()
	
	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.db.Close()
	defer server.redis.Close()

	if err := server.initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	router := server.setupRoutes()
	
	httpServer := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      router,
		ReadTimeout:  70 * time.Second, 
		WriteTimeout: 70 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on port %s", config.ServerPort)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	log.Println("Server stopped")
}