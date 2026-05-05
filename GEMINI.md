# KOPDS - Professional Lightweight OPDS Server

KOPDS is a high-performance, lightweight OPDS (Open Publication Distribution System) server designed specifically for self-hosting Calibre libraries. It is optimized for large libraries (10,000+ books) hosted on high-latency network shares (e.g., Nextcloud, SMB, NFS) and is tailored for the KOReader ecosystem.

## Project Overview

- **Purpose:** Provide a fast, reliable KOReader compatible OPDS 1.2 interface to a Calibre library.
- **Core Technologies:**
  - **Language:** Go (Golang) for a single-binary, low-memory footprint.
  - **Database:** Pure Go SQLite (`modernc.org/sqlite`) for local indexing and multi-user support.
  - **Web Framework:** `go-chi/chi/v5` for lightweight routing.
  - **Image Processing:** `disintegration/imaging` for on-the-fly cover resizing.
- **Architecture:** 
  - **Clean Architecture:** Separation of domain logic, use cases, and infrastructure.
  - **Background Indexing:** A synchronization engine that mirrors Calibre's `metadata.db` to a local index for instant querying.
  - **Image Pipeline:** On-the-fly thumbnail generation with a local LRU file cache.
  - **Deployment:** Ships as a standalone, single-executable binary for bare-metal execution, alongside a lightweight, CGO-free Docker image (scratch or alpine) for seamless home lab containerization.

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

  - [x] **Step 3.9 (Agent K):** Implement `SearchFeedHandler` in `internal/api/handlers.go`.
    - *Task:* Connect to repository search method and render results as Atom entries.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.

  - [x] **Step 3.10 (QA):** Verify XML output against standard OPDS 1.2 validators.
    - *Task:* Run integration tests for generated feeds.
    - *Action:* Commit changes to git and update GEMINI.md roadmap status.



- [ ] **Phase 4: Image & File Delivery**

  - [x] **Step 4.1:**
    - Create `internal/image/resizer.go` and implement `Resize(src io.Reader, width, height int) ([]byte, error)`.
    - Use `disintegration/imaging` to perform high-quality, efficient thumbnail generation.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.

  - [x] **Step 4.2:**
    - Implement `internal/image/cache.go` for LRU (Least Recently Used) disk-based caching.
    - Create a `DiskCache` struct that manages a directory of images, ensuring we don't exceed a defined max size/count.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.

  - [x] **Step 4.3:**
    - Implement `CoverHandler` in `internal/api/handlers.go`.
    - Create the endpoint `/opds/v1.2/cover/{bookID}` that checks the cache first, resizes if missing, and streams the image.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.

  - [x] **Step 4.4:**
    - Implement `BookFileHandler` in `internal/api/handlers.go`.
    - Create the endpoint `/opds/v1.2/download/{bookID}/{format}` to stream book files from the Calibre library.
    - Set correct `Content-Type` and `Content-Disposition` headers.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.

  - [x] **Step 4.5:**
    - Verify image caching and streaming performance.
    - Perform load tests on the image cache and verify successful book downloads in KOReader.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.


- [x] **Phase 5: Multi-User & Security**

  - [x] **Step 5.1:**
    - Implement `internal/api/auth.go` for password hashing and verification.
    - Use `golang.org/x/crypto/bcrypt` to implement `HashPassword(password string)` and `CheckPasswordHash(password, hash string)`.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.

  - [x] **Step 5.2:**
    - Implement `AuthMiddleware` in `internal/api/middleware.go`.
    - Implement an HTTP Basic Auth middleware that checks user credentials against the database.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.

  - [x] **Step 5.3:**
    - Implement `UserRepository` logic for user management in `internal/database/user_repository.go`.
    - Complete the implementation of `Save`, `GetByUsername`, and `DeleteUser`.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.

  - [x] **Step 5.4:**
    - Implement `AdminHandler` for basic user creation.
    - Create a secure command-line tool (e.g., `./kopds create-user`) for creating the initial admin user.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.

  - [x] **Step 5.5:**
    - Perform security audit.
    - Verify that all routes (except health check) are protected and that credentials are not logged.
    - Update GEMINI.md roadmap status.
    - Commit changes to git with an appropriate message.


- [x] **Phase 6: Deployment and Packaging**

  - [x] **Step 6.1:**
    - Create a multi-stage Dockerfile.
    - Use a golang builder image to compile the application with CGO_ENABLED=0 to ensure a static binary, then copy it into a minimal alpine or scratch runtime image.
  
  - [x] **Step 6.2:**
    - Create a docker-compose.yml template for users.
    - Map external volumes for the Calibre Library (Read-Only) and the local KOPDS SQLite index database (Read-Write).
    - Expose the necessary environment variables (via viper) for configuration.

  - [x] **Step 6.3:**
    - Add deployment documentation.
    - Write clear instructions emphasizing that KOPDS should be deployed behind a reverse proxy (e.g., Caddy, Traefik, Nginx) with HTTPS enabled, as Basic Auth transmits credentials in plain text.

- [ ] **Phase 7: Security Hardening & Reliability Review**

  - [x] **Step 7.1:**
    - Remove the vulnerable generic TIFF decode path from cover resizing.
    - Bound requested cover dimensions and input image size to prevent authenticated image DoS.
    - Upgrade `golang.org/x/image` to a fixed release.
    - Commit changes to git with an appropriate message.

## Development Conventions

- **Code Style:** Follow standard Go idioms and `gofmt`.
- **Concurrency:** Use background workers for indexing; ensure the API remains non-blocking.
- **Database:** Treat the Calibre `metadata.db` as read-only. All writes must occur in the local index database.
- **Error Handling:** Use structured logging with `rs/zerolog` and avoid swallowing errors.
- **Testing:** Aim for high unit test coverage in the `internal/domain` and `internal/opds` packages. Use a mock library for integration tests.
- **Docker & Storage:** When containerized, ensure the local KOPDS SQLite index is stored on a local volume attached to the host, not on the mounted high-latency network share (SMB/NFS), to prevent SQLite database locking issues and corruption.
