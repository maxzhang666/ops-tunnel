下面是一份可以直接拿去做“AI 编码”的需求草案（偏工程落地，含数据模型/API/核心流程/验收标准）。默认技术路线：**Go core（headless service）+ Web UI（SPA）+ 桌面壳（Wails 仅负责启动/托盘/打开 WebView）+ Docker 直接跑 core 并暴露端口**。

---

# 0. 项目目标与范围

## 目标

实现一个跨平台 SSH Tunnel 管理器，支持：

- **三类转发**：本地端口转发（Local `-L`）、远程端口转发（Remote `-R`）、动态转发（Dynamic SOCKS5 `-D`）
- **多跳板机链路**：每条隧道可配置 0..N 个跳板机（Jump Hosts），并且**每一跳可单独配置认证**
- **深层嵌套网络**：允许 jump1 -> jump2 -> ... -> target 的链式连接；提供断线重连、KeepAlive、可观测日志
- **交付形态**：
  - Desktop：同一套前端 UI + 本机 core
  - Docker：同一套前端 UI + 容器内 core（隧道端口可映射给其他机器使用）

## 非目标（第一期）

- 不做复杂的凭据保险箱（先做“可用但不完美”的密钥/口令存储）
- 不强制兼容系统 OpenSSH config 全量语法（可以后续做导入）
- 不实现 SSH agent 转发、Kerberos、PKCS11 等高级认证（留扩展点）

---

# 1. 运行形态与架构

## 1.1 Core 服务（必须）

- 一个可独立运行的二进制：`tunnel-core`
- 提供：
  - HTTP REST API（管理配置、启动停止、查询状态）
  - WebSocket（推送实时日志/事件/状态变更）
  - 静态文件托管（提供前端构建产物 `/ui`，可配置关闭）
- 可在：
  - 本机运行（Desktop）
  - Docker 容器运行（Headless + Web UI）

## 1.2 Desktop 壳（Wails）

- 职责：
  - 启动/停止本机 `tunnel-core`（同进程或子进程皆可；建议子进程便于和 Docker 形态一致）
  - 提供托盘菜单（Start/Stop、打开 UI、退出）
  - 打开 WebView 指向 `http://127.0.0.1:<port>/`
- Desktop 不直接承载隧道逻辑（隧道逻辑在 core）

## 1.3 Docker

- 镜像启动后运行 `tunnel-core --listen 0.0.0.0:8080 --ui`
- 用户通过浏览器访问 `http://<docker-host>:8080/`
- 隧道端口对外可用：
  - Local/动态转发监听地址可配置 `0.0.0.0`，并通过 `-p` 映射到宿主或直接 host network
  - Remote 转发由远端监听，需清晰展示其绑定地址和可达性

---

# 2. 数据模型（建议 JSON/YAML）

核心对象：**Tunnel**。每条 Tunnel 包含：连接链路（Hops）、目标（Target）、转发规则（Forwards）、运行策略（Policy）。

## 2.1 枚举与结构

### 2.1.1 Auth（认证）

支持类型：

- `password`
- `privateKey`（可选 passphrase）
- `keyboardInteractive`（第一期可当作 password 处理或提示“不支持”）
- `none`（仅用于内部测试）

建议结构：

```json
{
  "type": "privateKey",
  "username": "ubuntu",
  "password": null,
  "privateKey": {
    "format": "pem",
    "keyPem": "-----BEGIN ...",
    "passphrase": null
  },
  "agent": {
    "enabled": false
  }
}
```

> 第一版：keyPem 可直接存配置里（不安全但可用）；后续再加密或外接 keychain。

### 2.1.2 Endpoint

```json
{ "host": "10.0.0.5", "port": 22 }
```

### 2.1.3 Hop

每一跳一个 Hop：

```json
{
  "id": "hop-1",
  "name": "jump-1",
  "endpoint": { "host": "1.2.3.4", "port": 22 },
  "auth": { "...": "..." },
  "hostKeyVerification": {
    "mode": "acceptNew|strict|insecure",
    "knownHosts": null
  },
  "dialTimeoutMs": 10000,
  "keepAlive": {
    "intervalMs": 15000,
    "maxMissed": 3
  }
}
```

### 2.1.4 Target

Target 本质上也是 Hop 的同构结构（最后一跳）：

```json
{
  "name": "prod-bastion-inner",
  "endpoint": { "host": "10.10.0.10", "port": 22 },
  "auth": { "...": "..." },
  "hostKeyVerification": { "mode": "acceptNew" }
}
```

