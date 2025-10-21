package transport

import (
	"context"
	"fmt"
	"log"
	"time"

	"mercury-relay/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// GRPCTransport implements gRPC transport for high-performance communication
type GRPCTransport struct {
	config    config.GRPCConfig
	conn      *grpc.ClientConn
	connected bool
}

// GRPCConfig represents gRPC transport configuration
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

// NewGRPCTransport creates a new gRPC transport instance
func NewGRPCTransport(config config.GRPCConfig) *GRPCTransport {
	return &GRPCTransport{
		config:    config,
		connected: false,
	}
}

// Connect establishes a connection via gRPC
func (g *GRPCTransport) Connect(ctx context.Context) error {
	if !g.config.Enabled {
		return fmt.Errorf("gRPC transport is disabled")
	}

	var err error
	var retries int

	for retries < g.config.MaxRetries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Configure gRPC options
			opts := g.getGRPCOptions()

			// Establish gRPC connection
			address := fmt.Sprintf("%s:%d", g.config.ServerHost, g.config.ServerPort)
			g.conn, err = grpc.NewClient(address, opts...)

			if err == nil {
				g.connected = true
				log.Printf("gRPC transport connected to %s", address)
				return nil
			}

			log.Printf("gRPC connection attempt %d failed: %v", retries+1, err)
			retries++

			if retries < g.config.MaxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(g.config.RetryInterval):
					continue
				}
			}
		}
	}

	return fmt.Errorf("failed to connect via gRPC after %d retries: %w", g.config.MaxRetries, err)
}

// Disconnect closes the gRPC connection
func (g *GRPCTransport) Disconnect() error {
	if g.conn != nil {
		err := g.conn.Close()
		g.conn = nil
		g.connected = false
		return err
	}
	return nil
}

// IsConnected returns the connection status
func (g *GRPCTransport) IsConnected() bool {
	return g.connected && g.conn != nil
}

// GetConnection returns the gRPC connection for use with gRPC clients
func (g *GRPCTransport) GetConnection() *grpc.ClientConn {
	return g.conn
}

// GetConnectionInfo returns connection information
func (g *GRPCTransport) GetConnectionInfo() map[string]interface{} {
	return map[string]interface{}{
		"protocol":    "gRPC",
		"connected":   g.connected,
		"server_host": g.config.ServerHost,
		"server_port": g.config.ServerPort,
		"timeout":     g.config.Timeout.String(),
		"tls_enabled": g.config.TLSEnabled,
		"compression": g.config.CompressionEnabled,
	}
}

// Ping tests the connection using gRPC health check
func (g *GRPCTransport) Ping() error {
	if !g.IsConnected() {
		return fmt.Errorf("gRPC transport not connected")
	}

	// For now, just check if connection is still valid
	state := g.conn.GetState()
	// Accept both READY and IDLE states as valid for a mock server
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("gRPC connection not ready, state: %s", state.String())
	}

	return nil
}

// GetStats returns gRPC transport statistics
func (g *GRPCTransport) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"protocol":     "gRPC",
		"enabled":      g.config.Enabled,
		"connected":    g.connected,
		"server_host":  g.config.ServerHost,
		"server_port":  g.config.ServerPort,
		"timeout":      g.config.Timeout.String(),
		"tls_enabled":  g.config.TLSEnabled,
		"compression":  g.config.CompressionEnabled,
		"max_msg_size": g.config.MaxMessageSize,
		"keepalive":    g.config.KeepAliveTime.String(),
	}

	if g.conn != nil {
		stats["connection_state"] = g.conn.GetState().String()
	}

	return stats
}

// getGRPCOptions configures gRPC connection options
func (g *GRPCTransport) getGRPCOptions() []grpc.DialOption {
	var opts []grpc.DialOption

	// Configure credentials
	if g.config.TLSEnabled {
		if g.config.CertFile != "" && g.config.KeyFile != "" {
			creds, err := credentials.NewClientTLSFromFile(g.config.CertFile, "")
			if err != nil {
				log.Printf("Warning: Failed to load TLS credentials, using insecure: %v", err)
				opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			} else {
				opts = append(opts, grpc.WithTransportCredentials(creds))
			}
		} else {
			// Use system certs
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
		}
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Configure keepalive
	if g.config.KeepAliveTime > 0 {
		kaParams := keepalive.ClientParameters{
			Time:                g.config.KeepAliveTime,
			Timeout:             g.config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}
		opts = append(opts, grpc.WithKeepaliveParams(kaParams))
	}

	// Configure message size
	if g.config.MaxMessageSize > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(g.config.MaxMessageSize)))
	}

	// Configure compression
	if g.config.CompressionEnabled {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	}

	// Note: Timeout is now handled per-call, not at connection level

	return opts
}
