package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"authorization-server/internal/models"

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
	redisRepo SessionRepository // Используем интерфейс из auth_service.go
}

func NewOAuthService(cfg *Config, redisRepo SessionRepository) *OAuthService {
	return &OAuthService{
		config:    cfg,
		redisRepo: redisRepo,
	}
}

func (s *OAuthService) GetGitHubAuthURL(loginToken string) (string, error) {
	state := uuid.New().String()

	session := &AuthSession{
		ID:        state,
		UserID:    "",
		Token:     loginToken,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}

	err := s.redisRepo.SaveAuthSession(state, session)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email",
		s.config.GitHubClientID,
		"http://localhost:8080/auth/callback/github",
		state,
	)

	return url, nil
}

func (s *OAuthService) GetYandexAuthURL(loginToken string) (string, error) {
	state := uuid.New().String()

	session := &AuthSession{
		ID:        state,
		UserID:    "",
		Token:     loginToken,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}

	err := s.redisRepo.SaveAuthSession(state, session)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(
		"https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&state=%s",
		s.config.YandexClientID,
		state,
	)

	return url, nil
}

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

	client := &http.Client{}
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

func (s *OAuthService) GetGitHubUserInfo(accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
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

func (s *OAuthService) ExchangeYandexCode(code string) (string, error) {
	return "dummy_yandex_access_token", nil
}

func (s *OAuthService) GetYandexUserInfo(accessToken string) (*models.YandexUserInfo, error) {
	userInfo := &models.YandexUserInfo{
		Email:     "test@yandex.ru",
		Name:      "Test Yandex User",
		ID:        "12345",
		Login:     "testuser",
		FirstName: "Test",
		LastName:  "User",
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
