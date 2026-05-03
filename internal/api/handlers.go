package api

import (
	"encoding/xml"
	"net/http"

	"github.com/nlafevers/kopds/internal/opds"
	"github.com/nlafevers/kopds/internal/service"
	"github.com/nlafevers/kopds/pkg/utils"
)

// Handler handles HTTP requests for the OPDS API.
type Handler struct {
	BookService   *service.BookService
	LinkGenerator *utils.LinkGenerator
}

// NewHandler creates a new Handler.
func NewHandler(bookService *service.BookService, linkGenerator *utils.LinkGenerator) *Handler {
	return &Handler{
		BookService:   bookService,
		LinkGenerator: linkGenerator,
	}
}

// NavigationFeedHandler returns the root navigation feed for the OPDS catalog.
func (h *Handler) NavigationFeedHandler(w http.ResponseWriter, r *http.Request) {
	links := []opds.Link{
		{
			Rel:   "self",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.RootCatalog(),
			Title: "Root Catalog",
		},
		{
			Rel:   "subsection",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.AuthorsList(0),
			Title: "Authors",
		},
		{
			Rel:   "subsection",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.SeriesList(0),
			Title: "Series",
		},
		{
			Rel:   "http://opds-spec.org/sort/new",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.NewestBooks(0),
			Title: "Newest Books",
		},
		{
			Rel:   "search",
			Type:  "application/opensearchdescription+xml",
			Href:  h.LinkGenerator.OpenSearchDescriptor(),
			Title: "Search",
		},
	}

	feed := opds.NewFeed("KOPDS Root Catalog", "root-catalog", links)

	w.Header().Set("Content-Type", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		// Log error and return internal server error
		http.Error(w, "Failed to encode feed", http.StatusInternalServerError)
	}
}
