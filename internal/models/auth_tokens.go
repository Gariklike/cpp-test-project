package models

import (
	"time"
)

type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"` // или ExpiresIn int
	TokenType    string    `json:"token_type,omitempty"`
}
