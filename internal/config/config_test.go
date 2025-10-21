package config

import (
	"os"
	"testing"
	"time"

	"mercury-relay/test/helpers"
)

func TestConfigLoading(t *testing.T) {
	t.Run("Load from YAML", func(t *testing.T) {
		// Create temporary config file
		configContent := `
server:
  host: "localhost"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"

access:
  owner_npub: "npub1test"
  update_interval: "1h"
  relay_url: "https://relay.damus.io"
  allow_public_read: true
  allow_public_write: false

quality:
  max_content_length: 10000
  rate_limit_per_minute: 100
  spam_threshold: 0.7

cache:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  ttl: "28h"

queue:
  host: "localhost"
  port: 5672
  username: "guest"
  password: "guest"
  vhost: "/"

streaming:
  enabled: true
  upstream_relays:
    - url: "wss://relay.damus.io"
      enabled: true
      priority: 1
  reconnect_interval: "30s"
  timeout: "60s"
  transport_methods:
    websocket: true
    http_streaming: false
    sse: false
    tor: false
    i2p: false

rest_api:
  enabled: true
  port: 8082
  cors_enabled: true

admin:
  enabled: true
  port: 8081

tor:
  enabled: false
  socks_port: 9050
  control_port: 9051

i2p:
  enabled: false
  sam_port: 7656

xftp:
  enabled: false
  host: "localhost"
  port: 443
  max_file_size: "50MB"
`

		// Write to temporary file
		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		helpers.AssertNoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(configContent)
		helpers.AssertNoError(t, err)
		tmpFile.Close()

		// Load config
		cfg, err := Load(tmpFile.Name())
		helpers.AssertNoError(t, err)

		// Verify server config
		helpers.AssertStringEqual(t, "localhost", cfg.Server.Host)
		helpers.AssertIntEqual(t, 8080, cfg.Server.Port)
		helpers.AssertStringEqual(t, "30s", cfg.Server.ReadTimeout.String())
		helpers.AssertStringEqual(t, "30s", cfg.Server.WriteTimeout.String())

		// Verify access config
		helpers.AssertStringEqual(t, "npub1test", cfg.Access.OwnerNpub)
		helpers.AssertStringEqual(t, "1h", cfg.Access.UpdateInterval.String())
		helpers.AssertStringEqual(t, "https://relay.damus.io", cfg.Access.RelayURL)
		helpers.AssertBoolEqual(t, true, cfg.Access.AllowPublicRead)
		helpers.AssertBoolEqual(t, false, cfg.Access.AllowPublicWrite)

		// Verify quality config
		helpers.AssertIntEqual(t, 10000, cfg.Quality.MaxContentLength)
		helpers.AssertIntEqual(t, 100, cfg.Quality.RateLimitPerMinute)
		helpers.AssertFloat64Equal(t, 0.7, cfg.Quality.SpamThreshold, 0.01)

		// Verify cache config
		helpers.AssertStringEqual(t, "localhost", cfg.Redis.Host)
		helpers.AssertStringEqual(t, "", cfg.Redis.Password)
		helpers.AssertIntEqual(t, 0, cfg.Redis.DB)
		helpers.AssertStringEqual(t, "28h", cfg.Redis.TTL.String())

		// Verify queue config
		helpers.AssertStringEqual(t, "amqp://guest:guest@localhost:5672/", cfg.RabbitMQ.URL)
		helpers.AssertStringEqual(t, "events", cfg.RabbitMQ.ExchangeName)
		helpers.AssertStringEqual(t, "events_queue", cfg.RabbitMQ.QueueName)
		helpers.AssertStringEqual(t, "events_dlx", cfg.RabbitMQ.DLXName)
		helpers.AssertStringEqual(t, "24h", cfg.RabbitMQ.TTL.String())

		// Verify streaming config
		helpers.AssertBoolEqual(t, true, cfg.Streaming.Enabled)
		helpers.AssertIntEqual(t, 1, len(cfg.Streaming.UpstreamRelays))
		helpers.AssertStringEqual(t, "wss://relay.damus.io", cfg.Streaming.UpstreamRelays[0].URL)
		helpers.AssertBoolEqual(t, true, cfg.Streaming.UpstreamRelays[0].Enabled)
		helpers.AssertIntEqual(t, 1, cfg.Streaming.UpstreamRelays[0].Priority)
		helpers.AssertStringEqual(t, "30s", cfg.Streaming.ReconnectInterval.String())
		helpers.AssertStringEqual(t, "60s", cfg.Streaming.Timeout.String())

		// Verify REST API config
		helpers.AssertBoolEqual(t, true, cfg.RESTAPI.Enabled)
		helpers.AssertIntEqual(t, 8082, cfg.RESTAPI.Port)
		helpers.AssertBoolEqual(t, true, cfg.RESTAPI.CORSEnabled)

		// Verify admin config
		helpers.AssertBoolEqual(t, true, cfg.Admin.Enabled)
		helpers.AssertIntEqual(t, 8081, cfg.Admin.Port)

		// Verify transport configs
		helpers.AssertBoolEqual(t, false, cfg.Tor.Enabled)
		helpers.AssertIntEqual(t, 9050, cfg.Tor.SocksPort)
		helpers.AssertIntEqual(t, 9051, cfg.Tor.ControlPort)

		helpers.AssertBoolEqual(t, false, cfg.I2P.Enabled)
		helpers.AssertIntEqual(t, 7656, cfg.I2P.SAMPort)

		// Verify XFTP config
		helpers.AssertBoolEqual(t, false, cfg.XFTP.Enabled)
		helpers.AssertStringEqual(t, "http://localhost:443", cfg.XFTP.ServerURL)
		helpers.AssertStringEqual(t, "/tmp/xftp", cfg.XFTP.StorageDir)
		helpers.AssertStringEqual(t, "48h", cfg.XFTP.TTL)
	})

	t.Run("Environment variable override", func(t *testing.T) {
		// Set environment variables
		os.Setenv("OWNER_NPUB", "npub1env")
		os.Setenv("NOSTR_RELAY_PORT", "9090")
		os.Setenv("STREAMING_ENABLED", "false")
		defer func() {
			os.Unsetenv("OWNER_NPUB")
			os.Unsetenv("NOSTR_RELAY_PORT")
			os.Unsetenv("STREAMING_ENABLED")
		}()

		// Create minimal config file
		configContent := `
server:
  host: "localhost"
  port: 8080

access:
  owner_npub: "npub1default"

streaming:
  enabled: true
`

		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		helpers.AssertNoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(configContent)
		helpers.AssertNoError(t, err)
		tmpFile.Close()

		// Load config
		cfg, err := Load(tmpFile.Name())
		helpers.AssertNoError(t, err)

		// Verify environment variables override YAML values
		helpers.AssertStringEqual(t, "npub1env", cfg.Access.OwnerNpub)
		helpers.AssertIntEqual(t, 9090, cfg.Server.Port)
		helpers.AssertBoolEqual(t, false, cfg.Streaming.Enabled)
	})
}

