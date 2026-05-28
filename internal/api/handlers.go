package api

import (
	"encoding/xml"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nlafevers/kopds/internal/domain"
	"github.com/nlafevers/kopds/internal/image"
	"github.com/nlafevers/kopds/internal/opds"
	"github.com/nlafevers/kopds/internal/service"
	"github.com/nlafevers/kopds/pkg/utils"
)

// Handler handles HTTP requests for the OPDS API.
type Handler struct {
	BookService   *service.BookService
	UserRepo      domain.UserRepository
	LinkGenerator *utils.LinkGenerator
	ImageCache    *image.DiskCache
	LibraryPath   string
}

// NewHandler creates a new Handler.
func NewHandler(bookService *service.BookService, userRepo domain.UserRepository, linkGenerator *utils.LinkGenerator, imageCache *image.DiskCache, libraryPath string) *Handler {
	return &Handler{
		BookService:   bookService,
		UserRepo:      userRepo,
		LinkGenerator: linkGenerator,
		ImageCache:    imageCache,
		LibraryPath:   libraryPath,
	}
}

// NavigationFeedHandler returns the root navigation feed for the OPDS catalog.
func (h *Handler) NavigationFeedHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	feed := opds.Feed{
		Opds:    opds.OPDSNamespace,
		Title:   "KOPDS Root Catalog",
		ID:      "urn:kopds:catalog:root",
		Updated: now,
		Links: []opds.Link{
			{
				Rel:   "self",
				Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
				Href:  h.LinkGenerator.RootCatalog(),
				Title: "Root Catalog",
			},
			{
				Rel:   "start",
				Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
				Href:  h.LinkGenerator.RootCatalog(),
				Title: "Root Catalog",
			},
			{
				Rel:   "search",
				Type:  "application/opensearchdescription+xml",
				Href:  h.LinkGenerator.OpenSearchDescriptor(),
				Title: "Search KOPDS",
			},
		},
	}

	// Add Navigation Entries
	navItems := []struct {
		title   string
		id      string
		rel     string
		href    string
		summary string
	}{
		{
			title:   "Authors",
			id:      "urn:kopds:catalog:authors",
			rel:     "subsection",
			href:    h.LinkGenerator.AuthorsList(0),
			summary: "Browse books by author",
		},
		{
			title:   "Series",
			id:      "urn:kopds:catalog:series",
			rel:     "subsection",
			href:    h.LinkGenerator.SeriesList(0),
			summary: "Browse books by series",
		},
		{
			title:   "Tags",
			id:      "urn:kopds:catalog:tags",
			rel:     "subsection",
			href:    h.LinkGenerator.TagsList(0),
			summary: "Browse books by tag",
		},
		{
			title:   "Newest Books",
			id:      "urn:kopds:catalog:newest",
			rel:     "http://opds-spec.org/sort/new",
			href:    h.LinkGenerator.NewestBooks(0),
			summary: "Recently added books",
		},
	}

	for _, e := range navItems {
		entry := &opds.Entry{
			ID:      e.id,
			Title:   e.title,
			Updated: now,
			Summary: &opds.Content{
				Text: e.summary,
			},
			Links: []opds.Link{
				{
					Rel:   e.rel,
					Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
					Href:  e.href,
					Title: e.title,
				},
			},
		}
		feed.Entries = append(feed.Entries, entry)
	}
	h.sendFeed(w, r, feed)
}
func (h *Handler) addSearchLink(feed *opds.Feed) {
	feed.Links = append(feed.Links, opds.Link{
		Rel:   "search",
		Type:  "application/opensearchdescription+xml",
		Href:  h.LinkGenerator.OpenSearchDescriptor(),
		Title: "Search KOPDS",
	})
}

// AuthorsFeedHandler returns a paginated list of authors in the OPDS catalog.
func (h *Handler) AuthorsFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := getPage(r)
	authors, total, err := h.BookService.GetAuthors(r.Context(), page)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	links := h.generatePaginationLinks(h.LinkGenerator.AuthorsList, page, lastPage, "Authors")
	feed := opds.NewFeed("Authors", "authors-list", links)
	h.addSearchLink(&feed)

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
	h.sendFeed(w, r, feed)
}

// SeriesFeedHandler returns a paginated list of series in the OPDS catalog.
func (h *Handler) SeriesFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := getPage(r)
	series, total, err := h.BookService.GetSeries(r.Context(), page)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	links := h.generatePaginationLinks(h.LinkGenerator.SeriesList, page, lastPage, "Series")
	feed := opds.NewFeed("Series", "series-list", links)
	h.addSearchLink(&feed)

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
	h.sendFeed(w, r, feed)
}

