package services

import (
	"errors"
)

type PermissionService struct {
	// Здесь могут быть зависимости: репозиторий, кэш и т.д.
}

func NewPermissionService() *PermissionService {
	return &PermissionService{}
}

// GetPermissionsForUser возвращает разрешения для пользователя
func (s *PermissionService) GetPermissionsForUser(userID int) ([]string, error) {
	// Здесь должна быть логика получения разрешений из базы данных
	// Пока возвращаем заглушку

	// Пример: разные разрешения для разных ID
	if userID == 1 {
		return []string{"admin", "read", "write", "delete"}, nil
	}

	// По умолчанию базовые разрешения
	return []string{"read", "write"}, nil
}

// CheckPermission проверяет, есть ли у пользователя указанное разрешение
func (s *PermissionService) CheckPermission(userID int, permission string) bool {
	permissions, err := s.GetPermissionsForUser(userID)
	if err != nil {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	return false
}

// CheckPermissionWithError - версия с возвратом ошибки
func (s *PermissionService) CheckPermissionWithError(userID int, permission string) (bool, error) {
	permissions, err := s.GetPermissionsForUser(userID)
	if err != nil {
		return false, err
	}

	for _, p := range permissions {
		if p == permission {
			return true, nil
		}
	}

	return false, nil
}

// ValidatePermissions проверяет валидность списка разрешений
func (s *PermissionService) ValidatePermissions(permissions []string) error {
	validPermissions := map[string]bool{
		"read":   true,
		"write":  true,
		"delete": true,
		"admin":  true,
		"view":   true,
		"edit":   true,
	}

	for _, p := range permissions {
		if !validPermissions[p] {
			return errors.New("invalid permission: " + p)
		}
	}

	return nil
}

// AddPermission добавляет разрешение пользователю (заглушка)
func (s *PermissionService) AddPermission(userID int, permission string) error {
	// В реальности: добавление в БД
	return nil
}

// RemovePermission удаляет разрешение у пользователя (заглушка)
func (s *PermissionService) RemovePermission(userID int, permission string) error {
	// В реальности: удаление из БД
	return nil
}

// GetUserRole возвращает роль пользователя
func (s *PermissionService) GetUserRole(userID int) (string, error) {
	permissions, err := s.GetPermissionsForUser(userID)
	if err != nil {
		return "", err
	}

	// Определяем роль на основе разрешений
	for _, p := range permissions {
		if p == "admin" {
			return "admin", nil
		}
	}

	return "user", nil
}