func TestConfigDefaultValues(t *testing.T) {
	t.Run("Missing optional config", func(t *testing.T) {
		// Create minimal config file
		configContent := `
server:
  host: "localhost"
`

		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		helpers.AssertNoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(configContent)
		helpers.AssertNoError(t, err)
		tmpFile.Close()

		// Load config
		cfg, err := Load(tmpFile.Name())
		helpers.AssertNoError(t, err)

		// Verify defaults are applied
		helpers.AssertStringEqual(t, "localhost", cfg.Server.Host)
		helpers.AssertIntEqual(t, 8080, cfg.Server.Port)                      // Default port
		helpers.AssertStringEqual(t, "30s", cfg.Server.ReadTimeout.String())  // Default timeout
		helpers.AssertStringEqual(t, "30s", cfg.Server.WriteTimeout.String()) // Default timeout

		// Verify other defaults
		helpers.AssertBoolEqual(t, true, cfg.Access.AllowPublicRead)        // Default
		helpers.AssertBoolEqual(t, false, cfg.Access.AllowPublicWrite)      // Default
		helpers.AssertIntEqual(t, 10000, cfg.Quality.MaxContentLength)      // Default
		helpers.AssertIntEqual(t, 100, cfg.Quality.RateLimitPerMinute)      // Default
		helpers.AssertFloat64Equal(t, 0.7, cfg.Quality.SpamThreshold, 0.01) // Default
	})

	t.Run("Invalid config values", func(t *testing.T) {
		// Create config with invalid values
		configContent := `
server:
  host: "localhost"
  port: -1  # Invalid negative port

access:
  owner_npub: "invalid-npub"  # Invalid npub format
`

		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		helpers.AssertNoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(configContent)
		helpers.AssertNoError(t, err)
		tmpFile.Close()

		// Load config - should still work but with defaults for invalid values
		cfg, err := Load(tmpFile.Name())
		helpers.AssertNoError(t, err)

		// Invalid port should fall back to default
		helpers.AssertIntEqual(t, 8080, cfg.Server.Port) // Default port, not -1

		// Invalid npub should still be stored (validation happens elsewhere)
		helpers.AssertStringEqual(t, "invalid-npub", cfg.Access.OwnerNpub)
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("Valid config", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Host:         "localhost",
				Port:         8080,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			Access: AccessConfig{
				OwnerNpub:        "npub1valid",
				UpdateInterval:   1 * time.Hour,
				RelayURL:         "https://relay.damus.io",
				AllowPublicRead:  true,
				AllowPublicWrite: false,
			},
			Quality: QualityConfig{
				MaxContentLength:   10000,
				RateLimitPerMinute: 100,
				SpamThreshold:      0.7,
			},
		}

		err := cfg.Validate()
		helpers.AssertNoError(t, err)
	})

	t.Run("Invalid server config", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Host:         "",
				Port:         -1,
				ReadTimeout:  -1 * time.Second,
				WriteTimeout: -1 * time.Second,
			},
		}

		err := cfg.Validate()
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "invalid server config")
	})

	t.Run("Invalid access config", func(t *testing.T) {
		cfg := &Config{
			Access: AccessConfig{
				OwnerNpub:        "",
				UpdateInterval:   -1 * time.Hour,
				RelayURL:         "invalid-url",
				AllowPublicRead:  true,
				AllowPublicWrite: false,
			},
		}

		err := cfg.Validate()
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "invalid access config")
	})

	t.Run("Invalid quality config", func(t *testing.T) {
		cfg := &Config{
			Quality: QualityConfig{
				MaxContentLength:   -1,
				RateLimitPerMinute: -1,
				SpamThreshold:      -1.0,
			},
		}

		err := cfg.Validate()
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "invalid quality config")
	})
}

