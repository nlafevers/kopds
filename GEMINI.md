# KOPDS - Professional Lightweight OPDS Server

KOPDS is a high-performance, lightweight OPDS (Open Publication Distribution System) server designed specifically for self-hosting Calibre libraries. It is optimized for large libraries (10,000+ books) hosted on high-latency network shares (e.g., Nextcloud, SMB, NFS) and is tailored for the KOReader ecosystem.

## Project Overview

- **Purpose:** Provide a fast, reliable, and professional-grade OPDS 1.2 interface to a Calibre library.
- **Core Technologies:**
  - **Language:** Go (Golang) for a single-binary, low-memory footprint.
  - **Database:** Pure Go SQLite (`modernc.org/sqlite`) for local indexing and multi-user support.
  - **Web Framework:** `go-chi/chi/v5` for lightweight routing.
  - **Image Processing:** `disintegration/imaging` for on-the-fly cover resizing.
- **Architecture:** 
  - **Clean Architecture:** Separation of domain logic, use cases, and infrastructure.
  - **Background Indexing:** A synchronization engine that mirrors Calibre's `metadata.db` to a local index for instant querying.
  - **Image Pipeline:** On-the-fly thumbnail generation with a local LRU file cache.

## Roadmap Status

- [x] **Phase 1: Foundation & Infrastructure** (Complete)
  - [x] Initialize Go module and project structure.
  - [x] Implement domain entities and interfaces.
  - [x] Setup configuration (`viper`) and logging (`zerolog`).
  - [x] Implement SQLite database layer with migrations.
  - [x] Basic HTTP server with health checks and graceful shutdown.

- [x] **Phase 2: Metadata Synchronization (The Indexer)** (Complete)
  - [x] Implement Calibre `metadata.db` reader (Read-only).
  - [x] Build incremental sync logic (mirror Calibre to local index).
  - [x] Implement search indexing with SQLite FTS5.
  - [x] Add background worker to trigger sync on file changes or timer.

- [x] **Phase 3: OPDS 1.2 Implementation**

  - [x] **Step 3.0 (Agent I):** Implement `pkg/utils/link.go` as a `LinkGenerator`.
    - *Task:* Create a struct that takes the base URL and provides methods to build navigation and acquisition URLs for the OPDS catalog. Support dynamic page index manipulation.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.0b (Agent I):** Implement `internal/service/book_service.go` as a mediation layer.
    - *Task:* Define a service struct that orchestrates repository data and link generation, providing clean interfaces for handlers.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.1 (Agent I):** Implement `internal/opds/atom.go` for base Atom XML types (`Feed`, `Entry`, `Link`, `Author`). 
    - *Task:* Define Go structs with proper `xml:"..."` tags that match OPDS 1.2 spec. Proactively include support for `opds:indirectAcquisition` to future-proof the acquisition pipeline.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.2 (Agent I):** Implement helper function `NewFeed(title, id string, links []Link) Feed` in `internal/opds/atom.go`.
    - *Task:* Ensure it correctly sets standard XML namespaces.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.3 (Agent J):** Implement `NavigationFeedHandler` in `internal/api/handlers.go`.
    - *Task:* Create the root catalog feed containing links to Authors, Series, Tags, and Newest feeds.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.4 (Agent J):** Implement `AuthorsFeedHandler` in `internal/api/handlers.go`.
    - *Task:* Retrieve authors via repository (ensure repo method returns book counts) and convert them into an Atom feed with `<entry>` elements per author.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.5 (Agent J):** Implement `SeriesFeedHandler` in `internal/api/handlers.go`.
    - *Task:* Retrieve series via repository (ensure repo method returns book counts) and convert into an Atom feed.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.6 (Agent J):** Implement Pagination support in `ListRecent` repository methods and feed handlers.
    - *Task:* Add `?page=N` logic to navigation links (first, next, prev).
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.7 (Agent K):** Implement `BookDetailHandler` in `internal/api/handlers.go`.
    - *Task:* Build an acquisition feed entry for a specific book, including download links for formats.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.8 (Agent K):** Implement `OpenSearchDescriptor` endpoint.
    - *Task:* Serve `osd.xml` that defines the search query template for KOReader.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 3.9 (Agent K):** Implement `SearchFeedHandler` in `internal/api/handlers.go`.
    - *Task:* Connect to repository search method and render results as Atom entries.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 3.10 (QA):** Verify XML output against standard OPDS 1.2 validators.
    - *Task:* Run integration tests for generated feeds.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.



- [ ] **Phase 4: Image & File Delivery**

  - **Step 4.1 (Agent L):** Create `internal/image/resizer.go` and implement `Resize(src io.Reader, width, height int) ([]byte, error)`.
    - *Task:* Use `disintegration/imaging` to perform high-quality, efficient thumbnail generation.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 4.2 (Agent L):** Implement `internal/image/cache.go` for LRU (Least Recently Used) disk-based caching.
    - *Task:* Create a `DiskCache` struct that manages a directory of images, ensuring we don't exceed a defined max size/count.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 4.3 (Agent L):** Implement `CoverHandler` in `internal/api/handlers.go`.
    - *Task:* Create the endpoint `/opds/v1.2/cover/{bookID}` that checks the cache first, resizes if missing, and streams the image.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 4.4 (Agent M):** Implement `BookFileHandler` in `internal/api/handlers.go`.
    - *Task:* Create the endpoint `/opds/v1.2/download/{bookID}/{format}` to stream book files from the Calibre library.
    - *Action:* Set correct `Content-Type` and `Content-Disposition` headers.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 4.5 (QA):** Verify image caching and streaming performance.
    - *Task:* Perform load tests on the image cache and verify successful book downloads in KOReader.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.


- [ ] **Phase 5: Multi-User & Security**

  - **Step 5.1 (Agent N):** Implement `internal/api/auth.go` for password hashing and verification.
    - *Task:* Use `golang.org/x/crypto/bcrypt` to implement `HashPassword(password string)` and `CheckPasswordHash(password, hash string)`.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 5.2 (Agent N):** Implement `AuthMiddleware` in `internal/api/middleware.go`.
    - *Task:* Implement an HTTP Basic Auth middleware that checks user credentials against the database.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 5.3 (Agent N):** Implement `UserRepository` logic for user management in `internal/database/user_repository.go`.
    - *Task:* Complete the implementation of `Save`, `GetByUsername`, and `DeleteUser`.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 5.4 (Agent N):** Implement `AdminHandler` for basic user creation.
    - *Task:* Create a secure command-line or internal endpoint for creating the initial admin user.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - **Step 5.5 (QA):** Perform security audit.
    - *Task:* Verify that all routes (except health check) are protected and that credentials are not logged.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.


## Development Conventions

- **Code Style:** Follow standard Go idioms and `gofmt`.
- **Concurrency:** Use background workers for indexing; ensure the API remains non-blocking.
- **Database:** Treat the Calibre `metadata.db` as read-only. All writes must occur in the local index database.
- **Error Handling:** Use structured logging with `rs/zerolog` and avoid swallowing errors.
- **Testing:** Aim for high unit test coverage in the `internal/domain` and `internal/opds` packages. Use a mock library for integration tests.