// TagsFeedHandler returns a paginated list of tags in the OPDS catalog.
func (h *Handler) TagsFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := getPage(r)
	tags, total, err := h.BookService.GetTags(r.Context(), page)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	links := h.generatePaginationLinks(h.LinkGenerator.TagsList, page, lastPage, "Tags")
	feed := opds.NewFeed("Tags", "tags-list", links)
	h.addSearchLink(&feed)

	for _, t := range tags {
		summary := fmt.Sprintf("%d books", t.BookCount)
		entry := &opds.Entry{
			ID:    fmt.Sprintf("tag:%d", t.ID),
			Title: t.Name,
			Summary: &opds.Content{
				Text: summary,
			},
			Links: []opds.Link{
				{
					Rel:   "subsection",
					Type:  "application/atom+xml;profile=opds-catalog;kind=navigation",
					Href:  h.LinkGenerator.TagDetail(strconv.FormatInt(t.ID, 10), 0),
					Title: t.Name,
				},
			},
		}
		feed.Entries = append(feed.Entries, entry)
	}
	h.sendFeed(w, r, feed)
}

// NewestFeedHandler returns a paginated list of the newest books in the OPDS catalog.
func (h *Handler) NewestFeedHandler(w http.ResponseWriter, r *http.Request) {
	page := getPage(r)
	books, total, err := h.BookService.GetRecentBooks(r.Context(), page)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	links := h.generatePaginationLinks(h.LinkGenerator.NewestBooks, page, lastPage, "Newest Books")
	feed := opds.NewFeed("Newest Books", "newest-list", links)
	h.addSearchLink(&feed)

	h.appendBookEntries(&feed, books)
	h.sendFeed(w, r, feed)
}

// AuthorBooksHandler returns a paginated list of books by a specific author.
func (h *Handler) AuthorBooksHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	page := getPage(r)

	books, total, err := h.BookService.GetBooksByAuthor(r.Context(), id, page)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	linkFunc := func(p int) string { return h.LinkGenerator.AuthorDetail(idStr, p) }
	links := h.generatePaginationLinks(linkFunc, page, lastPage, "Books by Author")
	feed := opds.NewFeed("Books by Author", "author-books-"+idStr, links)

	h.appendBookEntries(&feed, books)
	h.sendFeed(w, r, feed)
}

// SeriesBooksHandler returns a paginated list of books in a specific series.
func (h *Handler) SeriesBooksHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	page := getPage(r)

	books, total, err := h.BookService.GetBooksBySeries(r.Context(), id, page)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	linkFunc := func(p int) string { return h.LinkGenerator.SeriesDetail(idStr, p) }
	links := h.generatePaginationLinks(linkFunc, page, lastPage, "Books in Series")
	feed := opds.NewFeed("Books in Series", "series-books-"+idStr, links)

	h.appendBookEntries(&feed, books)
	h.sendFeed(w, r, feed)
}

// TagBooksHandler returns a paginated list of books with a specific tag.
func (h *Handler) TagBooksHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	page := getPage(r)

	books, total, err := h.BookService.GetBooksByTag(r.Context(), id, page)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	linkFunc := func(p int) string { return h.LinkGenerator.TagDetail(idStr, p) }
	links := h.generatePaginationLinks(linkFunc, page, lastPage, "Books with Tag")
	feed := opds.NewFeed("Books with Tag", "tag-books-"+idStr, links)

	h.appendBookEntries(&feed, books)
	h.sendFeed(w, r, feed)
}

// BookDetailHandler returns a detail entry for a specific book.
func (h *Handler) BookDetailHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	book, err := h.BookService.GetBookByID(r.Context(), id)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
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
	h.sendFeed(w, r, feed)
}

// SearchFeedHandler returns a paginated list of books matching the search query.
func (h *Handler) SearchFeedHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page := getPage(r)

	if query == "" {
		// Return empty feed or bad request? Standard OPDS usually just returns empty feed.
		feed := opds.NewFeed("Search Results", "search-results", nil)
		h.sendFeed(w, r, feed)
		return
	}

	books, total, err := h.BookService.SearchBooks(r.Context(), query, page)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lastPage := calculateLastPage(total)
	linkFunc := func(p int) string {
		return addQueryParam(h.LinkGenerator.Search(p), "q", query)
	}
	links := h.generatePaginationLinks(linkFunc, page, lastPage, "Search Results")
	feed := opds.NewFeed("Search Results: "+query, "search-results", links)

	h.appendBookEntries(&feed, books)
	h.sendFeed(w, r, feed)
}

