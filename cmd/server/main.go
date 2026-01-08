package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Инициализация конфигурации
	cfg := config.Load()

	// ============= ПОДКЛЮЧЕНИЕ К POSTGRESQL =============
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}
	defer db.Close()

	// Проверка соединения с PostgreSQL
	if err := db.Ping(); err != nil {
		log.Fatal("PostgreSQL connection failed:", err)
	}
	log.Println("Connected to PostgreSQL")

	// ============= ПОДКЛЮЧЕНИЕ К MONGODB =============
	var mongoClient *mongo.Client
	var mongoDB *mongo.Database

	if cfg.MongoURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURL))
		if err != nil {
			log.Printf("Warning: Failed to connect to MongoDB: %v", err)
			log.Println("Continuing without MongoDB...")
		} else {
			// Проверка соединения с MongoDB
			err = mongoClient.Ping(ctx, nil)
			if err != nil {
				log.Printf("Warning: MongoDB ping failed: %v", err)
			} else {
				mongoDB = mongoClient.Database(cfg.MongoDatabase)
				log.Printf("Connected to MongoDB: %s", cfg.MongoDatabase)
			}
		}
	}

	// Отложенное закрытие соединения с MongoDB
	if mongoClient != nil {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := mongoClient.Disconnect(ctx); err != nil {
				log.Printf("Error disconnecting from MongoDB: %v", err)
			}
		}()
	}

	// ============= ПОДКЛЮЧЕНИЕ К REDIS =============
	redisRepo, err := redis.NewSessionRepository(cfg.RedisURL)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisRepo.Close()
	log.Println("Connected to Redis")

	// ============= ИНИЦИАЛИЗАЦИЯ РЕПОЗИТОРИЕВ =============
	userRepo := &PostgresUserRepositoryAdapter{db: db}
	tokenRepo := &PostgresTokenRepositoryAdapter{db: db}
	sessionRepoAdapter := &SessionRepoAdapter{repo: redisRepo}

	// MongoDB репозиторий (если подключено)
	var mongoRepo *MongoRepository
	if mongoDB != nil {
		mongoRepo = &MongoRepository{
			client:     mongoClient,
			database:   mongoDB,
			collection: mongoDB.Collection("auth_logs"),
		}
		log.Println("MongoDB repository initialized")
	}

	// ============= КОНФИГУРАЦИЯ OAuth =============
	oauthConfig := &services.Config{
		GitHubClientID:     cfg.GitHubClientID,
		GitHubClientSecret: cfg.GitHubClientSecret,
		YandexClientID:     cfg.YandexClientID,
		YandexClientSecret: cfg.YandexClientSecret,
		ServerPort:         cfg.ServerPort, // Добавляем порт в конфиг
	}

	// ============= ИНИЦИАЛИЗАЦИЯ СЕРВИСОВ =============
	authService := services.NewAuthService(db, userRepo, tokenRepo)
	tokenService := services.NewTokenService(
		cfg.JWTSecret,
		cfg.JWTSecret+"_refresh",
		1*time.Hour,
		24*7*time.Hour,
	)
	oauthService := services.NewOAuthService(oauthConfig, sessionRepoAdapter)
	permissionService := services.NewPermissionService()

	// ============= ИНИЦИАЛИЗАЦИЯ ОБРАБОТЧИКОВ =============
	authHandler := handlers.NewAuthHandler(authService, tokenService, oauthService, permissionService)
	tokenHandler := handlers.NewTokenHandler(tokenService, authService)
	codeAuthHandler := handlers.NewCodeAuthHandler(authService)

	// ============= НАСТРОЙКА РОУТЕРА =============
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.RecoveryWithWriter(gin.DefaultWriter))

	// Загружаем HTML-шаблоны (включая index.html, login.html, register.html)
	router.LoadHTMLGlob("templates/*")

	// ============= РАЗДАЧА СТАТИЧЕСКИХ ФАЙЛОВ =============
	// Раздача статических файлов из папки templates (для шаблонов)
	router.Static("/static", "./static")

	// ============= ДОБАВЛЕНО: Раздача файлов с вопросами =============
	// Раздача статических файлов из папки pkg/question-site
	router.Static("/questions", "./pkg/question-site")
	// ================================================================

	// === Главная страница ===
	router.GET("/", authHandler.HomePage)

	// === Страница логина ===
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"title": "Login",
		})
	})

	// === Страница регистрации ===
	// GET для отображения формы регистрации
	router.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"title": "Register",
		})
	})

	// POST для обработки формы регистрации
	router.POST("/register", authHandler.Register)
	router.POST("/auth/register", authHandler.Register)

	// === Страница успешной авторизации ===
	router.GET("/success", authHandler.SuccessPage)

	// Middleware для логирования в MongoDB
	if mongoRepo != nil {
		router.Use(mongoLoggingMiddleware(mongoRepo))
	}

	// Public endpoints
	router.GET("/health", func(c *gin.Context) {
		if mongoRepo != nil {
			go mongoRepo.LogHealthCheck(c.ClientIP())
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "mongodb": mongoRepo != nil})
	})

	// Auth endpoints
	router.GET("/auth/:type", authHandler.InitAuth)
	router.GET("/auth/callback/github", authHandler.GitHubCallback)
	router.GET("/auth/callback/yandex", authHandler.YandexCallback)
	router.POST("/auth/code/verify", codeAuthHandler.VerifyCode)

	// Token endpoints
	router.POST("/token/refresh", tokenHandler.RefreshToken)
	router.POST("/token/validate", tokenHandler.ValidateToken)
	router.POST("/logout", tokenHandler.Logout)
	router.GET("/logout", tokenHandler.LogoutGet)

	// MongoDB debug endpoint
	if mongoRepo != nil {
		router.GET("/debug/mongo", func(c *gin.Context) {
			stats, err := mongoRepo.GetCollectionStats()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, stats)
		})
	}

	// ============= ЗАПУСК СЕРВЕРА =============
	port := cfg.ServerPort
	if port == "" {
		port = "8000"
	}

	log.Printf("Authorization server starting on port %s", port)
	log.Printf("PostgreSQL: ✓")
	log.Printf("Redis: ✓")
	if mongoRepo != nil {
		log.Printf("MongoDB: ✓")
	} else {
		log.Printf("MongoDB: ✗ (not connected)")
	}
	log.Printf("Questions available at: http://localhost:%s/questions/index.html", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// ============= ОСТАЛЬНОЙ КОД =============

type MongoRepository struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

func (r *MongoRepository) LogAuthEvent(userID, email, eventType, provider string) error {
	if r.collection == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := map[string]interface{}{
		"user_id":    userID,
		"email":      email,
		"event_type": eventType,
		"provider":   provider,
		"timestamp":  time.Now(),
		"ip_address": "",
	}

	_, err := r.collection.InsertOne(ctx, event)
	return err
}

func (r *MongoRepository) LogHealthCheck(clientIP string) error {
	if r.collection == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := map[string]interface{}{
		"event_type": "health_check",
		"timestamp":  time.Now(),
		"ip_address": clientIP,
	}

	_, err := r.collection.InsertOne(ctx, event)
	return err
}

func (r *MongoRepository) GetCollectionStats() (map[string]interface{}, error) {
	if r.collection == nil {
		return map[string]interface{}{"error": "MongoDB not connected"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(ctx, map[string]interface{}{}, options.Find().SetSort(map[string]interface{}{"timestamp": -1}).SetLimit(5))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var recentEvents []map[string]interface{}
	for cursor.Next(ctx) {
		var event map[string]interface{}
		if err := cursor.Decode(&event); err != nil {
			continue
		}
		recentEvents = append(recentEvents, event)
	}

	return map[string]interface{}{
		"collection": r.collection.Name(),
		"count":      count,
		"recent":     recentEvents,
	}, nil
}

func mongoLoggingMiddleware(repo *MongoRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/debug/mongo" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		if repo != nil && repo.collection != nil {
			go func() {
				event := map[string]interface{}{
					"path":        c.Request.URL.Path,
					"method":      c.Request.Method,
					"status":      c.Writer.Status(),
					"duration_ms": time.Since(start).Milliseconds(),
					"timestamp":   time.Now(),
					"client_ip":   c.ClientIP(),
					"user_agent":  c.Request.UserAgent(),
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				repo.collection.InsertOne(ctx, event)
			}()
		}
	}
}

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

type PostgresUserRepositoryAdapter struct {
	db *sql.DB
}

func (r *PostgresUserRepositoryAdapter) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, roles, is_active, created_at, updated_at, password_hash, login_method
		FROM users
		WHERE email = $1
	`

	var user models.User
	var rolesStr string
	var passwordHash sql.NullString
	var loginMethod sql.NullString

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&passwordHash,
		&loginMethod,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if loginMethod.Valid {
		user.LoginMethod = loginMethod.String
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

	// ИСПРАВЛЕННЫЙ ЗАПРОС - добавлены password_hash и login_method
	query := `
		INSERT INTO users (
			id, email, full_name, roles, is_active, 
			created_at, updated_at, password_hash, login_method
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		user.PasswordHash,
		user.LoginMethod,
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
		SELECT id, email, full_name, roles, is_active, created_at, updated_at, password_hash, login_method
		FROM users
		WHERE id = $1
	`

	var user models.User
	var rolesStr string
	var passwordHash sql.NullString
	var loginMethod sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&passwordHash,
		&loginMethod,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if loginMethod.Valid {
		user.LoginMethod = loginMethod.String
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

	// ИСПРАВЛЕННЫЙ ЗАПРОС - добавлены password_hash и login_method
	query := `
		UPDATE users
		SET 
			email = $2, 
			full_name = $3, 
			roles = $4, 
			is_active = $5, 
			updated_at = $6,
			password_hash = $7,
			login_method = $8
		WHERE id = $1
	`

	_, err = r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		string(rolesJSON),
		user.IsActive,
		user.UpdatedAt,
		user.PasswordHash,
		user.LoginMethod,
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
		SELECT id, email, full_name, roles, is_active, created_at, updated_at, password_hash, login_method
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
		var passwordHash sql.NullString
		var loginMethod sql.NullString

		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&rolesStr,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
			&passwordHash,
			&loginMethod,
		); err != nil {
			return nil, err
		}

		if passwordHash.Valid {
			user.PasswordHash = passwordHash.String
		}
		if loginMethod.Valid {
			user.LoginMethod = loginMethod.String
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
