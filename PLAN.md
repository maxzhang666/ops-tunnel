# OpsTunnel — 详细实施计划书

## 1. 产品概述

跨平台 SSH Tunnel 管理器，支持本地转发(-L)、远程转发(-R)、动态 SOCKS5 转发(-D)，支持多跳板机链路，提供可视化管理界面。

**交付形态（按优先级）：**
1. Desktop 应用（Wails，macOS/Windows/Linux）
2. Docker + Web UI（Headless 模式）

**定位：** 开源 + 商业化

---

## 2. 技术栈

| 层 | 选型 | 理由 |
|----|------|------|
| Core 语言 | Go 1.22+ | 单二进制、跨平台、SSH 生态成熟 |
| SSH | `golang.org/x/crypto/ssh` | Go 标准扩展库，唯一选择 |
| HTTP Router | `github.com/go-chi/chi/v5` | 轻量、中间件链、路由分组，Go 社区事实标准 |
| WebSocket | `github.com/coder/websocket` | gorilla/websocket 继任者，支持 context，API 更现代 |
| 日志 | `log/slog`（stdlib） | Go 1.21+ 内置，零依赖，结构化日志 |
| ID 生成 | `github.com/rs/xid` | 短、有序、URL 安全，适合隧道/转发 ID |
| 桌面壳 | Wails v2 | Go 原生桌面框架，内嵌 WebView |
| 前端框架 | React 18 + TypeScript | shadcn/ui 的基础 |
| 构建工具 | Vite | 快、Wails 原生支持 |
| UI 组件 | shadcn/ui（preset: b1a1cZciO）+ Tailwind CSS + Radix UI | 高质量、可定制、商业友好 |
| 模块路径 | `github.com/maxzhang666/ops-tunnel` | |

---

## 3. 架构

### 3.1 运行模式

```
Desktop 模式（单进程）：
┌─────────────────────────────────────┐
│            Wails 进程                │
│  ┌───────────┐  ┌────────────────┐  │
│  │  Engine    │  │  HTTP Server   │  │
│  │  (隧道引擎) │→│  REST + WS     │  │
│  └───────────┘  │  + 静态文件托管   │  │
│                  └───────┬────────┘  │
│                          ↓           │
│                 WebView (localhost)   │
│  ┌─────────────────────────────────┐ │
│  │  托盘图标 (Start/Stop/Quit)      │ │
│  └─────────────────────────────────┘ │
└─────────────────────────────────────┘

Docker/Headless 模式（单进程）：
┌─────────────────────────────────────┐
│         tunnel-server 进程           │
│  ┌───────────┐  ┌────────────────┐  │
│  │  Engine    │→│  HTTP Server   │  │
│  │            │  │  REST + WS     │  │
│  └───────────┘  │  + 静态文件托管   │  │
│                  └───────┬────────┘  │
│                          ↓           │
│                 浏览器访问 :8080      │
└─────────────────────────────────────┘
```

**关键约束：前端统一走 HTTP + WebSocket，不使用 Wails binding。两个入口共享 `internal/` 全部核心代码。**

### 3.2 目录结构

