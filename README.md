# Mana

Mana is a small, responsive conference catering guide. It presents scheduled meals, drinks, and permanently available snacks at one central serving station. German and English are fully supported.

## Features

- responsive and accessible design without a frontend framework
- server-rendered HTML
- DE/EN switch that follows the browser's language preferences and falls back to German
- YAML as the single source of truth for menu content
- templates and assets embedded in one Go binary
- gzip compression and long-lived browser caching for slow or crowded Wi-Fi
- health endpoint and secure HTTP headers
- multi-stage Docker image running as a non-root user
- hardened Docker Compose configuration with a read-only container filesystem

## Start with Docker Compose

```bash
docker compose up --build
```

Mana is then available at [http://localhost:8080](http://localhost:8080). The health endpoint is exposed at `/healthz`.

## Edit the menu

All content lives in `content/menu.yaml`. Every user-facing value has a German and an English variant:

```yaml
name:
  de: Gemüse-Curry
  en: Vegetable curry
```

Docker Compose bind-mounts the `content` directory from the host. Edit `content/menu.yaml` normally with any host-side editor. Mana checks for updates at most twice per second, so no container restart or rebuild is required. If an edit temporarily produces invalid YAML, Mana keeps serving the most recent valid menu.

The directory mount is read-only (`:ro`) from the container's perspective. This prevents the application from accidentally changing the source file while it remains fully editable on the host, including with editors that save by replacing the file. Reload the page in the browser to see an update.

Without `MENU_PATH`, the application uses the menu file embedded at build time.

## Local development

Go 1.24 or newer is required.

```bash
go mod download
go run .
```

Run tests and create a local binary:

```bash
go test ./...
go build -o bin/mana .
```

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port used by the web server |
| `MENU_PATH` | empty | Optional path to an external YAML menu file |
