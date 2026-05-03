package utils

import (
	"testing"
)

func TestLinkGenerator(t *testing.T) {
	baseURL := "http://localhost:8080"
	lg := NewLinkGenerator(baseURL)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"RootCatalog", lg.RootCatalog(), "http://localhost:8080/opds/v1.2/catalog"},
		{"AuthorsList", lg.AuthorsList(0), "http://localhost:8080/opds/v1.2/authors"},
		{"AuthorsList Page 1", lg.AuthorsList(1), "http://localhost:8080/opds/v1.2/authors?page=1"},
		{"AuthorDetail", lg.AuthorDetail("123", 0), "http://localhost:8080/opds/v1.2/authors/123"},
		{"AuthorDetail Page 2", lg.AuthorDetail("123", 2), "http://localhost:8080/opds/v1.2/authors/123?page=2"},
		{"SeriesList", lg.SeriesList(0), "http://localhost:8080/opds/v1.2/series"},
		{"SeriesDetail", lg.SeriesDetail("456", 0), "http://localhost:8080/opds/v1.2/series/456"},
		{"NewestBooks", lg.NewestBooks(5), "http://localhost:8080/opds/v1.2/newest?page=5"},
		{"BookDetail", lg.BookDetail("789"), "http://localhost:8080/opds/v1.2/books/789"},
		{"Search", lg.Search(1), "http://localhost:8080/opds/v1.2/search?page=1"},
		{"OpenSearchDescriptor", lg.OpenSearchDescriptor(), "http://localhost:8080/opds/v1.2/opensearch.xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %s, expected %s", tt.got, tt.expected)
			}
		})
	}
}

func TestLinkGeneratorTrailingSlash(t *testing.T) {
	baseURL := "http://localhost:8080/"
	lg := NewLinkGenerator(baseURL)
	got := lg.RootCatalog()
	expected := "http://localhost:8080/opds/v1.2/catalog"
	if got != expected {
		t.Errorf("got %s, expected %s", got, expected)
	}
}