### 2.1.5 Forward（转发规则）

三类：

#### Local (-L)

```json
{
  "id": "fwd-1",
  "type": "local",
  "listen": { "host": "0.0.0.0", "port": 15432 },
  "connect": { "host": "127.0.0.1", "port": 5432 },
  "notes": "Expose remote postgres to outside via docker port mapping"
}
```

#### Remote (-R)

```json
{
  "id": "fwd-2",
  "type": "remote",
  "listen": { "host": "0.0.0.0", "port": 18080 },
  "connect": { "host": "127.0.0.1", "port": 8080 }
}
```

#### Dynamic SOCKS5 (-D)

```json
{
  "id": "fwd-3",
  "type": "dynamic",
  "listen": { "host": "0.0.0.0", "port": 1080 },
  "socks5": {
    "auth": "none|userpass",
    "username": null,
    "password": null,
    "allowCIDRs": ["10.0.0.0/8", "192.168.0.0/16"],
    "denyCIDRs": ["0.0.0.0/0"]
  }
}
```

> 说明：动态转发建议实现“本地 SOCKS5 server + 通过 SSH dial 出口”的方式；并提供 ACL，避免在 Docker 场景不小心开放代理造成安全风险。

### 2.1.6 Policy（运行策略）

```json
{
  "autoStart": false,
  "autoRestart": true,
  "restartBackoffMs": { "min": 500, "max": 15000, "factor": 1.7 },
  "maxRestartsPerHour": 60,
  "gracefulStopTimeoutMs": 5000
}
```

## 2.2 Tunnel 完整示例

```json
{
  "id": "tun-prod-1",
  "name": "prod-deep-net",
  "enabled": true,
  "hops": [
    {
      "id": "hop-1",
      "name": "jump-public",
      "endpoint": { "host": "1.2.3.4", "port": 22 },
      "auth": {
        "type": "privateKey",
        "username": "ec2-user",
        "privateKey": {
          "format": "pem",
          "keyPem": "-----BEGIN...",
          "passphrase": null
        }
      },
      "hostKeyVerification": { "mode": "acceptNew" },
      "dialTimeoutMs": 10000,
      "keepAlive": { "intervalMs": 15000, "maxMissed": 3 }
    },
    {
      "id": "hop-2",
      "name": "jump-inner",
      "endpoint": { "host": "10.0.0.10", "port": 22 },
      "auth": { "type": "password", "username": "ubuntu", "password": "xxx" },
      "hostKeyVerification": { "mode": "acceptNew" },
      "dialTimeoutMs": 10000,
      "keepAlive": { "intervalMs": 15000, "maxMissed": 3 }
    }
  ],
  "target": {
    "name": "target-app",
    "endpoint": { "host": "10.0.1.20", "port": 22 },
    "auth": {
      "type": "privateKey",
      "username": "svc",
      "privateKey": {
        "format": "pem",
        "keyPem": "-----BEGIN...",
        "passphrase": "pass"
      }
    },
    "hostKeyVerification": { "mode": "strict" }
  },
  "forwards": [
    {
      "id": "fwd-1",
      "type": "local",
      "listen": { "host": "0.0.0.0", "port": 15432 },
      "connect": { "host": "127.0.0.1", "port": 5432 }
    },
    {
      "id": "fwd-3",
      "type": "dynamic",
      "listen": { "host": "0.0.0.0", "port": 1080 },
      "socks5": {
        "auth": "none",
        "allowCIDRs": ["10.0.0.0/8"],
        "denyCIDRs": ["0.0.0.0/0"]
      }
    }
  ],
  "policy": {
    "autoStart": false,
    "autoRestart": true,
    "restartBackoffMs": { "min": 500, "max": 15000, "factor": 1.7 },
    "maxRestartsPerHour": 60,
    "gracefulStopTimeoutMs": 5000
  }
}
```

---

# 3. Core API 需求（REST + WebSocket）

Base URL：`/api/v1`

## 3.1 配置管理

- `GET /tunnels` -> 返回隧道列表（不含敏感字段或用 `redacted=true`）
- `POST /tunnels` -> 创建隧道
- `GET /tunnels/{id}` -> 获取隧道详情
- `PUT /tunnels/{id}` -> 更新隧道（全量）
- `PATCH /tunnels/{id}` -> 局部更新（可选）
- `DELETE /tunnels/{id}`

配置持久化：

- 默认存储：单机文件 `data/config.json`
- 要求：写入原子性（先写 temp 再 rename）

## 3.2 运行控制

