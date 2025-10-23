package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Tor       TorConfig       `yaml:"tor"`
	I2P       I2PConfig       `yaml:"i2p"`
	SSH       SSHConfig       `yaml:"ssh"`
	RabbitMQ  RabbitMQConfig  `yaml:"rabbitmq"`
	Redis     RedisConfig     `yaml:"redis"`
	XFTP      XFTPConfig      `yaml:"xftp"`
	Postgres  PostgresConfig  `yaml:"postgres"`
	Quality   QualityConfig   `yaml:"quality"`
	Access    AccessConfig    `yaml:"access"`
	Admin     AdminConfig     `yaml:"admin"`
	GRPC      GRPCConfig      `yaml:"grpc"`
	RESTAPI   RESTAPIConfig   `yaml:"rest_api"`
	Streaming StreamingConfig `yaml:"streaming"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type TorConfig struct {
	Enabled           bool   `yaml:"enabled"`
	DataDir           string `yaml:"data_dir"`
	ControlPort       int    `yaml:"control_port"`
	SocksPort         int    `yaml:"socks_port"`
	HiddenServiceDir  string `yaml:"hidden_service_dir"`
	HiddenServicePort int    `yaml:"hidden_service_port"`
}

type I2PConfig struct {
	Enabled    bool   `yaml:"enabled"`
	SAMPort    int    `yaml:"sam_port"`
	SAMHost    string `yaml:"sam_host"`
	TunnelName string `yaml:"tunnel_name"`
	TunnelPort int    `yaml:"tunnel_port"`
}

type SSHConfig struct {
	Enabled           bool              `yaml:"enabled"`
	KeyStorage        SSHKeyStorage     `yaml:"key_storage"`
	Connection        SSHConnection     `yaml:"connection"`
	TerminalInterface TerminalInterface `yaml:"terminal_interface"`
	Authentication    SSHAuthentication `yaml:"authentication"`
}

type SSHAuthentication struct {
	RequireAuth       bool     `yaml:"require_auth"`
	APIKey           string   `yaml:"api_key"`
	BasicAuthUser    string   `yaml:"basic_auth_user"`
	BasicAuthPass    string   `yaml:"basic_auth_pass"`
	AuthorizedPubkeys []string `yaml:"authorized_pubkeys"`
	AllowLocalhost    bool     `yaml:"allow_localhost"`
}

type SSHKeyStorage struct {
	KeyDir        string `yaml:"key_dir"`
	PrivateKeyExt string `yaml:"private_key_ext"`
	PublicKeyExt  string `yaml:"public_key_ext"`
	KeySize       int    `yaml:"key_size"`
	KeyType       string `yaml:"key_type"`
}

type SSHConnection struct {
	Host        string        `yaml:"host"`
	Port        int           `yaml:"port"`
	Username    string        `yaml:"username"`
	Timeout     time.Duration `yaml:"timeout"`
	KeepAlive   time.Duration `yaml:"keep_alive"`
	MaxRetries  int           `yaml:"max_retries"`
	RetryDelay  time.Duration `yaml:"retry_delay"`
	Compression bool          `yaml:"compression"`
	Banner      string        `yaml:"banner"`
}

type TerminalInterface struct {
	Enabled     bool   `yaml:"enabled"`
	Port        int    `yaml:"port"`
	Host        string `yaml:"host"`
	Interactive bool   `yaml:"interactive"`
	LogLevel    string `yaml:"log_level"`
}

type RabbitMQConfig struct {
	URL          string        `yaml:"url"`
	ExchangeName string        `yaml:"exchange_name"`
	QueueName    string        `yaml:"queue_name"`
	DLXName      string        `yaml:"dlx_name"`
	TTL          time.Duration `yaml:"ttl"`
}

type RedisConfig struct {
	Host     string        `yaml:"host"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	TTL      time.Duration `yaml:"ttl"`
}

type XFTPConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ServerURL  string `yaml:"server_url"`
	StorageDir string `yaml:"storage_dir"`
	TTL        string `yaml:"ttl"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type QualityConfig struct {
	SpamThreshold        float64 `yaml:"spam_threshold"`
	RateLimitPerMinute   int     `yaml:"rate_limit_per_minute"`
	MaxContentLength     int     `yaml:"max_content_length"`
	QuarantineSuspicious bool    `yaml:"quarantine_suspicious"`
}

type AccessConfig struct {
	OwnerNpub        string        `yaml:"owner_npub"`
	UpdateInterval   time.Duration `yaml:"update_interval"`
	RelayURL         string        `yaml:"relay_url"`
	AllowPublicRead  bool          `yaml:"allow_public_read"`
	AllowPublicWrite bool          `yaml:"allow_public_write"`
}

type AdminConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	APIKey  string `yaml:"api_key"`
}

type GRPCConfig struct {
	Enabled            bool          `yaml:"enabled"`
	ServerHost         string        `yaml:"server_host"`
	ServerPort         int           `yaml:"server_port"`
	Timeout            time.Duration `yaml:"timeout"`
	MaxRetries         int           `yaml:"max_retries"`
	RetryInterval      time.Duration `yaml:"retry_interval"`
	TLSEnabled         bool          `yaml:"tls_enabled"`
	CertFile           string        `yaml:"cert_file"`
	KeyFile            string        `yaml:"key_file"`
	KeepAliveTime      time.Duration `yaml:"keepalive_time"`
	KeepAliveTimeout   time.Duration `yaml:"keepalive_timeout"`
	MaxMessageSize     int           `yaml:"max_message_size"`
	CompressionEnabled bool          `yaml:"compression_enabled"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file"`
}

type RESTAPIConfig struct {
	Enabled            bool             `yaml:"enabled"`
	Port               int              `yaml:"port"`
	CORSEnabled        bool             `yaml:"cors_enabled"`
	CORSOrigins        []string         `yaml:"cors_origins"`
	RateLimitPerMinute int              `yaml:"rate_limit_per_minute"`
	Endpoints          RESTAPIEndpoints `yaml:"endpoints"`
}

type RESTAPIEndpoints struct {
	Events  string `yaml:"events"`
	Query   string `yaml:"query"`
	Publish string `yaml:"publish"`
	Health  string `yaml:"health"`
	Stats   string `yaml:"stats"`
}

type StreamingConfig struct {
	Enabled            bool             `yaml:"enabled"`
	UpstreamRelays     []UpstreamRelay  `yaml:"upstream_relays"`
	TransportMethods   TransportMethods `yaml:"transport_methods"`
	ConnectionPoolSize int              `yaml:"connection_pool_size"`
	ReconnectInterval  time.Duration    `yaml:"reconnect_interval"`
	Timeout            time.Duration    `yaml:"timeout"`
}

type UpstreamRelay struct {
	URL      string `yaml:"url"`
	Enabled  bool   `yaml:"enabled"`
	Priority int    `yaml:"priority"`
}

type TransportMethods struct {
	WebSocket     bool `yaml:"websocket"`
	Tor           bool `yaml:"tor"`
	I2P           bool `yaml:"i2p"`
	SSH           bool `yaml:"ssh"`
	HTTPStreaming bool `yaml:"http_streaming"`
	SSE           bool `yaml:"sse"`
	GRPC          bool `yaml:"grpc"`
}

