package services

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"manifold-test/internal/models"
)

type UserService struct {
	db *sql.DB
}

type RequestService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

func NewRequestService(db *sql.DB) *RequestService {
	return &RequestService{db: db}
}

func (s *UserService) GetOrCreateUser(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	query := `SELECT user_id, words_left, total_words, created_at, updated_at FROM users WHERE user_id = ?`
	
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.UserID, &user.WordsLeft, &user.TotalWords, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		// Create new user with 1M words
		insertQuery := `INSERT INTO users (user_id, words_left, total_words) VALUES (?, ?, ?)`
		_, err = s.db.ExecContext(ctx, insertQuery, userID, 1000000, 1000000)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		
		user = models.User{
			UserID:     userID,
			WordsLeft:  1000000,
			TotalWords: 1000000,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return &user, nil
}

func (s *UserService) UpdateWordsLeft(ctx context.Context, userID string, wordsUsed int) error {
	query := `UPDATE users SET words_left = GREATEST(0, words_left - ?), updated_at = NOW() WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, query, wordsUsed, userID)
	if err != nil {
		return fmt.Errorf("failed to update words left: %w", err)
	}
	return nil
}

func (s *UserService) GetUserStats(ctx context.Context, userID string) (*models.UserStats, error) {
	var stats models.UserStats
	query := `SELECT user_id, words_left, total_words FROM users WHERE user_id = ?`
	
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&stats.UserID, &stats.WordsLeft, &stats.TotalWords)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	
	stats.WordsUsed = stats.TotalWords - stats.WordsLeft
	return &stats, nil
}

func (s *RequestService) SaveRequest(ctx context.Context, userID, data string, duration float64) error {
	query := `INSERT INTO requests (user_id, data, duration) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, userID, data, duration)
	if err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}
	return nil
}

// Generate random words for streaming
func GenerateRandomWords(count int) string {
	words := []string{
		"the", "be", "to", "of", "and", "a", "in", "that", "have", "I",
		"it", "for", "not", "on", "with", "he", "as", "you", "do", "at",
		"this", "but", "his", "by", "from", "they", "we", "say", "her", "she",
		"or", "an", "will", "my", "one", "all", "would", "there", "their", "what",
		"so", "up", "out", "if", "about", "who", "get", "which", "go", "me",
		"when", "make", "can", "like", "time", "no", "just", "him", "know", "take",
		"people", "into", "year", "your", "good", "some", "could", "them", "see", "other",
		"than", "then", "now", "look", "only", "come", "its", "over", "think", "also",
		"back", "after", "use", "two", "how", "our", "work", "first", "well", "way",
		"even", "new", "want", "because", "any", "these", "give", "day", "most", "us",
	}
	
	var result []string
	for i := 0; i < count; i++ {
		result = append(result, words[rand.Intn(len(words))])
	}
	
	return strings.Join(result, " ")
}

func init() {
	rand.Seed(time.Now().UnixNano())
} 