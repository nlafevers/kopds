package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlafevers/kopds/internal/api"
	"github.com/nlafevers/kopds/internal/database"
)

func TestPasswordFromArgsReadsStdin(t *testing.T) {
	password, err := passwordFromArgs([]string{"--password-stdin"}, strings.NewReader("secret\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("passwordFromArgs failed: %v", err)
	}
	if password != "secret" {
		t.Fatalf("expected trimmed password, got %q", password)
	}
}

func TestPasswordFromArgsRejectsPositionalPassword(t *testing.T) {
	_, err := passwordFromArgs([]string{"secret"}, strings.NewReader(""), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected positional password to be rejected")
	}
}

func TestPasswordFromArgsRejectsEmptyStdinPassword(t *testing.T) {
	_, err := passwordFromArgs([]string{"--password-stdin"}, strings.NewReader("\n"), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected empty stdin password to be rejected")
	}
}

func TestCLIUserManagement(t *testing.T) {
	exe := "./kopds_test_bin"
	cmd := exec.Command("go", "build", "-o", exe, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}
	defer os.Remove(exe)

	dbPath := filepath.Join(t.TempDir(), "cli_test.db")

	os.Setenv("KOPDS_DATABASE_PATH", dbPath)
	defer os.Unsetenv("KOPDS_DATABASE_PATH")

	t.Run("Create User", func(t *testing.T) {
		cmd := exec.Command(exe, "create-user", "clitest", "--password-stdin")
		cmd.Stdin = bytes.NewBufferString("clipass\n")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("create-user failed: %v, output: %s", err, output)
		}

		if bytes.Contains(output, []byte("Using database:")) || bytes.Contains(output, []byte("Using log:")) {
			t.Errorf("unexpected config path output: %s", output)
		}
		if !bytes.Contains(output, []byte("User 'clitest' created/updated successfully.")) {
			t.Errorf("unexpected output: %s", output)
		}

		db, err := database.NewSQLite(dbPath)
		if err != nil {
			t.Fatalf("failed to open db: %v", err)
		}
		defer db.Close()

		repo := database.NewUserRepository(db)
		user, err := repo.GetByUsername(context.Background(), "clitest")
		if err != nil {
			t.Fatalf("failed to get user: %v", err)
		}
		if user == nil {
			t.Fatal("user not found in db")
		}
		if !api.CheckPassword(user.Password, "clipass") {
			t.Error("password mismatch")
		}
	})

	t.Run("Change Password", func(t *testing.T) {
		cmd := exec.Command(exe, "change-password", "clitest", "--password-stdin")
		cmd.Stdin = bytes.NewBufferString("newclipass\n")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("change-password failed: %v, output: %s", err, output)
		}

		if !bytes.Contains(output, []byte("Password for user 'clitest' updated successfully.")) {
			t.Errorf("unexpected output: %s", output)
		}

		db, _ := database.NewSQLite(dbPath)
		defer db.Close()
		repo := database.NewUserRepository(db)
		user, _ := repo.GetByUsername(context.Background(), "clitest")
		if !api.CheckPassword(user.Password, "newclipass") {
			t.Error("password update failed")
		}
	})

	t.Run("Create Existing User Updates Password", func(t *testing.T) {
		cmd := exec.Command(exe, "create-user", "clitest", "--password-stdin")
		cmd.Stdin = bytes.NewBufferString("upsertpass\n")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("create existing user failed: %v, output: %s", err, output)
		}

		if !bytes.Contains(output, []byte("User 'clitest' created/updated successfully.")) {
			t.Errorf("unexpected output: %s", output)
		}

		db, _ := database.NewSQLite(dbPath)
		defer db.Close()
		repo := database.NewUserRepository(db)
		user, _ := repo.GetByUsername(context.Background(), "clitest")
		if !api.CheckPassword(user.Password, "upsertpass") {
			t.Error("create existing user did not update password")
		}
	})

	t.Run("Delete User", func(t *testing.T) {
		cmd := exec.Command(exe, "delete-user", "clitest")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("delete-user failed: %v, output: %s", err, output)
		}

		if !bytes.Contains(output, []byte("User 'clitest' deleted successfully.")) {
			t.Errorf("unexpected output: %s", output)
		}

		db, _ := database.NewSQLite(dbPath)
		defer db.Close()
		repo := database.NewUserRepository(db)
		user, err := repo.GetByUsername(context.Background(), "clitest")
		if err != nil {
			t.Fatalf("GetByUsername after delete failed: %v", err)
		}
		if user != nil {
			t.Error("user still exists after deletion")
		}
	})

	t.Run("Delete Non-Existent User", func(t *testing.T) {
		cmd := exec.Command(exe, "delete-user", "noone")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("expected failure for non-existent user, but it succeeded")
		}
		if !bytes.Contains(output, []byte("Failed to delete user: user not found")) {
			t.Errorf("expected 'Failed to delete user: user not found', got: %s", output)
		}
	})

	t.Run("Missing DB Is Created", func(t *testing.T) {
		os.Remove(dbPath)
		cmd := exec.Command(exe, "delete-user", "noone")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("expected failure for non-existent user, but it succeeded")
		}
		if !bytes.Contains(output, []byte("Failed to delete user: user not found")) {
			t.Errorf("expected 'Failed to delete user: user not found', got: %s", output)
		}
		if _, err := os.Stat(dbPath); err != nil {
			t.Errorf("expected CLI to create database, got: %v", err)
		}
	})
}
