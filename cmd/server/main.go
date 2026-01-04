package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"authorization-server/internal/config"
	"authorization-server/internal/handlers"
	"authorization-server/internal/models"
	"authorization-server/internal/repository/redis"
	"authorization-server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Инициализация конфигурации
	cfg := config.Load()

	// Подключение к PostgreSQL
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Проверка соединения с БД
	if err := db.Ping(); err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Инициализация репозиториев через адаптеры
	userRepo := &PostgresUserRepositoryAdapter{db: db}
	tokenRepo := &PostgresTokenRepositoryAdapter{db: db}

	// Redis репозиторий для сессий
	redisRepo, err := redis.NewSessionRepository(cfg.RedisURL)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisRepo.Close()

	// Создаем адаптер для Redis
	sessionRepoAdapter := &SessionRepoAdapter{repo: redisRepo}

	// Конфигурация для OAuth
	oauthConfig := &services.Config{
		GitHubClientID:     cfg.GitHubClientID,
		GitHubClientSecret: cfg.GitHubClientSecret,
		YandexClientID:     cfg.YandexClientID,
		YandexClientSecret: cfg.YandexClientSecret,
	}

	// Инициализация сервисов
	authService := services.NewAuthService(db, userRepo, tokenRepo)
	tokenService := services.NewTokenService(
		cfg.JWTSecret,
		cfg.JWTSecret+"_refresh",
		1*time.Hour,
		24*7*time.Hour,
	)

	oauthService := services.NewOAuthService(oauthConfig, sessionRepoAdapter)
	permissionService := services.NewPermissionService()

	// Инициализация обработчиков
	authHandler := handlers.NewAuthHandler(authService, tokenService, oauthService, permissionService)
	tokenHandler := handlers.NewTokenHandler(tokenService, authService)
	codeAuthHandler := handlers.NewCodeAuthHandler(authService)

	// Настройка роутера
	router := gin.Default()

	// Добавляем статические файлы (HTML страницы)
	router.LoadHTMLGlob("templates/*")

	// Public endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth endpoints
	router.GET("/auth/:type", authHandler.InitAuth) // type: github, yandex, code
	router.GET("/auth/callback/github", authHandler.GitHubCallback)
	router.GET("/auth/callback/yandex", authHandler.YandexCallback)
	router.POST("/auth/code/verify", codeAuthHandler.VerifyCode)

	// Token endpoints
	router.POST("/token/refresh", tokenHandler.RefreshToken)
	router.POST("/token/validate", tokenHandler.ValidateToken)
	router.POST("/logout", tokenHandler.Logout)

	// Статические файлы
	router.Static("/static", "./static")

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Authorization server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// ============= АДАПТЕРЫ =============

// SessionRepoAdapter - адаптер для redis.SessionRepository
type SessionRepoAdapter struct {
	repo *redis.SessionRepository
}

func (a *SessionRepoAdapter) SaveAuthSession(state string, session *services.AuthSession) error {
	modelSession := models.AuthSession{
		ID:        session.ID,
		UserID:    session.UserID,
		Token:     session.Token,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
	return a.repo.SaveAuthSession(state, modelSession)
}

func (a *SessionRepoAdapter) GetAuthSession(state string) (*services.AuthSession, error) {
	modelSession, err := a.repo.GetAuthSession(state)
	if err != nil {
		return nil, err
	}
	if modelSession == nil {
		return nil, nil
	}

	return &services.AuthSession{
		ID:        modelSession.ID,
		UserID:    modelSession.UserID,
		Token:     modelSession.Token,
		CreatedAt: modelSession.CreatedAt,
		ExpiresAt: modelSession.ExpiresAt,
	}, nil
}

func (a *SessionRepoAdapter) DeleteAuthSession(state string) error {
	return a.repo.DeleteAuthSession(state)
}

// PostgresUserRepositoryAdapter - адаптер для postgres UserRepository
type PostgresUserRepositoryAdapter struct {
	db *sql.DB
}

func (r *PostgresUserRepositoryAdapter) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, roles, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	var rolesStr string

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *PostgresUserRepositoryAdapter) Create(ctx context.Context, user *models.User) (*models.User, error) {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO users (id, email, full_name, roles, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, full_name, roles, is_active, created_at, updated_at
	`

	var rolesStr string
	err = r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		string(rolesJSON),
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *PostgresUserRepositoryAdapter) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, roles, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	var rolesStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *PostgresUserRepositoryAdapter) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}

	query := `
		UPDATE users
		SET email = $2, full_name = $3, roles = $4, is_active = $5, updated_at = $6
		WHERE id = $1
	`

	_, err = r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		string(rolesJSON),
		user.IsActive,
		user.UpdatedAt,
	)

	return err
}

func (r *PostgresUserRepositoryAdapter) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresUserRepositoryAdapter) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, email, full_name, roles, is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var rolesStr string

		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&rolesStr,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
			return nil, err
		}

		users = append(users, &user)
	}

	return users, nil
}

func (r *PostgresUserRepositoryAdapter) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *PostgresUserRepositoryAdapter) Activate(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = true, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *PostgresUserRepositoryAdapter) Deactivate(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = false, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// PostgresTokenRepositoryAdapter - адаптер для postgres TokenRepository
type PostgresTokenRepositoryAdapter struct {
	db *sql.DB
}

func (r *PostgresTokenRepositoryAdapter) SaveRefreshToken(userID int, token string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
		userID, token, expiresAt,
	)
	return err
}

func (r *PostgresTokenRepositoryAdapter) FindRefreshToken(token string) (*models.RefreshToken, error) {
	var t models.RefreshToken
	err := r.db.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at FROM refresh_tokens WHERE token = $1",
		token,
	).Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &t, nil
}

func (r *PostgresTokenRepositoryAdapter) DeleteRefreshToken(token string) error {
	_, err := r.db.Exec("DELETE FROM refresh_tokens WHERE token = $1", token)
	return err
}

func (r *PostgresTokenRepositoryAdapter) DeleteExpiredTokens() error {
	_, err := r.db.Exec("DELETE FROM refresh_tokens WHERE expires_at < NOW()")
	return err
}

func (r *PostgresTokenRepositoryAdapter) Close() error {
	return nil
}
