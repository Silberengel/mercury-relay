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
	if config.Access.AllowPublicRead == false && config.Access.AllowPublicWrite == false {
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
