# LibreRegistration

Event registration web app for French non-profit associations ("Associations Loi 1901").

## Tech Stack

- **Go** with chi router, templ templates, SQLite (modernc.org/sqlite, pure Go)
- **Tailwind CSS v4** (standalone CLI)
- No JavaScript frameworks, server-rendered HTML forms

## Architecture

```
cmd/server/main.go          → entry point
internal/config/             → env var loading
internal/database/           → stores (user, event, registration, setting) + migrations
internal/handlers/           → HTTP handlers (auth, event, registration, admin)
internal/services/           → business logic (auth, event, registration, settings)
internal/middleware/          → logging, session, csrf, auth, flash, method override
internal/models/             → domain structs
internal/slug/               → French-aware slug generation
internal/mail/               → optional SMTP sender
templates/                   → templ templates (layouts, public, admin)
static/css/                  → Tailwind input/output CSS
```

## Build & Run

```bash
make build    # templ generate + tailwindcss + go build
make run      # run the binary
make dev      # build + run
make clean    # remove build artifacts
```

Requires `templ` CLI: `go install github.com/a-h/templ/cmd/templ@latest`

## Environment Variables

See `.env.example`. Key ones:
- `ADMIN_USERNAME` / `ADMIN_PASSWORD` — seeds admin on first run
- `SESSION_SECRET` / `CSRF_KEY` — 32+ char secrets
- `DATABASE_PATH` — SQLite file path (default: `libreregistration.db`)

## Conventions

- All user-facing strings are in French
- HTML forms use POST with `_method` hidden field for PUT/DELETE
- CSRF protection on all forms via gorilla/csrf
- Sessions via gorilla/sessions (cookie-based)
- Markdown rendering for event descriptions via goldmark
- UUID primary keys everywhere