- `POST /tunnels/{id}/start`
- `POST /tunnels/{id}/stop`
- `POST /tunnels/{id}/restart`
- `GET /tunnels/{id}/status`

### Status 建议结构

```json
{
  "id": "tun-prod-1",
  "state": "stopped|starting|running|degraded|error|stopping",
  "since": "2026-03-24T12:00:00Z",
  "activeForwards": [
    {
      "id": "fwd-1",
      "state": "listening|error",
      "listen": "0.0.0.0:15432",
      "detail": null
    }
  ],
  "chain": [
    {
      "hopId": "hop-1",
      "state": "connected|connecting|error",
      "latencyMs": 120,
      "detail": null
    },
    { "hopId": "hop-2", "state": "connected", "latencyMs": 30, "detail": null },
    { "hopId": "target", "state": "connected", "latencyMs": 15, "detail": null }
  ],
  "lastError": null
}
```

## 3.3 事件与日志（WebSocket）

- `GET /ws`（或 `/ws/events`）
- 事件类型：
  - `tunnel.stateChanged`
  - `tunnel.log`
  - `tunnel.forwardListening`
  - `tunnel.forwardError`
  - `tunnel.chainConnected/chainError`
  - `core.health`

日志事件示例：

```json
{
  "type": "tunnel.log",
  "tunnelId": "tun-prod-1",
  "level": "info|warn|error|debug",
  "ts": "2026-03-24T12:00:01Z",
  "message": "hop-2 connected",
  "fields": { "host": "10.0.0.10", "port": 22 }
}
```

---

# 4. 隧道引擎（核心行为规范）

## 4.1 链式连接（多跳）

- 建立顺序：hop1 -> hop2 -> ... -> target
- 每一跳：
  - 使用其独立 `auth`（用户名/密码/私钥等）
  - 启用 keepalive（发送 ignore/global request 或自定义心跳）
  - 支持 dial timeout
- 连接下一跳的方式：
  - 通过上一跳的 SSH client 建立 `net.Conn`（在 Go 中通常是 `client.Dial("tcp", "next:22")`）再 `ssh.NewClientConn` 形成下一跳 client
- 要求：若某一跳断开，整体隧道进入 `degraded/error`，按 policy 自动重连

## 4.2 转发实现

### Local

- 在本地（core 运行环境：桌面或容器）监听 `listen.host:listen.port`
- 每个 incoming conn：
  - 通过 target 的 SSH client `Dial("tcp", connect.host:connect.port)` 建立到远端目标的连接
  - 双向 copy（带超时/关闭处理）

### Remote

- 在 target 侧请求 remote port forward（等价 `ssh -R`）
- 要求能够配置 `listen.host`（注意 OpenSSH 里对应 GatewayPorts；库实现里取决于服务端支持）
- 远端有连接进来时，回连到本地 `connect.host:connect.port`（这里“本地”指 core 所在网络）

### Dynamic (SOCKS5)

- 在本地监听 socks5
- SOCKS5 CONNECT 的出站连接通过 target ssh client dial 到目的地址
- ACL：
  - allowCIDRs/denyCIDRs 生效顺序：先 deny 再 allow（或明确写入规范）
  - 默认拒绝 `0.0.0.0/0`，避免误开放

## 4.3 端口占用检查

- start 前检查 local/dynamic 的 listen 端口是否可绑定
- 失败要出明确错误并进入 `error`

## 4.4 自动重连与退避

- `autoRestart=true` 时，断线/错误触发重连
- 退避：指数退避 + 上限
- 避免疯狂重连：`maxRestartsPerHour`

---

# 5. UI（前端）需求草案

## 5.1 页面结构

- 左侧：隧道列表（状态小圆点 + 名称 + 运行时长）
- 右侧：当前隧道详情（Tabs）
  1. Overview：链路图（hop1 -> hop2 -> target）、延迟、错误
  2. Forwards：转发规则列表 + 启停状态 + 一键复制地址（如 `socks5://host:1080`）
  3. Config：表单编辑 hops/target/auth/forwards/policy
  4. Logs：实时滚动日志（可过滤 tunnelId/level）

## 5.2 关键交互

- 创建隧道向导（可选）：一步步添加 hop、选择认证方式、添加 forward
- Start/Stop/Restart 按钮
- Docker 场景提示：
  - 如果 listen host 是 `127.0.0.1`，提示“外部机器不可访问；如需对外暴露请设为 0.0.0.0 并映射端口”
