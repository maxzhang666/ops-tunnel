# OpsTunnel Roadmap

## Phase 13: Config Import/Export

**Goal:** One-click backup/restore, cross-device migration.

**Scope:**
- Export: download current `config.json` (with optional redaction of sensitive fields)
- Import: upload JSON, validate, merge or replace
- UI: Settings > About section, "Export" / "Import" buttons

**Design considerations:**
- Export should strip `id` fields or keep them? Keep — enables exact restore
- Sensitive data (passwords, private keys): offer "Export with secrets" / "Export redacted" toggle
- Import conflict resolution: if SSH connection IDs collide, overwrite or skip?
- File format: raw config.json (v1 schema) — no custom format, directly restorable
- Desktop: use Wails `runtime.SaveFileDialog` / `runtime.OpenFileDialog`
- Server/Docker: standard download/upload via browser

**API:**
- `GET /api/v1/config/export?redact=true|false` — returns config JSON
- `POST /api/v1/config/import` — accepts config JSON body, validates, saves

**Estimated effort:** Small

---

## Phase 14: Traffic Statistics + Dashboard [DONE]

**Goal:** Dashboard as the default homepage with real-time bandwidth curve, per-tunnel traffic, and global status overview.

### 14.1 Backend: Traffic Counters

**Byte counting in forwarders:**
- Replace `biCopy(local, remote)` with `biCopyCount(local, remote, &bytesIn, &bytesOut)`
- Use `atomic.Int64` counters on each forwarder (bytesIn, bytesOut)
- Extend `forward.Status` with `BytesIn`, `BytesOut` fields
- Extend `engine.MappingStatus` and `TunnelStatus` with aggregated traffic

**Real-time bandwidth sampling:**
- New `TrafficSampler` goroutine: every 1s, snapshot all tunnel bytesIn/bytesOut deltas
- Store samples in in-memory ring buffer (last 300 samples = 5 minutes)
- Expose via new API endpoint

### 14.2 Backend: Traffic Persistence (BBolt)

**Storage:** `go.etcd.io/bbolt` — pure Go embedded KV, no CGO dependency.

**Schema:**
```
Bucket: "traffic"
  Key:   "{tunnelID}:{timestamp_unix}"  (8+8 bytes binary key)
  Value: { bytesIn, bytesOut, conns }   (24 bytes binary value)
```

**Write strategy:**
- Flush aggregated per-tunnel stats every 60s to BBolt
- On tunnel stop, flush immediately
- Retention: auto-prune entries older than 30 days on startup

**Read strategy:**
- Dashboard chart queries: `GET /api/v1/traffic?range=24h&step=5m`
- Server aggregates BBolt entries into time buckets
- Returns `[{ ts, bytesIn, bytesOut }]` array

**File location:** `data/traffic.db` alongside `config.json`

### 14.3 API

```
GET /api/v1/traffic/realtime
  → { samples: [{ ts, bytesIn, bytesOut }], interval: 1 }
  → Last 300 1-second samples from in-memory ring buffer

GET /api/v1/traffic/history?range=24h&step=5m
  → { series: [{ ts, bytesIn, bytesOut }] }
  → Aggregated from BBolt, supports range: 1h/6h/24h/7d

GET /api/v1/tunnels/{id}/status
  → Extended with bytesIn, bytesOut per mapping and total
```

### 14.4 Frontend: Dashboard Page

**Route:** `/dashboard` as default homepage (replace current `/ssh` redirect)

**Layout:**
```
+--[ 3 Running ]--[ 1 Stopped ]--[ 0 Error ]--+
|                                               |
|  Global Stats Cards:                          |
|  [ Total Traffic: 4.2GB ] [ Connections: 142 ]|
|                                               |
+-----------------------------------------------+
|                                               |
|  Real-time Bandwidth Curve (5min)             |
|  ↑ Upload (blue)  ↓ Download (green)          |
|  Area chart, updates every 1s via polling     |
|                                               |
+-----------------------------------------------+
|                                               |
|  Tunnel Cards Grid:                           |
|  +--DB Tunnel----+  +--SOCKS Proxy--+         |
|  | ● Running  2m |  | ● Running 15m |         |
|  | ↑ 120MB ↓45MB |  | ↑ 800MB ↓2GB  |         |
|  | 12 conns      |  | 89 conns      |         |
|  +---------------+  +---------------+         |
|                                               |
+-----------------------------------------------+
```

**Chart library:** recharts (lightweight, React-native, supports area/line charts)

**Data fetching:**
- Status cards: existing `GET /tunnels` + status queries (3s poll)
- Real-time chart: `GET /traffic/realtime` polled every 2s
- Tunnel cards: reuse existing status data + traffic fields

### 14.5 Sidebar Update

- New "Dashboard" entry with chart icon at the top of sidebar nav
- `/` route redirects to `/dashboard` instead of `/ssh`

