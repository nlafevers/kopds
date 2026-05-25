# KOPDS - Professional Lightweight OPDS Server

KOPDS is a high-performance, lightweight OPDS (Open Publication Distribution System) server designed specifically for self-hosting Calibre libraries. It is optimized for large libraries (10,000+ books) hosted on high-latency network shares (e.g., Nextcloud, SMB, NFS) and is tailored for the KOReader ecosystem.

## Cross-Project Uniformity

KOPDS is maintained alongside KOSYNC with a maximum-uniformity goal. Functions that perform the same job in both repositories should use the same names and identical code wherever practical. See `UNIFORMITY.md` for the current inventory and boundaries. Keep CLI user management, password helpers, logger construction, config path resolution, SQLite opening, and storage-cap helper flow aligned unless a documented project-specific domain difference requires divergence.

## Project Overview

- **Purpose:** Provide a fast, reliable KOReader compatible OPDS 1.2 interface to a Calibre library.
- **Core Technologies:**
  - **Language:** Go (Golang) for a single-binary, low-memory footprint.
  - **Database:** Pure Go SQLite (`modernc.org/sqlite`) for local indexing and multi-user support.
  - **Web Framework:** Native `net/http.ServeMux` (Go 1.22+) for lightweight, dependency-free routing.
  - **Image Processing:** `disintegration/imaging` for on-the-fly cover resizing.
- **Architecture:** 
  - **Clean Architecture:** Separation of domain logic, use cases, and infrastructure.
  - **Background Indexing:** A synchronization engine that mirrors Calibre's `metadata.db` to a local index for instant querying.
  - **Image Pipeline:** On-the-fly thumbnail generation with a local LRU file cache.
  - **Deployment:** Ships as a standalone, single-executable binary for bare-metal execution, alongside a lightweight, CGO-free Docker image (scratch or alpine) for seamless home lab containerization.

## Development Conventions

- **Code Style:** Follow standard Go idioms and `gofmt`.
- **Concurrency:** Use background workers for indexing; ensure the API remains non-blocking.
- **Database:** Treat the Calibre `metadata.db` as read-only. All writes must occur in the local index database.
- **Error Handling:** Use structured logging with `rs/zerolog` and avoid swallowing errors.
- **Testing:** Aim for high unit test coverage in the `internal/domain` and `internal/opds` packages. Use a mock library for integration tests.
- **Docker & Storage:** When containerized, ensure the local KOPDS SQLite index is stored on a local volume attached to the host, not on the mounted high-latency network share (SMB/NFS), to prevent SQLite database locking issues and corruption.