- 安全提示：
  - 若 dynamic socks5 监听 `0.0.0.0` 且无鉴权且 allowCIDRs 过宽，提示高风险

---

# 6. 安全与最小合规（第一期最低限）

- API 可选支持 basic token（环境变量 `CORE_TOKEN`），未设置则不鉴权（本地模式）
- 日志禁止打印私钥/密码原文（必须 redact）
- host key 验证模式：
  - `insecure`：跳过（仅测试）
  - `acceptNew`：首次记录指纹，之后严格
  - `strict`：必须匹配已知指纹
- Docker 部署文档明确说明：
  - 强烈建议设置 token
  - SOCKS5 对外监听风险

---

# 7. 验收标准（可直接转测试用例）

1. **多跳认证**：配置 hop1 私钥 + hop2 密码 + target 私钥，能够成功连通并保持运行
2. **Local 转发**：在 Docker 中启动 local forward 监听 `0.0.0.0:15432`，通过 `-p 15432:15432` 映射后，另一台机器能连上并访问到远端服务
3. **Dynamic 转发**：在 Docker 中启动 SOCKS5 监听 `0.0.0.0:1080`，设置 allowCIDRs，外部机器通过该 socks5 访问允许网段地址成功，访问不允许网段被拒绝
4. **Remote 转发**：远端 `listen.host:port` 成功监听；从远端网络发起连接能回到 core 所在网络服务
5. **断线重连**：手动断开 hop2 网络，隧道进入 degraded/error，随后按退避自动恢复到 running
6. **可观测性**：UI Logs 能看到每条 hop 的连接/断开、每条 forward 的 listening/error，且不包含敏感字段

---

# 8. 工程拆分建议（方便你让 AI 分模块生成）

- `core/`
  - `config/`：模型定义 + 校验 + 存储（json）
  - `engine/`
    - `chain/`：建立多跳 SSH clients
    - `forward/`：local/remote/dynamic 三类实现
    - `supervisor/`：状态机、重连、退避
  - `api/`：REST + WS
  - `uiembed/`：嵌入静态资源（可选，docker 用文件也行）
- `ui/`：SPA
- `desktop/`：Wails 壳（启动 core、托盘、打开 URL）

---

下面给你两份“可直接拿去让 AI 编码”的材料：

1. **Go core 的 package 接口草图（建议的目录、关键类型、函数签名）**
2. **分阶段 AI 编码任务清单（按可提交的迭代/PR 粒度）**

我会尽量把边界、输入输出、状态机、事件流都写清楚，避免 AI 发散。

---

# A) Go core：Package 接口草图（可直接当架构契约）

## A1. 目录结构（建议）

```
core/
  cmd/tunnel-core/main.go
  internal/
    config/
      model.go
      validate.go
      store.go
      redact.go
    engine/
      engine.go
      supervisor.go
      state.go
      events.go
      chain/
        chain.go
        dial.go
        hostkey.go
        auth.go
      forward/
        forward.go
        local.go
        remote.go
        dynamic.go
        socks5.go
        acl.go
    api/
      server.go
      routes.go
      ws.go
      dto.go
    uiembed/
      uiembed.go (optional: //go:embed)
    util/
      backoff.go
      context.go
      net.go
      log.go
  data/ (runtime, not committed)
```

> 约定：`internal/` 内部包不被外部引用。后续做 Wails 壳时只需要启动 `tunnel-core` 二进制即可。

---

## A2. config/model.go（数据模型）