```
ops-tunnel/
├── cmd/
│   ├── tunnel-desktop/          # Wails 桌面入口
│   │   └── main.go
│   └── tunnel-server/           # Docker/Headless 入口
│       └── main.go
├── internal/
│   ├── config/                  # 配置层
│   │   ├── model.go             # Tunnel/Hop/Forward/Policy 数据结构
│   │   ├── defaults.go          # 默认值填充
│   │   ├── validate.go          # 校验规则
│   │   ├── store.go             # JSON 文件存储（原子写入）
│   │   └── redact.go            # 敏感字段脱敏
│   ├── engine/                  # 引擎层（编排）
│   │   ├── engine.go            # Engine 接口 + 实现
│   │   ├── supervisor.go        # 单隧道生命周期管理
│   │   ├── state.go             # 运行时状态类型
│   │   └── events.go            # EventBus（发布/订阅）
│   ├── ssh/                     # SSH 连接层
│   │   ├── chain.go             # 多跳链路构建
│   │   ├── auth.go              # config.Auth → ssh.AuthMethod
│   │   ├── hostkey.go           # Host Key 验证策略
│   │   └── keepalive.go         # KeepAlive 心跳
│   ├── forward/                 # 转发层
│   │   ├── forward.go           # Forward 接口
│   │   ├── local.go             # Local 转发 (-L)
│   │   ├── remote.go            # Remote 转发 (-R)
│   │   ├── dynamic.go           # Dynamic SOCKS5 (-D)
│   │   ├── socks5.go            # SOCKS5 协议最小实现
│   │   └── acl.go               # CIDR 白名单/黑名单
│   └── api/                     # API 层
│       ├── server.go            # HTTP server 启动/关闭
│       ├── routes.go            # 路由注册
│       ├── handler_tunnel.go    # Tunnel CRUD
│       ├── handler_control.go   # start/stop/restart/status
│       ├── ws.go                # WebSocket 事件推送
│       └── middleware.go        # Token 鉴权、CORS、日志
├── ui/                          # React SPA
│   ├── src/
│   │   ├── api/                 # API 客户端 + WebSocket hook
│   │   ├── components/          # shadcn/ui 组件
│   │   ├── pages/               # 页面
│   │   ├── hooks/               # 自定义 hooks
│   │   ├── lib/                 # 工具函数
│   │   ├── types/               # TypeScript 类型（与 Go model 对应）
│   │   └── App.tsx
│   ├── package.json
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   └── vite.config.ts
├── build/
│   ├── Dockerfile
│   └── docker-compose.yml
├── data/                        # 运行时数据（不提交）
├── go.mod
├── go.sum
├── wails.json
└── Makefile
```

### 3.3 数据流

```
用户操作 (UI)
    ↓ HTTP POST /api/v1/tunnels/{id}/start
API Handler
    ↓ engine.StartTunnel(id)
Engine
    ↓ supervisor.Start()
Supervisor
    ├→ ssh.BuildChain(hops, target)     # 建立多跳 SSH 链路
    ├→ forward.Start(targetClient)       # 启动转发规则
    └→ eventBus.Publish(stateChanged)    # 发布状态变更事件
                ↓
         WebSocket 推送到前端
                ↓
         UI 实时更新状态
```

---

## 4. 分阶段实施

### Phase 0：项目骨架

**目标：** 两个入口（desktop/server）都能编译运行，访问到 health 端点和空白前端页面。

**产物：**
- `go.mod` + 基础依赖
- `cmd/tunnel-server/main.go`：启动 HTTP server，支持 `--listen`、`--data-dir` flags
- `internal/api/server.go` + `routes.go`：`GET /healthz` → 200
- `ui/`：Vite + React + shadcn/ui 初始化，一个空白页面显示 "OpsTunnel"
- `cmd/tunnel-server` 托管 `ui/dist/` 静态文件
- `Makefile`：`make build-ui`、`make build-server`、`make dev`

**暂不做：** Wails 集成（Phase 9 做）。先用 tunnel-server 开发全部功能，最后接入 Wails。

**验收：**
- `make dev` 启动 server → 浏览器打开 `http://localhost:8080` 看到前端页面
- `curl http://localhost:8080/healthz` → `200 OK`

---

### Phase 1：Config 模型 + 存储 + 校验

**目标：** 定义完整的数据结构，能从 JSON 文件加载/保存配置，通过 API 进行 CRUD。

**产物：**
- `internal/config/model.go`：Tunnel、Hop、Target、Forward、Policy、Auth 等完整结构体
- `internal/config/defaults.go`：缺省值填充（如 DialTimeout 默认 10s，KeepAlive 默认 15s）
- `internal/config/validate.go`：结构化校验错误
- `internal/config/store.go`：FileStore，原子写入（temp → fsync → rename）
- `internal/config/redact.go`：脱敏函数，替换 password/keyPem/passphrase 为 `"***"`
- `internal/api/handler_tunnel.go`：CRUD 端点

**API 端点：**
```
GET    /api/v1/tunnels          # 列表（脱敏）
POST   /api/v1/tunnels          # 创建
GET    /api/v1/tunnels/{id}     # 详情（脱敏）
PUT    /api/v1/tunnels/{id}     # 全量更新
DELETE /api/v1/tunnels/{id}     # 删除
```

**校验规则：**
- ID 非空且全局唯一（创建时自动生成）
- Endpoint.Port ∈ [1, 65535]
- Auth.Type 决定必填字段（password 需要 Password，privateKey 需要 PrivateKey）
- Forward: local/dynamic 需要 Listen；remote 需要 Listen + Connect
- SOCKS5 安全警告：listen 0.0.0.0 + auth none + allowCIDRs 包含 0.0.0.0/0 → 返回 warning

