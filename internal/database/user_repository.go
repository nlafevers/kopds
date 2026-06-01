package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/nlafevers/kopds/internal/domain"
)

type sqliteUserRepository struct {
	db  *sql.DB
	log *slog.Logger
}

// NewUserRepository creates a new SQLite user repository.
func NewUserRepository(db *sql.DB, log *slog.Logger) domain.UserRepository {
	return &sqliteUserRepository{db: db, log: log}
}

func (r *sqliteUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	r.log.Debug("getting user by username", "username", username)
	query := `SELECT id, username, password FROM users WHERE username = ?`
	var user domain.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get user", "username", username, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *sqliteUserRepository) CreateUserIfNotExists(ctx context.Context, user *domain.User) error {
	r.log.Debug("creating user if not exists", "username", user.Username)
	existing, err := r.GetByUsername(ctx, user.Username)
	if err != nil {
		return err
	}
	if existing != nil {
		r.log.Warn("user already exists", "username", user.Username)
		return fmt.Errorf("user already exists")
	}

	query := `INSERT INTO users (username, password) VALUES (?, ?) RETURNING id`
	err = r.db.QueryRowContext(ctx, query, user.Username, user.Password).Scan(&user.ID)
	if err != nil {
		r.log.Error("failed to create user", "username", user.Username, "error", err)
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *sqliteUserRepository) DeleteUser(ctx context.Context, username string) error {
	r.log.Debug("deleting user", "username", username)
	res, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE username = ?", username)
	if err != nil {
		r.log.Error("failed to delete user", "username", username, "error", err)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		r.log.Error("failed to get rows affected", "operation", "delete_user", "username", username, "error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		r.log.Warn("user not found for deletion", "username", username)
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *sqliteUserRepository) UpdatePassword(ctx context.Context, username, password string) error {
	r.log.Debug("updating user password", "username", username)
	res, err := r.db.ExecContext(ctx, "UPDATE users SET password = ? WHERE username = ?", password, username)
	if err != nil {
		r.log.Error("failed to update password", "username", username, "error", err)
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		r.log.Error("failed to get rows affected", "operation", "update_password", "username", username, "error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		r.log.Warn("user not found for password update", "username", username)
		return fmt.Errorf("user not found")
	}

	return nil
}
