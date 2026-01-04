package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"internal/models"
	"internal/services"
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
	valid, err := h.authService.ValidateLoginToken(loginToken)
	if err != nil || !valid {
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

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

func (h *AuthHandler) initYandexAuth(c *gin.Context, loginToken string) {
	authURL, err := h.oauthService.GetYandexAuthURL(loginToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
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

	// Обмениваем code на access token
	githubToken, err := h.oauthService.ExchangeGitHubCode(code)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to exchange code for token",
		})
		return
	}

	// Получаем данные пользователя из GitHub
	userInfo, err := h.oauthService.GetGitHubUserInfo(githubToken)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to get user info from GitHub",
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
		user, err = h.authService.CreateUser(email, name)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"message": "Failed to create user",
			})
			return
		}
	}

	// Проверяем, заблокирован ли пользователь
	if !user.IsActive {
		h.oauthService.UpdateAuthStatus(state, models.AuthStatusDenied)
		c.HTML(http.StatusOK, "error.html", gin.H{
			"message": "User is blocked",
		})
		return
	}

	// Получаем разрешения пользователя
	permissions, err := h.authService.GetUserPermissions(user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to get user permissions",
		})
		return
	}

	// Генерируем токены
	accessToken, err := h.tokenService.GenerateAccessToken(user.ID, permissions)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to generate access token",
		})
		return
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to generate refresh token",
		})
		return
	}

	// Сохраняем refresh token в базе
	expiresAt := time.Now().Add(time.Hour * 24 * 7)
	err = h.authService.SaveRefreshToken(user.ID, refreshToken, expiresAt)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"message": "Failed to save refresh token",
		})
		return
	}

	// Обновляем статус авторизации
	h.oauthService.UpdateAuthStatus(state, models.AuthStatusGranted)
	h.oauthService.SetAuthTokens(state, accessToken, refreshToken)

	// Показываем страницу успеха
	c.HTML(http.StatusOK, "success.html", gin.H{
		"message": "Authorization successful! You can return to the application.",
	})
}

