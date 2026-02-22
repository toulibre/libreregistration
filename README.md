# LibreRegistration

A lightweight, self-hosted event registration platform. Let attendees sign up for your events through simple public forms — no account required, no tracking, no third-party dependencies. Deploy it on your own server in minutes with Docker.

Ideal for meetups, conferences, workshops, community gatherings, or any event where you need a simple and privacy-friendly registration page.

## Features

- **Event management** — create, edit, duplicate, and delete events with Markdown descriptions and image uploads
- **Privacy-friendly registration** — attendees only provide a name or nickname; email is optional
- **Self-service cancellation** — each registration gets a unique cancellation link, no account needed
- **Public attendee list** — optionally display the list of registered attendees on the event page
- **Multi-user admin panel** — dashboard with role-based access (admin and manager roles), global settings
- **CSV export** — download the attendee list for any event as a CSV file
- **Email notifications** — optional confirmation and cancellation emails via SMTP
- **No JavaScript required** — fully server-rendered HTML, works in any browser
- **SQLite or PostgreSQL** — use a single SQLite file for simplicity, or PostgreSQL for larger deployments

## Quick Start with Docker

### 1. Clone the repository

```bash
git clone https://github.com/toulibre/libreregistration.git
cd libreregistration
```

### 2. Configure the environment

```bash
cp .env.example .env
```

Edit `.env` and set at least these values:

```env
# Secrets (32+ characters each, must be changed)
SESSION_SECRET=your-random-session-secret-32ch
CSRF_KEY=your-random-csrf-key-of-32-chars

# Initial admin account (only used on first run)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=a-strong-password

# Public URL (adjust for production)
BASE_URL=http://localhost:8080
```

### 3. Start the application

```bash
docker compose up -d
```

The app is available at **http://localhost:8080**.

The admin panel is at **http://localhost:8080/admin/login**.

### 4. Stop the application

```bash
docker compose down
```

All data (SQLite database and uploads) is stored in a Docker volume named `data` and persists across restarts.

To delete all data:

```bash
docker compose down -v
```

## Database

LibreRegistration supports two database backends: **SQLite** (default) and **PostgreSQL**.

### SQLite (default)

No extra setup needed. Data is stored in a single file, easy to back up and migrate.

```env
DATABASE_DRIVER=sqlite
DATABASE_PATH=libreregistration.db
```

### PostgreSQL

For larger deployments or when you prefer a client-server database.

1. Start a PostgreSQL instance (e.g. with Docker):

```bash
docker run -d --name postgres \
  -e POSTGRES_DB=libreregistration \
  -e POSTGRES_PASSWORD=secret \
  -p 5432:5432 \
  postgres:17
```

2. Configure the environment:

```env
DATABASE_DRIVER=postgres
DATABASE_URL=postgres://postgres:secret@localhost:5432/libreregistration?sslmode=disable
```

When using `DATABASE_DRIVER=postgres`, the `DATABASE_PATH` variable is ignored and `DATABASE_URL` is used instead.

## Production Deployment

### Docker Compose

The provided `docker-compose.yml` is production-ready (SQLite):

```yaml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - data:/data
    environment:
      - UPLOAD_DIR=/data/uploads
    env_file:
      - .env
    restart: unless-stopped

volumes:
  data:
```

For PostgreSQL, add a `postgres` service and adjust the app environment:

```yaml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - uploads:/data/uploads
    environment:
      - UPLOAD_DIR=/data/uploads
      - DATABASE_DRIVER=postgres
      - DATABASE_URL=postgres://libreregistration:secret@postgres:5432/libreregistration?sslmode=disable
    env_file:
      - .env
    depends_on:
      - postgres
    restart: unless-stopped

  postgres:
    image: postgres:17
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=libreregistration
      - POSTGRES_USER=libreregistration
      - POSTGRES_PASSWORD=secret
    restart: unless-stopped

volumes:
  uploads:
  pgdata:
```

**Important notes:**

- With SQLite, the `data` volume holds both the database (`/data/libreregistration.db`) and uploaded files (`/data/uploads`)
- With PostgreSQL, only uploaded files need a volume — the database is managed by the PostgreSQL container
- Place a reverse proxy (nginx, Caddy, Traefik...) in front of the app for HTTPS
- Set `SESSION_SECRET` and `CSRF_KEY` to random strings of 32+ characters
- Set `BASE_URL` to your actual public URL (e.g. `https://registrations.myorg.fr`)
- SMTP variables are optional — the app works without email sending

### Reverse proxy example (Caddy)

```
registrations.myorg.fr {
    reverse_proxy localhost:8080
}
```

### Backups

**SQLite** — the database is a single file:

```bash
docker compose cp app:/data/libreregistration.db ./backup.db
```

**PostgreSQL** — use `pg_dump`:

```bash
docker compose exec postgres pg_dump -U libreregistration libreregistration > backup.sql
```

## Local Development (without Docker)

Prerequisites: Go 1.25+, [templ](https://templ.guide/)

```bash
# Install templ
go install github.com/a-h/templ/cmd/templ@latest

# Download the Tailwind CSS standalone CLI into bin/
mkdir -p bin
curl -sL https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.18/tailwindcss-macos-arm64 -o bin/tailwindcss
chmod +x bin/tailwindcss

# Configure and run
cp .env.example .env
export ADMIN_USERNAME=admin ADMIN_PASSWORD=changeme
make build
./bin/server
```

Open http://localhost:8080 in a browser.

### Make commands

| Command | Description |
|---|---|
| `make build` | Generate templ templates, compile CSS, build the binary |
| `make run` | Run the server |
| `make dev` | Build + run |
| `make css-watch` | Recompile CSS on every change (development) |
| `make clean` | Remove build artifacts |

## Configuration

| Variable | Description | Default |
|---|---|---|
| `PORT` | Server port | `8080` |
| `DATABASE_DRIVER` | Database backend (`sqlite` or `postgres`) | `sqlite` |
| `DATABASE_PATH` | SQLite database file path | `libreregistration.db` |
| `DATABASE_URL` | PostgreSQL connection URL | — |
| `BASE_URL` | Public URL of the site | `http://localhost:8080` |
| `SESSION_SECRET` | Session secret (32+ characters) | — |
| `CSRF_KEY` | CSRF key (32+ characters) | — |
| `ADMIN_USERNAME` | Initial admin username | — |
| `ADMIN_PASSWORD` | Initial admin password | — |
| `UPLOAD_DIR` | Directory for uploaded files | `uploads` |
| `SMTP_HOST` | SMTP server (optional) | — |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USER` | SMTP username | — |
| `SMTP_PASSWORD` | SMTP password | — |
| `SMTP_FROM` | Sender email address | — |

## Tech Stack

- **Go** with chi (router), templ (HTML templates), SQLite (modernc.org/sqlite) or PostgreSQL (pgx)
- **Tailwind CSS v4** (standalone CLI, no Node.js required)
- Server-rendered HTML, no JavaScript framework

## Built with AI

This project was built with the assistance of AI.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
