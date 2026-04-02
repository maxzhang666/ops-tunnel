# Phase 0: Project Scaffolding — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap OpsTunnel with a Go HTTP server (healthz endpoint), React+shadcn/ui frontend, and dev/build toolchain.

**Architecture:** `cmd/tunnel-server/main.go` parses flags and starts `internal/api.Server` (chi router). Frontend is a Vite+React SPA with shadcn/ui. In dev mode, Vite proxies API calls to Go. In production, Go serves the built frontend as static files.

**Tech Stack:** Go 1.26, chi v5, React 18, TypeScript, Vite, shadcn/ui (preset b1a1cZciO), pnpm, Makefile

---

## File Map

| File | Purpose |
|------|---------|
| `go.mod` | Go module definition |
| `cmd/tunnel-server/main.go` | CLI entry: parse flags, start server, handle signals |
| `internal/api/server.go` | Server struct, Run(), Shutdown() |
| `internal/api/routes.go` | Route registration, healthz handler, SPA static serving |
| `ui/` | React SPA (created by Vite + shadcn/ui init) |
| `ui/vite.config.ts` | Vite config with API proxy |
| `ui/src/App.tsx` | Root component with health check display |
| `Makefile` | Dev/build/clean targets |
| `.gitignore` | Ignore bin/, data/, node_modules/, dist/ |

---

## Task 1: Go Module + Dependencies

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go mod init github.com/maxzhang666/ops-tunnel
```

Expected: `go.mod` created with `module github.com/maxzhang666/ops-tunnel` and `go 1.26`.

- [ ] **Step 2: Add chi dependency**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go get github.com/go-chi/chi/v5
```

Expected: `go.mod` updated with `require github.com/go-chi/chi/v5` and `go.sum` created.

- [ ] **Step 3: Create .gitignore**

Create `.gitignore`:

```
bin/
data/
ui/node_modules/
ui/dist/
.DS_Store
*.exe
*.test
*.out
```

- [ ] **Step 4: Commit**

```bash
git init
git add go.mod go.sum .gitignore
git commit -m "chore: init Go module with chi dependency"
```

---

## Task 2: API Server + Healthz Endpoint

**Files:**
- Create: `internal/api/server.go`
- Create: `internal/api/routes.go`

- [ ] **Step 1: Create `internal/api/server.go`**

```go
package api

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Config struct {
	ListenAddr string
	UIDir      string
	Token      string
}

type Server struct {
	cfg    Config
	router chi.Router
	http   *http.Server
}

func NewServer(cfg Config) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	s := &Server{
		cfg:    cfg,
		router: r,
	}
	s.registerRoutes()
	return s
}

func (s *Server) Run(ctx context.Context) error {
	s.http = &http.Server{
		Addr:    s.cfg.ListenAddr,
		Handler: s.router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	slog.Info("server starting", "addr", s.cfg.ListenAddr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.http != nil {
		return s.http.Shutdown(ctx)
	}
	return nil
}
```

- [ ] **Step 2: Create `internal/api/routes.go`**

```go
package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (s *Server) registerRoutes() {
	s.router.Get("/healthz", s.handleHealthz)

	if s.cfg.UIDir != "" {
		s.serveSPA(s.cfg.UIDir)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"ts":     time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) serveSPA(dir string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return
	}

	fileServer := http.FileServer(http.Dir(absDir))

	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(absDir, r.URL.Path)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for non-file routes
		indexPath := filepath.Join(absDir, "index.html")
		if _, err := fs.Stat(os.DirFS(absDir), "index.html"); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		http.NotFound(w, r)
	})
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go build ./internal/api/
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/api/
git commit -m "feat: add API server with healthz endpoint and SPA serving"
```

---

## Task 3: CLI Entry Point

**Files:**
- Create: `cmd/tunnel-server/main.go`

- [ ] **Step 1: Create `cmd/tunnel-server/main.go`**

```go
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/api"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	dataDir := flag.String("data-dir", "./data", "data directory")
	uiDir := flag.String("ui-dir", "", "static UI files directory")
	token := flag.String("token", "", "bearer token for API auth")
	flag.Parse()

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		slog.Error("failed to create data dir", "path", *dataDir, "err", err)
		os.Exit(1)
	}

	srv := api.NewServer(api.Config{
		ListenAddr: *listen,
		UIDir:      *uiDir,
		Token:      *token,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Run(ctx); err != nil {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}
```

- [ ] **Step 2: Build and test**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go build -o bin/tunnel-server ./cmd/tunnel-server
```

Expected: `bin/tunnel-server` created.

```bash
bin/tunnel-server --listen 127.0.0.1:8080 &
SERVER_PID=$!
sleep 1
curl -s http://127.0.0.1:8080/healthz
kill $SERVER_PID
```

Expected output from curl: `{"status":"ok","ts":"2026-..."}`.

- [ ] **Step 3: Commit**

```bash
git add cmd/tunnel-server/
git commit -m "feat: add tunnel-server CLI entry point"
```

---

## Task 4: React + Vite + shadcn/ui Frontend

**Files:**
- Create: `ui/` (full Vite project)
- Modify: `ui/vite.config.ts` (add proxy)
- Modify: `ui/src/App.tsx` (health check UI)

- [ ] **Step 1: Scaffold Vite React project**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
pnpm create vite@latest ui -- --template react-ts
cd ui
pnpm install
```

