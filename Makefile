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
