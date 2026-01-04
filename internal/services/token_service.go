package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	jwtSecret []byte
}

func NewTokenService(jwtSecret string) *TokenService {
	return &TokenService{
		jwtSecret: []byte(jwtSecret),
	}
}

// GenerateAccessToken создает JWT access token
func (s *TokenService) GenerateAccessToken(userID string, permissions []string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":     userID,
		"permissions": permissions,
		"exp":         time.Now().Add(time.Minute * 15).Unix(), // 15 минут
		"iat":         time.Now().Unix(),
		"type":        "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// GenerateRefreshToken создает JWT refresh token
func (s *TokenService) GenerateRefreshToken(userID, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 дней
		"iat":     time.Now().Unix(),
		"type":    "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateToken проверяет JWT токен
func (s *TokenService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ExtractUserIDFromToken извлекает user_id из токена
func (s *TokenService) ExtractUserIDFromToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	if userID, ok := claims["user_id"].(string); ok {
		return userID, nil
	}

	return "", errors.New("user_id not found in token")
}

// ExtractPermissionsFromToken извлекает permissions из токена
func (s *TokenService) ExtractPermissionsFromToken(tokenString string) ([]string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if permsInterface, ok := claims["permissions"].([]interface{}); ok {
		permissions := make([]string, len(permsInterface))
		for i, p := range permsInterface {
			if str, ok := p.(string); ok {
				permissions[i] = str
			}
		}
		return permissions, nil
	}

	return []string{}, nil
}

// IsTokenExpired проверяет истек ли срок действия токена
func (s *TokenService) IsTokenExpired(tokenString string) bool {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return true
	}

	if exp, ok := claims["exp"].(float64); ok {
		return time.Unix(int64(exp), 0).Before(time.Now())
	}

	return true
}
