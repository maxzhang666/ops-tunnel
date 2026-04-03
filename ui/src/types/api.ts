export interface Endpoint {
  host: string
  port: number
}

export type AuthType = 'password' | 'privateKey' | 'none'
export type PrivateKeySource = 'inline' | 'file'

export interface PrivateKey {
  source: PrivateKeySource
  keyPem?: string
  filePath?: string
  passphrase?: string
}

export interface Auth {
  type: AuthType
  username: string
  password?: string
  privateKey?: PrivateKey
}

export type HostKeyVerifyMode = 'insecure' | 'acceptNew' | 'strict'

export interface HostKeyVerification {
  mode: HostKeyVerifyMode
}

export interface KeepAlive {
  intervalMs: number
  maxMissed: number
}

export interface SSHConnection {
  id: string
  name: string
  endpoint: Endpoint
  auth: Auth
  hostKeyVerification: HostKeyVerification
  dialTimeoutMs: number
  keepAlive: KeepAlive
}

export type TunnelMode = 'local' | 'remote' | 'dynamic'
export type Socks5Auth = 'none' | 'userpass'

export interface Socks5Cfg {
  auth: Socks5Auth
  username?: string
  password?: string
  allowCIDRs?: string[]
  denyCIDRs?: string[]
}

export interface Mapping {
  id: string
  listen: Endpoint
  connect?: Endpoint
  socks5?: Socks5Cfg
  notes?: string
}

export interface RestartBackoff {
  minMs: number
  maxMs: number
  factor: number
}

export interface Policy {
  autoStart: boolean
  autoRestart: boolean
  restartBackoff: RestartBackoff
  maxRestartsPerHour: number
  gracefulStopTimeoutMs: number
}

export interface Tunnel {
  id: string
  name: string
  enabled: boolean
  mode: TunnelMode
  chain: string[]
  mappings: Mapping[]
  policy: Policy
}

export type TunnelState = 'stopped' | 'starting' | 'running' | 'degraded' | 'error' | 'stopping'

export interface HopStatus {
  sshConnId: string
  state: string
  latencyMs?: number
  detail?: string
}

export interface MappingStatus {
  mappingId: string
  state: string
  listen: string
  detail?: string
}

export interface TunnelStatus {
  id: string
  state: TunnelState
  since: string
  chain: HopStatus[]
  mappings: MappingStatus[]
  lastError?: string
}

export interface TestResult {
  status: 'ok' | 'error'
  message: string
  latencyMs?: number
}

export interface TunnelEvent {
  type: string
  tunnelId?: string
  level?: string
  ts: string
  message: string
  fields?: Record<string, unknown>
}

export interface ApiErrorBody {
  error: string
  details?: { field: string; message: string }[]
}

export interface GeneralConfig {
  logLevel: string
  language: string
  autoStart: boolean
}

export interface AppearanceConfig {
  theme: string
}

export interface DesktopConfig {
  closeAction: string
}

export interface Settings {
  general: GeneralConfig
  appearance: AppearanceConfig
  desktop: DesktopConfig
}

export interface LatestRelease {
  version: string
  url: string
  publishedAt: string
}

export interface VersionInfo {
  version: string
  mode: 'server' | 'desktop'
  latest: LatestRelease | null
}
