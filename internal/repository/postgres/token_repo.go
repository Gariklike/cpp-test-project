package postgres

import (
	"authorization-server/internal/models"
	"database/sql"
	"time"

	_ "github.com/lib/pq" // ВАЖНО: добавьте этот импорт!
)

type TokenRepository struct {
	db *sql.DB
}

func NewTokenRepository(databaseURL string) (*TokenRepository, error) {
	// 1. Открываем соединение с PostgreSQL
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	// 2. Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// 3. Создаем таблицу если она не существует
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS refresh_tokens (
            id SERIAL PRIMARY KEY,
            user_id INTEGER NOT NULL,
            token TEXT NOT NULL UNIQUE,
            expires_at TIMESTAMP NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		return nil, err
	}

	// 4. Создаем индекс для быстрого поиска
	_, err = db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token 
        ON refresh_tokens(token)
    `)
	if err != nil {
		return nil, err
	}

	return &TokenRepository{db: db}, nil
}

func (r *TokenRepository) Close() error {
	return r.db.Close()
}

func (r *TokenRepository) SaveRefreshToken(userID int, token string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
		userID, token, expiresAt,
	)
	return err
}

func (r *TokenRepository) FindRefreshToken(token string) (*models.RefreshToken, error) {
	var t models.RefreshToken
	err := r.db.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at FROM refresh_tokens WHERE token = $1",
		token,
	).Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (r *TokenRepository) DeleteRefreshToken(token string) error {
	_, err := r.db.Exec("DELETE FROM refresh_tokens WHERE token = $1", token)
	return err
}

func (r *TokenRepository) DeleteExpiredTokens() error {
	_, err := r.db.Exec("DELETE FROM refresh_tokens WHERE expires_at < NOW()")
	return err
}
