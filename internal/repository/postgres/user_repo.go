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

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &UserRepository{db: db}, nil
}

func (r *UserRepository) Close() error {
	return r.db.Close()
}

// Create — создаёт пользователя и сохраняет password_hash
func (r *UserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return nil, err
	}

	// Добавлено password_hash в INSERT и RETURNING
	query := `
		INSERT INTO users (id, email, full_name, password_hash, roles, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, email, full_name, password_hash, roles, is_active, created_at, updated_at
	`

	var rolesStr string
	var passwordHash sql.NullString // для случая, когда NULL

	err = r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		user.PasswordHash, // ← Теперь сохраняется!
		string(rolesJSON),
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&passwordHash,
		&rolesStr,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Восстанавливаем password_hash (может быть NULL для OAuth)
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	} else {
		user.PasswordHash = ""
	}

	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID — теперь читает password_hash
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, password_hash, roles, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	var rolesStr string
	var passwordHash sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&passwordHash,
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

	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	} else {
		user.PasswordHash = ""
	}

	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByEmail — теперь читает password_hash
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, password_hash, roles, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	var rolesStr string
	var passwordHash sql.NullString

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&passwordHash,
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

	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	} else {
		user.PasswordHash = ""
	}

	if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
		return nil, err
	}

	return &user, nil
}

// Остальные методы (Update, Delete, List и т.д.) остаются без изменений
// (в Update password_hash не меняется, поэтому не трогаем)

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

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, email, full_name, password_hash, roles, is_active, created_at, updated_at
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
		var passwordHash sql.NullString

		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&passwordHash,
			&rolesStr,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if passwordHash.Valid {
			user.PasswordHash = passwordHash.String
		} else {
			user.PasswordHash = ""
		}

		if err := json.Unmarshal([]byte(rolesStr), &user.Roles); err != nil {
			return nil, err
		}

		users = append(users, &user)
	}

	return users, nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *UserRepository) Activate(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = true, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *UserRepository) Deactivate(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = false, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}
