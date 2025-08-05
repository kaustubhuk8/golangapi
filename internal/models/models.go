package models

import (
	"time"
)

type User struct {
	UserID     string    `json:"user_id" db:"user_id"`
	WordsLeft  int       `json:"words_left" db:"words_left"`
	TotalWords int       `json:"total_words" db:"total_words"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type Request struct {
	ID        int       `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Data      string    `json:"data" db:"data"`
	Duration  float64   `json:"duration" db:"duration"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type UserStats struct {
	UserID     string `json:"user_id"`
	WordsLeft  int    `json:"words_left"`
	TotalWords int    `json:"total_words"`
	WordsUsed  int    `json:"words_used"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
	Redis     string `json:"redis"`
} 