Expected: `ui/` directory with React+TS template, `node_modules/` populated.

- [ ] **Step 2: Initialize shadcn/ui**

```bash
cd /Users/maxzhang/Tools/ops-tunnel/ui
pnpm dlx shadcn@latest init --preset b1a1cZciO
```

Follow prompts to complete initialization. This sets up Tailwind CSS, `components.json`, and the shadcn/ui foundation.

- [ ] **Step 3: Add Card component from shadcn/ui**

```bash
cd /Users/maxzhang/Tools/ops-tunnel/ui
pnpm dlx shadcn@latest add card badge
```

- [ ] **Step 4: Update `ui/vite.config.ts` with API proxy**

Replace the entire file content with:

```typescript
import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
    },
  },
})
```

Note: The `resolve.alias` and `plugins` sections may already exist from shadcn/ui init. Merge the `server.proxy` block into whatever is already there. Keep all existing content and only add the `server` block.

- [ ] **Step 5: Replace `ui/src/App.tsx`**

```tsx
import { useEffect, useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

function App() {
  const [status, setStatus] = useState<"checking" | "connected" | "disconnected">("checking")

  useEffect(() => {
    fetch("/healthz")
      .then((res) => {
        if (res.ok) setStatus("connected")
        else setStatus("disconnected")
      })
      .catch(() => setStatus("disconnected"))
  }, [])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <Card className="w-80">
        <CardHeader>
          <CardTitle className="text-center text-2xl">OpsTunnel</CardTitle>
        </CardHeader>
        <CardContent className="flex justify-center">
          {status === "checking" && <Badge variant="outline">Checking...</Badge>}
          {status === "connected" && <Badge variant="default">Connected</Badge>}
          {status === "disconnected" && <Badge variant="destructive">Disconnected</Badge>}
        </CardContent>
      </Card>
    </div>
  )
}

export default App
```

- [ ] **Step 6: Remove unused boilerplate files**

```bash
cd /Users/maxzhang/Tools/ops-tunnel/ui
rm -f src/App.css src/assets/react.svg public/vite.svg
```

Also clean up `ui/src/index.css` — shadcn/ui init should have already set this up with the `@import "tailwindcss"` directive. If the Vite boilerplate CSS is still there, replace the contents with just the shadcn/ui imports.

- [ ] **Step 7: Verify frontend builds**

```bash
cd /Users/maxzhang/Tools/ops-tunnel/ui
pnpm build
```

Expected: `ui/dist/` created with `index.html` and JS/CSS assets.

- [ ] **Step 8: Commit**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
git add ui/
git commit -m "feat: add React frontend with shadcn/ui and health check"
```

---

## Task 5: Makefile

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Create `Makefile`**

```makefile
.PHONY: dev dev-server dev-ui build build-ui build-server clean install-ui

# Development
dev-server:
	go run ./cmd/tunnel-server --listen 127.0.0.1:8080

dev-ui:
	cd ui && pnpm dev

dev:
	$(MAKE) dev-server & $(MAKE) dev-ui & wait

# Install frontend dependencies
install-ui:
	cd ui && pnpm install

# Build
build-ui:
	cd ui && pnpm build

build-server:
	go build -o bin/tunnel-server ./cmd/tunnel-server

build: build-ui build-server

# Clean
clean:
	rm -rf bin/ ui/dist/
```

- [ ] **Step 2: Test `make build`**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
make build
```

Expected: `ui/dist/` built and `bin/tunnel-server` compiled.

- [ ] **Step 3: Test production mode (Go serving built frontend)**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
bin/tunnel-server --listen 127.0.0.1:8080 --ui-dir ui/dist &
SERVER_PID=$!
sleep 1
curl -s http://127.0.0.1:8080/healthz
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/
kill $SERVER_PID
```

Expected: healthz returns JSON, root returns `200` with HTML.

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile for dev/build workflows"
```

---

## Task 6: End-to-End Verification

- [ ] **Step 1: Full dev mode test**

Terminal 1:
```bash
cd /Users/maxzhang/Tools/ops-tunnel
make dev-server
```

Terminal 2:
```bash
cd /Users/maxzhang/Tools/ops-tunnel
make dev-ui
```

Open browser at `http://localhost:5173`. Verify:
- Page shows "OpsTunnel" heading
- Badge shows "Connected" (green)

- [ ] **Step 2: Full production mode test**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
make build
bin/tunnel-server --listen 127.0.0.1:8080 --ui-dir ui/dist
```

Open browser at `http://localhost:8080`. Verify same result.

- [ ] **Step 3: Add remaining project files and final commit**

Add `init.md`, `PLAN.md`, and docs to the repository:

```bash
cd /Users/maxzhang/Tools/ops-tunnel
git add init.md PLAN.md docs/
git commit -m "docs: add project plan and Phase 0 spec"
```
