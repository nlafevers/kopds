package service

import (
	"context"

	"github.com/nlafevers/kopds/internal/domain"
	"github.com/nlafevers/kopds/pkg/utils"
)

const DefaultPageSize = 50

// BookService handles business logic for books and library navigation.
type BookService struct {
	repo          domain.BookRepository
	linkGenerator *utils.LinkGenerator
}

// NewBookService creates a new BookService.
func NewBookService(repo domain.BookRepository, linkGenerator *utils.LinkGenerator) *BookService {
	return &BookService{
		repo:          repo,
		linkGenerator: linkGenerator,
	}
}

func (s *BookService) GetRecentBooks(ctx context.Context, page int) ([]domain.Book, int, error) {
	limit := DefaultPageSize
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListRecent(ctx, limit, offset)
}

func (s *BookService) GetAuthors(ctx context.Context, page int) ([]domain.AuthorWithCount, int, error) {
	limit := DefaultPageSize
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListAuthors(ctx, limit, offset)
}

func (s *BookService) GetSeries(ctx context.Context, page int) ([]domain.SeriesWithCount, int, error) {
	limit := DefaultPageSize
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListSeries(ctx, limit, offset)
}

func (s *BookService) GetTags(ctx context.Context, page int) ([]domain.TagWithCount, int, error) {
	limit := DefaultPageSize
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListTags(ctx, limit, offset)
}

func (s *BookService) GetBooksByAuthor(ctx context.Context, authorID int64, page int) ([]domain.Book, int, error) {
	limit := DefaultPageSize
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByAuthor(ctx, authorID, limit, offset)
}

func (s *BookService) GetBooksBySeries(ctx context.Context, seriesID int64, page int) ([]domain.Book, int, error) {
	limit := DefaultPageSize
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListBySeries(ctx, seriesID, limit, offset)
}

func (s *BookService) GetBooksByTag(ctx context.Context, tagID int64, page int) ([]domain.Book, int, error) {
	limit := DefaultPageSize
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByTag(ctx, tagID, limit, offset)
}

func (s *BookService) SearchBooks(ctx context.Context, query string, page int) ([]domain.Book, int, error) {
	limit := DefaultPageSize
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.repo.Search(ctx, query, limit, offset)
}

func (s *BookService) GetBookByID(ctx context.Context, id int64) (*domain.Book, error) {
	return s.repo.GetByID(ctx, id)
}

// GetLinkGenerator returns the link generator for use in handlers.
func (s *BookService) GetLinkGenerator() *utils.LinkGenerator {
	return s.linkGenerator
}
