package database

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/nlafevers/kopds/internal/domain"
)

func TestStorageUserMethods(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "kopds-storage-user-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := NewSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	s := NewStorage(db, slog.Default())

	// CreateUserIfNotExists - creates a new user
	if err := s.CreateUserIfNotExists("alice", "hash-alice"); err != nil {
		t.Fatalf("CreateUserIfNotExists failed: %v", err)
	}

	// CreateUserIfNotExists - fails on duplicate
	if err := s.CreateUserIfNotExists("alice", "hash-alice-2"); err == nil {
		t.Fatal("expected error for duplicate user, got nil")
	}

	// GetUserHash - returns the stored hash
	hash, err := s.GetUserHash("alice")
	if err != nil {
		t.Fatalf("GetUserHash failed: %v", err)
	}
	if hash != "hash-alice" {
		t.Errorf("expected hash-alice, got %s", hash)
	}

	// SaveUser - upserts (create new)
	if err := s.SaveUser("bob", "hash-bob"); err != nil {
		t.Fatalf("SaveUser (create) failed: %v", err)
	}

	// SaveUser - upserts (update existing)
	if err := s.SaveUser("alice", "hash-alice-updated"); err != nil {
		t.Fatalf("SaveUser (update) failed: %v", err)
	}
	hash, err = s.GetUserHash("alice")
	if err != nil {
		t.Fatalf("GetUserHash after SaveUser update failed: %v", err)
	}
	if hash != "hash-alice-updated" {
		t.Errorf("expected hash-alice-updated after SaveUser, got %s", hash)
	}

	// UpdatePassword - updates existing user
	if err := s.UpdatePassword("alice", "hash-alice-v2"); err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}
	hash, err = s.GetUserHash("alice")
	if err != nil {
		t.Fatalf("GetUserHash after UpdatePassword failed: %v", err)
	}
	if hash != "hash-alice-v2" {
		t.Errorf("expected hash-alice-v2 after UpdatePassword, got %s", hash)
	}

	// UpdatePassword - fails for nonexistent user
	if err := s.UpdatePassword("nonexistent", "hash-x"); err == nil {
		t.Fatal("expected error for UpdatePassword on nonexistent user, got nil")
	}

	// DeleteUser - removes existing user
	if err := s.DeleteUser("bob"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// GetUserHash - fails for deleted user
	if _, err := s.GetUserHash("bob"); err == nil {
		t.Fatal("expected error for GetUserHash on deleted user, got nil")
	}

	// DeleteUser - fails for nonexistent user
	if err := s.DeleteUser("nonexistent"); err == nil {
		t.Fatal("expected error for DeleteUser on nonexistent user, got nil")
	}
}

func TestUserRepository(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "kopds-user-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := NewSQLite(dbPath, true)
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

	// Test Save
	if err := repo.Save(ctx, user); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	if user.ID == 0 {
		t.Fatal("user ID should not be 0 after save")
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

	// Test Update password (ON CONFLICT)
	user.Password = "new-hashed-password"
	if err := repo.Save(ctx, user); err != nil {
		t.Fatalf("failed to update user: %v", err)
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
