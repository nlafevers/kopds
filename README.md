# KOPDS - Lightweight OPDS Server

KOPDS is a high-performance, lightweight OPDS (Open Publication Distribution System) server designed specifically for self-hosting Calibre libraries. It is engineered for large libraries (10,000+ books) hosted on high-latency network shares (e.g., Nextcloud, SMB, NFS) and is perfectly tailored for the KOReader ecosystem.

## Why KOPDS?

While many OPDS servers exist, KOPDS focuses on three core pillars:

1.  **High Performance:** By mirroring your Calibre `metadata.db` to a local, optimized SQLite index, KOPDS provides near-instant search and navigation, even when your library is stored on a slow network share.
2.  **Resource Efficiency:** Built in pure Go, KOPDS has a minimal memory footprint and compiles to a single, portable binary, making it ideal for low-power devices like Raspberry Pis or home servers.
3.  **KOReader Optimization:** Designed with the specific quirks and requirements of KOReader in mind, ensuring a seamless book discovery and acquisition experience.

## Core Features

- **OPDS 1.2 Support:** Fully compatible with KOReader and other standard OPDS clients.
- **Background Synchronization:** Automatically detects changes in your Calibre library and keeps the local index up-to-date without blocking API requests.
- **Instant Search:** Powered by SQLite FTS5 for rapid, full-text search across titles, authors, tags, and series.
- **Production-Ready:** Structured logging, multi-user support, and comprehensive test coverage.
- **Zero-Dependency Architecture:** Minimal external requirements; perfect for containerized deployments.
- **Clean Architecture Approach:** Domain logic is separated from infrastructure concerns. It features a background scanner that incrementally synchronizes your library, an optimized media delivery pipeline, and a robust API layer for OPDS delivery.

## Getting Started

### Docker (Recommended)

The easiest way to run KOPDS is via Docker by building the image locally.

1.  Create a `docker-compose.yml` file in the project root:
    ```yaml
    services:
      kopds:
        build: .
        container_name: kopds
        restart: unless-stopped
        ports:
          - "8080:8080"
        volumes:
          - /path/to/your/calibre/library:/library:ro
          - ./data:/data
          - ./cache:/cache
        environment:
          - KOPDS_LIBRARY_PATH=/library
          - KOPDS_DATABASE_PATH=/data/kopds.db
          - KOPDS_IMAGE_CACHE_PATH=/cache/images
    ```
2.  Start the container:
    ```bash
    docker compose up -d
    ```
3.  Create your initial admin user:
    ```bash
    printf '%s\n' 'yourpassword' | docker exec -i kopds ./kopds create-user admin --password-stdin
    ```
    To avoid exposing passwords to shell history or process listings, the create-user UX uses a hidden terminal prompt for inputting the password.  However, to allow Docker automation `--password-stdin` is an option.

### Binary Installation

1.  Build the binary:
    ```bash
    go build -o kopds ./cmd/kopds
    ```
2.  Configure `config.yaml` (see sample in repo).
3.  Create your admin user:
    ```bash
    ./kopds create-user admin
    ```
4.  Run the server:
    ```bash
    ./kopds
    ```

## Deployment Guidelines

### Security & Reverse Proxy

KOPDS uses **HTTP Basic Authentication** for simplicity and compatibility with KOReader. Since Basic Auth transmits credentials in plain text, you **must** deploy KOPDS behind a reverse proxy (e.g., Caddy, Nginx, Traefik) that provides **HTTPS**.

### Storage Performance

For optimal performance, ensure the `data` directory (which holds the SQLite index) is stored on **local high-speed storage** (SSD/NVMe). While your Calibre library can reside on a high-latency network share (SMB/NFS), the KOPDS index database must be on local storage to prevent SQLite locking issues and ensure rapid response times.

## License
GPL-3.0 license
