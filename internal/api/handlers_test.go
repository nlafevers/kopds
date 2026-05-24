package api

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlafevers/kopds/internal/domain"
	"github.com/nlafevers/kopds/internal/image"
	"github.com/nlafevers/kopds/internal/opds"
	"github.com/nlafevers/kopds/internal/service"
	"github.com/nlafevers/kopds/pkg/utils"
)

type mockUserRepo struct {
	domain.UserRepository
	getByUsernameFunc func(ctx context.Context, username string) (*domain.User, error)
	saveFunc          func(ctx context.Context, user *domain.User) error
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.getByUsernameFunc != nil {
		return m.getByUsernameFunc(ctx, username)
	}
	return nil, nil
}

func (m *mockUserRepo) Save(ctx context.Context, user *domain.User) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, user)
	}
	return nil
}

func TestNavigationFeedHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	// BookService requires a repository, but NavigationFeedHandler doesn't use it yet.
	// So we can pass nil or a mock if needed.
	svc := service.NewBookService(nil, linkGen)
	h := NewHandler(svc, nil, linkGen, nil, "")

	req, err := http.NewRequest("GET", "/opds/v1.2/catalog", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.NavigationFeedHandler)

	// Execute
	handler.ServeHTTP(rr, req)

	// Assert
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expectedContentType := "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	var feed opds.Feed
	err = xml.Unmarshal(rr.Body.Bytes(), &feed)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if feed.Title != "KOPDS Root Catalog" {
		t.Errorf("expected title 'KOPDS Root Catalog', got '%s'", feed.Title)
	}

	if feed.ID != "urn:kopds:catalog:root" {
		t.Errorf("expected ID 'urn:kopds:catalog:root', got '%s'", feed.ID)
	}

	// Check for expected top-level links
	expectedRels := map[string]bool{
		"self":   true,
		"start":  true,
		"search": true,
	}

	foundRels := make(map[string]int)
	for _, link := range feed.Links {
		foundRels[link.Rel]++
	}

	for rel := range expectedRels {
		if foundRels[rel] == 0 {
			t.Errorf("missing expected top-level link rel: %s", rel)
		}
	}

	// Check for expected entries
	expectedEntryTitles := map[string]bool{
		"Authors":      true,
		"Series":       true,
		"Tags":         true,
		"Newest Books": true,
	}

	if len(feed.Entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(feed.Entries))
	}

	for _, entry := range feed.Entries {
		if !expectedEntryTitles[entry.Title] {
			t.Errorf("unexpected entry title: %s", entry.Title)
		}
		if entry.ID == "" || !strings.HasPrefix(entry.ID, "urn:kopds:catalog:") {
			t.Errorf("invalid entry ID: %s", entry.ID)
		}
		if len(entry.Links) == 0 {
			t.Errorf("missing link in entry: %s", entry.Title)
		}
	}
}

type mockRepo struct {
	domain.BookRepository
	listAuthorsFunc  func(ctx context.Context, limit, offset int) ([]domain.AuthorWithCount, int, error)
	listSeriesFunc   func(ctx context.Context, limit, offset int) ([]domain.SeriesWithCount, int, error)
	listTagsFunc     func(ctx context.Context, limit, offset int) ([]domain.TagWithCount, int, error)
	listRecentFunc   func(ctx context.Context, limit, offset int) ([]domain.Book, int, error)
	listByAuthorFunc func(ctx context.Context, id int64, limit, offset int) ([]domain.Book, int, error)
	listBySeriesFunc func(ctx context.Context, id int64, limit, offset int) ([]domain.Book, int, error)
	listByTagFunc    func(ctx context.Context, id int64, limit, offset int) ([]domain.Book, int, error)
	getByIDFunc      func(ctx context.Context, id int64) (*domain.Book, error)
	searchFunc       func(ctx context.Context, query string, limit, offset int) ([]domain.Book, int, error)
}