```go
package config

type TunnelState string
const (
  TunnelStopped  TunnelState = "stopped"
  TunnelStarting TunnelState = "starting"
  TunnelRunning  TunnelState = "running"
  TunnelDegraded TunnelState = "degraded"
  TunnelError    TunnelState = "error"
  TunnelStopping TunnelState = "stopping"
)

type HostKeyVerifyMode string
const (
  HostKeyInsecure  HostKeyVerifyMode = "insecure"
  HostKeyAcceptNew HostKeyVerifyMode = "acceptNew"
  HostKeyStrict    HostKeyVerifyMode = "strict"
)

type AuthType string
const (
  AuthPassword AuthType = "password"
  AuthPrivateKey AuthType = "privateKey"
  AuthNone AuthType = "none"
)

type Endpoint struct {
  Host string `json:"host"`
  Port int    `json:"port"`
}

type PrivateKey struct {
  Format     string `json:"format"`      // "pem"
  KeyPEM     string `json:"keyPem"`      // raw pem (phase1)
  Passphrase string `json:"passphrase"`  // optional
}

type AgentCfg struct {
  Enabled bool `json:"enabled"`
}

type Auth struct {
  Type     AuthType    `json:"type"`
  Username string      `json:"username"`
  Password string      `json:"password,omitempty"`
  PrivateKey *PrivateKey `json:"privateKey,omitempty"`
  Agent    *AgentCfg   `json:"agent,omitempty"`
}

type HostKeyVerification struct {
  Mode HostKeyVerifyMode `json:"mode"`
  // Later: KnownHostsPath / InlineKnownHosts
}

type KeepAlive struct {
  IntervalMs int `json:"intervalMs"`
  MaxMissed  int `json:"maxMissed"`
}

type Hop struct {
  ID   string `json:"id"`
  Name string `json:"name"`
  Endpoint Endpoint `json:"endpoint"`
  Auth Auth `json:"auth"`

  HostKeyVerification HostKeyVerification `json:"hostKeyVerification"`

  DialTimeoutMs int `json:"dialTimeoutMs"`
  KeepAlive KeepAlive `json:"keepAlive"`
}

type Target = Hop // same structure, but semantically last hop

type ForwardType string
const (
  ForwardLocal   ForwardType = "local"
  ForwardRemote  ForwardType = "remote"
  ForwardDynamic ForwardType = "dynamic"
)

type Socks5Auth string
const (
  Socks5None    Socks5Auth = "none"
  Socks5UserPass Socks5Auth = "userpass"
)

type CIDRList struct {
  Allow []string `json:"allowCIDRs"`
  Deny  []string `json:"denyCIDRs"`
}

type Socks5Cfg struct {
  Auth Socks5Auth `json:"auth"`
  Username string `json:"username,omitempty"`
  Password string `json:"password,omitempty"`
  AllowCIDRs []string `json:"allowCIDRs,omitempty"`
  DenyCIDRs  []string `json:"denyCIDRs,omitempty"`
}

type Forward struct {
  ID   string `json:"id"`
  Type ForwardType `json:"type"`
  Listen  Endpoint `json:"listen"`  // host+port
  Connect Endpoint `json:"connect"` // for local/remote

  Socks5 *Socks5Cfg `json:"socks5,omitempty"` // for dynamic only
  Notes  string `json:"notes,omitempty"`
}

type RestartBackoff struct {
  MinMs  int     `json:"min"`
  MaxMs  int     `json:"max"`
  Factor float64 `json:"factor"`
}

type Policy struct {
  AutoStart bool `json:"autoStart"`
  AutoRestart bool `json:"autoRestart"`
  RestartBackoffMs RestartBackoff `json:"restartBackoffMs"`
  MaxRestartsPerHour int `json:"maxRestartsPerHour"`
  GracefulStopTimeoutMs int `json:"gracefulStopTimeoutMs"`
}

type Tunnel struct {
  ID string `json:"id"`
  Name string `json:"name"`
  Enabled bool `json:"enabled"`

  Hops []Hop `json:"hops"`
  Target Target `json:"target"`

  Forwards []Forward `json:"forwards"`
  Policy Policy `json:"policy"`
}

type Config struct {
  Version int `json:"version"`
  Tunnels []Tunnel `json:"tunnels"`
}
```

---

## A3. config/validate.go（校验规则）

```go
package config

type ValidationError struct {
  Field string `json:"field"`
  Msg   string `json:"msg"`
}
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string

func ValidateTunnel(t Tunnel) error
func ValidateConfig(c Config) error
```

强制校验点（第一期就要做）：

- ID 非空且唯一
- Endpoint port 1..65535
- Forward:
  - local/dynamic：listen.port 必须；listen.host 默认 `127.0.0.1`（如果空）
  - remote：listen.port 必须；connect 必须
  - dynamic：Socks5 必须存在；若 listen.host 是 `0.0.0.0` 且 auth=none 且 allowCIDRs 为空或包含 `0.0.0.0/0` -> 返回 warning（可用事件/字段传出）
- Hop/Target：auth.type 决定所需字段（password/privateKey）
- policy：backoff min/max/factor 合法

---

## A4. config/store.go（文件存储）

```go
package config

type Store interface {
  Load(ctx context.Context) (Config, error)
  Save(ctx context.Context, cfg Config) error
}

type FileStore struct {
  Path string
}

func NewFileStore(path string) *FileStore
func (s *FileStore) Load(ctx context.Context) (Config, error)
func (s *FileStore) Save(ctx context.Context, cfg Config) error
```

要求：保存原子写（temp + fsync + rename）。

---