**验收：**
- 启动时 `data/config.json` 不存在则自动创建空配置 `{"version":1,"tunnels":[]}`
- `POST /tunnels` 创建隧道 → `GET /tunnels` 能列出（敏感字段已脱敏）
- 重启后数据仍在
- 校验失败返回 400 + 结构化错误

---

### Phase 2：EventBus + Engine 骨架 + WebSocket

**目标：** Engine 管理隧道生命周期（状态机），EventBus 发布事件，WebSocket 实时推送。start/stop 暂为 stub（只改状态，不建 SSH 连接）。

**产物：**
- `internal/engine/events.go`：EventBus 实现（fan-out，每 subscriber 一个带 buffer 的 channel）
- `internal/engine/state.go`：TunnelStatus、HopRuntimeStatus、ForwardRuntimeStatus
- `internal/engine/engine.go`：Engine 接口 + 实现（持有 config.Store + EventBus + supervisors map）
- `internal/engine/supervisor.go`：TunnelSupervisor stub（Start → running，Stop → stopped）
- `internal/api/handler_control.go`：`POST start/stop/restart`，`GET status`
- `internal/api/ws.go`：WebSocket 端点，订阅 EventBus 并推送

**状态机：**
```
stopped ──start──→ starting ──→ running
   ↑                              │
   │                         (error)
   │                              ↓
   └──stop──── stopping ←── degraded/error
                                  │
                            (auto restart)
                                  ↓
                              starting
```

**API 端点（新增）：**
```
POST /api/v1/tunnels/{id}/start
POST /api/v1/tunnels/{id}/stop
POST /api/v1/tunnels/{id}/restart
GET  /api/v1/tunnels/{id}/status
GET  /ws                              # WebSocket
```

**WebSocket 消息格式：**
```json
{
  "type": "tunnel.stateChanged",
  "tunnelId": "xxx",
  "level": "info",
  "ts": "2026-04-02T12:00:00Z",
  "message": "tunnel started",
  "fields": { "state": "running" }
}
```

**验收：**
- `POST /tunnels/{id}/start` → `GET /status` 返回 `"state":"running"`
- WebSocket 客户端（wscat 或浏览器）能实时收到 stateChanged 事件
- `POST /stop` → 状态变为 stopped，WS 收到事件

---

### Phase 3：SSH 多跳链路

**目标：** 实现真实的 SSH 连接：单跳和多跳（hop1 → hop2 → ... → target），每跳独立认证。

**产物：**
- `internal/ssh/auth.go`：config.Auth → []ssh.AuthMethod（支持 password、privateKey、privateKey+passphrase）
- `internal/ssh/hostkey.go`：HostKeyCallback 工厂（insecure / acceptNew / strict）+ HostKeyStore 接口（JSON 文件实现）
- `internal/ssh/keepalive.go`：KeepAlive goroutine（SendRequest "keepalive@openssh.com"）
- `internal/ssh/chain.go`：
  ```
  BuildChain(ctx, hops, target, eventSink) → (*ChainResult, error)
  ```
  - 逐跳建立：dial 第一跳 → 通过第一跳 client.Dial 连接第二跳 → ... → target
  - 每跳启动 KeepAlive goroutine
  - 任何一跳失败返回 error + 已建立的连接列表（供 cleanup）
  - ChainResult 包含所有 ssh.Client + TargetClient() 便捷方法

**关键实现细节：**
```
Hop1: net.Dial("tcp", hop1.endpoint) → ssh.NewClientConn → ssh.NewClient
Hop2: hop1Client.Dial("tcp", hop2.endpoint) → ssh.NewClientConn → ssh.NewClient
Target: hop2Client.Dial("tcp", target.endpoint) → ssh.NewClientConn → ssh.NewClient
```

**集成：** 修改 supervisor.Start() 调用 ssh.BuildChain 替代 stub。

**验收：**
- 配置 1 个 hop + 1 个 target（password 认证）→ start 后 status 显示 chain 各跳 connected
- 配置 2 个 hops + 1 个 target（混合认证：hop1 私钥，hop2 密码）→ 连通
- 错误场景：hop2 地址错误 → status 显示 hop2 error，事件日志包含错误详情
- KeepAlive 工作：连接维持 > 1 分钟不断

