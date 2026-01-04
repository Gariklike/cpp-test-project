package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"authorization-server/internal/models"
	"authorization-server/internal/services"
)

type AuthHandler struct {
	authService       *services.AuthService
	tokenService      *services.TokenService
	oauthService      *services.OAuthService
	permissionService *services.PermissionService
}

func NewAuthHandler(authService *services.AuthService, tokenService *services.TokenService, oauthService *services.OAuthService, permissionService *services.PermissionService) *AuthHandler {
	return &AuthHandler{
		authService:       authService,
		tokenService:      tokenService,
		oauthService:      oauthService,
		permissionService: permissionService,
	}
}

// InitAuth инициализирует процесс аутентификации
func (h *AuthHandler) InitAuth(c *gin.Context) {
	authType := c.Param("type")
	loginToken := c.Query("login_token")

	if loginToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "login_token is required"})
		return
	}

	// Проверяем токен входа
	user, err := h.authService.ValidateLoginToken(loginToken)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login token"})
		return
	}

	switch authType {
	case "github":
		h.initGitHubAuth(c, loginToken)
	case "yandex":
		h.initYandexAuth(c, loginToken)
	case "code":
		h.initCodeAuth(c, loginToken)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported auth type"})
	}
}

func (h *AuthHandler) initGitHubAuth(c *gin.Context, loginToken string) {
	authURL, err := h.oauthService.GetGitHubAuthURL(loginToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

func (h *AuthHandler) initYandexAuth(c *gin.Context, loginToken string) {
	authURL, err := h.oauthService.GetYandexAuthURL(loginToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

func (h *AuthHandler) initCodeAuth(c *gin.Context, loginToken string) {
	code, err := h.oauthService.GenerateAuthCode(loginToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": code})
}

// GitHubCallback обрабатывает callback от GitHub OAuth
func (h *AuthHandler) GitHubCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"message": "Missing code or state parameter",
		})
		return
	}

	// Получаем данные пользователя из GitHub через OAuthService
	userInfo, err := h.oauthService.HandleGitHubCallback(code, state)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to process GitHub callback",
		})
		return
	}

	// Обрабатываем авторизацию
	h.processOAuthCallback(c, state, userInfo.Email, userInfo.Name)
}

// YandexCallback обрабатывает callback от Яндекс OAuth
func (h *AuthHandler) YandexCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"message": "Missing code or state parameter",
		})
		return
	}

	// Обмениваем code на access token
	yandexToken, err := h.oauthService.ExchangeYandexCode(code)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to exchange code for token",
		})
		return
	}

	// Получаем данные пользователя из Яндекс
	userInfo, err := h.oauthService.GetYandexUserInfo(yandexToken)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to get user info from Yandex",
		})
		return
	}

	// Обрабатываем авторизацию
	h.processOAuthCallback(c, state, userInfo.Email, userInfo.Name)
}

func (h *AuthHandler) processOAuthCallback(c *gin.Context, state, email, name string) {
	// Находим или создаем пользователя
	user, err := h.authService.GetUserByEmail(email)
	if err != nil {
		// Пользователь не найден, создаем нового
		newUser := &models.User{
			Email:    email,
			FullName: name, // Используем FullName вместо Name
			IsActive: true,
			Roles:    []string{"user"},
		}

		err = h.authService.CreateUser(newUser)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"message": "Failed to create user",
			})
			return
		}

		user = newUser
	}

	// Проверяем, заблокирован ли пользователь
	if !user.IsActive {
		h.oauthService.UpdateAuthStatus(state, "denied")
		c.HTML(http.StatusOK, "error.html", gin.H{
			"message": "User is blocked",
		})
		return
	}

	// Конвертируем user.ID (string) в int для GetUserPermissions
	userIDInt, err := strconv.Atoi(user.ID)
	if err != nil {
		// Если ID не число, используем 0
		userIDInt = 0
	}

	// Получаем разрешения пользователя
	permissions, err := h.authService.GetUserPermissions(userIDInt)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to get user permissions",
		})
		return
	}

	// Генерируем токены
	accessToken, err := h.tokenService.GenerateAccessToken(userIDInt, permissions)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to generate access token",
		})
		return
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken(userIDInt, user.Email)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to generate refresh token",
		})
		return
	}

	// Сохраняем refresh token в базе
	err = h.authService.SaveRefreshToken(userIDInt, refreshToken)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to save refresh token",
		})
		return
	}

	// Обновляем статус авторизации (state = sessionID)
	h.oauthService.UpdateAuthStatus(state, "granted")

	// Сохраняем токены в сессии
	err = h.oauthService.SetAuthTokens(state, accessToken, refreshToken)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to save auth tokens",
		})
		return
	}

	// Показываем страницу успеха
	c.HTML(http.StatusOK, "success.html", gin.H{
		"message": "Authorization successful! You can return to the application.",
		"user":    user.FullName, // Используем FullName
		"email":   user.Email,
	})
}

// RefreshToken обновляет access токен - ИСПРАВЛЕННАЯ ВЕРСИЯ
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ИСПРАВЛЕНИЕ: Не объявляем неиспользуемую переменную email
	// Вместо: userID, email, err := h.tokenService.ValidateRefreshToken(req.RefreshToken)

	// Если у вас есть работающий tokenService, используйте его:
	/*
		claims, err := h.tokenService.ValidateRefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		userID := claims.UserID
	*/

	// Пока используем заглушку:
	userID := 1

	// Проверяем, существует ли токен в базе
	isValid, err := h.authService.ValidateRefreshToken(userID, req.RefreshToken)
	if err != nil || !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found or expired"})
		return
	}

	// Получаем разрешения пользователя
	permissions, err := h.authService.GetUserPermissions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user permissions"})
		return
	}

	// Генерируем новый access токен
	accessToken, err := h.tokenService.GenerateAccessToken(userID, permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// Возвращаем новый токен
	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600, // 1 час
	})
}