## A5. engine/events.go（事件总线：WS/日志都从这里来）

```go
package engine

type EventType string
const (
  EventTunnelStateChanged EventType = "tunnel.stateChanged"
  EventTunnelLog          EventType = "tunnel.log"
  EventForwardListening   EventType = "tunnel.forwardListening"
  EventForwardError       EventType = "tunnel.forwardError"
  EventChainConnected     EventType = "tunnel.chainConnected"
  EventChainError         EventType = "tunnel.chainError"
  EventCoreHealth         EventType = "core.health"
)

type Event struct {
  Type EventType              `json:"type"`
  TunnelID string             `json:"tunnelId,omitempty"`
  Level string                `json:"level,omitempty"` // debug/info/warn/error
  TS time.Time                `json:"ts"`
  Message string              `json:"message"`
  Fields map[string]any       `json:"fields,omitempty"`
}

type EventSink interface {
  Publish(e Event)
}

type EventBus interface {
  EventSink
  Subscribe() (ch <-chan Event, cancel func())
}

func NewEventBus(buffer int) EventBus
```

> 实现建议：每个 subscriber 一个 channel；Publish fan-out；慢订阅丢弃或断开（需要策略）。

---

## A6. engine/state.go（运行时状态）

```go
package engine

import "core/internal/config"

type HopRuntimeStatus struct {
  HopID string `json:"hopId"`
  State string `json:"state"` // connected/connecting/error
  LatencyMs int `json:"latencyMs,omitempty"`
  Detail string `json:"detail,omitempty"`
}

type ForwardRuntimeStatus struct {
  ID string `json:"id"`
  State string `json:"state"` // listening/error/stopped
  Listen string `json:"listen"`
  Detail string `json:"detail,omitempty"`
}

type TunnelStatus struct {
  ID string `json:"id"`
  State config.TunnelState `json:"state"`
  Since time.Time `json:"since"`

  Chain []HopRuntimeStatus `json:"chain"`
  ActiveForwards []ForwardRuntimeStatus `json:"activeForwards"`

  LastError string `json:"lastError,omitempty"`
}
```

---

## A7. engine/engine.go（外部可调用的引擎接口）

```go
package engine

import "core/internal/config"

type Engine interface {
  UpsertTunnel(ctx context.Context, t config.Tunnel) error
  DeleteTunnel(ctx context.Context, id string) error

  StartTunnel(ctx context.Context, id string) error
  StopTunnel(ctx context.Context, id string) error
  RestartTunnel(ctx context.Context, id string) error

  GetTunnel(ctx context.Context, id string) (config.Tunnel, bool)
  ListTunnels(ctx context.Context) []config.Tunnel

  GetStatus(ctx context.Context, id string) (TunnelStatus, bool)
  ListStatus(ctx context.Context) []TunnelStatus

  Events() EventBus

  Shutdown(ctx context.Context) error
}

type EngineOptions struct {
  // for known_hosts / hostkey store in future
}

func NewEngine(bus EventBus, opts EngineOptions) Engine
```

---

## A8. engine/supervisor.go（每条隧道一个 supervisor）

```go
package engine

import "core/internal/config"

type TunnelSupervisor interface {
  ID() string
  Start(ctx context.Context) error
  Stop(ctx context.Context) error
  Status() TunnelStatus
  UpdateConfig(t config.Tunnel) error // hot update policy/forwards? first phase: stop+apply+start optional
}

func newTunnelSupervisor(t config.Tunnel, bus EventSink) TunnelSupervisor
```

行为规范：

- Start：进入 starting -> 建链 -> 启动 forwards -> running
- Stop：停止 forwards -> 断开链路 -> stopped
- 自动重连：由 supervisor 内部 goroutine 管理，遵循 policy

---

## A9. engine/chain/chain.go（多跳链路）

```go
package chain

import (
  "core/internal/config"
  "golang.org/x/crypto/ssh"
)

type Chain struct {
  // holds clients per hop
}

type BuildResult struct {
  Clients []*ssh.Client // hop clients + last is target client
}

func Build(ctx context.Context, hops []config.Hop, target config.Target, bus EventSink) (*BuildResult, error)
func Close(result *BuildResult) error
func TargetClient(result *BuildResult) *ssh.Client
```

### auth.go（将 config.Auth -> ssh.AuthMethod）

```go
package chain

import "core/internal/config"

func AuthMethods(a config.Auth) ([]ssh.AuthMethod, error)
```

### hostkey.go（host key 策略）

