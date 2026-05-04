package api

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nlafevers/kopds/internal/domain"
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
		http.Error(w, "Failed to encode feed", http.StatusInternalServerError)
	}
}

// AuthorsFeedHandler returns a paginated list of authors in the OPDS catalog.
func (h *Handler) AuthorsFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := getPage(r)
	authors, total, err := h.BookService.GetAuthors(r.Context(), page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	links := h.generatePaginationLinks(h.LinkGenerator.AuthorsList, page, lastPage, "Authors")
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

	h.sendFeed(w, feed)
}

// SeriesFeedHandler returns a paginated list of series in the OPDS catalog.
func (h *Handler) SeriesFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := getPage(r)
	series, total, err := h.BookService.GetSeries(r.Context(), page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	links := h.generatePaginationLinks(h.LinkGenerator.SeriesList, page, lastPage, "Series")
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

	h.sendFeed(w, feed)
}

// NewestFeedHandler returns a paginated list of the newest books in the OPDS catalog.
func (h *Handler) NewestFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := getPage(r)
	books, total, err := h.BookService.GetRecentBooks(r.Context(), page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	links := h.generatePaginationLinks(h.LinkGenerator.NewestBooks, page, lastPage, "Newest Books")
	feed := opds.NewFeed("Newest Books", "newest-list", links)

	h.appendBookEntries(&feed, books)
	h.sendFeed(w, feed)
}

// AuthorBooksHandler returns a paginated list of books by a specific author.
func (h *Handler) AuthorBooksHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	page := getPage(r)

	books, total, err := h.BookService.GetBooksByAuthor(r.Context(), id, page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	linkFunc := func(p int) string { return h.LinkGenerator.AuthorDetail(idStr, p) }
	links := h.generatePaginationLinks(linkFunc, page, lastPage, "Books by Author")
	feed := opds.NewFeed("Books by Author", "author-books-"+idStr, links)

	h.appendBookEntries(&feed, books)
	h.sendFeed(w, feed)
}

// SeriesBooksHandler returns a paginated list of books in a specific series.
func (h *Handler) SeriesBooksHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	page := getPage(r)

	books, total, err := h.BookService.GetBooksBySeries(r.Context(), id, page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	linkFunc := func(p int) string { return h.LinkGenerator.SeriesDetail(idStr, p) }
	links := h.generatePaginationLinks(linkFunc, page, lastPage, "Books in Series")
	feed := opds.NewFeed("Books in Series", "series-books-"+idStr, links)

	h.appendBookEntries(&feed, books)
	h.sendFeed(w, feed)
}

// BookDetailHandler returns a detail entry for a specific book.
func (h *Handler) BookDetailHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	book, err := h.BookService.GetBookByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.NotFound(w, r)
		return
	}

	links := []opds.Link{
		{
			Rel:  "self",
			Type: "application/atom+xml;profile=opds-catalog;kind=acquisition",
			Href: h.LinkGenerator.BookDetail(idStr),
		},
	}
feed := opds.NewFeed(book.Title, "book-detail-"+idStr, links)
h.appendBookEntries(&feed, []domain.Book{*book})
h.sendFeed(w, feed)
}

// SearchFeedHandler returns a paginated list of books matching the search query.
func (h *Handler) SearchFeedHandler(w http.ResponseWriter, r *http.Request) {
query := r.URL.Query().Get("q")
page := getPage(r)

if query == "" {
	// Return empty feed or bad request? Standard OPDS usually just returns empty feed.
	feed := opds.NewFeed("Search Results", "search-results", nil)
	h.sendFeed(w, feed)
	return
}

books, total, err := h.BookService.SearchBooks(r.Context(), query, page)
if err != nil {
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	return
}

lastPage := calculateLastPage(total)
linkFunc := func(p int) string {
	return fmt.Sprintf("%s?q=%s", h.LinkGenerator.Search(p), query)
}
links := h.generatePaginationLinks(linkFunc, page, lastPage, "Search Results")
feed := opds.NewFeed("Search Results: "+query, "search-results", links)

h.appendBookEntries(&feed, books)
h.sendFeed(w, feed)
}

