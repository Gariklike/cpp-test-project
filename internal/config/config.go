package config

import (
	"os"
)

type Config struct {
	// Базы данных
	DatabaseURL   string // PostgreSQL для пользователей и токенов
	RedisURL      string // Redis для сессий
	MongoURL      string // MongoDB для логов/аналитики (опционально)
	MongoDatabase string // Имя базы данных MongoDB

	// JWT
	JWTSecret string

	// OAuth провайдеры
	GitHubClientID     string
	GitHubClientSecret string
	YandexClientID     string
	YandexClientSecret string

	// Сервер
	ServerPort  string
	ServerHost  string
	Environment string // development, staging, production
}

func Load() *Config {
	return &Config{
		// Базы данных
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/auth_db?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
		MongoURL:      getEnv("MONGO_URL", "mongodb://localhost:27017"),
		MongoDatabase: getEnv("MONGO_DATABASE", "auth_service"),

		// JWT
		JWTSecret: getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),

		// OAuth провайдеры
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		YandexClientID:     getEnv("YANDEX_CLIENT_ID", ""),
		YandexClientSecret: getEnv("YANDEX_CLIENT_SECRET", ""),

		// Сервер
		ServerPort:  getEnv("PORT", "8080"),
		ServerHost:  getEnv("HOST", "localhost"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsProduction проверяет, production ли среда
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment проверяет, development ли среда
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// GetMongoConnectionString возвращает полную строку подключения к MongoDB
func (c *Config) GetMongoConnectionString() string {
	return c.MongoURL
}

// GetServerAddress возвращает адрес сервера (host:port)
func (c *Config) GetServerAddress() string {
	return c.ServerHost + ":" + c.ServerPort
}
