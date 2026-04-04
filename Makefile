.PHONY: dev dev-server dev-ui build build-ui build-server clean install-ui

VERSION ?= dev

# Development
dev-server:
	go run ./cmd/tunnel-server --listen 127.0.0.1:9876

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

build-server: build-ui
	rm -rf cmd/tunnel-server/dist
	cp -r ui/dist cmd/tunnel-server/dist
	go build -ldflags="-X main.version=$(VERSION)" -o bin/tunnel-server ./cmd/tunnel-server

build: build-server

# Desktop
DESKTOP_LDFLAGS := -s -w -X main.version=$(VERSION)
ifeq ($(OS),Windows_NT)
  DESKTOP_LDFLAGS += -H=windowsgui
else ifeq ($(shell uname -s),Darwin)
  export CGO_LDFLAGS := -framework UniformTypeIdentifiers
endif

build-desktop: build-ui
	rm -rf cmd/tunnel-desktop/dist
	cp -r ui/dist cmd/tunnel-desktop/dist
	go build -tags desktop,production -ldflags="$(DESKTOP_LDFLAGS)" -o bin/tunnel-desktop ./cmd/tunnel-desktop

dev-desktop: build-desktop
	./bin/tunnel-desktop

# Clean
clean:
	rm -rf bin/ ui/dist/ cmd/tunnel-desktop/dist cmd/tunnel-server/dist