### Implementation Order

1. Backend: byte counting in `biCopy` + forwarder Status extension
2. Backend: TrafficSampler + realtime API
3. Backend: BBolt persistence + history API
4. Frontend: Dashboard page + recharts chart + tunnel cards
5. Frontend: sidebar + routing update

**Estimated effort:** Medium-Large (split into sub-tasks)

---

## Phase 15: System Proxy Integration + PAC

**Goal:** D-type tunnels auto-configure system proxy or generate PAC scripts.

**Scope:**
- Desktop (macOS): use `networksetup` to set/clear SOCKS proxy on tunnel start/stop
- Desktop (Windows): modify registry `Internet Settings` for SOCKS proxy
- Desktop (Linux): set `GNOME`/`KDE` proxy via gsettings/kwriteconfig
- Server/Docker: generate PAC script file accessible via HTTP
- UI: per-tunnel toggle "Set as system proxy" (Desktop), "PAC URL" display (Server)

**Design considerations:**
- Only one tunnel can be system proxy at a time — need exclusive toggle
- On tunnel stop/crash, must always clear system proxy (cleanup hook)
- PAC script: `function FindProxyForURL(url, host) { return "SOCKS5 addr:port; DIRECT"; }`
- Serve PAC at `/proxy.pac` endpoint — browsers can auto-configure via this URL
- Desktop cleanup on crash: register OS-level cleanup or clear proxy on app startup
- Security: PAC endpoint should not be behind token auth (browsers need direct access)

**PAC generation:**
```javascript
function FindProxyForURL(url, host) {
  // Generated by OpsTunnel
  return "SOCKS5 127.0.0.1:9122; SOCKS 127.0.0.1:9122; DIRECT";
}
```

**API:**
- `GET /proxy.pac` — serves PAC for the first running dynamic tunnel (no auth)
- Per-tunnel toggle in config: `policy.systemProxy: bool`

**Estimated effort:** Medium (platform-specific code for Desktop, PAC is straightforward)

---

## Phase 16: Desktop Auto-Start [DONE]

**Goal:** Launch OpsTunnel on system startup via Settings toggle.

**Scope:**
- macOS: LaunchAgent plist in `~/Library/LaunchAgents/`
- Windows: Startup folder shortcut
- Linux: XDG `.desktop` file in `~/.config/autostart/`
- UI: Settings > General > "Auto Start" toggle (desktop-only)
- Syncs OS state on boot and on settings change via EventBus

**Estimated effort:** Small

---

## Phase 17: SSH Connection Pooling

**Goal:** Multiple tunnels sharing the same SSH hop reuse a single connection.

**Design considerations:**
- Reference-counted connection pool keyed by SSH connection ID
- Pool lives in Engine, forwarders receive the shared `*ssh.Client`
- Release when last tunnel using a connection stops
- Reconnect propagates to all dependent tunnels
- Reduces authentication overhead and server-side connection limits

**Estimated effort:** Medium

---

## Phase 18: Latency Monitoring

**Goal:** Real-time RTT display per tunnel hop.

**Design considerations:**
- Periodic SSH `SendRequest("keepalive@openssh.com")` round-trip measurement
- Store last N samples in ring buffer, expose via status API
- Frontend: small latency badge on tunnel card and detail view
- Reuse existing keepalive goroutine, add timing measurement

**Estimated effort:** Small

---

## Phase 19: SSH Key Generation

**Goal:** Generate SSH key pairs in-app, optionally deploy to remote host.

**Scope:**
- Generate ed25519 (default) or RSA keys
- Display public key for manual copy
- Optional: `ssh-copy-id` equivalent — deploy public key to remote `authorized_keys`
- UI: SSH Connections page, "Generate Key" button

**Estimated effort:** Small

---

## Phase 20: Config Encryption

**Goal:** Encrypt sensitive fields (passwords, private keys) at rest.

**Design considerations:**
- AES-256-GCM encryption with a master key
- Master key derived from user passphrase via Argon2
- Only encrypt sensitive fields, keep config.json human-readable for non-sensitive parts
- Prompt for passphrase on startup if encryption is enabled
- Migration path: existing plaintext configs auto-encrypt on first save

**Estimated effort:** Medium

---

## Phase 21: Tunnel Clone / Templates

**Goal:** Duplicate existing tunnel configs, save reusable templates.

**Scope:**
- Clone: copy tunnel config with new ID and "(Copy)" name suffix
- Templates: save/load tunnel presets (e.g., "SOCKS proxy via jumpbox")
- UI: clone button on tunnel card, template picker in new tunnel dialog

**Estimated effort:** Small

---

## Phase 22: Tunnel Groups + Bulk Operations

**Goal:** Organize tunnels by project/environment, start/stop by group.

