# Stage 1: Build frontend
FROM node:22-alpine AS build-ui
WORKDIR /src/ui
RUN corepack enable && corepack prepare pnpm@9 --activate
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY ui/ ./
RUN pnpm build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS build-server
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=build-ui /src/ui/dist ./cmd/tunnel-server/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o /tunnel-server ./cmd/tunnel-server
RUN mkdir -p /data && chown 65532:65532 /data

# Stage 3: Runtime
FROM gcr.io/distroless/static-debian12
COPY --from=build-server /tunnel-server /tunnel-server
COPY --from=build-server --chown=65532:65532 /data /data
EXPOSE 9876
VOLUME /data
USER nonroot:nonroot
ENTRYPOINT ["/tunnel-server", "--data-dir", "/data", "--listen", "0.0.0.0:9876"]