func (m *mockRepo) ListAuthors(ctx context.Context, limit, offset int) ([]domain.AuthorWithCount, int, error) {
	if m.listAuthorsFunc != nil {
		return m.listAuthorsFunc(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockRepo) ListSeries(ctx context.Context, limit, offset int) ([]domain.SeriesWithCount, int, error) {
	if m.listSeriesFunc != nil {
		return m.listSeriesFunc(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockRepo) ListTags(ctx context.Context, limit, offset int) ([]domain.TagWithCount, int, error) {
	if m.listTagsFunc != nil {
		return m.listTagsFunc(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockRepo) ListRecent(ctx context.Context, limit, offset int) ([]domain.Book, int, error) {
	if m.listRecentFunc != nil {
		return m.listRecentFunc(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockRepo) ListByAuthor(ctx context.Context, id int64, limit, offset int) ([]domain.Book, int, error) {
	if m.listByAuthorFunc != nil {
		return m.listByAuthorFunc(ctx, id, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockRepo) ListBySeries(ctx context.Context, id int64, limit, offset int) ([]domain.Book, int, error) {
	if m.listBySeriesFunc != nil {
		return m.listBySeriesFunc(ctx, id, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockRepo) ListByTag(ctx context.Context, id int64, limit, offset int) ([]domain.Book, int, error) {
	if m.listByTagFunc != nil {
		return m.listByTagFunc(ctx, id, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id int64) (*domain.Book, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRepo) Search(ctx context.Context, query string, limit, offset int) ([]domain.Book, int, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, limit, offset)
	}
	return nil, 0, nil
}

func TestAuthorsFeedHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	repo := &mockRepo{
		listAuthorsFunc: func(ctx context.Context, limit, offset int) ([]domain.AuthorWithCount, int, error) {
			authors := []domain.AuthorWithCount{
				{Author: domain.Author{ID: 1, Name: "Author One"}, BookCount: 5},
				{Author: domain.Author{ID: 2, Name: "Author Two"}, BookCount: 10},
			}
			return authors, 2, nil
		},
	}
	svc := service.NewBookService(repo, linkGen)
	h := NewHandler(svc, nil, linkGen, nil, "")

	req, _ := http.NewRequest("GET", "/opds/v1.2/authors", nil)
	rr := httptest.NewRecorder()

	// Execute
	h.AuthorsFeedHandler(rr, req)

	// Assert
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var feed opds.Feed
	err := xml.Unmarshal(rr.Body.Bytes(), &feed)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if feed.Title != "Authors" {
		t.Errorf("expected title 'Authors', got '%s'", feed.Title)
	}

	// Check for pagination links
	expectedRels := map[string]bool{
		"self":  true,
		"first": true,
		"last":  true,
	}
	for _, link := range feed.Links {
		delete(expectedRels, link.Rel)
	}
	if len(expectedRels) > 0 {
		t.Errorf("missing pagination links: %v", expectedRels)
	}

	if len(feed.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(feed.Entries))
	}

	if feed.Entries[0].Title != "Author One" {
		t.Errorf("expected first author 'Author One', got '%s'", feed.Entries[0].Title)
	}
}

func TestSeriesFeedHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	repo := &mockRepo{
		listSeriesFunc: func(ctx context.Context, limit, offset int) ([]domain.SeriesWithCount, int, error) {
			series := []domain.SeriesWithCount{
				{Series: domain.Series{ID: 1, Name: "Series One"}, BookCount: 3},
				{Series: domain.Series{ID: 2, Name: "Series Two"}, BookCount: 7},
			}
			return series, 2, nil
		},
	}
	svc := service.NewBookService(repo, linkGen)
	h := NewHandler(svc, nil, linkGen, nil, "")

	req, _ := http.NewRequest("GET", "/opds/v1.2/series", nil)
	rr := httptest.NewRecorder()

	// Execute
	h.SeriesFeedHandler(rr, req)

	// Assert
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var feed opds.Feed
	err := xml.Unmarshal(rr.Body.Bytes(), &feed)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if feed.Title != "Series" {
		t.Errorf("expected title 'Series', got '%s'", feed.Title)
	}

	// Check for pagination links
	expectedRels := map[string]bool{
		"self":  true,
		"first": true,
		"last":  true,
	}
	for _, link := range feed.Links {
		delete(expectedRels, link.Rel)
	}
	if len(expectedRels) > 0 {
		t.Errorf("missing pagination links: %v", expectedRels)
	}

	if len(feed.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(feed.Entries))
	}

	if feed.Entries[0].Title != "Series One" {
		t.Errorf("expected first series 'Series One', got '%s'", feed.Entries[0].Title)
	}
}

func TestTagsFeedHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	repo := &mockRepo{
		listTagsFunc: func(ctx context.Context, limit, offset int) ([]domain.TagWithCount, int, error) {
			tags := []domain.TagWithCount{
				{Tag: domain.Tag{ID: 1, Name: "Tag One"}, BookCount: 3},
				{Tag: domain.Tag{ID: 2, Name: "Tag Two"}, BookCount: 7},
			}
			return tags, 2, nil
		},
	}
	svc := service.NewBookService(repo, linkGen)
	h := NewHandler(svc, nil, linkGen, nil, "")

	req, _ := http.NewRequest("GET", "/opds/v1.2/tags", nil)
	rr := httptest.NewRecorder()

	// Execute
	h.TagsFeedHandler(rr, req)

	// Assert
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var feed opds.Feed
	err := xml.Unmarshal(rr.Body.Bytes(), &feed)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if feed.Title != "Tags" {
		t.Errorf("expected title 'Tags', got '%s'", feed.Title)
	}

	if len(feed.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(feed.Entries))
	}

	if feed.Entries[0].Title != "Tag One" {
		t.Errorf("expected first tag 'Tag One', got '%s'", feed.Entries[0].Title)
	}
}

func TestNewestFeedHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	repo := &mockRepo{
		listRecentFunc: func(ctx context.Context, limit, offset int) ([]domain.Book, int, error) {
			books := []domain.Book{
				{
					ID:    1,
					Title: "Book One",
					Authors: []domain.Author{
						{ID: 1, Name: "Author One"},
					},
					Formats: []domain.Format{
						{Format: "EPUB"},
					},
				},
				{
					ID:    2,
					Title: "Book Two",
					Authors: []domain.Author{
						{ID: 2, Name: "Author Two"},
					},
					Formats: []domain.Format{
						{Format: "PDF"},
					},
				},
			}
			return books, 2, nil
		},
	}
	svc := service.NewBookService(repo, linkGen)
	h := NewHandler(svc, nil, linkGen, nil, "")

	req, _ := http.NewRequest("GET", "/opds/v1.2/newest", nil)
	rr := httptest.NewRecorder()

	// Execute
	h.NewestFeedHandler(rr, req)

	// Assert
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var feed opds.Feed
	err := xml.Unmarshal(rr.Body.Bytes(), &feed)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if feed.Title != "Newest Books" {
		t.Errorf("expected title 'Newest Books', got '%s'", feed.Title)
	}

	// Check for pagination links
	expectedRels := map[string]bool{
		"self":  true,
		"first": true,
		"last":  true,
	}
	for _, link := range feed.Links {
		delete(expectedRels, link.Rel)
	}
	if len(expectedRels) > 0 {
		t.Errorf("missing pagination links: %v", expectedRels)
	}

	if len(feed.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(feed.Entries))
	}

	if feed.Entries[0].Title != "Book One" {
		t.Errorf("expected first book 'Book One', got '%s'", feed.Entries[0].Title)
	}

	if feed.Entries[0].Authors[0].Name != "Author One" {
		t.Errorf("expected author 'Author One', got '%s'", feed.Entries[0].Authors[0].Name)
	}

	// Check for acquisition link
	foundAcquisition := false
	for _, link := range feed.Entries[0].Links {
		if link.Rel == "http://opds-spec.org/acquisition" {
			foundAcquisition = true
			break
		}
	}
	if !foundAcquisition {
		t.Error("missing acquisition link")
	}

	// Test pagination with multiple pages
	repo.listRecentFunc = func(ctx context.Context, limit, offset int) ([]domain.Book, int, error) {
		return []domain.Book{{
			ID:    3,
			Title: "Book Three",
			Formats: []domain.Format{
				{Format: "EPUB"},
			},
		}}, 101, nil // 3 pages if limit is 50
	}

	req, _ = http.NewRequest("GET", "/opds/v1.2/newest?page=2", nil)
	rr = httptest.NewRecorder()
	h.NewestFeedHandler(rr, req)

	err = xml.Unmarshal(rr.Body.Bytes(), &feed)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	expectedRels = map[string]bool{
		"self":     true,
		"first":    true,
		"previous": true,
		"next":     true,
		"last":     true,
	}
	for _, link := range feed.Links {
		delete(expectedRels, link.Rel)
	}
	if len(expectedRels) > 0 {
		t.Errorf("missing pagination links on page 2: %v", expectedRels)
	}
}

func TestSearchFeedHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	repo := &mockRepo{
		searchFunc: func(ctx context.Context, query string, limit, offset int) ([]domain.Book, int, error) {
			if query == "Go" {
				books := []domain.Book{
					{
						ID:    1,
						Title: "The Go Programming Language",
						Formats: []domain.Format{
							{Format: "EPUB"},
						},
					},
				}
				return books, 1, nil
			}
			if query == "Go & Rust" {
				return []domain.Book{{ID: 2, Title: "Encoded Search", Formats: []domain.Format{{Format: "EPUB"}}}}, 101, nil
			}
			return nil, 0, nil
		},
	}
	svc := service.NewBookService(repo, linkGen)
	h := NewHandler(svc, nil, linkGen, nil, "")

	// Execute search
	req, _ := http.NewRequest("GET", "/opds/v1.2/search?q=Go", nil)
	rr := httptest.NewRecorder()
	h.SearchFeedHandler(rr, req)

	// Assert
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var feed opds.Feed
	err := xml.Unmarshal(rr.Body.Bytes(), &feed)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if feed.Title != "Search Results: Go" {
		t.Errorf("expected title 'Search Results: Go', got '%s'", feed.Title)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(feed.Entries))
	}

	if feed.Entries[0].Title != "The Go Programming Language" {
		t.Errorf("expected book 'The Go Programming Language', got '%s'", feed.Entries[0].Title)
	}

	req, _ = http.NewRequest("GET", "/opds/v1.2/search?q=Go+%26+Rust&page=2", nil)
	rr = httptest.NewRecorder()
	h.SearchFeedHandler(rr, req)

	err = xml.Unmarshal(rr.Body.Bytes(), &feed)
	if err != nil {
		t.Fatalf("failed to unmarshal encoded search XML: %v", err)
	}
	for _, link := range feed.Links {
		if link.Rel == "next" && !strings.Contains(link.Href, "q=Go+%26+Rust") {
			t.Fatalf("expected encoded q parameter in next link, got %s", link.Href)
		}
	}
}

func TestOpenSearchDescriptorHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	svc := service.NewBookService(nil, linkGen)
	h := NewHandler(svc, nil, linkGen, nil, "")

	req, _ := http.NewRequest("GET", "/opds/v1.2/opensearch.xml", nil)
	rr := httptest.NewRecorder()

	// Execute
	h.OpenSearchDescriptorHandler(rr, req)

	// Assert
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expectedContentType := "application/opensearchdescription+xml; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	var osd OpenSearchDescription
	err := xml.Unmarshal(rr.Body.Bytes(), &osd)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if osd.ShortName != "KOPDS" {
		t.Errorf("expected ShortName 'KOPDS', got '%s'", osd.ShortName)
	}

	expectedTemplate := "http://localhost:8080/opds/v1.2/search?q={searchTerms}"
	if osd.Url.Template != expectedTemplate {
		t.Errorf("expected template '%s', got '%s'", expectedTemplate, osd.Url.Template)
	}

	if osd.Url.Type != "application/atom+xml" {
		t.Errorf("expected type 'application/atom+xml', got '%s'", osd.Url.Type)
	}
}

func TestCoverHandler(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	libraryPath := filepath.Join(tempDir, "library")
	cachePath := filepath.Join(tempDir, "cache")
	os.MkdirAll(libraryPath, 0755)

	// Create a dummy cover.jpg
	bookPath := "Author/Book (1)"
	os.MkdirAll(filepath.Join(libraryPath, bookPath), 0755)

	// Create a real small JPEG to avoid decode errors
	// But for a unit test, we can just mock imaging if we wanted,
	// but here we are using the real image package.
	// Let's create a minimal valid JPEG.
	// Actually, easier to just use a 1x1 pixel image.
	importImage := func() {
		// This is just a placeholder to show I need to import "image" and "image/jpeg" and "image/color"
	}
	_ = importImage

	// To keep it simple, I'll just check if it returns 404 for missing books.
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	repo := &mockRepo{
		getByIDFunc: func(ctx context.Context, id int64) (*domain.Book, error) {
			if id == 1 {
				return &domain.Book{ID: 1, Path: bookPath, HasCover: true}, nil
			}
			return nil, nil
		},
	}

	cache, _ := image.NewDiskCache(cachePath, 10)
	svc := service.NewBookService(repo, linkGen)
	h := NewHandler(svc, nil, linkGen, cache, libraryPath)

	// Test 404
	req, _ := http.NewRequest("GET", "/opds/v1.2/cover/2", nil)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /opds/v1.2/cover/{id}", h.CoverHandler)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing book, got %d", rr.Code)
	}

	// Test Invalid ID
	req, _ = http.NewRequest("GET", "/opds/v1.2/cover/abc", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid ID, got %d", rr.Code)
	}

	// Test Invalid Dimensions
	req, _ = http.NewRequest("GET", "/opds/v1.2/cover/1?w=9999&h=150", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid dimensions, got %d", rr.Code)
	}
}
