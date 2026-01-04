package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"internal/services"
)

type TokenHandler struct {
	tokenService *services.TokenService
	authService  *services.AuthService
}

func NewTokenHandler(tokenService *services.TokenService, authService *services.AuthService) *TokenHandler {
	return &TokenHandler{
		tokenService: tokenService,
		authService:  authService,
	}
}

// RefreshToken обновляет access token
func (h *TokenHandler) RefreshToken(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем refresh token
	refreshToken, err := h.authService.ValidateRefreshToken(request.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// Проверяем срок действия
	if refreshToken.IsExpired() {
		// Удаляем просроченный токен
		h.authService.DeleteRefreshToken(request.RefreshToken)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token expired"})
		return
	}

	// Получаем пользователя
	user, err := h.authService.GetUserByEmail(refreshToken.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	// Получаем разрешения пользователя
	permissions, err := h.authService.GetUserPermissions(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get permissions"})
		return
	}

	// Генерируем новую пару токенов
	newAccessToken, err := h.tokenService.GenerateAccessToken(user.ID, permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	newRefreshToken, err := h.tokenService.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Сохраняем новый refresh token
	err = h.authService.SaveRefreshToken(user.ID, newRefreshToken, refreshToken.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save refresh token"})
		return
	}

	// Удаляем старый refresh token
	h.authService.DeleteRefreshToken(request.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    900, // 15 минут в секундах
	})
}

// ValidateToken проверяет токен
func (h *TokenHandler) ValidateToken(c *gin.Context) {
	var request struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидируем токен
	claims, err := h.tokenService.ValidateToken(request.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false, "error": err.Error()})
		return
	}

	// Проверяем тип токена
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false, "error": "not an access token"})
		return
	}

	// Проверяем срок действия
	if exp, ok := claims["exp"].(float64); ok {
		// Можно добавить дополнительную проверку срока действия
		_ = exp
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"claims": claims,
	})
}

// Logout выходит из системы
func (h *TokenHandler) Logout(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
		AllDevices   bool   `json:"all_devices"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	// Получаем информацию о токене
	refreshToken, err := h.authService.ValidateRefreshToken(request.RefreshToken)
	if err == nil && refreshToken != nil {
		userID := refreshToken.UserID

		if request.AllDevices {
			// Удаляем все токены пользователя
			err = h.authService.DeleteAllUserTokens(userID)
		} else {
			// Удаляем только текущий токен
			err = h.authService.DeleteRefreshToken(request.RefreshToken)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tokens"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