**Scope:**
- Add optional `group` string field to Tunnel config
- UI: group headers in tunnel list, collapsible sections
- Bulk start/stop per group
- API: `POST /tunnels/bulk/start` with `{ group: "production" }`

**Estimated effort:** Medium

---

## Phase 23: Search & Filter

**Goal:** Quick lookup in tunnel and SSH connection lists.

**Scope:**
- Frontend: search bar above lists, filter by name/host/state/mode
- Client-side filtering (no API change needed)
- Keyboard shortcut: Ctrl+K or `/` to focus search

**Estimated effort:** Small

---

## Phase 24: Webhook Notifications

**Goal:** Push tunnel events to external services (Discord, Slack, etc.).

**Scope:**
- Config: webhook URL + event types to send
- Payload: JSON with event type, tunnel name, state, timestamp
- Retry with backoff on failure
- UI: Settings > Notifications, add webhook URL + event selector

**Estimated effort:** Medium

---

## Phase 25: Prometheus Metrics

**Goal:** Expose `/metrics` for integration with existing monitoring stacks.

**Scope:**
- Tunnel state gauges (running/stopped/error per tunnel)
- Traffic counters (bytes in/out, connections total)
- SSH connection health (connected/disconnected)
- Latency histograms (if Phase 18 implemented)
- No auth on `/metrics` (or optional separate token)

**Estimated effort:** Small (if traffic stats from Phase 14 exist)

---

## Phase 26: Uptime History + Audit Log

**Goal:** Track tunnel availability over time and user actions.

**Scope:**
- Uptime: record state transitions with timestamps in a ring buffer or SQLite
- Audit: log all API mutations (create/update/delete/start/stop) with timestamp
- UI: timeline view in tunnel detail, audit log page in settings
- Retention: configurable, default 7 days

**Estimated effort:** Medium

---

## Phase 27: Web UI Authentication [DONE]

**Goal:** Proper login page instead of just bearer token.

**Scope:**
- Username/password login with session cookie
- Admin credentials via `TUNNEL_ADMIN_PASSWORD` env var (no setup wizard)
- Bearer token auth kept as API-only alternative
- Session expiry (24h default, 30d with "Remember me") and logout

**Estimated effort:** Medium

---

## Phase 28: API IP Whitelist

**Goal:** Restrict API access by source IP.

**Scope:**
- Config: `general.allowedIPs: ["127.0.0.1/8", "192.168.0.0/16"]`
- Middleware: check `X-Forwarded-For` / `RemoteAddr` against whitelist
- Empty list = allow all (backward compatible)

**Estimated effort:** Small

---

## Phase 29: CLI Companion Tool

**Goal:** Command-line interface for scripting and headless management.

**Scope:**
- `opstunnel status` — list all tunnels and states
- `opstunnel start/stop/restart <name|id>` — control tunnels
- `opstunnel ssh list/add/remove` — manage SSH connections
- Connects to server API via `--addr` and `--token` flags
- Single static binary, no dependencies

**Estimated effort:** Medium

---

## Phase 30: Browser Extension

**Goal:** One-click proxy switching tied to OpsTunnel tunnels.

**Scope:**
- Chrome/Firefox extension
- Query OpsTunnel API for running dynamic tunnels
- Switch browser proxy to selected tunnel's SOCKS5/HTTP address
- Badge shows connected tunnel name

**Estimated effort:** Large (separate project)

---

## Phase 31: Multi-Instance Management

**Goal:** Central dashboard managing OpsTunnel instances on multiple machines.

**Scope:**
- Agent mode: OpsTunnel instances register with a central server
- Central UI: view/control all registered instances
- Requires stable API + authentication between instances

**Estimated effort:** Large (architectural change)

---

## Phase 32: Config Sync

**Goal:** Synchronize tunnel configs across devices.

**Scope:**
- Options: Git-based sync, cloud storage (S3/WebDAV), or built-in P2P
- Conflict resolution for concurrent edits
- Selective sync (choose which tunnels to sync)

**Estimated effort:** Large

---

## Phase 33: WoL (Wake-on-LAN)

**Goal:** Wake remote host before SSH connection.

**Scope:**
- Per SSH connection config: `wol.macAddress`, `wol.broadcastAddr`
- Before dial: send magic packet, wait configurable delay
- UI: WoL fields in SSH connection advanced settings

**Estimated effort:** Small

---

## Phase 34: SSH Host Monitoring (Opt-in)

**Goal:** Leverage active SSH connections to collect basic remote host metrics.

**Scope:**
- Opt-in per SSH connection: `monitoring.enabled: bool`
- Execute lightweight commands over existing SSH session (no extra connections)
- Metrics: load average, memory usage, disk usage, system uptime
- Display in tunnel detail view under SSH chain section
- Sampling interval: 30s (configurable)

