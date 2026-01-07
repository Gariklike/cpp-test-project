package models

import (
	"context"
	"time"
)

type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	FullName     string    `json:"full_name" db:"full_name"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Roles        []string  `json:"roles" db:"roles"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	LoginMethod  string    `json:"login_method" db:"login_method"` // "local" или "oauth"
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type RefreshToken struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
}

type AuthStatus int

const (
	AuthStatusPending AuthStatus = iota
	AuthStatusGranted
	AuthStatusDenied
	AuthStatusExpired
)

type AuthRequest struct {
	LoginToken string      `json:"login_token"`
	Type       string      `json:"type"`
	ExpiresAt  time.Time   `json:"expires_at"`
	Status     AuthStatus  `json:"status"`
	Tokens     *AuthTokens `json:"tokens,omitempty"`
}

type GitHubUserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type YandexUserInfo struct {
	ID            string `json:"id"`
	Login         string `json:"login"`
	Email         string `json:"default_email"`
	Name          string `json:"real_name"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	IsAvatarEmpty bool   `json:"is_avatar_empty"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*User, error)
	UpdateRoles(ctx context.Context, id string, roles []string) error
	UpdateStatus(ctx context.Context, id string, isActive bool) error
}

type TokenRepository interface {
	Save(ctx context.Context, token *RefreshToken) error
	GetByToken(ctx context.Context, token string) (*RefreshToken, error)
	GetByUserID(ctx context.Context, userID string) ([]*RefreshToken, error)
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID string) error
}

type SessionRepository interface {
	SaveLoginToken(ctx context.Context, token string, ttl time.Duration) error
	ValidateLoginToken(ctx context.Context, token string) (bool, error)
	SaveSession(ctx context.Context, key string, data map[string]interface{}, ttl time.Duration) error
	GetSession(ctx context.Context, key string) (map[string]interface{}, error)
	DeleteSession(ctx context.Context, key string) error
}
