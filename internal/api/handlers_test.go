package api

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

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