func TestConfigEnvironmentVariables(t *testing.T) {
	t.Run("All environment variables", func(t *testing.T) {
		// Set all environment variables
		envVars := map[string]string{
			"OWNER_NPUB":             "npub1env",
			"NOSTR_RELAY_PORT":       "9090",
			"ADMIN_PORT":             "9091",
			"REST_API_PORT":          "9092",
			"LOG_LEVEL":              "debug",
			"STREAMING_ENABLED":      "true",
			"UPSTREAM_RELAYS":        "wss://relay1.com,wss://relay2.com",
			"TOR_ENABLED":            "true",
			"TOR_SOCKS_PORT":         "9050",
			"TOR_CONTROL_PORT":       "9051",
			"I2P_ENABLED":            "true",
			"I2P_SAM_PORT":           "7656",
			"XFTP_ENABLED":           "true",
			"XFTP_PORT":              "8443",
			"XFTP_MAX_FILE_SIZE":     "100MB",
			"API_KEY":                "secret-key",
			"CORS_ENABLED":           "true",
			"RATE_LIMIT_PER_MINUTE":  "200",
			"ACCESS_PUBLIC_READ":     "true",
			"ACCESS_PUBLIC_WRITE":    "false",
			"ACCESS_UPDATE_INTERVAL": "2h",
			"ACCESS_RELAY_URL":       "https://custom-relay.com",
			"REDIS_HOST":             "redis.example.com",
			"REDIS_PORT":             "6380",
			"REDIS_PASSWORD":         "redis-pass",
			"REDIS_DB":               "5",
			"RABBITMQ_HOST":          "rabbit.example.com",
			"RABBITMQ_PORT":          "5673",
			"RABBITMQ_USERNAME":      "rabbit-user",
			"RABBITMQ_PASSWORD":      "rabbit-pass",
			"RABBITMQ_VHOST":         "/custom",
		}

		for key, value := range envVars {
			os.Setenv(key, value)
		}

		defer func() {
			for key := range envVars {
				os.Unsetenv(key)
			}
		}()

		// Create minimal config file
		configContent := `
server:
  host: "localhost"
  port: 8080
`

		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		helpers.AssertNoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(configContent)
		helpers.AssertNoError(t, err)
		tmpFile.Close()

		// Load config
		cfg, err := Load(tmpFile.Name())
		helpers.AssertNoError(t, err)

		// Verify environment variables are applied
		helpers.AssertStringEqual(t, "npub1env", cfg.Access.OwnerNpub)
		helpers.AssertIntEqual(t, 9090, cfg.Server.Port)
		helpers.AssertIntEqual(t, 9091, cfg.Admin.Port)
		helpers.AssertIntEqual(t, 9092, cfg.RESTAPI.Port)
		helpers.AssertBoolEqual(t, true, cfg.Streaming.Enabled)
		helpers.AssertBoolEqual(t, true, cfg.Tor.Enabled)
		helpers.AssertBoolEqual(t, true, cfg.I2P.Enabled)
		helpers.AssertBoolEqual(t, true, cfg.XFTP.Enabled)
		helpers.AssertStringEqual(t, "secret-key", cfg.Admin.APIKey)
		helpers.AssertBoolEqual(t, true, cfg.RESTAPI.CORSEnabled)
		helpers.AssertIntEqual(t, 200, cfg.Quality.RateLimitPerMinute)
		helpers.AssertBoolEqual(t, true, cfg.Access.AllowPublicRead)
		helpers.AssertBoolEqual(t, false, cfg.Access.AllowPublicWrite)
		helpers.AssertStringEqual(t, "2h", cfg.Access.UpdateInterval.String())
		helpers.AssertStringEqual(t, "https://custom-relay.com", cfg.Access.RelayURL)
		helpers.AssertStringEqual(t, "redis.example.com", cfg.Redis.Host)
		helpers.AssertStringEqual(t, "redis-pass", cfg.Redis.Password)
		helpers.AssertIntEqual(t, 5, cfg.Redis.DB)
		helpers.AssertStringEqual(t, "amqp://rabbit-user:rabbit-pass@rabbit.example.com:5673/custom", cfg.RabbitMQ.URL)
		helpers.AssertStringEqual(t, "events", cfg.RabbitMQ.ExchangeName)
		helpers.AssertStringEqual(t, "events_queue", cfg.RabbitMQ.QueueName)
		helpers.AssertStringEqual(t, "events_dlx", cfg.RabbitMQ.DLXName)
	})
}

