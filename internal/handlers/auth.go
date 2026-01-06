package handlers

import (
	"log"
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

func NewAuthHandler(
	authService *services.AuthService,
	tokenService *services.TokenService,
	oauthService *services.OAuthService,
	permissionService *services.PermissionService,
) *AuthHandler {
	return &AuthHandler{
		authService:       authService,
		tokenService:      tokenService,
		oauthService:      oauthService,
		permissionService: permissionService,
	}
}

// InitAuth — начало OAuth
func (h *AuthHandler) InitAuth(c *gin.Context) {
	authType := c.Param("type")

	switch authType {
	case "github":
		h.initGitHubAuth(c)
	case "yandex":
		h.initYandexAuth(c)
	case "code":
		loginToken := c.Query("login_token")
		if loginToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "login_token is required for code auth"})
			return
		}
		h.initCodeAuth(c, loginToken)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported auth type"})
	}
}

func (h *AuthHandler) initGitHubAuth(c *gin.Context) {
	authURL, err := h.oauthService.GetGitHubAuthURL("")
	if err != nil {
		log.Printf("GitHub auth URL error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Не удалось начать авторизацию через GitHub"})
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) initYandexAuth(c *gin.Context) {
	authURL, err := h.oauthService.GetYandexAuthURL("")
	if err != nil {
		log.Printf("Yandex auth URL error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Не удалось начать авторизацию через Яндекс"})
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) initCodeAuth(c *gin.Context, loginToken string) {
	code, err := h.oauthService.GenerateAuthCode(loginToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": code})
}

// GitHub Callback
func (h *AuthHandler) GitHubCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "Отсутствует code или state"})
		return
	}

	userInfo, err := h.oauthService.HandleGitHubCallback(code, state)
	if err != nil {
		log.Printf("GitHub callback error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Ошибка обработки GitHub"})
		return
	}

	h.processOAuthCallback(c, state, userInfo.Email, userInfo.Name)
}

// Yandex Callback
func (h *AuthHandler) YandexCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "Отсутствует code или state"})
		return
	}

	accessToken, err := h.oauthService.ExchangeYandexCode(code)
	if err != nil {
		log.Printf("Yandex token error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Не удалось получить токен Яндекса"})
		return
	}

	userInfo, err := h.oauthService.GetYandexUserInfo(accessToken)
	if err != nil {
		log.Printf("Yandex userinfo error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Не удалось получить данные пользователя"})
		return
	}

	h.processOAuthCallback(c, state, userInfo.Email, userInfo.Name)
}

// Общая обработка после OAuth
func (h *AuthHandler) processOAuthCallback(c *gin.Context, state, email, name string) {
	if email == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "Email не получен от провайдера"})
		return
	}

	// Ищем пользователя
	user, err := h.authService.GetUserByEmail(email)
	var isNewUser bool

	if err != nil || user == nil {
		// Создаём нового
		newUser := &models.User{
			Email:    email,
			FullName: name,
			IsActive: true,
			Roles:    []string{"user"},
		}

		if err := h.authService.CreateUser(newUser); err != nil {
			log.Printf("Ошибка создания пользователя: %v", err)
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Failed to create user: " + err.Error()})
			return
		}
		user = newUser
		isNewUser = true
		log.Printf("Создан новый пользователь: %s (%s)", email, user.ID)
	}

	if !user.IsActive {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"message": "Пользователь заблокирован"})
		return
	}

	// ID теперь int64 из БД (предполагаем, что в модели User.ID — string или int64)
	// Если в модели ID string — преобразуем, если int64 — используем напрямую
	userID := 1 // заглушка на случай ошибки
	if id, err := strconv.ParseInt(user.ID, 10, 64); err == nil {
		userID = int(id)
	}

	// Разрешения
	permissions, _ := h.authService.GetUserPermissions(userID)

	// Токены
	accessToken, err := h.tokenService.GenerateAccessToken(userID, permissions)
	if err != nil {
		log.Printf("Access token error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Не удалось создать access token"})
		return
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken(userID, user.Email)
	if err != nil {
		log.Printf("Refresh token error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Не удалось создать refresh token"})
		return
	}

	// Сохраняем refresh token
	if err := h.authService.SaveRefreshToken(userID, refreshToken); err != nil {
		log.Printf("Save refresh token error: %v", err)
	}

	// Сохраняем статус и токены в сессии
	h.oauthService.UpdateAuthStatus(state, "granted")
	h.oauthService.SetAuthTokens(state, accessToken, refreshToken)

	// УСПЕХ!
	msg := "Авторизация успешна!"
	if isNewUser {
		msg = "Регистрация и авторизация успешны!"
	}

	c.HTML(http.StatusOK, "success.html", gin.H{
		"message": msg,
		"user":    user.FullName,
		"email":   user.Email,
	})
}

// RefreshToken
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Здесь нужна реальная валидация — пока заглушка
	userID := 1

	isValid, err := h.authService.ValidateRefreshToken(userID, req.RefreshToken)
	if err != nil || !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	permissions, _ := h.authService.GetUserPermissions(userID)
	accessToken, _ := h.tokenService.GenerateAccessToken(userID, permissions)

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}