func Load(path string) (*Config, error) {
	var config Config

	// Load from file if it exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Set defaults for any unset fields
	setDefaults(&config)

	// Apply environment variable overrides
	applyEnvOverrides(&config)

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(config *Config) {
	// Server defaults
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}
	if config.Server.Port <= 0 {
		config.Server.Port = 8080
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}

	// Access defaults
	if !config.Access.AllowPublicRead && !config.Access.AllowPublicWrite {
		config.Access.AllowPublicRead = true
		config.Access.AllowPublicWrite = false
	}
	if config.Access.UpdateInterval == 0 {
		config.Access.UpdateInterval = time.Hour
	}

	// Quality defaults
	if config.Quality.MaxContentLength == 0 {
		config.Quality.MaxContentLength = 10000
	}
	if config.Quality.RateLimitPerMinute == 0 {
		config.Quality.RateLimitPerMinute = 100
	}
	if config.Quality.SpamThreshold == 0 {
		config.Quality.SpamThreshold = 0.7
	}

	// RabbitMQ defaults
	if config.RabbitMQ.ExchangeName == "" {
		config.RabbitMQ.ExchangeName = "events"
	}
	if config.RabbitMQ.QueueName == "" {
		config.RabbitMQ.QueueName = "events_queue"
	}
	if config.RabbitMQ.DLXName == "" {
		config.RabbitMQ.DLXName = "events_dlx"
	}

	// SSH defaults
	if config.SSH.KeyStorage.KeyDir == "" {
		config.SSH.KeyStorage.KeyDir = "./ssh-keys"
	}
	if config.SSH.KeyStorage.PrivateKeyExt == "" {
		config.SSH.KeyStorage.PrivateKeyExt = ".pem"
	}
	if config.SSH.KeyStorage.PublicKeyExt == "" {
		config.SSH.KeyStorage.PublicKeyExt = ".pub"
	}
	if config.SSH.KeyStorage.KeySize == 0 {
		config.SSH.KeyStorage.KeySize = 2048
	}
	if config.SSH.KeyStorage.KeyType == "" {
		config.SSH.KeyStorage.KeyType = "rsa"
	}
	if config.SSH.Connection.Port == 0 {
		config.SSH.Connection.Port = 22
	}
	if config.SSH.Connection.Timeout == 0 {
		config.SSH.Connection.Timeout = 30 * time.Second
	}
	if config.SSH.Connection.KeepAlive == 0 {
		config.SSH.Connection.KeepAlive = 30 * time.Second
	}
	if config.SSH.Connection.MaxRetries == 0 {
		config.SSH.Connection.MaxRetries = 3
	}
	if config.SSH.Connection.RetryDelay == 0 {
		config.SSH.Connection.RetryDelay = 5 * time.Second
	}
	if config.SSH.TerminalInterface.Port == 0 {
		config.SSH.TerminalInterface.Port = 2222
	}
	if config.SSH.TerminalInterface.Host == "" {
		config.SSH.TerminalInterface.Host = "localhost"
	}
	if config.SSH.TerminalInterface.LogLevel == "" {
		config.SSH.TerminalInterface.LogLevel = "info"
	}
	
	// SSH authentication defaults
	if config.SSH.Authentication.APIKey == "" {
		config.SSH.Authentication.APIKey = "admin-ssh-key-2024"
	}
	if config.SSH.Authentication.BasicAuthUser == "" {
		config.SSH.Authentication.BasicAuthUser = "admin"
	}
	if config.SSH.Authentication.BasicAuthPass == "" {
		config.SSH.Authentication.BasicAuthPass = "mercury-ssh-2024"
	}
	if !config.SSH.Authentication.RequireAuth {
		config.SSH.Authentication.RequireAuth = true // Default to requiring auth
	}
	if !config.SSH.Authentication.AllowLocalhost {
		config.SSH.Authentication.AllowLocalhost = true // Default to allowing localhost
	}
}

