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

const appName = "kopds"

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
			fmt.Printf("Usage: %s %s <username> [--password-stdin]\n", appName, command)
			os.Exit(1)
		}
		username := os.Args[2]
		password, err := passwordFromArgs(os.Args[3:], os.Stdin, os.Stdout)
		if err != nil {
			logger.LogCLIFailure(nil, command, username, "failed to read password: "+err.Error())
			fmt.Printf("Failed to read password: %v\n", err)
			os.Exit(1)
		}
		createUser(cfg, username, password)
	case "delete-user":
		if len(os.Args) < 3 {
			fmt.Printf("Usage: %s %s <username>\n", appName, command)
			os.Exit(1)
		}
		username := os.Args[2]
		deleteUser(cfg, username)
	case "change-password":
		if len(os.Args) < 3 {
			fmt.Printf("Usage: %s %s <username> [--password-stdin]\n", appName, command)
			os.Exit(1)
		}
		username := os.Args[2]
		password, err := passwordFromArgs(os.Args[3:], os.Stdin, os.Stdout)
		if err != nil {
			logger.LogCLIFailure(nil, command, username, "failed to read password: "+err.Error())
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
	fmt.Printf("  %s                          Run the server\n", appName)
	fmt.Printf("  %s create-user <username>   Create a new user\n", appName)
	fmt.Printf("  %s delete-user <username>   Delete a user\n", appName)
	fmt.Printf("  %s change-password <user>   Change a user's password\n", appName)
	fmt.Println("\nOptions for user commands:")
	fmt.Println("  --password-stdin                Read password from stdin")
}

func createUser(cfg *config.Config, username, password string) {
	operation := "create-user"
	db, err := database.NewSQLite(cfg.DatabasePath)
	if err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to connect to database: "+err.Error())
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to run migrations: "+err.Error())
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	userRepo := database.NewUserRepository(db, slog.Default())
	hash, err := api.HashPassword(password)
	if err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to hash password: "+err.Error())
		fmt.Printf("Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	user := &domain.User{
		Username: username,
		Password: hash,
	}

	if err := userRepo.CreateUserIfNotExists(context.Background(), user); err != nil {
		if err.Error() == "user already exists" {
			logger.LogCLIFailure(nil, operation, username, "user already exists")
			fmt.Printf("Error: User '%s' already exists\n", username)
			os.Exit(1)
		}
		logger.LogCLIFailure(nil, operation, username, "failed to save user: "+err.Error())
		fmt.Printf("Failed to save user: %v\n", err)
		os.Exit(1)
	}

	logger.LogCLISuccess(nil, operation, username)
	fmt.Printf("User '%s' created successfully.\n", username)
}

func deleteUser(cfg *config.Config, username string) {
	operation := "delete-user"
	db, err := database.NewSQLite(cfg.DatabasePath)
	if err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to connect to database: "+err.Error())
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to run migrations: "+err.Error())
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	userRepo := database.NewUserRepository(db, slog.Default())
	if err := userRepo.DeleteUser(context.Background(), username); err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to delete user: "+err.Error())
		fmt.Printf("Failed to delete user: %v\n", err)
		os.Exit(1)
	}

	logger.LogCLISuccess(nil, operation, username)
	fmt.Printf("User '%s' deleted successfully.\n", username)
}
func changePassword(cfg *config.Config, username, password string) {
	operation := "change-password"
	db, err := database.NewSQLite(cfg.DatabasePath)
	if err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to connect to database: "+err.Error())
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to run migrations: "+err.Error())
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	userRepo := database.NewUserRepository(db, slog.Default())
	hash, err := api.HashPassword(password)
	if err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to hash password: "+err.Error())
		fmt.Printf("Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	if err := userRepo.UpdatePassword(context.Background(), username, hash); err != nil {
		logger.LogCLIFailure(nil, operation, username, "failed to update password: "+err.Error())
		fmt.Printf("Failed to update password: %v\n", err)
		os.Exit(1)
	}

	logger.LogCLISuccess(nil, operation, username)
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
	log.Info("Starting KOPDS",
		"app_name", appName,
		"port", cfg.Port,
		"database_path", cfg.DatabasePath,
		"log_level", cfg.LogLevel,
		"json_log", cfg.JSONLog,
		"log_path", cfg.LogPath,
	)

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
	log.Info("Database initialized",
		"database_path", cfg.DatabasePath,
		"migration_status", "success",
		"storage_cap_mb", cfg.StorageCapMB,
	)

	// 5. Initialize Scanner
	bookRepo := database.NewBookRepository(db, log)
	userRepo := database.NewUserRepository(db, log)
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
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// OPDS Routes
	protected := http.NewServeMux()
	protected.HandleFunc("GET /opds/v1.2/catalog", h.NavigationFeedHandler)
	protected.HandleFunc("GET /opds/v1.2/authors", h.AuthorsFeedHandler)
	protected.HandleFunc("GET /opds/v1.2/authors/{id}", h.AuthorBooksHandler)
	protected.HandleFunc("GET /opds/v1.2/series", h.SeriesFeedHandler)
	protected.HandleFunc("GET /opds/v1.2/series/{id}", h.SeriesBooksHandler)
	protected.HandleFunc("GET /opds/v1.2/tags", h.TagsFeedHandler)
	protected.HandleFunc("GET /opds/v1.2/tags/{id}", h.TagBooksHandler)
	protected.HandleFunc("GET /opds/v1.2/newest", h.NewestFeedHandler)
	protected.HandleFunc("GET /opds/v1.2/books/{id}", h.BookDetailHandler)
	protected.HandleFunc("GET /opds/v1.2/search", h.SearchFeedHandler)
	protected.HandleFunc("GET /opds/v1.2/cover/{id}", h.CoverHandler)
	protected.HandleFunc("GET /opds/v1.2/download/{id}/{format}", h.BookFileHandler)
	protected.HandleFunc("GET /opds/v1.2/opensearch.xml", h.OpenSearchDescriptorHandler)

	mux.Handle("/opds/v1.2/", api.BasicAuth(userRepo, protected))

	// 8. Start Server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: api.LoggingMiddleware(mux),
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

	sig := <-stop
	log.Info("Shutdown signal received", "signal", sig.String())
	workerCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("Shutting down server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed", "error", err)
	} else {
		log.Info("Server exited cleanly")
	}
}
