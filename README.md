# KOPDS - Lightweight OPDS Server

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/nlafevers/kopds)](https://goreportcard.com/report/github.com/nlafevers/kopds)

<p align="center">
<img width="768" height="780" alt="KOPDS screenshot" src="https://github.com/user-attachments/assets/84e829a7-77e4-45d7-bbc8-3d4f61f50b9f" />
</p>

KOPDS is a high-performance, lightweight OPDS (Open Publication Distribution System) server designed specifically for self-hosting Calibre libraries. It is optimized for large libraries (10,000+ books) hosted on high-latency network shares (e.g., Nextcloud, SMB, NFS) and is tailored for the KOReader ecosystem.

---

## 📖 Table of Contents

1.  [Why KOPDS?](#-why-kopds)
2.  [Key Features](#-key-features)
3.  [Prerequisites](#-prerequisites)
4.  [Quick Start (Docker)](#-quick-start-docker)
5.  [Usage with KOReader](#-usage-with-koreader)
6.  [Native Installation](#-native-installation)
7.  [Configuration Reference](#-configuration-reference)
8.  [Advanced Logging](#-advanced-logging)
9.  [Technical Architecture](#-technical-architecture)
10. [Security & Deployment](#-security--deployment)
11. [Troubleshooting](#-troubleshooting)
12. [License](#-license)

---

## 🚀 Why KOPDS?

While many OPDS servers exist, KOPDS focuses on three core pillars:

1.  **High Performance:** By mirroring your Calibre `metadata.db` to a local, optimized SQLite index, KOPDS provides near-instant search and navigation, even when your library is stored on a slow network share. **Note:** Only metadata and resized thumbnails are mirrored; your book files stay exactly where they are until requested.
2.  **Resource Efficiency:** Built in pure Go, KOPDS has a minimal memory footprint and compiles to a single, portable binary (~15MB), making it ideal for low-power devices like Raspberry Pis, home servers, or free-tier cloud VMs.
3.  **KOReader Optimization:** Designed with the specific requirements of KOReader in mind, ensuring a seamless book discovery and acquisition experience.

---

## ✨ Key Features

- **OPDS 1.2 Support:** Fully compatible with KOReader and other standard OPDS clients.
- **Background Synchronization:** Automatically detects changes in your Calibre library and keeps the local index up-to-date without blocking API requests.
- **Instant Search:** Powered by SQLite FTS5 for rapid, full-text search across titles, authors, tags, and series.
- **Efficient Image Pipeline:** On-the-fly cover resizing with high-quality Lanczos resampling and an LRU disk cache.
- **Multi-User Support:** Secure your library with HTTP Basic Authentication and bcrypt-hashed passwords.
- **Production-Ready:** Structured logging, graceful shutdown, and containerized deployment options.

---

## 📋 Prerequisites

Before you begin, ensure your environment meets the following requirements.

### Data Requirements
- **Calibre Library:** A folder containing your books and the `metadata.db` file.

### Software Requirements

#### 1. If using Docker
- **Docker and Docker Compose:** These need to be installed on the host machine. To check if you have them, run:
```bash
docker --version
docker compose version
```
- *If you don't have them, follow the [official Docker installation guide](https://docs.docker.com/get-docker/).*

#### 2. If installing Natively
- **Go compiler:** you need version 1.25+. To check your version, run:
```bash
go version
```
- *If you don't have it, download it from [go.dev](https://go.dev/dl/). No C compiler is required as KOPDS uses a pure-Go SQLite driver.*

#### 3. Cloud/Remote Calibre Library Synchronization
- KOPDS is designed for serving a remote Calibre library efficiently, but you will need a way to sync the remote library to the machine running KOPDS.  Rclone is recommended over davfs2 due to better stability, and more graceful handling of network blips.  Rclone also offers finer control allowing you to throttle the connection if needed to avoid overwhelming a resource constrained server during the initial library metadata scan.  Rclone is also written in Go, maintaining a pure Go environment.

- *To install Rclone, see the [official documentation](https://rclone.org/install/).*

#### 4. Reverse Proxy ([recommended](#reverse-proxy))
- While KOPDS itself uses HTTP Basic Authentication according to the OPDS 1.2 spec, for security reasons you should place it behind a reverse proxy.  Caddy is recommended to keep a pure-Go environment.  Additionally, other services you might want to run off the same server might need HTTPS, such as the KOReader sync server.
> [!NOTE]
> A reverse proxy alone does not make your server completely secure.  You are responsible for properly configuring your server to meet your security needs.

- *To install Caddy, see the [official documentation](https://caddyserver.com/docs/install).*

### Hardware Requirements

One reason to prefer deploying natively with a Go binary is to minimize resource usage in constrained server setups.  A free-tier GCP e2-micro VM only has 1 GB of memory, and early Raspberry Pi's have even less.  Even if the overhead consumed by Docker is as low as often claimed 100-200 MB (and not closer 300-400 MB), that is still a significant proportion of your available RAM on a micro cloud VM or early-generation Raspberry Pi.  The Go binary running natively should consume less than half that (~90 MB).  Running your entire stack natively, if using Caddy (20-30 MB) and Rclone (20-40 MB), would consume less RAM than the Docker overhead by itself.

The other hardware requirements are potato-tier.  See recommended below:

| Specification | Native Dual (kopds + kosync) | Docker Dual (kopds + kosync) | Native kosync | Native kopds |
| :-----------: | :--------------------------: | :--------------------------: | :----------:  | :----------: |
| CPU           | 1 Core (1.0 GHz)             | 1 Core (1.0 GHz)    | 1 Core (Any speed) | 1 Core (1.0 GHz) |
| RAM (Idle)    | ~100 MB                      | ~350 MB                      | < 15 MB       | ~90 MB       |
| RAM (Minimum) | 512 MB*                      | 1 GB*<sup>†</sup>            | 64 MB         | 512 MB*      |
| Storage Space | ~250 MB                      | ~1.5 GB                      | ~25 MB        | ~200 MB      |
| Network       | 1+ Mbps                      | 1+ Mbps                      |	< 1 Mbps      | 1+ Mbps      |

_*Assumes rclone is used to mount remote storage. A swap file is highly recommended to prevent Out-of-Memory (OOM) crashes during initial directory scans._

_†1 GB will likely not be sufficient if you intend to build your own Docker image locally_

---

## 🐳 Quick Start (Docker)

The easiest way to run KOPDS is via Docker. This method ensures all dependencies are handled and simplifies updates.

### 1. Prepare Your Environment
Create a directory for KOPDS and move into it:
```bash
mkdir ~/kopds && cd ~/kopds
```

### 2. Create Docker Compose File
Create a file named `docker-compose.yml` and paste the following content. **Make sure to edit the host path to your Calibre library and `KOPDS_BASE_URL`.**

```yaml
services:
  kopds:
    image: ghcr.io/nlafevers/kopds:latest # or build: .
    container_name: kopds
    restart: unless-stopped
    ports:
      - "8080:8080"
    read_only: true
    tmpfs:
      - /tmp
    volumes:
      # Path to your Calibre library (keep this read-only)
      - /path/to/your/calibre/library:/library:ro # [HOST_DIR:CONTAINER_DIR:OPTIONS]

      # Persistence for KOPDS index and cache (should be on local SSD)
      - kopds_data:/data
      - kopds_cache:/cache
    environment:
      - KOPDS_LIBRARY_PATH=/library
      - KOPDS_DATABASE_PATH=/data/kopds.db
      - KOPDS_IMAGE_CACHE_PATH=/cache/images
      - KOPDS_BASE_URL=http://your-server-ip:8080 # Change to your IP/Domain
      - KOPDS_LOG_LEVEL=info
      - KOPDS_PORT=8080
      - KOPDS_JSON_LOG=true # Recommended for Docker

volumes:
  kopds_data:
  kopds_cache:
```

### 3. Launch KOPDS
Start the server in the background:
```bash
docker compose up -d
```

### 4. Create Your Admin User
KOPDS requires authentication. Create your first user with the following command:
```bash
docker exec -it kopds ./kopds create-user admin
```
Follow the prompts to set a secure password.

> [!TIP]
> For automation, you can use the `--password-stdin` flag:
> `echo "mypassword" | docker exec -i kopds ./kopds create-user admin --password-stdin`

---

## 📱 Usage with KOReader

1.  Open **KOReader**.
2.  Tap the top menu (while in the file browser) and select the **Search** icon (magnifying glass).
3.  Select **OPDS catalog** -> **Add new catalog**.
4.  Enter a name (e.g., "Home Library").
5.  Enter the URL: `http://your-server-ip:8080/opds/v1.2/catalog`
6.  Enter the **Username** and **Password** you created in Quick Start - Step 4.
7.  Save.
8.  Tap your new catalog to browse and download your books!
> [!NOTE]
> For large libraries it is possible your sub-menus (navigation feeds in OPDS terminology, eg. Authors, Series, Tags, etc.) will extend to more than 3 or 4 pages.  A limitation of OPDS is that only the first few pages are loaded, so the total page count displayed in KOReader is not necessarily accurate when you first enter a sub-menu.  Additionally, if you navigate to the last page (`>>`), when the page count is not correct, you will stall there, and need to paginate backwards (`<`) then forwards (`>`) to access the later menu pages.

---

## 🛠 Native Installation

For users who prefer running KOPDS without Docker, you can use one of the provided binaries (see Releases), or build one yourself.

### 1. Build from Source
```bash
git clone https://github.com/nlafevers/kopds.git
```
or, to download only the latest branch without the entire commit history
```bash
git clone --depth 1 --branch $(curl -s https://api.github.com/repos/nlafevers/kopds/releases/latest | grep "tag_name" | cut -d '"' -f 4) https://github.com/nlafevers/kopds.git
```
then
```bash
cd kopds
go build -o kopds ./cmd/kopds
```

### 2. Configure
KOPDS can be configured via environment variables or a `config.yaml` file in the same directory (or a `./config` subdirectory). 

> [!NOTE]
> Environment variables always take precedence over settings in `config.yaml`. In Docker, environment variables are the standard way to configure the container, but you can also mount a `config.yaml` to `/app/config.yaml` if you prefer.

```bash
# Set required environment variables
export KOPDS_LIBRARY_PATH=/path/to/calibre
./kopds create-user admin
./kopds
```

---

## ⚙️ Configuration Reference

All settings can be provided as environment variables (prefixed with `KOPDS_`) or in a `config.yaml` file.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `KOPDS_LIBRARY_PATH` | **Required.** Path to your Calibre library folder. | - |
| `KOPDS_DATABASE_PATH` | Path where the local SQLite index will be stored. | `kopds.db` |
| `KOPDS_BASE_URL` | The external URL used for generating OPDS links. | `http://your-server-ip:8080` |
| `KOPDS_PORT` | The port the server listens on. | `8080` |
| `KOPDS_LOG_LEVEL` | Logging verbosity (`debug`, `info`, `warn`, `error`). | `info` |
| `KOPDS_JSON_LOG` | Enable structured JSON logging (best for ELK/Loki). | `false` |
| `KOPDS_SYNC_INTERVAL` | How often to scan Calibre for changes (e.g., `1h`, `30m`). | `30m` |
| `KOPDS_IMAGE_CACHE_PATH` | Directory for resized cover thumbnails. | `cache/images` |
| `KOPDS_IMAGE_CACHE_MAX_COUNT` | Maximum number of images to keep in cache. | `1000` |

---

## 📊 Advanced Logging

KOPDS uses structured logging via `zerolog` to provide clear and actionable insights into the server's operation.

### Log Formats
- **Human-Readable (Default):** Optimized for terminal viewing with colors and formatted timestamps. Best for local development and native deployments.
- **JSON:** Structured output that is easy to parse by log aggregators like **Promtail/Loki**, **Elasticsearch**, or **CloudWatch**. Enable this with `KOPDS_JSON_LOG=true`.

### Log Levels
You can adjust the verbosity of the logs using the `KOPDS_LOG_LEVEL` setting:

- **`debug`:** Use this when troubleshooting. It provides granular details about the background scanner (e.g., which books are being indexed) and internal routing.
- **`info`:** The recommended level for production. Reports server startup, synchronization batches, and incoming requests.
- **`warn`:** Only logs non-critical issues, such as failed cover resizing for a specific book or minor synchronization skips.
- **`error`:** Only logs critical failures that require attention, such as database connection issues or inability to access the library share.

---

## 🏗 Technical Architecture

KOPDS is designed for speed and reliability, especially in home lab environments where libraries are often stored on network-attached storage (NAS).

### Hybrid Database Strategy
Calibre's `metadata.db` is a complex SQLite database that isn't optimized for OPDS serving and can be slow over network shares.
- KOPDS treats `metadata.db` as a **read-only source of truth**.
- It maintains a **local SQLite index** using FTS5 for blazing-fast search.
- **This does NOT mirror your entire library.** Only the metadata (titles, authors, etc.) is copied to the local index. Your actual EPUB/PDF files are streamed directly from the source library only when you click "Download."

### Background Incremental Sync
The scanner engine uses a multi-tier change detection system:
1.  **File Stats:** Checks the modification time and size of `metadata.db`.
2.  **Timestamp Comparison:** If the file changed, it compares the `last_modified` timestamps of individual books to perform an incremental update, rather than a full re-index.
3.  **Pruning:** Automatically removes books from the local index that have been deleted from Calibre.

### Optimized Image Pipeline
To ensure cover thumbnails load instantly on e-ink devices:
- **Resizing:** Uses the `disintegration/imaging` library with Lanczos resampling for high-quality, sharp thumbnails.
- **Caching:** Implements a disk-based LRU (Least Recently Used) cache. Once a cover is resized, it's served instantly from the local SSD for subsequent requests. This prevents the server from having to read large image files across the network more than once.
- **Security:** Bounds image dimensions and input sizes to prevent DoS attacks.

---

## 🔒 Security & Deployment

### Reverse Proxy
KOPDS uses **HTTP Basic Authentication**. While simple and widely compatible, it transmits credentials in plain text. **You should always deploy KOPDS behind a reverse proxy** (like Caddy, Nginx, or Traefik) that provides **HTTPS**.

**Example Caddyfile:**
```caddy
kopds.example.com {
    reverse_proxy localhost:8080
}
```
> [!NOTE]
> A reverse proxy alone does not make your server completely secure.  You are responsible for properly configuring your server to meet your security needs.

### Storage Performance
For the best experience:
- **Calibre Library:** Can be on a slow HDD or network share (SMB/NFS).
- **KOPDS Data/Cache:** **Should** be on local high-speed storage (SSD/NVMe). This ensures the SQLite index and image cache are highly responsive.

---

## ❓ Troubleshooting

### "Unauthorized" error in KOReader
- Double-check your username and password.
- Ensure your `KOPDS_BASE_URL` is set correctly. If it doesn't match the URL you're using to access the server, some links might be broken.

### Books are not showing up
- Check the logs: `docker logs kopds`.
- Ensure `KOPDS_LIBRARY_PATH` points to a folder containing `metadata.db`.
- KOPDS might still be performing the initial scan. Large libraries can take a few minutes to index the first time.

### Covers are missing
- Calibre stores covers in book directories as `cover.jpg`. Ensure these files exist and are readable by KOPDS.
- Check that the `cache` directory is writable.

### 404 error when you attempt to download a book
- Double-check that your remote library is still mounted.  The OPDS server will feed a list of books from its own SQLite database, so it will appear as if there are books available, but when you try to download one you will get a 404 error if the library is not still mounted.

### Client tries to download a directory instead of a book
- This can happen if you give KOReader login credentials without actually creating a username and password on the KOPDS server.

---

## 📜 License

KOPDS is released under the **GPL-3.0 License**. See the [LICENSE](LICENSE) file for details.
