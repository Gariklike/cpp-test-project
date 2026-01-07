package handlers

import (
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

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

	user, err := h.authService.GetUserByEmail(email)
	if err != nil {
		log.Printf("Ошибка поиска пользователя по email: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Внутренняя ошибка сервера"})
		return
	}

	if user == nil {
		// Генерируем ID для нового пользователя
		userID := uuid.New().String()
		newUser := &models.User{
			ID:          userID,
			Email:       email,
			FullName:    name,
			IsActive:    true,
			Roles:       []string{"user"},
			LoginMethod: "oauth",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := h.authService.CreateUser(newUser); err != nil {
			log.Printf("Ошибка создания пользователя: %v", err)
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"message": "Не удалось создать пользователя",
				"details": err.Error(),
			})
			return
		}
		user = newUser
		log.Printf("Создан новый OAuth пользователь: %s (ID: %s)", email, user.ID)
	} else {
		log.Printf("Вход существующего OAuth пользователя: %s (ID: %s)", email, user.ID)
	}

	if !user.IsActive {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"message": "Пользователь заблокирован"})
		return
	}

	// Преобразуем string ID в int для токенов
	userIDInt := h.convertUserIDToInt(user.ID)

	permissions, _ := h.authService.GetUserPermissions(userIDInt)

	accessToken, err := h.tokenService.GenerateAccessToken(userIDInt, permissions)
	if err != nil {
		log.Printf("Access token error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Не удалось создать access token"})
		return
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken(userIDInt, user.Email)
	if err != nil {
		log.Printf("Refresh token error: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Не удалось создать refresh token"})
		return
	}

	if err := h.authService.SaveRefreshToken(userIDInt, refreshToken); err != nil {
		log.Printf("Save refresh token error: %v", err)
	}

	// Куки
	c.SetCookie("access_token", accessToken, 3600, "/", "", false, true)
	c.SetCookie("refresh_token", refreshToken, 86400*30, "/", "", false, true)

	h.oauthService.UpdateAuthStatus(state, "granted")
	h.oauthService.SetAuthTokens(state, accessToken, refreshToken)

	// Переход на страницу успеха
	c.Redirect(http.StatusFound, "/success")
}

// LocalLogin — GET: форма входа, POST: обработка
func (h *AuthHandler) LocalLogin(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.HTML(http.StatusOK, "login.html", gin.H{"title": "Вход"})
		return
	}

	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")

	if email == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"title": "Вход",
			"error": "Введите email и пароль",
			"email": email,
		})
		return
	}

	user, err := h.authService.GetUserByEmail(email)
	if err != nil {
		log.Printf("Ошибка поиска пользователя: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Внутренняя ошибка"})
		return
	}

	if user == nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Вход",
			"error": "Неверный email или пароль",
			"email": email,
		})
		return
	}

	if !user.IsActive {
		c.HTML(http.StatusForbidden, "login.html", gin.H{
			"title": "Вход",
			"error": "Аккаунт заблокирован",
			"email": email,
		})
		return
	}

	// Проверяем, есть ли пароль у пользователя (не OAuth-пользователь)
	if user.PasswordHash == "" || strings.TrimSpace(user.PasswordHash) == "" {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Вход",
			"error": "Этот аккаунт создан через социальный логин. Используйте его.",
			"email": email,
		})
		return
	}

	// Проверка пароля
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Вход",
			"error": "Неверный email или пароль",
			"email": email,
		})
		return
	}

	// Преобразуем string ID в int для токенов
	userIDInt := h.convertUserIDToInt(user.ID)

	permissions, _ := h.authService.GetUserPermissions(userIDInt)

	accessToken, err := h.tokenService.GenerateAccessToken(userIDInt, permissions)
	if err != nil {
		log.Printf("Ошибка генерации access token: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Ошибка авторизации"})
		return
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken(userIDInt, user.Email)
	if err != nil {
		log.Printf("Ошибка генерации refresh token: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Ошибка авторизации"})
		return
	}

	h.authService.SaveRefreshToken(userIDInt, refreshToken)

	c.SetCookie("access_token", accessToken, 3600, "/", "", false, true)
	c.SetCookie("refresh_token", refreshToken, 86400*30, "/", "", false, true)

	c.Redirect(http.StatusFound, "/success")
}

