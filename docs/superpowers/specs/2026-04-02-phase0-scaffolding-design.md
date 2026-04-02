# Phase 0: Project Scaffolding Design

## Goal

Bootstrap the OpsTunnel project: Go HTTP server with health endpoint, React SPA with shadcn/ui, and a dev/build toolchain that ties them together. After this phase, `make dev` starts both servers and you can see the frontend at localhost:5173 with API proxying to Go at 8080.

## Scope

**In scope:**
- Go module init (`github.com/maxzhang666/ops-tunnel`)
- `cmd/tunnel-server/main.go` with CLI flags
- `internal/api/` with chi router, healthz endpoint, static file serving
- `ui/` with Vite + React + TypeScript + shadcn/ui (preset `b1a1cZciO`)
- Makefile for dev/build workflows
- `.gitignore`

**Out of scope:**
- Wails desktop (Phase 10)
- Any tunnel logic, config storage, or WebSocket
- Tests (nothing meaningful to test yet)

## Architecture

```
cmd/tunnel-server/main.go
    │ parses flags: --listen, --data-dir, --ui-dir, --token
    │ creates api.Server
    │ handles SIGINT/SIGTERM → graceful shutdown
    ▼
internal/api/server.go
    │ Server struct holds chi.Router + config
    │ Run(ctx) starts http.Server
    │ Shutdown(ctx) graceful stop
    ▼
internal/api/routes.go
    │ GET /healthz → {"status":"ok","ts":"..."}
    │ Static file serving from ui-dir (if provided)
    ▼
ui/ (Vite + React + shadcn/ui)
    │ vite.config.ts: proxy /api/** and /healthz to Go server
    │ App.tsx: minimal page with "OpsTunnel" title
```

## Go Server Detail

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--listen` | `127.0.0.1:8080` | HTTP listen address |
| `--data-dir` | `./data` | Data directory (for future config storage) |
| `--ui-dir` | `""` | Static files directory (empty = no static serving) |
| `--token` | `""` | Bearer token (empty = no auth, placeholder for Phase 8) |

Use Go stdlib `flag` package. No external CLI framework needed at this stage.

### API Server

- Router: `chi.NewRouter()`
- Middleware (Phase 0 minimal): `chi.middleware.Logger`, `chi.middleware.Recoverer`
- `GET /healthz` returns:
  ```json
  {"status": "ok", "ts": "2026-04-02T12:00:00Z"}
  ```
- Static file serving: if `ui-dir` flag is set and directory exists, serve files at `/` using `http.FileServer`. SPA fallback: non-API routes that don't match a file return `index.html`.

### Graceful Shutdown

- `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`
- On signal: `httpServer.Shutdown(5s timeout)`

## Frontend Detail

### Init

```bash
cd ui
pnpm create vite@latest . -- --template react-ts
pnpm dlx shadcn@latest init --preset b1a1cZciO
```

### Vite Config

```typescript
export default defineConfig({
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
    },
  },
})
```

### App.tsx

Minimal page:
- Display "OpsTunnel" heading
- On mount, fetch `/healthz` and display connection status (connected/disconnected)
- Use shadcn/ui `Card` component for the status display

## Makefile

```makefile
.PHONY: dev dev-server dev-ui build build-ui build-server clean

dev-server:
	go run ./cmd/tunnel-server --listen 127.0.0.1:8080

dev-ui:
	cd ui && pnpm dev

dev:
	$(MAKE) dev-server & $(MAKE) dev-ui & wait

build-ui:
	cd ui && pnpm build

build-server:
	go build -o bin/tunnel-server ./cmd/tunnel-server

build: build-ui build-server

clean:
	rm -rf bin/ ui/dist/ ui/node_modules/ ui/.pnpm-store/
```

## .gitignore

```
bin/
data/
ui/node_modules/
ui/dist/
.DS_Store
```

## Acceptance Criteria

1. `make dev` starts Go server + Vite dev server without errors
2. `curl http://localhost:8080/healthz` returns `{"status":"ok",...}`
3. Browser at `http://localhost:5173` shows the OpsTunnel page
4. The page successfully calls `/healthz` through Vite proxy and shows "connected"
5. `make build` produces `bin/tunnel-server` binary
6. `bin/tunnel-server --ui-dir ui/dist` serves the built frontend at `http://localhost:8080/`
