package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWTManager управляет JWT токенами
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewJWTManager создает новый JWT менеджер
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
	}
}

// UserClaims кастомные claims для пользователя
type UserClaims struct {
	jwt.RegisteredClaims
	UserID      int      `json:"user_id"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions"`
}

// GenerateToken генерирует JWT токен для пользователя
func (manager *JWTManager) GenerateToken(userID int, email string, permissions []string) (string, error) {
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(manager.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:      userID,
		Email:       email,
		Permissions: permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(manager.secretKey))
}

// VerifyToken проверяет и парсит JWT токен
func (manager *JWTManager) VerifyToken(accessToken string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(
		accessToken,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(manager.secretKey), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// ExtractClaims извлекает claims из токена без валидации (только для отладки)
func ExtractClaims(tokenStr string) (jwt.MapClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// IsTokenExpired проверяет, истек ли срок действия токена
func IsTokenExpired(tokenStr string, secretKey string) (bool, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		if validationErr, ok := err.(*jwt.ValidationError); ok {
			if validationErr.Errors&jwt.ValidationErrorExpired != 0 {
				return true, nil
			}
		}
		return false, err
	}

	if !token.Valid {
		return true, nil
	}

	return false, nil
}

// RefreshToken обновляет токен
func (manager *JWTManager) RefreshToken(oldToken string) (string, error) {
	claims, err := manager.VerifyToken(oldToken)
	if err != nil {
		return "", err
	}

	// Проверяем, не истек ли токен (можно обновлять если истек не более часа назад)
	if time.Until(claims.ExpiresAt.Time) > -time.Hour {
		return manager.GenerateToken(claims.UserID, claims.Email, claims.Permissions)
	}

	return "", errors.New("token cannot be refreshed")
}
