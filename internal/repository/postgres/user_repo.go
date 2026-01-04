package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"authorization-server/internal/models"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(connectionString string) (*UserRepository, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	// Проверяем подключение
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &UserRepository{db: db}, nil
}

func (r *UserRepository) Close() error {
	return r.db.Close()
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Преобразуем roles в JSON
	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO users (id, email, full_name, roles, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, full_name, roles, is_active, created_at, updated_at
	`

	var rolesStr string
	err = r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		string(rolesJSON),
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Преобразуем roles из JSON
	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, roles, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	var rolesStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Преобразуем roles из JSON
	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, roles, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	var rolesStr string

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Преобразуем roles из JSON
	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}

	query := `
		UPDATE users
		SET email = $2, full_name = $3, roles = $4, is_active = $5, updated_at = $6
		WHERE id = $1
	`

	_, err = r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		string(rolesJSON),
		user.IsActive,
		user.UpdatedAt,
	)

	return err
}

// Delete удаляет пользователя по ID
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List возвращает список пользователей с пагинацией
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, email, full_name, roles, is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var rolesStr string

		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&rolesStr,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}

		// Преобразуем roles из JSON
		if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
			return nil, err
		}

		users = append(users, &user)
	}

	return users, nil
}

// Count возвращает общее количество пользователей
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// Activate активирует пользователя
func (r *UserRepository) Activate(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = true, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// Deactivate деактивирует пользователя
func (r *UserRepository) Deactivate(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = false, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}
