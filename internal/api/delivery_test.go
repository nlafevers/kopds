package api

import (
	"context"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nlafevers/kopds/internal/domain"
	img "github.com/nlafevers/kopds/internal/image"
	"github.com/nlafevers/kopds/internal/service"
	"github.com/nlafevers/kopds/pkg/utils"
)

func TestDeliveryIntegration(t *testing.T) {
	// 1. Setup temporary library and cache
	tempDir, err := os.MkdirTemp("", "kopds-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	libraryPath := filepath.Join(tempDir, "library")
	cachePath := filepath.Join(tempDir, "cache")
	os.MkdirAll(libraryPath, 0755)
	os.MkdirAll(cachePath, 0755)

	// Create a dummy book directory and file
	bookPath := "Test Book (1)"
	fullBookPath := filepath.Join(libraryPath, bookPath)
	os.MkdirAll(fullBookPath, 0755)

	// Create dummy cover
	coverPath := filepath.Join(fullBookPath, "cover.jpg")
	createDummyImage(t, coverPath)

	// Create dummy book file
	epubPath := filepath.Join(fullBookPath, "Test Book.epub")
	err = os.WriteFile(epubPath, []byte("fake epub content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Setup Handler
	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	repo := &mockRepoDelivery{
		getByIDFunc: func(ctx context.Context, id int64) (*domain.Book, error) {
			if id == 1 {
				return &domain.Book{
					ID:       1,
					Title:    "Test Book",
					Path:     bookPath,
					HasCover: true,
					Formats: []domain.Format{
						{Format: "EPUB", Name: "Test Book"},
					},
				}, nil
			}
			return nil, nil
		},
	}

	svc := service.NewBookService(repo, linkGen)
	imageCache, _ := img.NewDiskCache(cachePath, 10)
	h := NewHandler(svc, linkGen, imageCache, libraryPath)

	r := chi.NewRouter()
	r.Route("/opds/v1.2", func(r chi.Router) {
		r.Get("/cover/{id}", h.CoverHandler)
		r.Get("/download/{id}/{format}", h.BookFileHandler)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	// 3. Test CoverHandler
	t.Run("CoverHandler", func(t *testing.T) {
		res, err := http.Get(ts.URL + "/opds/v1.2/cover/1?w=100&h=150")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("expected OK, got %d", res.StatusCode)
		}
		if ct := res.Header.Get("Content-Type"); ct != "image/jpeg" {
			t.Errorf("expected image/jpeg, got %s", ct)
		}

		// Verify it's in cache
		cacheKey := "1_100x150.jpg"
		if _, err := imageCache.Get(cacheKey); err != nil {
			t.Errorf("expected image to be in cache: %v", err)
		}
	})

	// 4. Test BookFileHandler
	t.Run("BookFileHandler", func(t *testing.T) {
		res, err := http.Get(ts.URL + "/opds/v1.2/download/1/epub")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("expected OK, got %d", res.StatusCode)
		}
		if ct := res.Header.Get("Content-Type"); ct != "application/epub+zip" {
			t.Errorf("expected application/epub+zip, got %s", ct)
		}
		
		body, _ := io.ReadAll(res.Body)
		if string(body) != "fake epub content" {
			t.Errorf("expected 'fake epub content', got '%s'", string(body))
		}

		if cd := res.Header.Get("Content-Disposition"); cd != "attachment; filename=\"Test Book.epub\"" {
			t.Errorf("unexpected Content-Disposition: %s", cd)
		}
	})

	t.Run("BookFileHandler_NotFound", func(t *testing.T) {
		res, err := http.Get(ts.URL + "/opds/v1.2/download/1/pdf")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected NotFound, got %d", res.StatusCode)
		}
	})
}

func createDummyImage(t *testing.T, path string) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatal(err)
	}
}

type mockRepoDelivery struct {
	domain.BookRepository
	getByIDFunc func(ctx context.Context, id int64) (*domain.Book, error)
}

func (m *mockRepoDelivery) GetByID(ctx context.Context, id int64) (*domain.Book, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
