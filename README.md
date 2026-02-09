# FileHub

FileHub is a lightweight file management service with a Web UI and a CLI.

- Backend: Go + Gin + SQLite
- Storage: MinIO (server-side proxy)
- Frontend: Vue3 (built and embedded into the Go binary)
- Deploy: single binary or Docker

## Features

- Upload / list / preview / download / delete files
- Share links: open in browser and jump to `/file/{id}` (SPA route)
- Video streaming (Range requests)
- Auth
  - Web UI: JWT login + refresh tokens
  - CLI/Agent: `X-Local-Key` only (no username/password stored)
- Audit logs for key actions

## Quick Start (Docker)

Requirements: Docker + Docker Compose

```bash
docker-compose up -d
```

Open:
- Web UI: `http://localhost:8080`
- MinIO console: `http://localhost:9001`

MinIO lifecycle:
- The Docker setup initializes a rule to abort incomplete multipart uploads after 1 day.

Default dev config is in `config.yaml`.

## One-Click Install (Docker + CLI)

Linux/amd64, pulls the latest GitHub Release image from GHCR and installs `filehub-cli`.

```bash
curl -fsSL https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh | bash
```

Options example:
```bash
curl -fsSL https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh | bash -s -- \
  --port 18080 \
  --version v0.1.0
```

Notes:
- The installer keeps existing config and data if present.
- It pulls `kirydocker/filehub:latest` by default, and falls back to the latest release tag if needed.

## Recommended Server Layout

If you deploy on a server, keep all runtime state in a single directory:

```text
/opt/filehub/
  docker-compose.yml
  config.yaml
  data/
    filehub.db
    minio/
```

This makes upgrades and backups simple.

## Upgrade (Docker)

If you installed via the one-click script, the runtime directory is `~/.filehub` by default.

1) Backup (recommended)

```bash
cd ~/.filehub
tar -czf filehub-backup-$(date +%F).tar.gz config.yaml data/
```

2) Pull a new version and restart

```bash
cd ~/.filehub
docker compose pull
docker compose up -d
```

If you pin versions (recommended), update the `image:` tag in `docker-compose.yml` (e.g. `ghcr.io/kiry163/filehub:v0.1.1`) and then run the commands above.

## Backup & Restore

What to backup:
- `config.yaml`
- `data/` (SQLite DB + MinIO data)

Backup:

```bash
cd ~/.filehub
tar -czf filehub-backup-$(date +%F).tar.gz config.yaml data/
```

Restore:

```bash
cd ~/.filehub
docker compose down
rm -rf data config.yaml
tar -xzf filehub-backup-YYYY-MM-DD.tar.gz
docker compose up -d
```

## CLI

Initialize config:

```bash
filehub-cli config init \
  --endpoint http://localhost:8080 \
  --local-key filehub-local-key \
  --public-endpoint http://localhost:8080
```

Commands:

```bash
# upload
filehub-cli upload ./myfile.zip

# list
filehub-cli list --limit 10

# share (prints browser URL)
filehub-cli share filehub://<id>

# download
filehub-cli download filehub://<id> --output ./downloads

# delete
filehub-cli delete filehub://<id>

# backup (compress ~/.filehub/data)
filehub-cli backup

# backup excludes MinIO internal metadata (.minio.sys)
```

## Configuration

Server config file: `config.yaml`

Key fields:
- `server.port`: HTTP port
- `database.path`: SQLite path
- `auth.admin_username` / `auth.admin_password`: Web login
- `auth.jwt_secret`: JWT signing secret
- `auth.local_key`: CLI key (`X-Local-Key`)
- `minio.*`: MinIO connection

Environment variables override YAML (examples):

```bash
FILEHUB_SERVER_PORT=8080
FILEHUB_DATABASE_PATH=./data/filehub.db
FILEHUB_AUTH_LOCAL_KEY=your-local-key
FILEHUB_MINIO_ENDPOINT=minio:9000
```

## Web Routes

- `/` files list
- `/upload` upload page
- `/file/{id}` file detail
- `/login` login page

The Go server serves embedded static assets and uses SPA fallback (non-`/api/*` routes return `index.html`).

## API

Base: `/api/v1`

Auth:
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`

Files:
- `POST /files` (upload)
- `GET /files` (list)
- `GET /files/{id}` (meta)
- `GET /files/{id}/download` (download, supports Range)
- `DELETE /files/{id}`
- `GET /files/{id}/share` (returns public download URL, valid for 7 days)
- `GET /files/{id}/preview` (returns a short-lived stream URL)
- `GET /files/stream?token=...` (streaming endpoint)

## Build

Local dev:

```bash
# backend
go run ./cmd/filehub

# frontend (dev)
cd web-ui
npm ci
npm run dev
```

Production single binary (embed web-ui):

```bash
cd web-ui
npm ci
npm run build
rm -rf ../web/dist
cp -r dist ../web/dist

cd ..
go build -o filehub ./cmd/filehub
```

## CI/CD

- CI: `/.github/workflows/ci.yml`
- Release (tag `v*`): builds and pushes `ghcr.io/kiry163/filehub:<tag>` and creates a GitHub Release with binaries.