---

### Phase 4：Local Forward (-L)

**目标：** 实现本地端口转发。在 core 本地监听端口，通过 SSH 链路连接到远端服务。

**产物：**
- `internal/forward/forward.go`：Forward 接口
  ```go
  type Forwarder interface {
      Start(ctx context.Context, sshClient *ssh.Client) error
      Stop(ctx context.Context) error
      Status() ForwardStatus
  }
  ```
- `internal/forward/local.go`：
  - 绑定前检查端口可用性
  - 监听 TCP → 每连接: sshClient.Dial("tcp", connect) → 双向 io.Copy
  - 连接计数、错误追踪
  - 优雅关闭：停止 accept，等待活跃连接 drain

**集成：** supervisor 在 chain 建立后启动所有 forwards。

**验收：**
- 配置 local forward：listen 127.0.0.1:15432 → connect 127.0.0.1:5432
- 通过 SSH 链路 start 后，`psql -h 127.0.0.1 -p 15432` 能连到远端 PostgreSQL（或任何 TCP 服务）
- stop 后端口释放，`lsof -i :15432` 无结果
- 端口被占用时 start → 明确错误

---

### Phase 5：Remote Forward (-R)

**目标：** 实现远程端口转发。在远端 SSH 服务器上监听端口，流量回连到 core 本地。

**产物：**
- `internal/forward/remote.go`：
  - 使用 `sshClient.Listen("tcp", remoteAddr)` 请求远端监听
  - 远端连接进来 → 本地 net.Dial(connect) → 双向 io.Copy
  - 处理 GatewayPorts 限制：如果请求 0.0.0.0 但远端只给了 127.0.0.1，在 status/log 中明确提示

**验收：**
- 配置 remote forward：远端 listen 0.0.0.0:18080 → 本地 connect 127.0.0.1:8080
- 从远端网络访问 target:18080 → 流量到达 core 本机的 8080
- sshd 不允许 GatewayPorts 时，日志和 status 中有明确提示

---

### Phase 6：Dynamic SOCKS5 Forward (-D)

**目标：** 实现本地 SOCKS5 代理，出站流量通过 SSH 链路。

**产物：**
- `internal/forward/socks5.go`：SOCKS5 服务端实现
  - 支持 SOCKS5 协议（RFC 1928）：版本协商、方法选择
  - 支持命令：CONNECT（主动连接目标）+ BIND（等待目标回连，用于 FTP 等场景）
  - 支持地址类型：IPv4 + 域名（DOMAINNAME）+ IPv6
  - 鉴权方式：none / username+password（RFC 1929）
- `internal/forward/acl.go`：CIDR ACL
  - 规则：先检查 deny → 再检查 allow → 不在 allow 中则拒绝
  - allowCIDRs 为空 = 全部拒绝（安全默认）
- `internal/forward/dynamic.go`：组装 SOCKS5 server + ACL + SSH dial

**不实现：** UDP ASSOCIATE（第一期不需要）。

**验收：**
- 配置 dynamic forward：listen 0.0.0.0:1080，allowCIDRs: ["10.0.0.0/8"]
- `curl --socks5 127.0.0.1:1080 http://10.0.0.5:80` → 成功（通过 SSH 出口）
- `curl --socks5 127.0.0.1:1080 http://8.8.8.8` → 被 ACL 拒绝，日志记录
- 配置 userpass 鉴权 → 无凭据被拒绝
- BIND：FTP 主动模式等回连场景能正常工作

---

### Phase 7：Supervisor 完整实现（自动重连 + 退避）

**目标：** 补全 supervisor 的生产级行为：断线检测、自动重连、指数退避、重启限速。

**产物：**
- `internal/engine/supervisor.go` 完善：
  - 内部循环：`buildChain → startForwards → 等待 error/ctx → cleanup → backoff → retry`
  - 指数退避：`delay = min(minMs * factor^n, maxMs)` + 小随机抖动
  - 限速：滑动窗口计数 maxRestartsPerHour，超限进入 error 状态并停止重试
  - stop 能打断重连循环（通过 context cancel）
  - gracefulStopTimeout：stop 时给活跃连接 drain 时间

**状态转换完善：**
```
running → (SSH 断开) → degraded → (auto restart) → starting → running
running → (SSH 断开) → degraded → (超过重试上限) → error（停止重试）
error → (手动 restart) → starting → running
```