func TestConfigFileNotFound(t *testing.T) {
	t.Run("Non-existent config file", func(t *testing.T) {
		_, err := Load("non-existent-config.yaml")
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "no such file")
	})
}

func TestConfigInvalidYAML(t *testing.T) {
	t.Run("Invalid YAML syntax", func(t *testing.T) {
		// Create config file with invalid YAML
		configContent := `
server:
  host: "localhost"
  port: 8080
invalid: yaml: syntax: [error
`

		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		helpers.AssertNoError(t, err)
		defer os.Remove(tmpFile.Name())

		helpers.AssertNoError(t, err)
		_, err = tmpFile.WriteString(configContent)
		helpers.AssertNoError(t, err)
		tmpFile.Close()

		// Load config should fail
		_, err = Load(tmpFile.Name())
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "yaml")
	})
}

func TestConfigEmptyFile(t *testing.T) {
	t.Run("Empty config file", func(t *testing.T) {
		// Create empty config file
		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		helpers.AssertNoError(t, err)
		defer os.Remove(tmpFile.Name())

		tmpFile.Close()

		// Load config should work with defaults
		cfg, err := Load(tmpFile.Name())
		helpers.AssertNoError(t, err)

		// Verify defaults are applied
		helpers.AssertIntEqual(t, 8080, cfg.Server.Port)
		helpers.AssertBoolEqual(t, true, cfg.Access.AllowPublicRead)
		helpers.AssertBoolEqual(t, false, cfg.Access.AllowPublicWrite)
	})
}