// Helpers
// OpenSearchDescriptorHandler serves the OpenSearch description XML.
func (h *Handler) OpenSearchDescriptorHandler(w http.ResponseWriter, r *http.Request) {
	searchURL := h.LinkGenerator.Search(0)
	osd := OpenSearchDescription{
		ShortName:      "KOPDS",
		Description:    "Search the KOPDS Catalog",
		InputEncoding:  "UTF-8",
		OutputEncoding: "UTF-8",
		Url: OSDUrl{
			Type:     "application/atom+xml",
			Template: fmt.Sprintf("%s?q={searchTerms}", searchURL),
		},
	}

	w.Header().Set("Content-Type", "application/opensearchdescription+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(osd); err != nil {
		http.Error(w, "Failed to encode OpenSearch Description", http.StatusInternalServerError)
	}
}

// OpenSearchDescription represents the OpenSearch Description Document.
type OpenSearchDescription struct {
	XMLName        xml.Name `xml:"http://a9.com/-/spec/opensearch/1.1/ OpenSearchDescription"`
	ShortName      string   `xml:"ShortName"`
	Description    string   `xml:"Description"`
	InputEncoding  string   `xml:"InputEncoding"`
	OutputEncoding string   `xml:"OutputEncoding"`
	Url            OSDUrl   `xml:"Url"`
}

// OSDUrl represents the Url element in the OpenSearch Description Document.
type OSDUrl struct {
	Type     string `xml:"type,attr"`
	Template string `xml:"template,attr"`
}

// Helpers

func getPage(r *http.Request) int {
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			return val
		}
	}
	return 1
}

func calculateLastPage(total int) int {
	lastPage := (total + service.DefaultPageSize - 1) / service.DefaultPageSize
	if lastPage == 0 {
		return 1
	}
	return lastPage
}

func (h *Handler) generatePaginationLinks(linkFunc func(int) string, page, lastPage int, title string) []opds.Link {
	links := []opds.Link{
		{
			Rel:   "self",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  linkFunc(page),
			Title: title,
		},
		{
			Rel:   "first",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  linkFunc(1),
			Title: "First Page",
		},
	}

	if page > 1 {
		links = append(links, opds.Link{
			Rel:   "previous",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  linkFunc(page - 1),
			Title: "Previous Page",
		})
	}

	if page < lastPage {
		links = append(links, opds.Link{
			Rel:   "next",
			Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
			Href:  linkFunc(page + 1),
			Title: "Next Page",
		})
	}

	links = append(links, opds.Link{
		Rel:   "last",
		Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
		Href:  linkFunc(lastPage),
		Title: "Last Page",
	})

	return links
}

func (h *Handler) appendBookEntries(feed *opds.Feed, books []domain.Book) {
	for _, book := range books {
		idStr := strconv.FormatInt(book.ID, 10)
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
				URI:  h.LinkGenerator.AuthorDetail(strconv.FormatInt(author.ID, 10), 0),
			})
		}

		// Cover link
		if book.HasCover {
			entry.Links = append(entry.Links, opds.Link{
				Rel:  "http://opds-spec.org/image",
				Type: "image/jpeg",
				Href: h.LinkGenerator.Cover(idStr),
			})
			entry.Links = append(entry.Links, opds.Link{
				Rel:  "http://opds-spec.org/image/thumbnail",
				Type: "image/jpeg",
				Href: h.LinkGenerator.Cover(idStr), // We'll handle resizing in Phase 4
			})
		}

		// Acquisition links
		for _, format := range book.Formats {
			entry.Links = append(entry.Links, opds.Link{
				Rel:   "http://opds-spec.org/acquisition",
				Type:  getMimeType(format.Format),
				Href:  h.LinkGenerator.Download(idStr, format.Format),
				Title: format.Format,
			})
		}

		// Self link to detail
		entry.Links = append(entry.Links, opds.Link{
			Rel:  "alternate",
			Type: "application/atom+xml;profile=opds-catalog;kind=acquisition",
			Href: h.LinkGenerator.BookDetail(idStr),
		})

		feed.Entries = append(feed.Entries, entry)
	}
}

func (h *Handler) sendFeed(w http.ResponseWriter, feed opds.Feed) {
	w.Header().Set("Content-Type", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		http.Error(w, "Failed to encode feed", http.StatusInternalServerError)
	}
}

func getMimeType(format string) string {
	switch strings.ToLower(format) {
	case "epub":
		return "application/epub+zip"
	case "pdf":
		return "application/pdf"
	case "mobi":
		return "application/x-mobipocket-ebook"
	case "azw3":
		return "application/vnd.amazon.mobi8-ebook"
	case "cbz":
		return "application/x-cbz"
	case "cbr":
		return "application/x-cbr"
	default:
		return "application/octet-stream"
	}
}
