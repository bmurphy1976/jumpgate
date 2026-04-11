# Jumpgate

A self-hosted bookmark dashboard. Single Go binary, SQLite database, color themes.

## Features

- Categorized bookmarks with a favorites section
- Full-text search (by name, URL, keywords)
- Built-in color themes
- Weather widget with browser geolocation
- Admin panel with drag-and-drop reordering
- Private links and categories (shown only when authenticated)
- Mobile URL support (alternate URLs for mobile devices)
- YAML import/export for initial setup and backup
- Single binary deployment
- Docker support

## Quick Start

### Docker

```bash
docker build -t jumpgate .
docker run -p 8080:8080 -v ./data:/app/data jumpgate
```

### Local

```bash
make build
./bin/jumpgate server
```

Dashboard at `http://localhost:8080`, admin at `http://localhost:8080/admin`.

For local development, use `make debug` to start with auth checks disabled.

Versioning details live in [VERSIONING.md](VERSIONING.md). Release notes live in [RELEASE_NOTES.md](RELEASE_NOTES.md).

## Release Commands

```bash
make version         # Print the stamped service version
make next-version    # Print the next CalVer tag version
make release-prepare # Create the release branch, commit VERSION, push, and open the PR
make release-publish # Tag the merged release commit on main and create the GitHub Release
```

## CLI

```
jumpgate server [--db PATH] [--addr ADDR] [--no-auth]
jumpgate import [--force] [--db PATH] config.yaml
jumpgate export [--force] [--db PATH] [output.yaml]
```

**Environment variables:**
- `DB_FILE` — SQLite database path (default: `data/jumpgate.db`)
- `LISTEN_ADDR` — Listen address (default: `:8080`)

## Configuration

Jumpgate stores all data in a SQLite database. Manage everything through the admin panel, or import bookmarks from a YAML file (see `bookmarks.yaml.example`).

### Weather

Configure weather location and units in the admin panel settings. The weather widget uses the [Open-Meteo API](https://open-meteo.com/) and supports browser geolocation to override the default coordinates.

## Themes

Built-in themes are the `.css` files in `static/themes/`.

### Creating a Theme

1. Copy an existing file in `static/themes/` (e.g., `monokai.css` → `mytheme.css`)
2. Set the CSS custom properties:
   - `--color-background`, `--color-primary`, `--color-accent`, `--color-hover`
   - `--color-surface`, `--color-border`, `--color-danger`, `--color-success`
3. Rebuild — the theme will be auto-discovered

## Authentication

Jumpgate delegates authentication to a reverse proxy. The admin panel and private content require one of these headers to be set:

- `X-Authorized-User`
- `X-User`
- `X-Remote-User`

Works with Authelia, OAuth2 Proxy, Traefik Forward Auth, or any proxy that sets user headers.

**Important:** Your reverse proxy must strip these headers from incoming requests before setting them, so end users cannot forge authentication. Documentation and configuration examples will be provided in the future.

## Demo Mode Proxy Allowlist

Demo mode isolates sessions by cookie and client IP. Set `demo.allowed_proxies` when demo mode runs behind a proxy or CDN and should use `X-Forwarded-For`. Set `demo.disable_proxy_headers: true` to ignore forwarded headers and use the direct connection address instead.

Migration details for this breaking change are in [RELEASE_NOTES.md](RELEASE_NOTES.md).

## Project Structure

```
cmd/jumpgate/       CLI entry point (server, import, export)
handlers/           HTTP handlers (dashboard + admin)
views/              Templ templates (.templ files)
model/              Go data structures
storage/            Datasource interface + SQLite implementation
config/             YAML config parsing (import/export)
icons/              MDI icon loader and cache
static/             Embedded assets (CSS, JS, themes)
data/               Runtime data (SQLite DB, icon cache)
```
