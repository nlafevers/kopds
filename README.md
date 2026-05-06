# KOPDS - Lightweight OPDS Server

KOPDS is a high-performance, lightweight OPDS (Open Publication Distribution System) server designed specifically for self-hosting Calibre libraries. It is engineered for large libraries (10,000+ books) hosted on high-latency network shares (e.g., Nextcloud, SMB, NFS) and is tailored for the KOReader ecosystem.

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

## Prerequisites

- **Docker** (or Podman) if deploying via container.
- **Go (1.25+)** if building your own binary.
- A **reverse proxy** like Nginx, Caddy, or Traefik is **highly recommended**.

## Getting Started

The easiest way to run KOPDS is via Docker.  You can either use an image from GitHub Container Registry (if available, check Packages for this repository), or build one locally.  If building locally you'll need to replace `image: ghcr.io/yourusername/kopds:latest` with `build: .` in `docker-compose.yml`.

You can also build the binary from source and deploy natively on your host.

### Method 1:
**Docker deployment using a pre-built image (Recommended)**

1.  Create a project root directory:
    ```bash
    mkdir PROJECT_ROOT && cd PROJECT_ROOT
    ```
2.  Create a `docker-compose.yml` file in the project root.  Make sure to change the path to your Calibre library.  Make sure to change the KOPDS_BASE_URL to match the IP address or domain name that your Koreader devices use to connect.
    ```yaml
    services:
      kopds:
        image: ghcr.io/nlafevers/kopds:latest
        container_name: kopds
        restart: unless-stopped
        ports:
          - "8080:8080"
        read_only: true
        tmpfs:
          - /tmp
        volumes: # [HOST_PATH:CONTAINER_PATH:OPTIONS]
          # BIND MOUNT: Users map their existing Calibre library here (Read-Only)
          - /path/to/your/calibre/library:/library:ro
      
          # NAMED VOLUMES: Docker manages these automatically. No permission issues.
          - kopds_data:/data
          - kopds_cache:/cache
        environment:
          - KOPDS_LIBRARY_PATH=/library
          - KOPDS_DATABASE_PATH=/data/kopds.db
          - KOPDS_IMAGE_CACHE_PATH=/cache/images
          - KOPDS_LOG_LEVEL=info
          - KOPDS_PORT=8080
          - KOPDS_BASE_URL=http://DOMAIN_NAME:8080

      # You must declare named volumes at the bottom of the file
      volumes:
        kopds_data:
        kopds_cache:
    ```
3.  Start the container:
    ```bash
    docker compose up -d
    ```
4.  Create your initial admin user:
    ```bash
    printf '%s\n' 'yourpassword' | docker exec -i kopds ./kopds create-user admin --password-stdin
    ```
    To avoid exposing passwords to shell history or process listings, the create-user UX uses a hidden terminal prompt for inputting the password.  However, to allow Docker automation `--password-stdin` is an option.

### Method 2:
**Docker deployment using a locally built image**

1.  Download the latest release (and extract it), or clone the repository (from the latest tag):
    ```bash
    curl -s https://api.github.com/repos/nlafevers/kopds/releases/latest \
    | grep "tarball_url" \
    | cut -d : -f 2,3 \
    | tr -d \" \
    | xargs curl -L -o kopds-latest.tar.gz && mkdir kopds && \
    tar -xzf kopds-latest.tar.gz -C kopds --strip-components=1
    ```
    or
    
    ```bash
    wget -qO- https://api.github.com/repos/nlafevers/kopds/releases/latest \
    | grep tarball_url \
    | cut -d '"' -f 4 \
    | xargs wget -O kopds.tar.gz && mkdir kopds && \
    tar -xzf kopds.tar.gz -C kopds --strip-components=1
    ```
    or
    ```bash
    git clone --depth 1 --branch $(curl -s https://api.github.com/repos/nlafevers/kopds/releases/latest | grep "tag_name" | cut -d '"' -f 4) https://github.com/nlafevers/kopds.git
    ```
2.  Change to the project directory `cd $(ls -td kopds* | head -n 1)` and create a `docker-compose.yml` file.  Make sure to change the path to your Calibre library.  Make sure to change the KOPDS_BASE_URL to match the IP address or domain name that your Koreader devices use to connect.
    ```yaml
    services:
      kopds:
        build: .
        container_name: kopds
        restart: unless-stopped
        ports:
          - "8080:8080"
        read_only: true
        tmpfs:
          - /tmp
        volumes: # [HOST_PATH:CONTAINER_PATH:OPTIONS]
          # BIND MOUNT: Users map their existing Calibre library here (Read-Only)
          - /path/to/your/calibre/library:/library:ro
      
          # NAMED VOLUMES: Docker manages these automatically. No permission issues.
          - kopds_data:/data
          - kopds_cache:/cache
        environment:
          - KOPDS_LIBRARY_PATH=/library
          - KOPDS_DATABASE_PATH=/data/kopds.db
          - KOPDS_IMAGE_CACHE_PATH=/cache/images
          - KOPDS_LOG_LEVEL=info
          - KOPDS_PORT=8080
          - KOPDS_BASE_URL=http://DOMAIN_NAME:8080

      # You must declare named volumes at the bottom of the file
      volumes:
        kopds_data:
        kopds_cache:
3.  Start the container and create an admin user as in Steps 3 and 4 above.

### Method 3:
**Host-based deployment using a stand-alone binary**

1.  Download the source code as in Method 2: Step 1.
2.  Go to the project directory:
    ```bash
    cd $(ls -td kopds* | head -n 1)
    ```
3.  Build the binary:
    ```bash
    go build -o kopds ./cmd/kopds
    ```
4.  Configure `config.yaml`  (make sure to change the path to your Calibre library).
5.  Create your admin user:
    ```bash
    ./kopds create-user admin
    ```
6.  Run the server:
    ```bash
    ./kopds
    ```

## Deployment Guidelines

### Security & Reverse Proxy

KOPDS uses **HTTP Basic Authentication** for simplicity and compatibility with KOReader. Since Basic Auth transmits credentials in plain text, you **should** deploy KOPDS behind a reverse proxy (e.g., Caddy, Nginx, Traefik) that provides **HTTPS**.

### Storage Performance

For optimal performance, ensure the `data` directory (which holds the SQLite index) is stored on **local high-speed storage** (SSD/NVMe). While your Calibre library can reside on a high-latency network share (SMB/NFS), the KOPDS index database must be on local storage to prevent SQLite locking issues and ensure rapid response times.

## License
GPL-3.0 license
