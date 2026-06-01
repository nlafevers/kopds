package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db  *sql.DB
	log *slog.Logger
}

func OpenSQLite(path string, allowCreate bool) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if !allowCreate {
			return nil, fmt.Errorf("database file does not exist: %s", path)
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to create db file with 0600: %w", err)
		}
		file.Close()
	} else if err == nil {
		if err := os.Chmod(path, 0600); err != nil {
			return nil, fmt.Errorf("failed to chmod 0600 on existing db file: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Migrate applies the schema to the database.
func Migrate(db *sql.DB) error {
	var columnName string
	err := db.QueryRow("SELECT name FROM pragma_table_info('books') WHERE name='series_id'").Scan(&columnName)
	if err != nil && err != sql.ErrNoRows {
		// Ignore
	}
	if columnName == "" {
		_, _ = db.Exec("DROP TABLE IF EXISTS books")
		_, _ = db.Exec("DROP TABLE IF EXISTS books_search")
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS series (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS books (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			sort TEXT,
			author_sort TEXT,
			timestamp DATETIME,
			pub_date DATETIME,
			series_id INTEGER,
			series_index REAL DEFAULT 1,
			last_modified DATETIME,
			path TEXT NOT NULL,
			has_cover BOOLEAN DEFAULT 0,
			calibre_id INTEGER UNIQUE,
			description TEXT,
			FOREIGN KEY(series_id) REFERENCES series(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS authors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			sort TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sync_state (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS formats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			book_id INTEGER NOT NULL,
			format TEXT NOT NULL,
			uncompressed_size INTEGER,
			name TEXT,
			FOREIGN KEY(book_id) REFERENCES books(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS books_authors_link (
			book_id INTEGER,
			author_id INTEGER,
			PRIMARY KEY(book_id, author_id),
			FOREIGN KEY(book_id) REFERENCES books(id) ON DELETE CASCADE,
			FOREIGN KEY(author_id) REFERENCES authors(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS books_tags_link (
			book_id INTEGER,
			tag_id INTEGER,
			PRIMARY KEY(book_id, tag_id),
			FOREIGN KEY(book_id) REFERENCES books(id) ON DELETE CASCADE,
			FOREIGN KEY(tag_id) REFERENCES tags(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS books_search USING fts5(
			title,
			authors,
			series,
			tags
		);`,
	}

	for i, stmt := range statements {
		_, err = db.Exec(stmt)
		if err != nil {
			return fmt.Errorf("failed to apply migration statement %d: %w", i, err)
		}
	}

	return nil
}

func (s *Storage) logger() *slog.Logger {
	if s != nil && s.log != nil {
		return s.log
	}
	return slog.Default()
}

// EnforceStorageCap checks if the database file exceeds the size limit.
func (s *Storage) EnforceStorageCap(path string, capMB int) (bool, error) {
	log := s.logger()
	if capMB <= 0 {
		log.Debug("storage cap disabled, skipping enforcement", "database_path", path)
		return false, nil
	}

	log.Debug("checking storage cap", "database_path", path, "cap_mb", capMB)

	pruned, err := enforceStorageCap(path, capMB, s.pruneStorageCapRecords, s.vacuum)
	if err != nil {
		log.Error("failed to enforce storage cap", "database_path", path, "cap_mb", capMB, "error", err)
		return false, err
	}

	if pruned {
		log.Warn("storage cap exceeded", "database_path", path, "cap_mb", capMB)
		log.Info("storage cap enforced", "database_path", path, "cap_mb", capMB)
	}

	return pruned, nil
}

func (s *Storage) pruneStorageCapRecords() (int64, error) {
	log := s.logger()

	var rowCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sync_state").Scan(&rowCount)
	if err != nil {
		log.Error("failed to count sync state rows for pruning", "error", err)
		return 0, err
	}

	rowsTargeted := 0
	if rowCount > 0 {
		rowsTargeted = rowCount/5 + 1
	}

	if rowsTargeted > 0 {
		log.Warn("pruning storage cap records", "rows_targeted", rowsTargeted)
	}

	start := time.Now()
	res, err := s.db.Exec(`
		DELETE FROM sync_state
		WHERE key IN (
			SELECT key
			FROM sync_state
			ORDER BY key ASC
			LIMIT (SELECT COUNT(*) / 5 FROM sync_state) + 1
		)`)
	duration := time.Since(start)
	if err != nil {
		log.Error("failed to prune storage cap records", "rows_targeted", rowsTargeted, "duration", duration, "error", err)
		return 0, err
	}

	rowsDeleted, err := res.RowsAffected()
	if err != nil {
		log.Error("failed to count pruned rows", "rows_targeted", rowsTargeted, "duration", duration, "error", err)
		return 0, err
	}

	log.Info("storage cap records pruned", "rows_deleted", rowsDeleted, "rows_targeted", rowsTargeted, "duration", duration)
	return rowsDeleted, nil
}

func (s *Storage) vacuum() error {
	log := s.logger()
	start := time.Now()
	log.Debug("vacuuming database", "operation", "vacuum")

	_, err := s.db.Exec("VACUUM")
	duration := time.Since(start)
	if err != nil {
		log.Error("database vacuum failed", "operation", "vacuum", "duration", duration, "error", err)
		return err
	}

	log.Info("database vacuum completed", "operation", "vacuum", "duration", duration)
	return nil
}

func enforceStorageCap(path string, capMB int, prune func() (int64, error), vacuum func() error) (bool, error) {
	if capMB <= 0 {
		return false, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if info.Size() < int64(capMB)*1024*1024 {
		return false, nil
	}

	if _, err := prune(); err != nil {
		return false, err
	}

	if err := vacuum(); err != nil {
		return false, err
	}

	return true, nil
}
