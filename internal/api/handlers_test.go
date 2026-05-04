package api

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlafevers/kopds/internal/domain"
	"github.com/nlafevers/kopds/internal/opds"
	"github.com/nlafevers/kopds/internal/service"
	"github.com/nlafevers/kopds/pkg/utils"
)

func TestNavigationFeedHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	// BookService requires a repository, but NavigationFeedHandler doesn't use it yet.
	// So we can pass nil or a mock if needed.
	svc := service.NewBookService(nil, linkGen)
	h := NewHandler(svc, linkGen)

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

	// Check for expected links
	expectedRels := map[string]bool{
		"self":                         true,
		"subsection":                   true,
		"http://opds-spec.org/sort/new": true,
		"search":                       true,
	}

	foundRels := make(map[string]int)
	for _, link := range feed.Links {
		foundRels[link.Rel]++
	}

	for rel := range expectedRels {
		if foundRels[rel] == 0 {
			t.Errorf("missing expected link rel: %s", rel)
		}
	}
	
	if foundRels["subsection"] < 2 {
		t.Errorf("expected at least 2 subsection links (Authors, Series), got %d", foundRels["subsection"])
	}
}

type mockRepo struct {
	domain.BookRepository
	listAuthorsFunc  func(ctx context.Context, limit, offset int) ([]domain.AuthorWithCount, int, error)
	listSeriesFunc   func(ctx context.Context, limit, offset int) ([]domain.SeriesWithCount, int, error)
	listRecentFunc   func(ctx context.Context, limit, offset int) ([]domain.Book, int, error)
	listByAuthorFunc func(ctx context.Context, id int64, limit, offset int) ([]domain.Book, int, error)
	listBySeriesFunc func(ctx context.Context, id int64, limit, offset int) ([]domain.Book, int, error)
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
	h := NewHandler(svc, linkGen)

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
	h := NewHandler(svc, linkGen)

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
	h := NewHandler(svc, linkGen)

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
			return nil, 0, nil
		},
	}
	svc := service.NewBookService(repo, linkGen)
	h := NewHandler(svc, linkGen)

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
}

func TestOpenSearchDescriptorHandler(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	svc := service.NewBookService(nil, linkGen)
	h := NewHandler(svc, linkGen)

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
