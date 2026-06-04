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
8.  [CLI User Management](#-cli-user-management)
9.  [Logging](#-logging)
10. [Technical Overview](#-technical-overview)
11. [Security](#-security)
12. [Troubleshooting](#-troubleshooting)
13. [License](#-license)

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
- **Calibre Library:** A local or remote folder containing your books and the `metadata.db` file.

### Software Requirements

#### 1. If using Docker
- **Docker and Docker Compose:** These need to be installed on the host machine. To check if you have them, run:
```bash
docker --version
docker compose version
```
- *If you don't have them, follow the [official Docker installation guide](https://docs.docker.com/get-docker/).*

#### 2. If installing Natively
- **Go compiler:** you need version 1.25.x or later. To check your version, run:
```bash
go version
```
- *If you don't have it, download it from [go.dev](https://go.dev/dl/). No C compiler is required as KOPDS uses a pure-Go SQLite driver.*

#### 3. Cloud/Remote Calibre Library Synchronization
- KOPDS is designed for serving a remote Calibre library efficiently, but you will need a way to sync the remote library to the machine running KOPDS.  Rclone is recommended over davfs2 due to better stability, and more graceful handling of network blips.  Rclone also offers finer control allowing you to throttle the connection if needed to avoid overwhelming a resource constrained server during the initial library metadata scan.  Rclone is also written in Go, maintaining a pure Go environment.
- *To install Rclone, see the [official documentation](https://rclone.org/install/).*

#### 4. Reverse Proxy (recommended for production)
- KOPDS uses HTTP Basic Authentication, which transmits credentials in plain text. For any internet-facing deployment you should place it behind an HTTPS reverse proxy (Caddy is a good pure-Go choice). Reverse-proxy setup instructions live in the KOSERVER project deployment guide.
> [!NOTE]
> A reverse proxy alone does not make your server completely secure.  You are responsible for properly configuring your server to meet your security needs.

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
Create a file named `deploy/docker-compose.yml` and paste the following content. **Make sure to edit the host path to your Calibre library and `KOPDS_BASE_URL`.**

```yaml
services:
  kopds:
    build:
      context: ..
      dockerfile: build/Dockerfile
    image: ghcr.io/nlafevers/kopds:latest
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
To force a local build, rather than downloading the image from the Github Container Registry add the `--build` option.
```bash
docker compose up -d --build
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

### 2. Run as a non-root user
Create a dedicated system user to run the service securely, and give it ownership of the binary and the data directory.
```bash
sudo useradd -r -s /usr/sbin/nologin kopds
sudo mkdir -p data
sudo chown -R kopds:kopds kopds data
```

### 3. Configure and Run
KOPDS reads its settings from environment variables or a `config.yaml` file (see [Configuration Reference](#-configuration-reference) for every option). At minimum, set the path to your Calibre library, create a user, and start the server:
```bash
export KOPDS_LIBRARY_PATH=/path/to/calibre
sudo -u kopds ./kopds create-user admin
sudo -u kopds ./kopds
```

> [!NOTE]
> Environment variables always take precedence over settings in `config.yaml`. In Docker, environment variables are the standard way to configure the container, but you can also mount a `config.yaml` to `/app/config.yaml` if you prefer.

---

## ⚙️ Configuration Reference

All settings can be provided as environment variables (prefixed with `KOPDS_`) or in a `config.yaml` file placed in the working directory (or a `./config` subdirectory).

| Variable | Description | Default |
| :--- | :--- | :--- |
| `KOPDS_LIBRARY_PATH` | Path to your Calibre library folder. | - |
| `KOPDS_DATABASE_PATH` | Path where the local SQLite index will be stored. | `./data/kopds.db` |
| `KOPDS_BASE_URL` | The external URL used for generating OPDS links. | `http://localhost:8080` |
| `KOPDS_PORT` | The port the server listens on. | `8080` |
| `KOPDS_LOG_LEVEL` | Logging verbosity (`debug`, `info`, `warn`, `error`). | `info` |
| `KOPDS_JSON_LOG` | Enable structured JSON logging (best for ELK/Loki). | `false` |
| `KOPDS_LOG_PATH` | Optional log file. When set, the server writes logs to this file **and** stderr; CLI commands log to this file only. | - |
| `KOPDS_SYNC_INTERVAL` | How often to scan Calibre for changes (e.g., `1h`, `30m`). | `30m` |
| `KOPDS_IMAGE_CACHE_PATH` | Directory for resized cover thumbnails. | `cache/images` |
| `KOPDS_IMAGE_CACHE_MAX_COUNT` | Maximum number of images to keep in cache. | `1000` |
| `KOPDS_STORAGE_CAP_MB` | Maximum database size in MB (0 to disable). | `0` |
| `KOPDS_RATE_LIMIT_ENABLED` | Enable rate limiting on failed authentication attempts. | `true` |
| `KOPDS_RATE_LIMIT_PER_MINUTE` | Maximum failed auth attempts allowed per minute per IP. | `30` |
| `KOPDS_RATE_LIMIT_BURST` | Maximum burst size for failed auth rate limiting. | `10` |
| `KOPDS_TRUST_PROXY_HEADERS` | Trust `X-Forwarded-For` headers for client IP detection (enable only behind a trusted reverse proxy). | `false` |

---

## 🖥 CLI User Management

KOPDS includes a built-in CLI for managing users securely without exposing passwords in your shell history.

### Create a User
```bash
./kopds create-user <username>
```
You will be prompted to enter and confirm a password. The characters will not be visible. The command will fail if the user already exists. To change an existing user's password, use the `change-password` command.

### Change a Password
```bash
./kopds change-password <username>
```
Useful for resetting a user's password or regular security updates.

### Delete a User
```bash
./kopds delete-user <username>
```
This will permanently remove the user from the database.

### Automated Setup (Non-interactive)
For Docker initialization or scripts, you can use the `--password-stdin` flag:
```bash
echo "mypassword" | ./kopds create-user admin --password-stdin
```

User-management commands create and migrate the configured database automatically, so initial setup can create the first user before the server has been started.

---

## 📊 Logging

KOPDS uses structured logging via the Go standard library `slog` to provide clear and actionable insights into the server's operation. All logs include a `request_id` for correlating multiple events from a single request.

### Log Formats
- **Human-Readable (Default):** Optimized for terminal viewing. Best for local development and native deployments.
- **JSON:** Structured output that is easy to parse by log aggregators like **Promtail/Loki**, **Elasticsearch**, or **CloudWatch**. Enable this with `KOPDS_JSON_LOG=true`.

### Log Destinations
When `KOPDS_LOG_PATH` is set, the **server** writes structured logs to both stderr and that file. **CLI** commands (`create-user`, `delete-user`, `change-password`) write structured logs to the file only — or discard them when no path is set — so the terminal shows only the one-line human-readable result.

**Docker note:** `docker exec` runs in a separate process — its output goes directly to your terminal, not through Docker's logging driver. CLI user-management commands therefore never appear in `docker logs` regardless of log settings. If you need a persistent audit trail of CLI operations, set `KOPDS_LOG_PATH` to a path on a mounted volume (e.g., `/data/kopds.log`) and read that file directly.

### Log Levels
You can adjust the verbosity of the logs using the `KOPDS_LOG_LEVEL` setting:

- **`debug`:** troubleshooting detail. Shows query-level database diagnostics, authentication success details, and individual file sync events.
- **`info`:** Recommended for production. Reports server startup, database initialization, background sync milestones, and completed HTTP requests.
- **`warn`:** Logs handled problems, such as client authentication failures (401), invalid request paths (404), or storage cap pruning events.
- **`error`:** Logs critical failures requiring attention, such as database corruption, library access errors, or server-side crashes (500).

### Example Logs

**Healthy Request (INFO):**
`time=2026-05-27T10:00:00Z level=INFO msg="request completed" method=GET path="/opds/v1.2/catalog" request_id=eae37b8c status_code=200 duration=189ms remote_addr=192.168.1.50`

**Auth Failure (WARN):**
`time=2026-05-27T10:05:00Z level=WARN msg="client error" method=GET path="/opds/v1.2/catalog" request_id=5d8bd0a7 status_code=401 duration=194ms remote_addr=192.168.1.51`

**Scanner Diagnostic (DEBUG):**
`time=2026-05-27T10:10:00Z level=DEBUG msg="getting user by username" username=testuser request_id=eae37b8c`

### Troubleshooting with Logs
1. **Missing Books:** Set `KOPDS_LOG_LEVEL=debug` and check for `Initial sync failed` or `Periodic sync failed` messages.
2. **Slow Performance:** Check the `duration` field in `request completed` logs. If database queries are slow, check for `DEBUG` logs from the repository layer.
3. **Auth Issues:** Look for `WARN` logs with `status_code=401`. The `request_id` will help you find the corresponding `auth failure` or `getting user` logs.

---

## 🏗 Technical Overview

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

### Storage Performance
For the best experience:
- **Calibre Library:** Can be on a slow HDD or network share (SMB/NFS).
- **KOPDS Data/Cache:** **Should** be on local high-speed storage (SSD/NVMe). This ensures the SQLite index and image cache are highly responsive.

---

## 🔒 Security

KOPDS uses **HTTP Basic Authentication**. It is simple and widely compatible, but it transmits credentials in plain text, so you should **always run KOPDS behind an HTTPS reverse proxy** (such as Caddy, Nginx, or Traefik).

Step-by-step deployment instructions — reverse proxy, firewall, backups, and running KOPDS and KOSYNC together — live in the KOSERVER project deployment guide.

> [!NOTE]
> A reverse proxy alone does not make your server completely secure.  You are responsible for properly configuring your server to meet your security needs.

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
