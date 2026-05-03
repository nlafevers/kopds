package domain

import (
	"time"
)

// Book represents a book in the library.
type Book struct {
	ID             int64     `json:"id" db:"id"`
	UUID           string    `json:"uuid" db:"uuid"`
	Title          string    `json:"title" db:"title"`
	Sort           string    `json:"sort" db:"sort"`
	AuthorSort     string    `json:"author_sort" db:"author_sort"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
	PubDate        time.Time `json:"pub_date" db:"pub_date"`
	SeriesIndex    float64   `json:"series_index" db:"series_index"`
	LastModified   time.Time `json:"last_modified" db:"last_modified"`
	Path           string    `json:"path" db:"path"`
	HasCover       bool      `json:"has_cover" db:"has_cover"`
	CalibreID      int64     `json:"calibre_id" db:"calibre_id"`
	Description    string    `json:"description" db:"description"`
	Authors        []Author  `json:"authors"`
	Tags           []Tag     `json:"tags"`
	Series         *Series   `json:"series,omitempty"`
	Formats        []Format  `json:"formats"`
}

// Author represents a book author.
type Author struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
	Sort string `json:"sort" db:"sort"`
}

// AuthorWithCount represents an author with the number of books they have.
type AuthorWithCount struct {
	Author
	BookCount int `json:"book_count"`
}

// Tag represents a book tag/category.
type Tag struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// Series represents a book series.
type Series struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// SeriesWithCount represents a series with the number of books in it.
type SeriesWithCount struct {
	Series
	BookCount int `json:"book_count"`
}

// Format represents a physical file format of a book.
type Format struct {
	ID               int64  `json:"id" db:"id"`
	Format           string `json:"format" db:"format"`
	UncompressedSize int64  `json:"uncompressed_size" db:"uncompressed_size"`
	Name             string `json:"name" db:"name"`
}

// User represents a system user.
type User struct {
	ID       int64  `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Password string `json:"-" db:"password"` // Salted hash
}
