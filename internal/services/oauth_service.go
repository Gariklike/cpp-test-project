package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Config для OAuth
type Config struct {
	GitHubClientID     string
	GitHubClientSecret string
	YandexClientID     string
	YandexClientSecret string
}

// OAuthUserInfo для информации пользователя OAuth
type OAuthUserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type OAuthService struct {
	config    *Config
	redisRepo SessionRepository // Интерфейс из вашего проекта
}

func NewOAuthService(cfg *Config, redisRepo SessionRepository) *OAuthService {
	return &OAuthService{
		config:    cfg,
		redisRepo: redisRepo,
	}
}

// GetGitHubAuthURL — для веб-авторизации (loginToken теперь не обязателен)
func (s *OAuthService) GetGitHubAuthURL(_ string) (string, error) {
	state := uuid.New().String()

	session := &AuthSession{
		ID:        state,
		UserID:    "",
		Token:     "",
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}

	err := s.redisRepo.SaveAuthSession(state, session)
	if err != nil {
		return "", err
	}

	// ИЗМЕНЕНИЕ: добавлен параметр prompt=login для принудительной авторизации
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email&prompt=login",
		s.config.GitHubClientID,
		url.QueryEscape("http://localhost:8080/auth/callback/github"),
		state,
	)

	return authURL, nil
}

// GetYandexAuthURL — для веб-авторизации
func (s *OAuthService) GetYandexAuthURL(_ string) (string, error) {
	state := uuid.New().String()

	session := &AuthSession{
		ID:        state,
		UserID:    "",
		Token:     "",
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}

	err := s.redisRepo.SaveAuthSession(state, session)
	if err != nil {
		return "", err
	}

	// ИЗМЕНЕНИЕ: добавлен параметр force_confirm=true для принудительной авторизации
	authURL := fmt.Sprintf(
		"https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&redirect_uri=%s&state=%s&force_confirm=true",
		s.config.YandexClientID,
		url.QueryEscape("http://localhost:8080/auth/callback/yandex"),
		state,
	)

	return authURL, nil
}

// ExchangeGitHubCode — остаётся как было
func (s *OAuthService) ExchangeGitHubCode(code string) (string, error) {
	reqBody := strings.NewReader(fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s",
		s.config.GitHubClientID,
		s.config.GitHubClientSecret,
		code,
	))

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", reqBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// GetGitHubUserInfo — остаётся как было
func (s *OAuthService) GetGitHubUserInfo(accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &OAuthUserInfo{
		Email: userInfo.Email,
		Name:  userInfo.Name,
	}, nil
}

// ExchangeYandexCode — реальная реализация
func (s *OAuthService) ExchangeYandexCode(code string) (string, error) {
	if s.config.YandexClientID == "" || s.config.YandexClientSecret == "" {
		return "", fmt.Errorf("yandex oauth credentials not configured")
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", s.config.YandexClientID)
	data.Set("client_secret", s.config.YandexClientSecret)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.PostForm("https://oauth.yandex.ru/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("yandex token request failed with status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// GetYandexUserInfo — теперь возвращает *OAuthUserInfo (совместимо с callback)
func (s *OAuthService) GetYandexUserInfo(accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequest("GET", "https://login.yandex.ru/info", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yandex userinfo request failed with status %d", resp.StatusCode)
	}

	var yandexResp struct {
		Emails      []string `json:"emails"`
		RealName    string   `json:"real_name"`
		FirstName   string   `json:"first_name"`
		LastName    string   `json:"last_name"`
		DisplayName string   `json:"display_name"`
		Login       string   `json:"login"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&yandexResp); err != nil {
		return nil, err
	}

	email := ""
	if len(yandexResp.Emails) > 0 {
		email = yandexResp.Emails[0]
	}

	name := yandexResp.DisplayName
	if name == "" {
		name = yandexResp.RealName
	}
	if name == "" {
		name = strings.TrimSpace(yandexResp.FirstName + " " + yandexResp.LastName)
	}
	if name == "" {
		name = yandexResp.Login
	}

	return &OAuthUserInfo{
		Email: email,
		Name:  name,
	}, nil
}

// Остальные методы оставляем без изменений
func (s *OAuthService) UpdateAuthStatus(state string, status string) error {
	session, err := s.redisRepo.GetAuthSession(state)
	if err != nil || session == nil {
		return fmt.Errorf("session not found")
	}
	return s.redisRepo.SaveAuthSession(state, session)
}

func (s *OAuthService) GenerateAuthCode(loginToken string) (string, error) {
	code := uuid.New().String()[:8]

	session := &AuthSession{
		ID:        code,
		UserID:    "",
		Token:     loginToken,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}

	err := s.redisRepo.SaveAuthSession(code, session)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (s *OAuthService) HandleGitHubCallback(code, state string) (*OAuthUserInfo, error) {
	session, err := s.redisRepo.GetAuthSession(state)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	accessToken, err := s.ExchangeGitHubCode(code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	userInfo, err := s.GetGitHubUserInfo(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if userInfo.Email != "" {
		session.UserID = userInfo.Email
		s.redisRepo.SaveAuthSession(state, session)
	}

	return userInfo, nil
}

func (s *OAuthService) SetAuthTokens(state string, accessToken, refreshToken string) error {
	session, err := s.redisRepo.GetAuthSession(state)
	if err != nil || session == nil {
		return fmt.Errorf("session not found")
	}

	session.Token = accessToken
	session.ExpiresAt = time.Now().Add(1 * time.Hour).Unix()

	return s.redisRepo.SaveAuthSession(state, session)
}
