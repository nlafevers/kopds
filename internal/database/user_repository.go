package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nlafevers/kopds/internal/domain"
)

type sqliteUserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new SQLite user repository.
func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &sqliteUserRepository{db: db}
}

func (r *sqliteUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, username, password FROM users WHERE username = ?`
	var user domain.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *sqliteUserRepository) Save(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (username, password) VALUES (?, ?)
		ON CONFLICT(username) DO UPDATE SET password=excluded.password
		RETURNING id`
	err := r.db.QueryRowContext(ctx, query, user.Username, user.Password).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (r *sqliteUserRepository) DeleteUser(ctx context.Context, username string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
