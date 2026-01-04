package models

type AuthSession struct {
	ID        string `json:"id" redis:"id"`
	UserID    string `json:"user_id" redis:"user_id"` // string, чтобы соответствовать User.ID
	Token     string `json:"token" redis:"token"`
	CreatedAt int64  `json:"created_at" redis:"created_at"`
	ExpiresAt int64  `json:"expires_at" redis:"expires_at"`
}
