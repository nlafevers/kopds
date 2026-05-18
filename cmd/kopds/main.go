package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nlafevers/kopds/internal/api"
	"github.com/nlafevers/kopds/internal/config"
	"github.com/nlafevers/kopds/internal/database"
	"github.com/nlafevers/kopds/internal/domain"
	"github.com/nlafevers/kopds/internal/image"
	"github.com/nlafevers/kopds/internal/logger"
	"github.com/nlafevers/kopds/internal/scanner"
	"github.com/nlafevers/kopds/internal/service"
	"github.com/nlafevers/kopds/pkg/utils"
	"golang.org/x/term"
)

func main() {
	// 1. Load Config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Logger
	log := logger.New(cfg.LogLevel, cfg.JSONLog, cfg.LogPath)

	if len(os.Args) > 1 {
		runCLI(cfg)
		return
	}

	runServer(cfg, log)
}

func runCLI(cfg *config.Config) {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "create-user":
		if len(os.Args) < 3 {
			fmt.Printf("Usage: kopds %s <username> [--password-stdin]\n", command)
			os.Exit(1)
		}
		username := os.Args[2]
		password, err := passwordFromArgs(os.Args[3:], os.Stdin, os.Stdout)
		if err != nil {
			slog.Error("failed to read password", "username", username, "source", "CLI", "error", err)
			fmt.Printf("Failed to read password: %v\n", err)
			os.Exit(1)
		}
		createUser(cfg, username, password)
	case "delete-user":
		if len(os.Args) < 3 {
			fmt.Printf("Usage: kopds %s <username>\n", command)
			os.Exit(1)
		}
		username := os.Args[2]
		deleteUser(cfg, username)
	case "change-password":
		if len(os.Args) < 3 {
			fmt.Printf("Usage: kopds %s <username> [--password-stdin]\n", command)
			os.Exit(1)
		}
		username := os.Args[2]
		password, err := passwordFromArgs(os.Args[3:], os.Stdin, os.Stdout)
		if err != nil {
			slog.Error("failed to read password", "username", username, "source", "CLI", "error", err)
			fmt.Printf("Failed to read password: %v\n", err)
			os.Exit(1)
		}
		changePassword(cfg, username, password)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  kopds                          Run the server")
	fmt.Println("  kopds create-user <username>   Create a new user")
	fmt.Println("  kopds delete-user <username>   Delete a user")
	fmt.Println("  kopds change-password <user>   Change a user's password")
	fmt.Println("\nOptions for user commands:")
	fmt.Println("  --password-stdin                Read password from stdin")
}

