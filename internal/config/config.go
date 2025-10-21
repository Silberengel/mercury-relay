package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Tor       TorConfig       `yaml:"tor"`
	I2P       I2PConfig       `yaml:"i2p"`
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
	HTTPStreaming bool `yaml:"http_streaming"`
	SSE           bool `yaml:"sse"`
	GRPC          bool `yaml:"grpc"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Quality.MaxContentLength <= 0 {
		return fmt.Errorf("invalid max content length: %d", c.Quality.MaxContentLength)
	}
	if c.Quality.RateLimitPerMinute <= 0 {
		return fmt.Errorf("invalid rate limit: %d", c.Quality.RateLimitPerMinute)
	}
	return nil
}
