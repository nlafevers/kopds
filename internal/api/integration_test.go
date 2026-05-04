package api

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nlafevers/kopds/internal/domain"
	"github.com/nlafevers/kopds/internal/opds"
	"github.com/nlafevers/kopds/internal/service"
	"github.com/nlafevers/kopds/pkg/utils"
)

func TestOPDSIntegration(t *testing.T) {
	// Setup
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	repo := &mockRepo{
		listAuthorsFunc: func(ctx context.Context, limit, offset int) ([]domain.AuthorWithCount, int, error) {
			return []domain.AuthorWithCount{{Author: domain.Author{ID: 1, Name: "Test Author"}, BookCount: 1}}, 1, nil
		},
		listSeriesFunc: func(ctx context.Context, limit, offset int) ([]domain.SeriesWithCount, int, error) {
			return []domain.SeriesWithCount{{Series: domain.Series{ID: 1, Name: "Test Series"}, BookCount: 1}}, 1, nil
		},
		listRecentFunc: func(ctx context.Context, limit, offset int) ([]domain.Book, int, error) {
			return []domain.Book{{ID: 1, Title: "Test Book", Formats: []domain.Format{{Format: "EPUB"}}}}, 1, nil
		},
		searchFunc: func(ctx context.Context, query string, limit, offset int) ([]domain.Book, int, error) {
			return []domain.Book{{ID: 1, Title: "Search Book", Formats: []domain.Format{{Format: "EPUB"}}}}, 1, nil
		},
		getByIDFunc: func(ctx context.Context, id int64) (*domain.Book, error) {
			return &domain.Book{ID: id, Title: "Detail Book", Formats: []domain.Format{{Format: "EPUB"}}}, nil
		},
	}
	svc := service.NewBookService(repo, linkGen)
	h := NewHandler(svc, linkGen)

	r := chi.NewRouter()
	r.Route("/opds/v1.2", func(r chi.Router) {
		r.Get("/catalog", h.NavigationFeedHandler)
		r.Get("/authors", h.AuthorsFeedHandler)
		r.Get("/series", h.SeriesFeedHandler)
		r.Get("/newest", h.NewestFeedHandler)
		r.Get("/books/{id}", h.BookDetailHandler)
		r.Get("/search", h.SearchFeedHandler)
		r.Get("/opensearch.xml", h.OpenSearchDescriptorHandler)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	endpoints := []struct {
		name        string
		path        string
		contentType string
	}{
		{"Root", "/opds/v1.2/catalog", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8"},
		{"Authors", "/opds/v1.2/authors", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8"},
		{"Series", "/opds/v1.2/series", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8"},
		{"Newest", "/opds/v1.2/newest", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8"},
		{"Detail", "/opds/v1.2/books/1", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8"},
		{"Search", "/opds/v1.2/search?q=test", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8"},
		{"OSD", "/opds/v1.2/opensearch.xml", "application/opensearchdescription+xml; charset=utf-8"},
	}

	for _, tc := range endpoints {
		t.Run(tc.name, func(t *testing.T) {
			res, err := http.Get(ts.URL + tc.path)
			if err != nil {
				t.Fatalf("failed to GET %s: %v", tc.path, err)
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				t.Errorf("expected OK status for %s, got %d", tc.path, res.StatusCode)
			}

			if ct := res.Header.Get("Content-Type"); ct != tc.contentType {
				t.Errorf("expected Content-Type %s for %s, got %s", tc.contentType, tc.path, ct)
			}

			// Partial semantic validation: check if it's valid XML
			if tc.name != "OSD" {
				var feed opds.Feed
				if err := xml.NewDecoder(res.Body).Decode(&feed); err != nil {
					t.Errorf("failed to decode XML for %s: %v", tc.path, err)
				}
				if feed.ID == "" {
					t.Errorf("feed ID is empty for %s", tc.path)
				}
			} else {
				var osd OpenSearchDescription
				if err := xml.NewDecoder(res.Body).Decode(&osd); err != nil {
					t.Errorf("failed to decode OSD XML: %v", err)
				}
			}
		})
	}
}