**验收：**
- 运行中手动断开网络 10 秒恢复 → 隧道自动重连回 running，日志显示退避过程
- 持续断网 → 退避间隔递增 → 达到 maxRestartsPerHour 后进入 error
- error 状态手动 restart → 重新开始
- stop 能即时中止正在等待退避的 supervisor

---

### Phase 8：API 完善 + 安全

**目标：** API 生产级加固：Token 鉴权、CORS、输入校验、日志脱敏。

**产物：**
- `internal/api/middleware.go`：
  - Token 鉴权：`Authorization: Bearer <token>`，token 从 `--token` flag 或 `CORE_TOKEN` 环境变量读取；未设置则跳过鉴权（本地模式）
  - CORS：Desktop 模式允许 localhost，Docker 模式可配置
  - Request 日志（slog，不含敏感字段）
- DTO 层加固：所有响应经过 redact，永不泄露 password/keyPem/passphrase
- `PATCH /api/v1/tunnels/{id}`：局部更新（可选，方便前端）
- `GET /api/v1/tunnels` 支持 `?status=running` 过滤

**验收：**
- 设置 token → 无 token 请求返回 401
- API 响应中搜索不到任何密码/密钥原文
- curl 能完整管理隧道全生命周期

---

### Phase 9：Frontend（React + shadcn/ui）

**目标：** 可视化管理界面。

**页面结构：**
```
┌─────────────────────────────────────────────┐
│  OpsTunnel                        [+ New]   │
├──────────┬──────────────────────────────────┤
│          │  Overview | Forwards | Config | Logs │
│ tunnel-1 │──────────────────────────────────│
│  ● running│                                  │
│          │  Chain:                           │
│ tunnel-2 │  [hop1 ●] → [hop2 ●] → [target ●] │
│  ○ stopped│                                  │
│          │  Forwards:                        │
│ tunnel-3 │  L 0.0.0.0:15432 → 127.0.0.1:5432 ● │
│  ✕ error │  D 0.0.0.0:1080 (SOCKS5) ●       │
│          │                                   │
│          │  [Start] [Stop] [Restart]         │
└──────────┴──────────────────────────────────┘
```

**产物：**
- `ui/src/api/client.ts`：封装 REST 调用（fetch wrapper + error handling）
- `ui/src/api/ws.ts`：WebSocket 连接管理（自动重连、事件分发）
- `ui/src/types/`：TypeScript 类型定义（与 Go model 对齐）
- 页面：
  - **Tunnel List**（左侧）：状态指示灯、名称、运行时长
  - **Overview Tab**：链路可视化（hop 连线图）、延迟、错误信息
  - **Forwards Tab**：转发规则列表、状态、一键复制地址（如 `socks5://host:1080`）
  - **Config Tab**：表单编辑 hops/target/auth/forwards/policy
  - **Logs Tab**：实时滚动日志，支持按 level 过滤
- 创建隧道：表单（非向导，一页完成），支持动态添加 hop 和 forward
- 安全提示：SOCKS5 监听 0.0.0.0 无鉴权时显示警告
- Docker 提示：listen 127.0.0.1 时提示"外部不可访问"

**前端技术细节：**
- 状态管理：React Context + useReducer（无需 Redux，复杂度不到那个级别）
- 数据获取：TanStack Query（缓存、自动刷新、乐观更新）
- 表单：React Hook Form + Zod 校验
- 主题：shadcn/ui 内置 dark/light mode

**验收：**
- 能通过 UI 创建隧道、配置多跳+多转发、启动、查看状态
- Logs tab 实时滚动显示日志
- 响应式：在 Wails WebView 和浏览器中都正常显示

---

### Phase 10：Wails Desktop 集成

**目标：** 打包为桌面应用，托盘菜单，开箱即用。

**产物：**
- `cmd/tunnel-desktop/main.go`：
  - 初始化 Engine + API Server（随机可用端口）
  - 启动 Wails 窗口，WebView 指向 `http://localhost:<port>`
  - 窗口关闭时最小化到托盘（不退出）
  - 托盘菜单：Show UI / Hide / Quit
  - Quit 时：Engine.Shutdown() → 优雅关闭所有隧道 → 退出
- `wails.json` 配置
- 前端构建产物通过 `//go:embed` 嵌入 Go 二进制
- 支持 autoStart policy：应用启动时自动启动标记了 autoStart 的隧道