**Commands (Linux):**
```
cat /proc/loadavg          → load 1/5/15
cat /proc/meminfo          → MemTotal, MemAvailable
df -P /                    → disk usage
cat /proc/uptime           → system uptime
```

**Design considerations:**
- Default OFF — user must explicitly enable per connection
- Graceful fallback: if exec session fails (restricted SSH), disable silently
- Linux-first (covers most use cases), macOS/BSD support later via `sysctl`/`vm_stat`
- Lightweight: single exec session per sample, parse output in Go
- Cache last sample in memory, no persistence needed
- Display as small info cards or tooltip in tunnel detail's SSH chain area

**Estimated effort:** Medium

---

## Phase 35: WiFi-Based Tunnel Auto-Switch (Desktop)

**Goal:** Automatically start/stop tunnels based on current WiFi network.

**Config model:** Per-tunnel SSID rules with two modes:
```json
{
  "policy": {
    "wifiRules": {
      "mode": "include",
      "ssids": ["CorpWiFi", "CorpWiFi-5G"]
    }
  }
}
```

**Modes:**
- `include` — only start tunnel when connected to listed SSIDs (e.g., office-only tunnels)
- `exclude` — start tunnel on any WiFi EXCEPT listed SSIDs (e.g., always-on proxy but not at office)
- `null` / absent — no WiFi rule, tunnel follows manual control / `enabled` state

**Wildcard:** `"*"` in ssids list matches any WiFi (useful with `exclude` mode)

**Detection (platform-specific):**
- macOS: `CoreWLAN` framework via CGO, or shell out to `networksetup -getairportnetwork en0`
- Windows: `netsh wlan show interfaces` or Win32 WLAN API
- Linux: `iwgetid -r` or NetworkManager D-Bus

**Architecture:**
```
WiFiMonitor (goroutine, polls every 5s)
  ├── detects SSID change
  ├── evaluates each tunnel's wifiRules
  ├── starts tunnels that match current SSID
  └── stops tunnels that no longer match
```

**Design considerations:**
- Desktop-only feature, hidden in Server/Docker mode
- WiFiMonitor runs as a background goroutine in the desktop app
- On WiFi disconnect (no SSID): stop all WiFi-rule tunnels, or keep running? Configurable
- On startup: evaluate rules immediately against current SSID
- Manual start/stop should override WiFi rules until next SSID change
- UI: tunnel form > Policy section, WiFi rule selector with SSID input
- Tray menu: show current WiFi name and active rules

**Edge cases:**
- Ethernet (no WiFi): treat as "no SSID" — only `exclude` mode tunnels with `*` would run
- Multiple network interfaces: use the primary WiFi interface
- Rapid WiFi switching: debounce (ignore changes within 2s)
- VPN connection: WiFi SSID doesn't change, rules unaffected

**Estimated effort:** Medium

---

## Priority Matrix

### Tier 1 — High Value, Low Effort
| Phase | Feature | Effort |
|-------|---------|--------|
| 13 | Config Import/Export | Small |
| 18 | Latency Monitoring | Small |
| 19 | SSH Key Generation | Small |
| 21 | Tunnel Clone/Templates | Small |
| 23 | Search & Filter | Small |
| 28 | API IP Whitelist | Small |
| 33 | Wake-on-LAN | Small |

### Tier 2 — High Value, Medium Effort
| Phase | Feature | Effort |
|-------|---------|--------|
| 14 | Traffic Statistics + Dashboard | Medium |
| 15 | System Proxy + PAC | Medium |
| 16 | Desktop Auto-Start | Small |
| 17 | SSH Connection Pooling | Medium |
| 22 | Tunnel Groups + Bulk Ops | Medium |
| 25 | Prometheus Metrics | Small-Medium |
| 34 | SSH Host Monitoring | Medium |
| 35 | WiFi-Based Tunnel Auto-Switch | Medium |

### Tier 3 — Medium Value, Medium Effort
| Phase | Feature | Effort |
|-------|---------|--------|
| 20 | Config Encryption | Medium |
| 24 | Webhook Notifications | Medium |
| 26 | Uptime History + Audit Log | Medium |
| 27 | Web UI Authentication | Medium |
| 29 | CLI Companion Tool | Medium |

### Tier 4 — Large Scope, Future Vision
| Phase | Feature | Effort |
|-------|---------|--------|
| 30 | Browser Extension | Large |
| 31 | Multi-Instance Management | Large |
| 32 | Config Sync | Large |

---

## Notes

- All phases are independent and can be implemented in any order
- Each phase should be a separate branch/PR for clean review
- Config schema changes should be backward-compatible (new optional fields)
- Server/Docker features should work without Desktop-specific code paths
- Phases 25 (Prometheus) depends on Phase 14 (Traffic Stats) for full value
- Phases 30-32 are long-term vision, may warrant separate sub-projects
