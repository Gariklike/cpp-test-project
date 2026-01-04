package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"internal/models"
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

	// РџСЂРѕРІРµСЂСЏРµРј РїРѕРґРєР»СЋС‡РµРЅРёРµ
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

	// РџСЂРµРѕР±СЂР°Р·СѓРµРј roles РІ JSON
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

	// РџСЂРµРѕР±СЂР°Р·СѓРµРј roles РёР· JSON
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

	// РџСЂРµРѕР±СЂР°Р·СѓРµРј roles РёР· JSON
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

	// РџСЂРµРѕР±СЂР°Р·СѓРµРј roles РёР· JSON
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


