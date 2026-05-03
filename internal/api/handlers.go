package api

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"

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

// AuthorsFeedHandler returns a paginated list of authors in the OPDS catalog.
func (h *Handler) AuthorsFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	authors, total, err := h.BookService.GetAuthors(r.Context(), page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := (total + service.DefaultPageSize - 1) / service.DefaultPageSize
	if lastPage == 0 {
		lastPage = 1
	}

	links := []opds.Link{
		{
			Rel:   "self",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.AuthorsList(page),
			Title: "Authors",
		},
		{
			Rel:   "first",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.AuthorsList(1),
			Title: "First Page",
		},
	}

	if page > 1 {
		links = append(links, opds.Link{
			Rel:   "previous",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.AuthorsList(page - 1),
			Title: "Previous Page",
		})
	}

	if page < lastPage {
		links = append(links, opds.Link{
			Rel:   "next",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.AuthorsList(page + 1),
			Title: "Next Page",
		})
	}

	links = append(links, opds.Link{
		Rel:   "last",
		Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
		Href:  h.LinkGenerator.AuthorsList(lastPage),
		Title: "Last Page",
	})

	feed := opds.NewFeed("Authors", "authors-list", links)

	for _, author := range authors {
		summary := fmt.Sprintf("%d books", author.BookCount)
		entry := &opds.Entry{
			ID:    fmt.Sprintf("author:%d", author.ID),
			Title: author.Name,
			Summary: &opds.Content{
				Text: summary,
			},
			Links: []opds.Link{
				{
					Rel:   "subsection",
					Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
					Href:  h.LinkGenerator.AuthorDetail(strconv.FormatInt(author.ID, 10), 0),
					Title: author.Name,
				},
			},
		}
		feed.Entries = append(feed.Entries, entry)
	}

	w.Header().Set("Content-Type", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		http.Error(w, "Failed to encode feed", http.StatusInternalServerError)
	}
}

// SeriesFeedHandler returns a paginated list of series in the OPDS catalog.
func (h *Handler) SeriesFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	series, total, err := h.BookService.GetSeries(r.Context(), page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := (total + service.DefaultPageSize - 1) / service.DefaultPageSize
	if lastPage == 0 {
		lastPage = 1
	}

	links := []opds.Link{
		{
			Rel:   "self",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.SeriesList(page),
			Title: "Series",
		},
		{
			Rel:   "first",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.SeriesList(1),
			Title: "First Page",
		},
	}

	if page > 1 {
		links = append(links, opds.Link{
			Rel:   "previous",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.SeriesList(page - 1),
			Title: "Previous Page",
		})
	}

	if page < lastPage {
		links = append(links, opds.Link{
			Rel:   "next",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.SeriesList(page + 1),
			Title: "Next Page",
		})
	}

	links = append(links, opds.Link{
		Rel:   "last",
		Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
		Href:  h.LinkGenerator.SeriesList(lastPage),
		Title: "Last Page",
	})

	feed := opds.NewFeed("Series", "series-list", links)

	for _, s := range series {
		summary := fmt.Sprintf("%d books", s.BookCount)
		entry := &opds.Entry{
			ID:    fmt.Sprintf("series:%d", s.ID),
			Title: s.Name,
			Summary: &opds.Content{
				Text: summary,
			},
			Links: []opds.Link{
				{
					Rel:   "subsection",
					Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
					Href:  h.LinkGenerator.SeriesDetail(strconv.FormatInt(s.ID, 10), 0),
					Title: s.Name,
				},
			},
		}
		feed.Entries = append(feed.Entries, entry)
	}

	w.Header().Set("Content-Type", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		http.Error(w, "Failed to encode feed", http.StatusInternalServerError)
	}
}

// NewestFeedHandler returns a paginated list of the newest books in the OPDS catalog.
func (h *Handler) NewestFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	books, total, err := h.BookService.GetRecentBooks(r.Context(), page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := (total + service.DefaultPageSize - 1) / service.DefaultPageSize
	if lastPage == 0 {
		lastPage = 1
	}

	links := []opds.Link{
		{
			Rel:   "self",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.NewestBooks(page),
			Title: "Newest Books",
		},
		{
			Rel:   "first",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.NewestBooks(1),
			Title: "First Page",
		},
	}

	if page > 1 {
		links = append(links, opds.Link{
			Rel:   "previous",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.NewestBooks(page - 1),
			Title: "Previous Page",
		})
	}

	if page < lastPage {
		links = append(links, opds.Link{
			Rel:   "next",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  h.LinkGenerator.NewestBooks(page + 1),
			Title: "Next Page",
		})
	}

	links = append(links, opds.Link{
		Rel:   "last",
		Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
		Href:  h.LinkGenerator.NewestBooks(lastPage),
		Title: "Last Page",
	})

	feed := opds.NewFeed("Newest Books", "newest-list", links)

	for _, book := range books {
		entry := &opds.Entry{
			ID:      fmt.Sprintf("book:%d", book.ID),
			Title:   book.Title,
			Updated: book.LastModified,
			Summary: &opds.Content{
				Text: book.Description,
			},
		}

		for _, author := range book.Authors {
			entry.Authors = append(entry.Authors, opds.Author{
				Name: author.Name,
			})
		}

		// Acquisition links (Step 3.7 will enhance this)
		entry.Links = append(entry.Links, opds.Link{
			Rel:   "http://opds-spec.org/acquisition",
			Type:  "application/epub+zip", // Defaulting to epub for now
			Href:  h.LinkGenerator.BookDetail(strconv.FormatInt(book.ID, 10)),
			Title: "Download",
		})

		feed.Entries = append(feed.Entries, entry)
	}

	w.Header().Set("Content-Type", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		http.Error(w, "Failed to encode feed", http.StatusInternalServerError)
	}
}