**验收：**
- `wails dev` 启动 → 看到 UI 窗口
- 关闭窗口 → 托盘可见，隧道继续运行
- 托盘 Quit → 所有隧道优雅关闭
- `wails build` → 生成单二进制，双击即用

---

### Phase 11：Docker 支持

**目标：** Docker 镜像 + 文档。

**产物：**
- `build/Dockerfile`：多阶段构建（Go build + Node build → alpine 最终镜像）
- `build/docker-compose.yml`：示例配置
- `cmd/tunnel-server/main.go`：支持 `--listen 0.0.0.0:8080`、`--ui`、`--token`

**Dockerfile 核心：**
```dockerfile
# Stage 1: Build UI
FROM node:20-alpine AS ui-builder
# npm install + build

# Stage 2: Build Go
FROM golang:1.22-alpine AS go-builder
# embed ui/dist + go build

# Stage 3: Runtime
FROM alpine:3.19
# copy binary + expose ports
```

**验收：**
- `docker build -t ops-tunnel .`
- `docker run -p 8080:8080 -p 1080:1080 ops-tunnel --token=secret`
- 浏览器访问 :8080 看到 UI
- SOCKS5 1080 端口对外可用

---

### Phase 12：打磨与加固

**目标：** 生产就绪。

**内容：**
- 单元测试：config 校验、ACL 规则、退避算法、状态机转换
- 集成测试：使用 Docker SSH 容器进行多跳连接测试
- 错误处理统一：所有 API 错误格式一致
- 性能：大量并发连接场景的 forward 稳定性
- CI/CD：GitHub Actions（lint + test + build）
- 文档：README（安装、使用、Docker 部署、安全建议）

---

## 5. 依赖关系与执行顺序

```
Phase 0 (骨架)
    ↓
Phase 1 (Config)
    ↓
Phase 2 (EventBus + Engine stub + WS)
    ↓
Phase 3 (SSH Chain) ──────────────────┐
    ↓                                  │
Phase 4 (Local Forward)               │
    ↓                                  │
Phase 5 (Remote Forward)              │
    ↓                                  │
Phase 6 (Dynamic SOCKS5)              │
    ↓                                  │
Phase 7 (Supervisor) ←────────────────┘ (需要 chain + forward 都就绪)
    ↓
Phase 8 (API 加固)
    ↓
Phase 9 (Frontend) ←── 可从 Phase 2 之后开始并行开发基础 UI
    ↓
Phase 10 (Wails Desktop)
    ↓
Phase 11 (Docker)
    ↓
Phase 12 (打磨)
```

**可并行的部分：**
- Phase 9（前端）的基础框架和组件可以在 Phase 2 之后就开始，用 stub API 开发
- Phase 5 和 Phase 6 互不依赖，可并行
- Phase 10 和 Phase 11 互不依赖，可并行

---

## 6. 关键技术决策记录

| # | 决策 | 理由 |
|---|------|------|
| D1 | 前端统一走 HTTP+WS，不用 Wails binding | 一套前端代码，Desktop/Docker 通用 |
| D2 | Target = Hop（type alias） | 结构相同，减少冗余；如后续需要区分，改为独立 struct + 嵌入公共字段 |
| D3 | SOCKS5 自实现（仅 CONNECT） | 避免重依赖，需求明确，代码量约 200 行 |
| D4 | ACL: deny → allow → 默认拒绝 | 安全优先，避免 Docker 场景误开放代理 |
| D5 | 配置存 JSON 单文件 | 简单，原子写入保证一致性；后续可扩展为 SQLite |
| D6 | slog 做日志 | 零依赖，结构化，Go 1.21+ 标准库 |
| D7 | 先做 tunnel-server 开发全部功能，最后接 Wails | 降低开发环节复杂度，前期只需 Go + Node 环境 |
| D8 | ID 用 xid 而非 UUID | 更短（20 字符）、有时间排序、URL 安全 |

---

## 7. 第一期不做（明确排除）

- SSH Agent 转发、Kerberos、PKCS11 认证
- 导入系统 OpenSSH config
- 凭据加密存储（先明文 JSON，后续加 keychain 集成）
- 多用户/权限系统
- Web SaaS 模式
- 自动更新
- SOCKS5 UDP ASSOCIATE
