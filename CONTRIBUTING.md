# Contributing

## Prerequisites

- Go 1.26+
- Node 22+
- pnpm

## Development

```bash
# Install frontend dependencies
make install-ui

# Run server + UI dev mode
make dev

# Run desktop app
make dev-desktop

# Build production
make build                        # server + UI
VERSION=1.0.0 make build-desktop  # desktop binary with version

# Docker
docker compose up --build
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.26, chi/v5, golang.org/x/crypto/ssh |
| Frontend | React 19, TypeScript, Vite, Tailwind 4, shadcn/ui |
| Desktop | Wails v2, fyne.io/systray |
| State | TanStack Query v5, WebSocket |
| CI/CD | GitHub Actions, GHCR, Docker multi-stage |

## Project Structure

```
cmd/
  tunnel-server/    Headless HTTP+WS API server
  tunnel-desktop/   Wails desktop app with system tray
internal/
  config/           Data model, validation, file persistence
  ssh/              SSH auth, host keys, chain building, keepalive
  engine/           Tunnel supervisor, backoff, event bus
  forward/          Local/Remote/Dynamic forwarder implementations
  api/              HTTP API, WebSocket, middleware
ui/                 React SPA
```

## Conventions

- Default port: **9876**
- PATCH uses pointer-based structs (nil = skip)
- Frontend: all pages use `export default`, lazy-loaded via `React.lazy`
- Frontend: HTTP+WS for all API (not Wails bindings) — same code for Desktop/Server
- Config persisted to `data/config.json` via FileStore
- Comments in English
