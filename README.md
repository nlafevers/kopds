# KOPDS - Lightweight OPDS Server

KOPDS is a high-performance, lightweight OPDS (Open Publication Distribution System) server designed specifically for self-hosting Calibre libraries. It is engineered for large libraries (10,000+ books) hosted on high-latency network shares (e.g., Nextcloud, SMB, NFS) and is perfectly tailored for the KOReader ecosystem.

## Why KOPDS?

While many OPDS servers exist, KOPDS focuses on three core pillars:

1.  **Extreme Performance:** By mirroring your Calibre `metadata.db` to a local, optimized SQLite index, KOPDS provides near-instant search and navigation, even when your library is stored on a slow network share.
2.  **Resource Efficiency:** Built in pure Go, KOPDS has a minimal memory footprint and compiles to a single, portable binary, making it ideal for low-power devices like Raspberry Pis or home servers.
3.  **KOReader Optimization:** Designed with the specific quirks and requirements of KOReader in mind, ensuring a seamless book discovery and acquisition experience.

## Core Features

- **OPDS 1.2 Support:** Fully compatible with KOReader and other standard OPDS clients.
- **Background Synchronization:** Automatically detects changes in your Calibre library and keeps the local index up-to-date without blocking API requests.
- **Instant Search:** Powered by SQLite FTS5 for rapid, full-text search across titles, authors, tags, and series.
- **Production-Ready:** Structured logging, multi-user support, and comprehensive test coverage.
- **Zero-Dependency Architecture:** Minimal external requirements; perfect for containerized deployments.
- **Clean Architecture Approach:** Domain logic is separated from infrastructure concerns. It features a background scanner that incrementally synchronizes your library, an optimized media delivery pipeline, and a robust API layer for OPDS delivery.

## Quick Start

### Prerequisites
- Go 1.22 or higher.

### Installation
```bash
git clone https://github.com/nlafevers/kopds
cd kopds
go build -o kopds ./cmd/kopds
```

### Configuration
Create a `config.yaml` file in the project root:

```yaml
library_path: /path/to/your/calibre/library
database_path: ./data/kopds.db
port: 8080
log_level: info
```

### Running
```bash
./kopds --config config.yaml
```

### Development
This project is still under active development and is not yet ready for deployment.

## License
GPL-3.0 license