// Register — GET: форма, POST: обработка
func (h *AuthHandler) Register(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.HTML(http.StatusOK, "register.html", gin.H{"title": "Регистрация"})
		return
	}

	email := strings.TrimSpace(c.PostForm("email"))
	fullName := strings.TrimSpace(c.PostForm("full_name"))
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	// Валидация обязательных полей
	if email == "" || password == "" || fullName == "" || confirmPassword == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":    "Регистрация",
			"error":    "Заполните все поля",
			"email":    email,
			"fullName": fullName,
		})
		return
	}

	// Проверка совпадения паролей
	if password != confirmPassword {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":    "Регистрация",
			"error":    "Пароли не совпадают",
			"email":    email,
			"fullName": fullName,
		})
		return
	}

	// Проверка формата email
	if !isValidEmail(email) {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":    "Регистрация",
			"error":    "Некорректный формат email",
			"email":    email,
			"fullName": fullName,
		})
		return
	}

	// Проверка длины пароля
	if len(password) < 6 {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":    "Регистрация",
			"error":    "Пароль должен содержать минимум 6 символов",
			"email":    email,
			"fullName": fullName,
		})
		return
	}

	// Проверка существующего пользователя
	existing, err := h.authService.GetUserByEmail(email)
	if err != nil {
		log.Printf("Ошибка проверки email: %v", err)
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"title":    "Регистрация",
			"error":    "Ошибка проверки данных. Попробуйте позже.",
			"email":    email,
			"fullName": fullName,
		})
		return
	}

	if existing != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":    "Регистрация",
			"error":    "Пользователь с таким email уже существует",
			"email":    email,
			"fullName": fullName,
		})
		return
	}

	// Хеширование пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Ошибка хэширования пароля: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Ошибка сервера при обработке пароля"})
		return
	}

	// Генерация UUID для пользователя
	userID := uuid.New().String()

	// Устанавливаем роль "student" для соответствия вашей БД
	roles := []string{"student"}

	newUser := &models.User{
		ID:           userID,
		Email:        email,
		FullName:     fullName,
		PasswordHash: string(hashedPassword),
		IsActive:     true,
		Roles:        roles,
		LoginMethod:  "local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Создание пользователя в базе
	if err := h.authService.CreateUser(newUser); err != nil {
		log.Printf("Ошибка создания пользователя: %v", err)
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"title":    "Регистрация",
			"error":    "Не удалось создать пользователя: " + err.Error(),
			"email":    email,
			"fullName": fullName,
		})
		return
	}

	log.Printf("Успешная регистрация: %s (ID: %s, Name: %s)", email, userID, fullName)

	// После успешной регистрации редирект на страницу входа с сообщением
	c.Redirect(http.StatusFound, "/login?registered=success&email="+email)
}

// SuccessPage — страница успешной авторизации
func (h *AuthHandler) SuccessPage(c *gin.Context) {
	// Получаем параметры из URL (если они есть)
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDescription := c.Query("error_description")

	// Проверяем на ошибки OAuth
	if errorParam != "" {
		log.Printf("OAuth ошибка: %s - %s", errorParam, errorDescription)
		c.HTML(http.StatusOK, "error.html", gin.H{
			"title":   "Ошибка авторизации",
			"message": "Ошибка при авторизации через OAuth провайдер",
			"details": errorDescription,
		})
		return
	}

	// Проверяем, авторизован ли пользователь
	accessToken, err := c.Cookie("access_token")
	isAuthenticated := err == nil && accessToken != ""

	// Если есть код авторизации, но пользователь не авторизован
	if code != "" && state != "" && !isAuthenticated {
		log.Printf("OAuth callback на /success: code=%s, state=%s", code, state)
		// Пробуем обработать как OAuth callback
		// Это упрощенная версия - в реальности нужно вызывать обработчик callback
	}

	// Определяем сообщение в зависимости от аутентификации
	message := "Авторизация прошла успешно!"
	if !isAuthenticated {
		message = "Добро пожаловать! Вы успешно зарегистрировались."
	}

	c.HTML(http.StatusOK, "success.html", gin.H{
		"title":           "Успешная авторизация",
		"message":         message,
		"isAuthenticated": isAuthenticated,
		"code":            code,
		"state":           state,
	})
}

// Logout — выход
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// HomePage — главная страница (публичная)
func (h *AuthHandler) HomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Authorization Server",
	})
}

// ProfilePage — профиль
func (h *AuthHandler) ProfilePage(c *gin.Context) {
	_, err := c.Cookie("access_token")
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	c.HTML(http.StatusOK, "profile.html", gin.H{"title": "Профиль"})
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

// Вспомогательная функция для проверки email
func isValidEmail(email string) bool {
	// Простая проверка на наличие @ и точки, а также отсутствие пробелов
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") || strings.Contains(email, " ") {
		return false
	}

	// Проверка на двойные @ (ваша ошибка из базы)
	if strings.Count(email, "@") > 1 {
		return false
	}

	// Более строгая проверка через regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// convertUserIDToInt преобразует string ID в int для совместимости с токен-сервисом
func (h *AuthHandler) convertUserIDToInt(userID string) int {
	if userID == "" {
		return 1
	}

	// Пытаемся преобразовать напрямую, если это число
	if idNum, err := strconv.Atoi(userID); err == nil && idNum > 0 {
		return idNum
	}

	// Если это UUID или строка, создаем хеш
	hash := 0
	for _, char := range userID {
		hash = (hash*31 + int(char)) % 1000000
	}

	// Гарантируем положительное число больше 0
	if hash <= 0 {
		hash = 1
	}

	return hash
}