// applyEnvOverrides applies environment variable overrides
func applyEnvOverrides(config *Config) {
	// Server config
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if port := os.Getenv("NOSTR_RELAY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	// Access config
	if owner := os.Getenv("OWNER_NPUB"); owner != "" {
		config.Access.OwnerNpub = owner
	}
	if url := os.Getenv("ACCESS_RELAY_URL"); url != "" {
		config.Access.RelayURL = url
	}
	if read := os.Getenv("ACCESS_PUBLIC_READ"); read != "" {
		config.Access.AllowPublicRead = read == "true"
	}
	if write := os.Getenv("ACCESS_PUBLIC_WRITE"); write != "" {
		config.Access.AllowPublicWrite = write == "true"
	}
	if interval := os.Getenv("ACCESS_UPDATE_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			config.Access.UpdateInterval = d
		}
	}

	// Admin config
	if port := os.Getenv("ADMIN_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Admin.Port = p
		}
	}
	if key := os.Getenv("API_KEY"); key != "" {
		config.Admin.APIKey = key
	}

	// REST API config
	if port := os.Getenv("REST_API_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.RESTAPI.Port = p
		}
	}
	if cors := os.Getenv("CORS_ENABLED"); cors != "" {
		config.RESTAPI.CORSEnabled = cors == "true"
	}

	// Streaming config
	if streaming := os.Getenv("STREAMING_ENABLED"); streaming != "" {
		config.Streaming.Enabled = streaming == "true"
	}

	// Tor config
	if tor := os.Getenv("TOR_ENABLED"); tor != "" {
		config.Tor.Enabled = tor == "true"
	}
	if port := os.Getenv("TOR_SOCKS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Tor.SocksPort = p
		}
	}
	if port := os.Getenv("TOR_CONTROL_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Tor.ControlPort = p
		}
	}

	// I2P config
	if i2p := os.Getenv("I2P_ENABLED"); i2p != "" {
		config.I2P.Enabled = i2p == "true"
	}
	if port := os.Getenv("I2P_SAM_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.I2P.SAMPort = p
		}
	}

	// SSH config
	if ssh := os.Getenv("SSH_ENABLED"); ssh != "" {
		config.SSH.Enabled = ssh == "true"
	}
	if host := os.Getenv("SSH_HOST"); host != "" {
		config.SSH.Connection.Host = host
	}
	if port := os.Getenv("SSH_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.SSH.Connection.Port = p
		}
	}
	if username := os.Getenv("SSH_USERNAME"); username != "" {
		config.SSH.Connection.Username = username
	}
	if keyDir := os.Getenv("SSH_KEY_DIR"); keyDir != "" {
		config.SSH.KeyStorage.KeyDir = keyDir
	}
	if keyType := os.Getenv("SSH_KEY_TYPE"); keyType != "" {
		config.SSH.KeyStorage.KeyType = keyType
	}
	if keySize := os.Getenv("SSH_KEY_SIZE"); keySize != "" {
		if size, err := strconv.Atoi(keySize); err == nil {
			config.SSH.KeyStorage.KeySize = size
		}
	}
	if timeout := os.Getenv("SSH_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.SSH.Connection.Timeout = d
		}
	}
	if terminal := os.Getenv("SSH_TERMINAL_ENABLED"); terminal != "" {
		config.SSH.TerminalInterface.Enabled = terminal == "true"
	}
	if terminalPort := os.Getenv("SSH_TERMINAL_PORT"); terminalPort != "" {
		if p, err := strconv.Atoi(terminalPort); err == nil {
			config.SSH.TerminalInterface.Port = p
		}
	}

	// XFTP config
	if xftp := os.Getenv("XFTP_ENABLED"); xftp != "" {
		config.XFTP.Enabled = xftp == "true"
	}

	// Quality config
	if rate := os.Getenv("RATE_LIMIT_PER_MINUTE"); rate != "" {
		if r, err := strconv.Atoi(rate); err == nil {
			config.Quality.RateLimitPerMinute = r
		}
	}

	// Redis config
	if host := os.Getenv("REDIS_HOST"); host != "" {
		config.Redis.Host = host
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		config.Redis.Password = password
	}
	if db := os.Getenv("REDIS_DB"); db != "" {
		if d, err := strconv.Atoi(db); err == nil {
			config.Redis.DB = d
		}
	}

	// RabbitMQ config
	if host := os.Getenv("RABBITMQ_HOST"); host != "" {
		username := os.Getenv("RABBITMQ_USERNAME")
		password := os.Getenv("RABBITMQ_PASSWORD")
		port := os.Getenv("RABBITMQ_PORT")
		vhost := os.Getenv("RABBITMQ_VHOST")

		if username == "" {
			username = "guest"
		}
		if password == "" {
			password = "guest"
		}
		if port == "" {
			port = "5672"
		}
		if vhost == "" {
			vhost = "/"
		}

		config.RabbitMQ.URL = fmt.Sprintf("amqp://%s:%s@%s:%s%s", username, password, host, port, vhost)
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server config: port %d", c.Server.Port)
	}
	if c.Server.ReadTimeout < 0 {
		return fmt.Errorf("invalid server config: negative read timeout")
	}
	if c.Server.WriteTimeout < 0 {
		return fmt.Errorf("invalid server config: negative write timeout")
	}

	// Validate access config
	if c.Access.UpdateInterval < 0 {
		return fmt.Errorf("invalid access config: negative update interval")
	}

	// Validate quality config
	if c.Quality.MaxContentLength <= 0 {
		return fmt.Errorf("invalid quality config: max content length %d", c.Quality.MaxContentLength)
	}
	if c.Quality.RateLimitPerMinute <= 0 {
		return fmt.Errorf("invalid quality config: rate limit %d", c.Quality.RateLimitPerMinute)
	}
	if c.Quality.SpamThreshold < 0 || c.Quality.SpamThreshold > 1 {
		return fmt.Errorf("invalid quality config: spam threshold %f", c.Quality.SpamThreshold)
	}

	return nil
}
