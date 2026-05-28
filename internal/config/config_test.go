package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Set some env vars
	os.Setenv("KOPDS_PORT", "9090")
	os.Setenv("KOPDS_DATABASE_PATH", "/tmp/test.db")
	os.Setenv("KOPDS_LOG_LEVEL", "debug")
	os.Setenv("KOPDS_LIBRARY_PATH", "/tmp/lib")

	defer func() {
		os.Unsetenv("KOPDS_PORT")
		os.Unsetenv("KOPDS_DATABASE_PATH")
		os.Unsetenv("KOPDS_LOG_LEVEL")
		os.Unsetenv("KOPDS_LIBRARY_PATH")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("expected 9090, got %d", cfg.Port)
	}
	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("expected /tmp/test.db, got %s", cfg.DatabasePath)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected debug, got %s", cfg.LogLevel)
	}
	if cfg.LibraryPath != "/tmp/lib" {
		t.Errorf("expected /tmp/lib, got %s", cfg.LibraryPath)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Ensure env is clean
	os.Unsetenv("KOPDS_PORT")
	os.Unsetenv("KOPDS_DATABASE_PATH")
	os.Unsetenv("KOPDS_LIBRARY_PATH")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected 8080 default, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected info default, got %s", cfg.LogLevel)
	}

	// Test absolute path resolution for default DatabasePath
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	expectedDBPath := filepath.Join(exeDir, "data", "kopds.db")
	if cfg.DatabasePath != expectedDBPath {
		t.Errorf("expected %s, got %s", expectedDBPath, cfg.DatabasePath)
	}
}

func TestPathResolution(t *testing.T) {
	os.Setenv("KOPDS_DATABASE_PATH", "my.db")
	os.Setenv("KOPDS_LOG_PATH", "my.log")
	os.Setenv("KOPDS_IMAGE_CACHE_PATH", "mycache")

	defer func() {
		os.Unsetenv("KOPDS_DATABASE_PATH")
		os.Unsetenv("KOPDS_LOG_PATH")
		os.Unsetenv("KOPDS_IMAGE_CACHE_PATH")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	if cfg.DatabasePath != filepath.Join(exeDir, "my.db") {
		t.Errorf("expected %s, got %s", filepath.Join(exeDir, "my.db"), cfg.DatabasePath)
	}
	if cfg.LogPath != filepath.Join(exeDir, "my.log") {
		t.Errorf("expected %s, got %s", filepath.Join(exeDir, "my.log"), cfg.LogPath)
	}
	if cfg.ImageCachePath != filepath.Join(exeDir, "mycache") {
		t.Errorf("expected %s, got %s", filepath.Join(exeDir, "mycache"), cfg.ImageCachePath)
	}
}
