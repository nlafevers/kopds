package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// NewSQLite creates a new SQLite database connection.
func NewSQLite(path string) (*sql.DB, error) {
	// DSN with performance pragmas
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-2000)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Migrate applies the schema to the database.
// To handle schema evolution, we drop the 'books' table if it doesn't match our latest schema,
// or simply ensure we can run migrations.
func Migrate(db *sql.DB) error {
	// Check if the 'books' table has the 'series_id' column.
	// If not, we drop it to re-create it with the correct schema.
	var columnName string
	err := db.QueryRow("SELECT name FROM pragma_table_info('books') WHERE name='series_id'").Scan(&columnName)
	if err != nil || columnName == "" {
		_, _ = db.Exec("DROP TABLE IF EXISTS books")
		_, _ = db.Exec("DROP TABLE IF EXISTS books_search")
	}

	schema := `
	CREATE TABLE IF NOT EXISTS books (
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
	);

	CREATE TABLE IF NOT EXISTS authors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		sort TEXT
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS series (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sync_state (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE TABLE IF NOT EXISTS formats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL,
		format TEXT NOT NULL,
		uncompressed_size INTEGER,
		name TEXT,
		FOREIGN KEY(book_id) REFERENCES books(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS books_authors_link (
		book_id INTEGER,
		author_id INTEGER,
		PRIMARY KEY(book_id, author_id),
		FOREIGN KEY(book_id) REFERENCES books(id) ON DELETE CASCADE,
		FOREIGN KEY(author_id) REFERENCES authors(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS books_tags_link (
		book_id INTEGER,
		tag_id INTEGER,
		PRIMARY KEY(book_id, tag_id),
		FOREIGN KEY(book_id) REFERENCES books(id) ON DELETE CASCADE,
		FOREIGN KEY(tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);

	-- FTS5 Search Table
	CREATE VIRTUAL TABLE IF NOT EXISTS books_search USING fts5(
		title,
		authors,
		series,
		tags
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	return nil
}
