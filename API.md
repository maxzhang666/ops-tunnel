# API Reference

Base URL: `http://localhost:9876`

## Authentication

Requests to `/api/v1/*` require authentication when enabled.

**Session cookie** (browser): Sign in via `POST /api/v1/auth/login` — the server sets an `HttpOnly` session cookie.

**Bearer token** (scripts/API): Pass the `Authorization: Bearer <token>` header. The token is configured via `TUNNEL_TOKEN` env var or `--token` flag.

When neither `TUNNEL_ADMIN_PASSWORD` nor `TUNNEL_TOKEN` is set, all endpoints are accessible without authentication.

## Endpoints

```
GET    /healthz                              Health check
GET    /ws                                   WebSocket event stream

/api/v1/auth:
  POST   /login                              Sign in (returns session cookie)
  POST   /logout                             Sign out (clears session)
  GET    /check                              Check auth status

/api/v1/ssh-connections:
  GET    /                                   List all SSH connections
  POST   /                                   Create SSH connection
  POST   /test                               Test connection without saving
  GET    /{id}                               Get SSH connection
  PUT    /{id}                               Replace SSH connection
  PATCH  /{id}                               Partial update SSH connection
  DELETE /{id}                               Delete SSH connection
  POST   /{id}/test                          Test saved connection

/api/v1/tunnels:
  GET    /                                   List all tunnels
  POST   /                                   Create tunnel
  GET    /{id}                               Get tunnel
  PUT    /{id}                               Replace tunnel
  PATCH  /{id}                               Partial update tunnel
  DELETE /{id}                               Delete tunnel
  POST   /{id}/start                         Start tunnel
  POST   /{id}/stop                          Stop tunnel
  POST   /{id}/restart                       Restart tunnel
  GET    /{id}/status                        Get tunnel status

/api/v1:
  GET    /settings                           Get settings
  PATCH  /settings                           Update settings
  GET    /version                            Get version info
  GET    /traffic/realtime                   Real-time traffic samples
  GET    /traffic/history?range=24h&step=5m  Historical traffic data
  GET    /stats                              Global statistics
```