// CoverHandler serves the cover image for a book, resizing it if necessary and caching the result.
func (h *Handler) CoverHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		GetLogger(r.Context()).Warn("invalid book ID", "id", idStr)
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}
	width, height, err := getCoverDimensions(r)
	if err != nil {
		GetLogger(r.Context()).Warn("invalid cover dimensions", "error", err)
		http.Error(w, "Invalid cover dimensions", http.StatusBadRequest)
		return
	}
	cacheKey := fmt.Sprintf("%s_%dx%d.jpg", idStr, width, height)
	data, err := h.ImageCache.Get(cacheKey)
	if err == nil {
		GetLogger(r.Context()).Debug("image cache hit", "key", cacheKey)
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=604800") // 1 week
		w.Write(data)
		return
	}

	GetLogger(r.Context()).Debug("image cache miss", "key", cacheKey)
	// Not in cache, resize
	book, err := h.BookService.GetBookByID(r.Context(), id)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if book == nil || !book.HasCover {
		http.NotFound(w, r)
		return
	}

	coverPath, err := safeLibraryPath(h.LibraryPath, book.Path, "cover.jpg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(coverPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		GetLogger(r.Context()).Error("failed to open cover", "path", coverPath, "error", err)
		http.Error(w, "Failed to open cover", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	resizedData, err := image.Resize(file, width, height)
	if err != nil {
		GetLogger(r.Context()).Error("failed to resize image", "error", err)
		http.Error(w, "Failed to resize image", http.StatusInternalServerError)
		return
	}

	if err := h.ImageCache.Put(cacheKey, resizedData); err != nil {
		GetLogger(r.Context()).Warn("failed to cache resized image", "key", cacheKey, "error", err)
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=604800") // 1 week
	w.Write(resizedData)
}

// BookFileHandler streams a book file in the requested format.
func (h *Handler) BookFileHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		GetLogger(r.Context()).Warn("invalid book ID", "id", idStr)
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}
	requestedFormat := strings.ToUpper(r.PathValue("format"))
	if requestedFormat == "" {
		GetLogger(r.Context()).Warn("format is required")
		http.Error(w, "Format is required", http.StatusBadRequest)
		return
	}
	book, err := h.BookService.GetBookByID(r.Context(), id)
	if err != nil {
		GetLogger(r.Context()).Error("handler error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.NotFound(w, r)
		return
	}

	var targetFormat *domain.Format
	for _, f := range book.Formats {
		if strings.ToUpper(f.Format) == requestedFormat {
			targetFormat = &f
			break
		}
	}

	if targetFormat == nil {
		GetLogger(r.Context()).Warn("format not found for book", "id", id, "format", requestedFormat)
		http.Error(w, "Format not found for this book", http.StatusNotFound)
		return
	}
	fileName, err := formatFileName(targetFormat)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	filePath, err := safeLibraryPath(h.LibraryPath, book.Path, fileName)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		GetLogger(r.Context()).Warn("file not found on disk", "path", filePath)
		http.Error(w, "File not found on disk", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", getMimeType(targetFormat.Format))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fileName}))

	http.ServeFile(w, r, filePath)
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
		GetLogger(r.Context()).Error("failed to encode OpenSearch Description", "error", err)
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

func getCoverDimensions(r *http.Request) (int, int, error) {
	width := 300
	height := 450

	if wParam := r.URL.Query().Get("w"); wParam != "" {
		val, err := strconv.Atoi(wParam)
		if err != nil {
			return 0, 0, err
		}
		width = val
	}

	if hParam := r.URL.Query().Get("h"); hParam != "" {
		val, err := strconv.Atoi(hParam)
		if err != nil {
			return 0, 0, err
		}
		height = val
	}

	if err := image.ValidateDimensions(width, height); err != nil {
		return 0, 0, err
	}
	return width, height, nil
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

func (h *Handler) sendFeed(w http.ResponseWriter, r *http.Request, feed opds.Feed) {
	log := GetLogger(r.Context())
	w.Header().Set("Content-Type", "application/atom+xml;profile=opds-catalog;kind=navigation;charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		log.Error("failed to encode feed", "error", err)
		http.Error(w, "Failed to encode feed", http.StatusInternalServerError)
		return
	}
	log.Info("feed served", "title", feed.Title)
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

func addQueryParam(rawURL, key, value string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

func formatFileName(format *domain.Format) (string, error) {
	baseName := filepath.Base(format.Name)
	extension := strings.ToLower(format.Format)
	if baseName == "." || baseName == string(filepath.Separator) || baseName == "" || baseName != format.Name || strings.ContainsAny(format.Name, `/\`) {
		return "", fmt.Errorf("invalid format name")
	}
	if extension == "" || strings.ContainsAny(extension, `/\`) {
		return "", fmt.Errorf("invalid format extension")
	}
	return fmt.Sprintf("%s.%s", baseName, extension), nil
}

func safeLibraryPath(libraryPath string, parts ...string) (string, error) {
	root, err := filepath.Abs(libraryPath)
	if err != nil {
		return "", err
	}

	cleanParts := make([]string, 0, len(parts)+1)
	cleanParts = append(cleanParts, root)
	for _, part := range parts {
		if part == "" || filepath.IsAbs(part) {
			return "", fmt.Errorf("invalid library path component")
		}
		clean := filepath.Clean(part)
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("invalid library path component")
		}
		cleanParts = append(cleanParts, clean)
	}

	target, err := filepath.Abs(filepath.Join(cleanParts...))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path escapes library root")
	}
	return target, nil
}
