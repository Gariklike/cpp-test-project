package services

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"authorization-server/internal/models"
)

// Константы для статусов авторизации
const (
	AuthStatusPending = "pending"
	AuthStatusGranted = "granted"
	AuthStatusDenied  = "denied"
	AuthStatusExpired = "expired"
)

// Определяем интерфейсы для репозиториев
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, user *models.User) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	Count(ctx context.Context) (int, error)
	Activate(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
}

type TokenRepository interface {
	SaveRefreshToken(userID int, token string, expiresAt time.Time) error
	FindRefreshToken(token string) (*models.RefreshToken, error)
	DeleteRefreshToken(token string) error
	DeleteExpiredTokens() error
	Close() error
}

type AuthSession struct {
	ID        string
	UserID    string
	Token     string
	CreatedAt int64
	ExpiresAt int64
}

type SessionRepository interface {
	SaveAuthSession(state string, session *AuthSession) error
	GetAuthSession(state string) (*AuthSession, error)
	DeleteAuthSession(state string) error
}

type AuthService struct {
	db        *sql.DB
	userRepo  UserRepository
	tokenRepo TokenRepository
}

func NewAuthService(db *sql.DB, userRepo UserRepository, tokenRepo TokenRepository) *AuthService {
	return &AuthService{
		db:        db,
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
	}
}

func (s *AuthService) ValidateUser(username, password string) (*models.User, error) {
	ctx := context.Background()
	user, err := s.userRepo.GetByEmail(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *AuthService) ValidateLoginToken(token string) (*models.User, error) {
	if token == "" {
		return nil, errors.New("empty token")
	}
	user := &models.User{
		ID:       "1",
		Email:    "test@example.com",
		FullName: "Test User",
		IsActive: true,
		Roles:    []string{"user"},
	}
	return user, nil
}

func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
	ctx := context.Background()
	return s.userRepo.GetByEmail(ctx, email)
}

func (s *AuthService) GetUserByID(id string) (*models.User, error) {
	ctx := context.Background()
	return s.userRepo.GetByID(ctx, id)
}

func (s *AuthService) CreateUser(user *models.User) error {
	ctx := context.Background()
	_, err := s.userRepo.Create(ctx, user)
	return err
}

func (s *AuthService) UpdateUser(user *models.User) error {
	ctx := context.Background()
	return s.userRepo.Update(ctx, user)
}

func (s *AuthService) DeleteUser(id string) error {
	ctx := context.Background()
	return s.userRepo.Delete(ctx, id)
}

func (s *AuthService) GetUserPermissions(userID int) ([]string, error) {
	user, err := s.GetUserByID(strconv.Itoa(userID))
	if err != nil {
		return nil, err
	}
	return user.Roles, nil
}

func (s *AuthService) SaveRefreshToken(userID int, token string) error {
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	return s.tokenRepo.SaveRefreshToken(userID, token, expiresAt)
}

func (s *AuthService) ValidateRefreshToken(userID int, token string) (bool, error) {
	refreshToken, err := s.tokenRepo.FindRefreshToken(token)
	if err != nil {
		return false, err
	}

	if refreshToken == nil {
		return false, nil
	}

	// refreshToken.UserID уже int, всё ок
	if refreshToken.UserID != userID {
		return false, nil
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		s.tokenRepo.DeleteRefreshToken(token)
		return false, nil
	}

	return true, nil
}

func (s *AuthService) DeleteRefreshToken(token string) error {
	return s.tokenRepo.DeleteRefreshToken(token)
}

func (s *AuthService) VerifyAuthCode(code string) (bool, error) {
	return code != "", nil
}

func (s *AuthService) GetTokensByCode(code string) (*models.AuthTokens, error) {
	// БЕЗ ExpiresIn и TokenType
	return &models.AuthTokens{
		AccessToken:  "dummy_access_token_" + code,
		RefreshToken: "dummy_refresh_token_" + code,
	}, nil
}

func (s *AuthService) ListUsers(limit, offset int) ([]*models.User, error) {
	ctx := context.Background()
	return s.userRepo.List(ctx, limit, offset)
}

func (s *AuthService) CountUsers() (int, error) {
	ctx := context.Background()
	return s.userRepo.Count(ctx)
}

func (s *AuthService) ActivateUser(id string) error {
	ctx := context.Background()
	return s.userRepo.Activate(ctx, id)
}

func (s *AuthService) DeactivateUser(id string) error {
	ctx := context.Background()
	return s.userRepo.Deactivate(ctx, id)
}

func (s *AuthService) CleanupExpiredTokens() error {
	return s.tokenRepo.DeleteExpiredTokens()
}
