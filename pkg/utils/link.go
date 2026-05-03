package utils

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// LinkGenerator centralizes URL construction for OPDS feeds.
type LinkGenerator struct {
	baseURL string
}

// NewLinkGenerator creates a new LinkGenerator with the given base URL.
func NewLinkGenerator(baseURL string) *LinkGenerator {
	return &LinkGenerator{
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// buildURL is a helper to construct URLs with an optional page parameter.
func (lg *LinkGenerator) buildURL(path string, page int) string {
	fullURL := lg.baseURL + path
	if page > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return fullURL // Should not happen with valid paths
		}
		q := u.Query()
		q.Set("page", strconv.Itoa(page))
		u.RawQuery = q.Encode()
		return u.String()
	}
	return fullURL
}

// RootCatalog returns the URL for the root OPDS catalog.
func (lg *LinkGenerator) RootCatalog() string {
	return lg.buildURL("/opds/v1.2/catalog", 0)
}

// AuthorsList returns the URL for the authors list, optionally paginated.
func (lg *LinkGenerator) AuthorsList(page int) string {
	return lg.buildURL("/opds/v1.2/authors", page)
}

// AuthorDetail returns the URL for a specific author's books, optionally paginated.
func (lg *LinkGenerator) AuthorDetail(id string, page int) string {
	return lg.buildURL(fmt.Sprintf("/opds/v1.2/authors/%s", id), page)
}

// SeriesList returns the URL for the series list, optionally paginated.
func (lg *LinkGenerator) SeriesList(page int) string {
	return lg.buildURL("/opds/v1.2/series", page)
}

// SeriesDetail returns the URL for a specific series' books, optionally paginated.
func (lg *LinkGenerator) SeriesDetail(id string, page int) string {
	return lg.buildURL(fmt.Sprintf("/opds/v1.2/series/%s", id), page)
}

// NewestBooks returns the URL for the newest books feed, optionally paginated.
func (lg *LinkGenerator) NewestBooks(page int) string {
	return lg.buildURL("/opds/v1.2/newest", page)
}

// BookDetail returns the URL for a specific book's acquisition entry.
func (lg *LinkGenerator) BookDetail(id string) string {
	return lg.buildURL(fmt.Sprintf("/opds/v1.2/books/%s", id), 0)
}

// Search returns the URL for the search endpoint, optionally paginated.
func (lg *LinkGenerator) Search(page int) string {
	return lg.buildURL("/opds/v1.2/search", page)
}

// OpenSearchDescriptor returns the URL for the OpenSearch descriptor XML.
func (lg *LinkGenerator) OpenSearchDescriptor() string {
	return lg.buildURL("/opds/v1.2/opensearch.xml", 0)
}
