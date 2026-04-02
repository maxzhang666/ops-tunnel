package config

// Endpoint represents a host:port pair.
type Endpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// AuthType defines SSH authentication method.
type AuthType string

const (
	AuthPassword   AuthType = "password"
	AuthPrivateKey AuthType = "privateKey"
	AuthNone       AuthType = "none"
)

// PrivateKeySource defines where the key content comes from.
type PrivateKeySource string

const (
	KeySourceInline PrivateKeySource = "inline"
	KeySourceFile   PrivateKeySource = "file"
)

// PrivateKey holds SSH private key configuration.
type PrivateKey struct {
	Source     PrivateKeySource `json:"source"`
	KeyPEM     string          `json:"keyPem,omitempty"`
	FilePath   string          `json:"filePath,omitempty"`
	Passphrase string          `json:"passphrase,omitempty"`
}

// Auth holds SSH authentication configuration.
type Auth struct {
	Type       AuthType    `json:"type"`
	Username   string      `json:"username"`
	Password   string      `json:"password,omitempty"`
	PrivateKey *PrivateKey `json:"privateKey,omitempty"`
}

// HostKeyVerifyMode defines host key verification strategy.
type HostKeyVerifyMode string

const (
	HostKeyInsecure  HostKeyVerifyMode = "insecure"
	HostKeyAcceptNew HostKeyVerifyMode = "acceptNew"
	HostKeyStrict    HostKeyVerifyMode = "strict"
)

// HostKeyVerification holds host key verification settings.
type HostKeyVerification struct {
	Mode HostKeyVerifyMode `json:"mode"`
}

// KeepAlive holds SSH keep-alive settings.
type KeepAlive struct {
	IntervalMs int `json:"intervalMs"`
	MaxMissed  int `json:"maxMissed"`
}

// SSHConnection is an independently managed SSH connection configuration.
type SSHConnection struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Endpoint            Endpoint            `json:"endpoint"`
	Auth                Auth                `json:"auth"`
	HostKeyVerification HostKeyVerification `json:"hostKeyVerification"`
	DialTimeoutMs       int                 `json:"dialTimeoutMs"`
	KeepAlive           KeepAlive           `json:"keepAlive"`
}

// TunnelMode defines the forwarding type.
type TunnelMode string

const (
	ModeLocal   TunnelMode = "local"
	ModeRemote  TunnelMode = "remote"
	ModeDynamic TunnelMode = "dynamic"
)

// Socks5Auth defines SOCKS5 authentication method.
type Socks5Auth string

const (
	Socks5None     Socks5Auth = "none"
	Socks5UserPass Socks5Auth = "userpass"
)

// Socks5Cfg holds SOCKS5 proxy configuration for dynamic tunnels.
type Socks5Cfg struct {
	Auth       Socks5Auth `json:"auth"`
	Username   string     `json:"username,omitempty"`
	Password   string     `json:"password,omitempty"`
	AllowCIDRs []string   `json:"allowCIDRs,omitempty"`
	DenyCIDRs  []string   `json:"denyCIDRs,omitempty"`
}

// Mapping defines a single port forwarding rule within a tunnel.
type Mapping struct {
	ID      string     `json:"id"`
	Listen  Endpoint   `json:"listen"`
	Connect Endpoint   `json:"connect,omitempty"`
	Socks5  *Socks5Cfg `json:"socks5,omitempty"`
	Notes   string     `json:"notes,omitempty"`
}

// RestartBackoff holds exponential backoff parameters.
type RestartBackoff struct {
	MinMs  int     `json:"minMs"`
	MaxMs  int     `json:"maxMs"`
	Factor float64 `json:"factor"`
}

// Policy holds tunnel runtime behavior settings.
type Policy struct {
	AutoStart             bool           `json:"autoStart"`
	AutoRestart           bool           `json:"autoRestart"`
	RestartBackoff        RestartBackoff `json:"restartBackoff"`
	MaxRestartsPerHour    int            `json:"maxRestartsPerHour"`
	GracefulStopTimeoutMs int            `json:"gracefulStopTimeoutMs"`
}

// Tunnel references SSH connections and defines forwarding rules.
type Tunnel struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Mode     TunnelMode `json:"mode"`
	Chain    []string   `json:"chain"`
	Mappings []Mapping  `json:"mappings"`
	Policy   Policy     `json:"policy"`
}

// Config is the top-level configuration persisted to disk.
type Config struct {
	Version        int              `json:"version"`
	SSHConnections []SSHConnection  `json:"sshConnections"`
	Tunnels        []Tunnel         `json:"tunnels"`
}

// NewConfig returns an empty config with version set.
func NewConfig() *Config {
	return &Config{
		Version:        1,
		SSHConnections: []SSHConnection{},
		Tunnels:        []Tunnel{},
	}
}