```go
package chain

import "core/internal/config"

func HostKeyCallback(mode config.HostKeyVerifyMode, store HostKeyStore, host string) (ssh.HostKeyCallback, error)

type HostKeyStore interface {
  Get(hostport string) (fingerprint string, ok bool)
  Put(hostport string, fingerprint string) error
}
```

> 第一版 hostkey store 可以用本地 json 文件；acceptNew 逻辑放这里。

---

## A10. engine/forward/forward.go（转发抽象）

```go
package forward

import "golang.org/x/crypto/ssh"
import "core/internal/config"

type Runtime interface {
  ID() string
  Start(ctx context.Context, target *ssh.Client) error
  Stop(ctx context.Context) error
  Status() any // or a typed status
}

func NewRuntime(f config.Forward, bus EventSink) (Runtime, error)
```

### local.go

- 监听 TCP
- 每连接：target.Dial -> io.Copy 双向

### remote.go

- 使用 `target.Listen("tcp", addr)` 或 `(*ssh.Client).Listen`（具体 API 以 x/crypto/ssh 为准；某些情况下需要 `tcpip-forward` request）
- 接受 remote conn 后，本地 dial 到 connect，再 copy

### dynamic.go + socks5.go

- 本地 socks5 server（建议自己实现最小 SOCKS5 CONNECT；不引入太重依赖）
- ACL 检查：目的地址是否允许
- 出站：target.Dial("tcp", dst)

---

## A11. api/server.go（HTTP + WS）

```go
package api

import "core/internal/engine"

type ServerOptions struct {
  ListenAddr string // "127.0.0.1:8080" or "0.0.0.0:8080"
  UIRoot string     // path to static files, optional
  Token string      // optional bearer token
}

type Server struct {
  Eng engine.Engine
  Opt ServerOptions
}

func NewServer(eng engine.Engine, opt ServerOptions) *Server
func (s *Server) Run(ctx context.Context) error
func (s *Server) Shutdown(ctx context.Context) error
```

### routes.go（必须实现的路由）

- `GET /api/v1/tunnels`
- `POST /api/v1/tunnels`
- `GET /api/v1/tunnels/{id}`
- `PUT /api/v1/tunnels/{id}`
- `DELETE /api/v1/tunnels/{id}`
- `POST /api/v1/tunnels/{id}/start|stop|restart`
- `GET /api/v1/tunnels/{id}/status`
- `GET /ws`（升级 websocket，转发 eventbus）

### dto.go

- 处理 redaction：列表接口返回删掉 password/keyPem/passphrase（或用 `"***"`）

---

# B) 分阶段 AI 编码任务清单（按迭代/PR 粒度）

下面每一步都定义：**目标 / 产物 / 关键点 / 验收**。你可以把每个步骤原封不动给 AI，让它按步骤生成代码与提交说明。

---

## PR-0：初始化仓库与可运行骨架

**目标**

- `tunnel-core` 能启动 HTTP server，返回 health，提供空的 tunnels API（mock）

**产物**

- `cmd/tunnel-core/main.go` 支持 flags：
  - `--listen` 默认 `127.0.0.1:8080`
  - `--data-dir` 默认 `./data`
  - `--ui-dir` 可选
  - `--token` 可选
- `GET /healthz` -> `200 ok`

**验收**

- `go run ./cmd/tunnel-core --listen 127.0.0.1:8080` 可访问 `/healthz`

---

## PR-1：Config 模型 + 文件存储 + 校验 + redaction

**目标**

- 实现 `config` 包，能 load/save JSON，能校验 tunnel

**关键点**

- 原子写入
- 校验错误结构化输出
- redaction：返回给 API 的 Tunnel 不包含敏感信息

**验收**

- 启动时若 `data/config.json` 不存在则创建空 config
- 写入后重启仍能读取

---

## PR-2：Engine 内存管理 + 事件总线 + 状态 API（不建链）

**目标**

- Engine 能 CRUD tunnels；start/stop 只改变状态（stub）
- EventBus 可订阅，WS 能收到事件

**关键点**

- 状态机字段齐全
- WS 广播 `tunnel.stateChanged`

**验收**

- 调用 `POST /start` 后，`GET status` 显示 running（假）
- WS 能实时收到事件

---

## PR-3：实现 SSH 链式连接（multi-hop）但不做转发

**目标**

- `engine/chain` 能建立 hop1 -> hop2 -> target 的 _ssh.Client 链_
- 每 hop 独立认证（password/privateKey）
- host key 模式：insecure + acceptNew（先落地）

