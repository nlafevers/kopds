package database

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/nlafevers/kopds/internal/domain"
)

func TestUserRepository(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "kopds-user-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	repo := NewUserRepository(db, slog.Default())
	ctx := context.Background()

	user := &domain.User{
		Username: "admin",
		Password: "hashed-password",
	}

	// Test CreateUserIfNotExists
	if err := repo.CreateUserIfNotExists(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Test GetByUsername
	got, err := repo.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("failed to get user by username: %v", err)
	}

	if got == nil {
		t.Fatal("expected user, got nil")
	}

	if got.Username != user.Username {
		t.Errorf("expected username %s, got %s", user.Username, got.Username)
	}

	if got.Password != user.Password {
		t.Errorf("expected password %s, got %s", user.Password, got.Password)
	}

	// Test GetByUsername - Not Found
	got, err = repo.GetByUsername(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nonexistent user")
	}

	// Test UpdatePassword
	if err := repo.UpdatePassword(ctx, user.Username, "new-hashed-password"); err != nil {
		t.Fatalf("failed to update password: %v", err)
	}

	got, _ = repo.GetByUsername(ctx, "admin")
	if got.Password != "new-hashed-password" {
		t.Errorf("expected updated password, got %s", got.Password)
	}

	if err := repo.DeleteUser(ctx, "admin"); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	got, err = repo.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByUsername after delete failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for deleted user")
	}
}
