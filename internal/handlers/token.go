package handlers

import (
	"log" // ДОБАВЬТЕ ЭТОТ ИМПОРТ
	"net/http"

	"authorization-server/internal/services"

	"github.com/gin-gonic/gin"
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

func (h *TokenHandler) RefreshToken(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем refresh token
	claims, err := h.tokenService.ValidateRefreshToken(request.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Получаем разрешения пользователя
	permissions, err := h.authService.GetUserPermissions(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Генерируем новый access token
	accessToken, err := h.tokenService.GenerateAccessToken(claims.UserID, permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": request.RefreshToken,
	})
}

func (h *TokenHandler) ValidateToken(c *gin.Context) {
	var request struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидируем токен
	claims, err := h.tokenService.ValidateAccessToken(request.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid":   false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":       true,
		"user_id":     claims.UserID,
		"exp":         claims.ExpiresAt,
		"permissions": claims.Permissions,
	})
}

// Logout обрабатывает POST запрос для выхода (для API)
func (h *TokenHandler) Logout(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Удаляем refresh token из базы
	err := h.authService.DeleteRefreshToken(request.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Очищаем куки
	h.clearAuthCookies(c)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// LogoutGet обрабатывает GET запрос для выхода (для браузера)
func (h *TokenHandler) LogoutGet(c *gin.Context) {
	// Получаем refresh token из куки или query параметра
	refreshToken := c.Query("refresh_token")
	if refreshToken == "" {
		// Пробуем получить из куки
		refreshToken, _ = c.Cookie("refresh_token")
	}

	// Если есть refresh token, удаляем его
	if refreshToken != "" {
		err := h.authService.DeleteRefreshToken(refreshToken)
		if err != nil {
			// Логируем ошибку, но продолжаем выход
			log.Printf("Failed to delete refresh token: %v", err)
		}
	}

	// Очищаем все авторизационные куки
	h.clearAuthCookies(c)

	// Перенаправляем на главную страницу
	c.Redirect(http.StatusFound, "/")
}

// clearAuthCookies очищает все авторизационные куки
func (h *TokenHandler) clearAuthCookies(c *gin.Context) {
	authCookies := []string{
		"access_token",
		"refresh_token",
		"session_token",
		"session",
		"token",
		"auth_token",
		"oauth_state",
		"user_id",
		"auth_state",
	}

	for _, cookieName := range authCookies {
		c.SetCookie(cookieName, "", -1, "/", "", false, true)
	}
}
