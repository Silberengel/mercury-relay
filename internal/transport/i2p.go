package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"mercury-relay/internal/config"
)

type I2PTransport struct {
	config  config.I2PConfig
	address string
	healthy bool
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewI2PTransport(config config.I2PConfig) *I2PTransport {
	ctx, cancel := context.WithCancel(context.Background())
	return &I2PTransport{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (i *I2PTransport) Start(ctx context.Context) error {
	// Connect to I2P SAM bridge
	samAddr := fmt.Sprintf("%s:%d", i.config.SAMHost, i.config.SAMPort)
	conn, err := net.Dial("tcp", samAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to I2P SAM bridge: %w", err)
	}
	defer conn.Close()

	// Create tunnel
	if err := i.createTunnel(conn); err != nil {
		return fmt.Errorf("failed to create I2P tunnel: %w", err)
	}

	// Start health check
	go i.healthCheck()

	return nil
}

func (i *I2PTransport) createTunnel(conn net.Conn) error {
	// Send SAM session create command
	cmd := fmt.Sprintf("SESSION CREATE STYLE=STREAM ID=%s DESTINATION=TRANSIENT\n", i.config.TunnelName)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("failed to send SESSION CREATE: %w", err)
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read SESSION CREATE response: %w", err)
	}

	response := string(buffer[:n])
	if !strings.Contains(response, "SESSION STATUS RESULT=OK") {
		return fmt.Errorf("SESSION CREATE failed: %s", response)
	}

	// Send tunnel create command
	cmd = fmt.Sprintf("TUNNEL CREATE STYLE=STREAM ID=%s DESTINATION=TRANSIENT\n", i.config.TunnelName)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("failed to send TUNNEL CREATE: %w", err)
	}

	// Read response
	n, err = conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read TUNNEL CREATE response: %w", err)
	}

	response = string(buffer[:n])
	if !strings.Contains(response, "TUNNEL STATUS RESULT=OK") {
		return fmt.Errorf("TUNNEL CREATE failed: %s", response)
	}

	// Get tunnel address
	cmd = fmt.Sprintf("TUNNEL GET DESTINATION ID=%s\n", i.config.TunnelName)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("failed to send TUNNEL GET DESTINATION: %w", err)
	}

	// Read response
	n, err = conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read TUNNEL GET DESTINATION response: %w", err)
	}

	response = string(buffer[:n])
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "TUNNEL DESTINATION") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				i.address = parts[2]
				break
			}
		}
	}

	if i.address == "" {
		return fmt.Errorf("failed to get tunnel address")
	}

	log.Printf("I2P tunnel created: %s", i.address)
	return nil
}

func (i *I2PTransport) healthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-i.ctx.Done():
			return
		case <-ticker.C:
			// Check if I2P router is accessible
			samAddr := fmt.Sprintf("%s:%d", i.config.SAMHost, i.config.SAMPort)
			conn, err := net.DialTimeout("tcp", samAddr, 5*time.Second)
			if err != nil {
				i.healthy = false
				log.Printf("I2P health check failed: %v", err)
			} else {
				i.healthy = true
				conn.Close()
			}
		}
	}
}

func (i *I2PTransport) Stop() error {
	i.cancel()
	return nil
}

func (i *I2PTransport) GetAddress() string {
	return i.address
}

func (i *I2PTransport) IsHealthy() bool {
	return i.healthy
}
