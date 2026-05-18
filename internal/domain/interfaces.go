package domain

import "context"

// BookRepository defines the interface for book data access.
type BookRepository interface {
	GetByID(ctx context.Context, id int64) (*Book, error)
	Search(ctx context.Context, query string, limit, offset int) ([]Book, int, error)
	ListRecent(ctx context.Context, limit, offset int) ([]Book, int, error)
	ListByAuthor(ctx context.Context, authorID int64, limit, offset int) ([]Book, int, error)
	ListBySeries(ctx context.Context, seriesID int64, limit, offset int) ([]Book, int, error)
	ListAuthors(ctx context.Context, limit, offset int) ([]AuthorWithCount, int, error)
	ListSeries(ctx context.Context, limit, offset int) ([]SeriesWithCount, int, error)
	ListTags(ctx context.Context, limit, offset int) ([]TagWithCount, int, error)
	ListByTag(ctx context.Context, tagID int64, limit, offset int) ([]Book, int, error)
	Upsert(ctx context.Context, book *Book) error
	PruneMissingCalibreIDs(ctx context.Context, keepIDs []int64) (int64, error)
	GetSyncState(ctx context.Context, key string) (string, error)
	SetSyncState(ctx context.Context, key, value string) error
	EnforceStorageCap(ctx context.Context, path string, capMB int) (bool, error)
}

// UserRepository defines the interface for user data access.
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*User, error)
	Save(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, username string) error
	UpdatePassword(ctx context.Context, username, password string) error
}

// Indexer defines the interface for the background synchronization engine.
type Indexer interface {
	Sync(ctx context.Context) error
}
