package services

import (
	"context"
	"database/sql"
	"encoding/json"
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

// Интерфейсы репозиториев
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

// ValidateUser — для локального логина (если используется)
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

// ValidateLoginToken — оставляем минимальную заглушку (если используется в code-auth)
func (s *AuthService) ValidateLoginToken(token string) (*models.User, error) {
	if token == "" {
		return nil, errors.New("empty token")
	}
	// Заглушка — можно доработать под реальную проверку
	return &models.User{
		ID:       "1",
		Email:    "test@example.com",
		FullName: "Test User",
		IsActive: true,
		Roles:    []string{"user"},
	}, nil
}

func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
	ctx := context.Background()
	return s.userRepo.GetByEmail(ctx, email)
}

func (s *AuthService) GetUserByID(id string) (*models.User, error) {
	ctx := context.Background()
	return s.userRepo.GetByID(ctx, id)
}

// CreateUser — ИСПРАВЛЕНО: ID генерируется в БД
func (s *AuthService) CreateUser(user *models.User) error {
	ctx := context.Background()

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Roles == nil {
		user.Roles = []string{"user"}
	}
	if user.FullName == "" {
		user.FullName = "User"
	}

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}

	// Запрос без ID — БД сгенерирует его сама
	query := `
		INSERT INTO users (email, full_name, roles, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var newID int64
	err = s.db.QueryRowContext(ctx, query,
		user.Email,
		user.FullName,
		rolesJSON,
		true,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&newID)
	if err != nil {
		return err
	}

	// Присваиваем сгенерированный ID как строку
	user.ID = strconv.FormatInt(newID, 10)

	return nil
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
	if err != nil || user == nil {
		return []string{}, nil
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
