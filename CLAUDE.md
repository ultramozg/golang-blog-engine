# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests with race detection and coverage
go test -v -race -coverprofile=coverage.out ./...

# Run a single package's tests
go test -v -race ./app/...
go test -v -race ./model/...

# Run a specific test
go test -v -run TestName ./app/...

# Lint
golangci-lint run

# Security scan
gosec ./...

# Build for Linux/amd64 (deployment target: Orange Pi)
make build

# Build for ARM (alternative deployment target)
make build-arm

# Run locally (uses defaults: port :8080, SQLite at database/database.sqlite)
go run main.go
```

## Configuration

All config is read from environment variables at startup (see [app/config.go](app/config.go)). Key vars with defaults:

| Variable | Default |
|---|---|
| `DBURI` | `file:database/database.sqlite` |
| `ADMIN_PASSWORD` | `12345` |
| `HTTP_PORT` | `:8080` |
| `HTTPS_PORT` | `:8443` |
| `TEMPLATES` | `templates/*.gohtml` |
| `PRODUCTION` | `false` |
| `DOMAIN` | `` (empty = localhost) |

GitHub OAuth vars: `GITHUB_AUTHORIZE_URL`, `GITHUB_TOKEN_URL`, `REDIRECT_URL`, `CLIENT_ID`, `CLIENT_SECRET`.

## Architecture

**Entry point:** `main.go` → `app.NewApp()` → `a.Initialize()` → `a.Run()`

**`App` struct** ([app/app.go](app/app.go)) is the central object wiring everything together: HTTP router, SQLite DB, HTML templates, session store, OAuth config, and the three services.

**Request flow:**
```
HTTP request
  → LogMiddleware
  → securityMiddleware (auth gate for /create, /update, /delete, /upload-file; login required for comments)
  → PostRedirectMiddleware (301 /post?id=N → /p/<slug> for SEO)
  → GzipMiddleware
  → SetHeaderMiddleware
  → mux (routes)
```

**Packages:**

- **`model/`** — SQLite data layer. `Post`, `Comment`, `User`, `File` structs with direct `*sql.DB` methods. Schema is created and migrated in `MigrateDatabase` / `MigrateExistingDatabase` at every startup (idempotent `CREATE TABLE IF NOT EXISTS` + `ALTER TABLE` checks).

- **`services/`** — Business logic extracted from handlers:
  - `SlugService` — generates URL slugs from titles, ensures uniqueness
  - `FileService` — handles file uploads (10 MB limit), image thumbnail generation, UUID-based storage under `uploads/`
  - `SEOService` — generates meta tags, Open Graph tags, JSON-LD structured data, sitemap XML, robots.txt

- **`session/`** — In-memory session store (`map[string]model.User` behind `sync.RWMutex`). Sessions are not persisted; they are lost on restart. Two session types: `ADMIN` (local login) and `GITHUB` (OAuth).

- **`middleware/`** — Standard `http.Handler` wrappers: logging, gzip, cache-control, TLS redirect, and `PostRedirectMiddleware` which takes a `*sql.DB` to resolve post slugs.

- **`templates/`** — Go HTML templates (`.gohtml`). Two custom template functions registered in `app.go`: `processFileReferences` (converts `[file:filename]` markers to inline HTML) and `extractExcerpt` (strips HTML for list views).

- **`testutils/`** — Shared test helpers: `NewTestApp()` creates a fully wired `App` against a temporary SQLite DB, pointing at the real `templates/` directory relative to the test file location.

**Authentication model:** Admin logs in via username/password (bcrypt). GitHub OAuth is for comment authors — GitHub users can post comments but cannot administer the blog. The `securityMiddleware` enforces this at the route level.

**Database migrations** run at every startup. New columns are added via `ALTER TABLE` only if they don't already exist (checked via `pragma_table_info`). No migration framework — keep this pattern for schema changes.

**File embedding in posts:** Post bodies can contain `[file:filename]` tokens that `processFileReferences` resolves at render time by querying the `files` table. Images render as `<img>` tags with thumbnail support; other files render as download links.