func createUser(cfg *config.Config, username, password string) {
	db, err := database.NewSQLite(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to connect to database", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		slog.Error("failed to run migrations", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	userRepo := database.NewUserRepository(db)
	hash, err := api.HashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	user := &domain.User{
		Username: username,
		Password: hash,
	}

	if err := userRepo.Save(context.Background(), user); err != nil {
		slog.Error("failed to save user", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to save user: %v\n", err)
		os.Exit(1)
	}

	slog.Info("user created successfully", "username", username, "source", "CLI")
	fmt.Printf("User '%s' created/updated successfully.\n", username)
}

func deleteUser(cfg *config.Config, username string) {
	db, err := database.NewSQLite(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to connect to database", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	userRepo := database.NewUserRepository(db)
	if err := userRepo.DeleteUser(context.Background(), username); err != nil {
		slog.Error("failed to delete user", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to delete user: %v\n", err)
		os.Exit(1)
	}

	slog.Info("user deleted successfully", "username", username, "source", "CLI")
	fmt.Printf("User '%s' deleted successfully.\n", username)
}

func changePassword(cfg *config.Config, username, password string) {
	db, err := database.NewSQLite(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to connect to database", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	userRepo := database.NewUserRepository(db)
	hash, err := api.HashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	if err := userRepo.UpdatePassword(context.Background(), username, hash); err != nil {
		slog.Error("failed to update password", "username", username, "source", "CLI", "error", err)
		fmt.Printf("Failed to update password: %v\n", err)
		os.Exit(1)
	}

	slog.Info("user password updated successfully", "username", username, "source", "CLI")
	fmt.Printf("Password for user '%s' updated successfully.\n", username)
}

func passwordFromArgs(args []string, stdin io.Reader, stdout io.Writer) (string, error) {
	switch len(args) {
	case 0:
		return readPasswordInteractively(stdout)
	case 1:
		if args[0] != "--password-stdin" {
			return "", errors.New("password arguments are not supported; use interactive prompt or --password-stdin")
		}
		passwordBytes, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}
		password := strings.TrimRight(string(passwordBytes), "\r\n")
		if password == "" {
			return "", errors.New("password cannot be empty")
		}
		return password, nil
	default:
		return "", errors.New("too many arguments")
	}
}

func readPasswordInteractively(stdout io.Writer) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", errors.New("stdin is not a terminal; use --password-stdin for automation")
	}

	fmt.Fprint(stdout, "Password: ")
	first, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(stdout)
	if err != nil {
		return "", err
	}

	fmt.Fprint(stdout, "Confirm password: ")
	second, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(stdout)
	if err != nil {
		return "", err
	}

	if string(first) == "" {
		return "", errors.New("password cannot be empty")
	}
	if string(first) != string(second) {
		return "", errors.New("passwords do not match")
	}
	return string(first), nil
}

func runServer(cfg *config.Config, log *slog.Logger) {
	log.Info("Starting KOPDS server...")

	// 3. Validate Config
	if err := cfg.Validate(); err != nil {
		log.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	// 4. Initialize Database
	db, err := database.NewSQLite(cfg.DatabasePath)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}
	log.Info("Database initialized", "path", cfg.DatabasePath)

	// 5. Initialize Scanner
	bookRepo := database.NewBookRepository(db)
	userRepo := database.NewUserRepository(db)
	engine := scanner.NewSyncEngine(bookRepo, cfg.LibraryPath, cfg.DatabasePath, cfg.StorageCapMB, log)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	go scanner.StartWorker(workerCtx, engine, cfg.SyncInterval, log)

	// 6. Initialize Handlers
	imageCache, err := image.NewDiskCache(cfg.ImageCachePath, cfg.ImageCacheMaxCount)
	if err != nil {
		log.Error("Failed to initialize image cache", "error", err)
		os.Exit(1)
	}

	linkGen := utils.NewLinkGenerator(cfg.BaseURL)
	bookService := service.NewBookService(bookRepo, linkGen)
	h := api.NewHandler(bookService, userRepo, linkGen, imageCache, cfg.LibraryPath)

	// 7. Setup Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.GetHead)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// OPDS Routes
	r.Route("/opds/v1.2", func(r chi.Router) {
		r.Use(h.BasicAuth)
		r.Get("/catalog", h.NavigationFeedHandler)
		r.Get("/authors", h.AuthorsFeedHandler)
		r.Get("/authors/{id}", h.AuthorBooksHandler)
		r.Get("/series", h.SeriesFeedHandler)
		r.Get("/series/{id}", h.SeriesBooksHandler)
		r.Get("/tags", h.TagsFeedHandler)
		r.Get("/tags/{id}", h.TagBooksHandler)
		r.Get("/newest", h.NewestFeedHandler)
		r.Get("/books/{id}", h.BookDetailHandler)
		r.Get("/search", h.SearchFeedHandler)
		r.Get("/cover/{id}", h.CoverHandler)
		r.Get("/download/{id}/{format}", h.BookFileHandler)
		r.Get("/opensearch.xml", h.OpenSearchDescriptorHandler)
	})

	// 8. Start Server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: r,
	}

	go func() {
		log.Info("Server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("ListenAndServe failed", "error", err)
			os.Exit(1)
		}
	}()

	// 9. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Info("Shutting down server...")
	workerCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited cleanly")
}
