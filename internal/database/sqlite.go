package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
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

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;"); err != nil {
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

// NewStorage creates a new storage wrapper.
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// NewSQLite creates a new SQLite database connection.
func NewSQLite(path string) (*sql.DB, error) {
	return OpenSQLite(path, true)
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

// EnforceStorageCap checks if the database file exceeds the size limit.
func (s *Storage) EnforceStorageCap(path string, capMB int) (bool, error) {
	return enforceStorageCap(path, capMB, func() error {
		// Delete oldest 20% of sync state records (as a proxy for progress/old entries).
		_, err := s.db.Exec(`
		DELETE FROM sync_state
		WHERE key IN (
			SELECT key
			FROM sync_state
			ORDER BY key ASC
			LIMIT (SELECT COUNT(*) / 5 FROM sync_state) + 1
		)`)
		return err
	}, func() error {
		_, err := s.db.Exec("VACUUM")
		return err
	})
}

func enforceStorageCap(path string, capMB int, prune func() error, vacuum func() error) (bool, error) {
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

	if err := prune(); err != nil {
		return false, err
	}

	if err := vacuum(); err != nil {
		return false, err
	}

	return true, nil
}
