# OpsTunnel

Cross-platform SSH Tunnel Manager with desktop app, web UI, and Docker support.

Manage SSH connections, create Local (-L) / Remote (-R) / Dynamic SOCKS5 (-D) tunnels through multi-hop SSH chains, with real-time monitoring and auto-reconnect.

## Features

- **SSH Connection Management** - Create, test, and reuse SSH connections across tunnels
- **Multi-hop SSH Chains** - Chain multiple SSH servers with drag-and-drop ordering
- **Three Tunnel Modes** - Local (-L), Remote (-R), Dynamic SOCKS5 (-D) with CONNECT + BIND
- **Auto Reconnect** - Supervisor with exponential backoff, rate limiting, and graceful shutdown
- **Desktop App** - Native window with system tray (color-coded icon, tunnel submenus, clipboard copy)
- **Web UI** - Responsive app-like interface accessible from any browser
- **Docker** - Multi-stage distroless image, one-command deployment
- **Real-time Monitoring** - WebSocket-driven status updates and live log streaming
- **Settings** - Theme switching (Light/Dark/System), log level, version check
- **API Security** - Bearer token auth, CORS, request size limits

## Quick Start

### Desktop

Download the latest release from [Releases](https://github.com/maxzhang666/ops-tunnel/releases) and run the app.

### Docker

```bash
docker run -d --name ops-tunnel \
  -p 9876:9876 \
  -v tunnel-data:/data \
  ghcr.io/maxzhang666/ops-tunnel:latest
```

Open http://localhost:9876

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/maxzhang666/ops-tunnel/main/docker-compose.yml
docker compose up -d
```

### Server Binary

```bash
./tunnel-server --listen 127.0.0.1:9876 --data-dir ./data
```

Environment variables: `TUNNEL_LISTEN`, `TUNNEL_DATA_DIR`, `TUNNEL_TOKEN`

## Development

```bash
# Prerequisites: Go 1.26+, Node 22+, pnpm

# Install frontend dependencies
make install-ui

# Run server + UI dev mode
make dev

# Run desktop app
make dev-desktop

# Build
make build                       # server + UI
VERSION=1.0.0 make build-desktop # desktop binary with version
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

## API

Default port: `9876`

```
GET    /healthz
GET    /ws                                WebSocket event stream

/api/v1:
  GET/POST       /ssh-connections
  GET/PUT/PATCH/DELETE /ssh-connections/{id}
  POST           /ssh-connections/{id}/test
  GET/POST       /tunnels
  GET/PUT/PATCH/DELETE /tunnels/{id}
  POST           /tunnels/{id}/start|stop|restart
  GET            /tunnels/{id}/status
  GET/PATCH      /settings
  GET            /version
```

## License

MIT
