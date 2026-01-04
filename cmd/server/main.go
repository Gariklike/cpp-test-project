package main

import (
	"log"
	"net/http"
	"os"

	"internal/config"
	"internal/handlers"
	"internal/repository/postgres"
	"internal/repository/redis"
	"internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Инициализация конфигурации
	cfg := config.Load()

	// Инициализация репозиториев
	userRepo, err := postgres.NewUserRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer userRepo.Close()

	tokenRepo, err := postgres.NewTokenRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer tokenRepo.Close()

	sessionRepo, err := redis.NewSessionRepository(cfg.RedisURL)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer sessionRepo.Close()

	// Инициализация сервисов
	authService := services.NewAuthService(userRepo, tokenRepo, sessionRepo)
	tokenService := services.NewTokenService(cfg.JWTSecret)
	oauthService := services.NewOAuthService(cfg)
	permissionService := services.NewPermissionService()

	// Инициализация обработчиков
	authHandler := handlers.NewAuthHandler(authService, tokenService, oauthService, permissionService)
	tokenHandler := handlers.NewTokenHandler(tokenService, authService)
	codeAuthHandler := handlers.NewCodeAuthHandler(authService)

	// Настройка роутера
	router := gin.Default()

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

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Authorization server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