**关键点**

- 使用 `client.Dial("tcp", next)` 再 `ssh.NewClientConn` 形成下一 hop
- keepalive goroutine（每 hop）
- 超时控制

**验收**

- 通过集成测试/手动：配置 2 hops + target 成功连接并维持 1 分钟
- 错误能在 events/日志中看到是哪个 hop 失败

---

## PR-4：Local forward (-L) 全链路打通

**目标**

- 在 target client 上执行 `Dial`，在本地监听端口，完成 TCP 代理

**关键点**

- 端口占用检查
- 每连接的双向 copy，正确关闭
- 运行中统计 forward 状态（listening/error）

**验收**

- 在 Docker 或本机：访问 `listen` 端口能到达远端 `connect` 服务
- stop 后端口释放

---

## PR-5：Dynamic SOCKS5 (-D) + ACL

**目标**

- 本地 SOCKS5 server，CONNECT 请求通过 target ssh dial 出口
- allow/deny CIDR 生效
- 可选 user/pass 鉴权（第一期可以先只做 none，但强烈建议实现 userpass）

**关键点**

- SOCKS5 协议最小实现：握手、方法选择、CONNECT、IPv4/域名
- ACL：先 deny 后 allow；allow 为空表示默认拒绝（安全默认）

**验收**

- curl/wget 使用 socks5 能访问 allow 网段地址
- 访问 deny 或不在 allow 的地址被拒绝且有日志

---

## PR-6：Remote forward (-R)

**目标**

- 请求远端监听端口；远端连接进入后回连到本地 connect

**关键点**

- 兼容性：服务端是否允许 GatewayPorts/远端绑定 0.0.0.0
- 明确暴露错误：如果远端拒绝绑定或只允许 127.0.0.1，要在 status/log 提示

**验收**

- 在允许 remote forward 的 sshd 上：远端端口监听成功，从远端侧访问可打到 core 本地服务

---

## PR-7：Supervisor 自动重连 + 退避 + 限速

**目标**

- 断线后按 policy 自动重连
- 指数退避、每小时最大重启次数

**关键点**

- Supervisor 内部循环：build chain -> start forwards -> wait error/ctx done -> cleanup -> backoff -> retry
- stop 要能“打断重连循环”

**验收**

- 人为断网 10 秒恢复，隧道最终回到 running
- 重连次数超过阈值进入 error 并停止重试

---

## PR-8：API 完整化 + DTO + 过滤日志

**目标**

- API 返回结构稳定、字段 redacted
- 增加 `GET /tunnels/{id}`、`PUT`、`DELETE` 等
- WS 事件类型完善

**验收**

- 前端不做也能通过 curl 完整管理隧道
- 日志不泄露 password/keyPem

---

## PR-9：Docker 化

**目标**

- Dockerfile + 最小运行说明
- 支持 `-p` 暴露 forward 端口

**产物**

- `Dockerfile`
- `docker-compose.yml`（可选）
- README：如何配置 listen=0.0.0.0、如何映射端口

**验收**

- `docker run -p 8080:8080 -p 1080:1080 ...` 可访问 UI/API，SOCKS5 对外可用（按配置）

---

## PR-10：Web UI（最小可用）

**目标**

- 列表/详情/编辑/日志页面
- start/stop/restart
- WS 实时日志

**验收**

- UI 能创建隧道并启动，看到 forward listening

---

## PR-11：Wails 桌面壳

**目标**

- 托盘 + 启动 core + 打开 UI
- 自动选择可用端口启动 core（例如 0 表示随机端口），然后打开 WebView

**验收**

- 双击启动后能看到 UI；退出能关闭 core 进程

---

# C) 你可以直接给 AI 的“总提示词模板”（建议）

你后续让 AI 写代码时，可以用这个模板保证它按草案实现、不跑题：

> 你是资深 Go 工程师。请严格按照以下架构契约实现：
>
> - 使用 Go 1.22+，模块名为 `tunnel`（或你指定）
> - 实现 packages：config/engine/chain/forward/api
> - Engine 提供 CRUD + start/stop + status + eventbus
> - 多跳 SSH 链路通过上一跳 `ssh.Client.Dial` 建立下一跳连接；每跳独立认证
> - 实现 local/dynamic/remote 三类转发
> - 不得在日志或 API 中输出 password/keyPem/passphrase
> - 提供最小可运行示例（main.go + healthz + tunnels API）
>   输出：代码文件列表 + 每个文件内容；并附上运行命令与简单 curl 验证步骤。

---
