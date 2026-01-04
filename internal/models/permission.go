// ЗАМЕНИТЕ package auth НА package models
package models

// Ваш код...

import (
	"errors"
)

// Permission представляет разрешение в системе
type Permission struct {
	ID          string
	Name        string
	Description string
	Resource    string
	Action      string // например: read, write, delete
}

// PermissionService управляет разрешениями
type PermissionService struct {
	permissions map[string]Permission
}

func NewPermissionService() *PermissionService {
	return &PermissionService{
		permissions: make(map[string]Permission),
	}
}

func (ps *PermissionService) AddPermission(p Permission) error {
	if _, exists := ps.permissions[p.ID]; exists {
		return errors.New("permission already exists")
	}
	ps.permissions[p.ID] = p
	return nil
}

func (ps *PermissionService) HasPermission(userID, resource, action string) bool {
	// Здесь логика проверки прав пользователя
	// Например, проверка ролей, групп и т.д.
	return false // заглушка
}
