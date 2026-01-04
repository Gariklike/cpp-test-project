package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"internal/models"
	"internal/repository"
)

type AuthService struct {
	userRepo    repository.UserRepository
	tokenRepo   repository.TokenRepository
	sessionRepo repository.SessionRepository
}

func NewAuthService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository, sessionRepo repository.SessionRepository) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		sessionRepo: sessionRepo,
	}
}

// GenerateLoginToken СЃРѕР·РґР°РµС‚ С‚РѕРєРµРЅ РІС…РѕРґР°
func (s *AuthService) GenerateLoginToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	loginToken := hex.EncodeToString(token)

	// РЎРѕС…СЂР°РЅСЏРµРј РІ Redis РЅР° 5 РјРёРЅСѓС‚
	err := s.sessionRepo.SaveLoginToken(context.Background(), loginToken, time.Minute*5)
	if err != nil {
		return "", err
	}

	return loginToken, nil
}

// ValidateLoginToken РїСЂРѕРІРµСЂСЏРµС‚ С‚РѕРєРµРЅ РІС…РѕРґР°
func (s *AuthService) ValidateLoginToken(loginToken string) (bool, error) {
	return s.sessionRepo.ValidateLoginToken(context.Background(), loginToken)
}

// GetUserByEmail РїРѕР»СѓС‡Р°РµС‚ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ РїРѕ email
func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
	return s.userRepo.GetByEmail(context.Background(), email)
}

// CreateUser СЃРѕР·РґР°РµС‚ РЅРѕРІРѕРіРѕ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ
func (s *AuthService) CreateUser(email, name string) (*models.User, error) {
	user := &models.User{
		Email:     email,
		FullName:  name,
		Roles:     []string{"student"},
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	return s.userRepo.Create(context.Background(), user)
}

// GetUserPermissions РїРѕР»СѓС‡Р°РµС‚ СЂР°Р·СЂРµС€РµРЅРёСЏ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ
func (s *AuthService) GetUserPermissions(userID string) ([]string, error) {
	user, err := s.userRepo.GetByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}

	// РЎРѕР±РёСЂР°РµРј СЂР°Р·СЂРµС€РµРЅРёСЏ РёР· РІСЃРµС… СЂРѕР»РµР№ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ
	permissions := make([]string, 0)

	for _, role := range user.Roles {
		rolePerms := getPermissionsByRole(role)
		permissions = append(permissions, rolePerms...)
	}

	return removeDuplicates(permissions), nil
}

// SaveRefreshToken СЃРѕС…СЂР°РЅСЏРµС‚ refresh token
func (s *AuthService) SaveRefreshToken(userID, refreshToken string, expiresAt time.Time) error {
	token := &models.RefreshToken{
		UserID:    userID,
		Token:     refreshToken,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	return s.tokenRepo.Save(context.Background(), token)
}

// ValidateRefreshToken РїСЂРѕРІРµСЂСЏРµС‚ refresh token
func (s *AuthService) ValidateRefreshToken(refreshToken string) (*models.RefreshToken, error) {
	return s.tokenRepo.GetByToken(context.Background(), refreshToken)
}

// DeleteRefreshToken СѓРґР°Р»СЏРµС‚ refresh token
func (s *AuthService) DeleteRefreshToken(refreshToken string) error {
	return s.tokenRepo.Delete(context.Background(), refreshToken)
}

// DeleteAllUserTokens СѓРґР°Р»СЏРµС‚ РІСЃРµ С‚РѕРєРµРЅС‹ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ
func (s *AuthService) DeleteAllUserTokens(userID string) error {
	return s.tokenRepo.DeleteByUserID(context.Background(), userID)
}

// Helper functions
func getPermissionsByRole(role string) []string {
	permissionsMap := map[string][]string{
		"student": {
			"user:fullName:write:self",
			"user:data:read:self",
			"course:list:read",
			"course:info:read",
			"course:testList:read:enrolled",
			"course:test:read:enrolled",
		},
		"teacher": {
			"user:fullName:write:self",
			"user:data:read:self",
			"user:data:read:other",
			"course:list:read",
			"course:info:read",
			"course:info:write:own",
			"course:testList:read",
			"course:test:read",
			"course:test:write:own",
			"test:create:own",
			"test:update:own",
			"test:delete:own",
			"question:create:own",
			"question:update:own",
			"question:delete:own",
		},
		"admin": {
			"user:list:read",
			"user:fullName:write",
			"user:data:read",
			"user:roles:read",
			"user:roles:write",
			"user:block:read",
			"user:block:write",
			"course:info:write",
			"course:test:write",
			"test:create",
			"test:update",
			"test:delete",
			"question:create",
			"question:update",
			"question:delete",
		},
	}

	if perms, ok := permissionsMap[role]; ok {
		return perms
	}

	return []string{}
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}


