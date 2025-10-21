package transport

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mercury-relay/internal/config"
)

type TorTransport struct {
	config  config.TorConfig
	process *exec.Cmd
	address string
	healthy bool
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewTorTransport(config config.TorConfig) *TorTransport {
	ctx, cancel := context.WithCancel(context.Background())
	return &TorTransport{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (t *TorTransport) Start(ctx context.Context) error {
	// Create Tor data directory if it doesn't exist
	if err := os.MkdirAll(t.config.DataDir, 0700); err != nil {
		return fmt.Errorf("failed to create Tor data directory: %w", err)
	}

	// Create hidden service directory
	hiddenServiceDir := filepath.Join(t.config.DataDir, "mercury_relay")
	if err := os.MkdirAll(hiddenServiceDir, 0700); err != nil {
		return fmt.Errorf("failed to create hidden service directory: %w", err)
	}

	// Create Tor configuration
	torrcPath := filepath.Join(t.config.DataDir, "torrc")
	if err := t.createTorrc(torrcPath); err != nil {
		return fmt.Errorf("failed to create torrc: %w", err)
	}

	// Start Tor process
	t.process = exec.CommandContext(ctx, "tor", "-f", torrcPath)
	t.process.Stdout = os.Stdout
	t.process.Stderr = os.Stderr

	if err := t.process.Start(); err != nil {
		return fmt.Errorf("failed to start Tor: %w", err)
	}

	// Wait for Tor to start and get the hidden service address
	go t.waitForHiddenService()

	return nil
}

func (t *TorTransport) createTorrc(path string) error {
	config := fmt.Sprintf(`
DataDirectory %s
ControlPort %d
SocksPort %d
HiddenServiceDir %s
HiddenServicePort %d 127.0.0.1:8080
`,
		t.config.DataDir,
		t.config.ControlPort,
		t.config.SocksPort,
		filepath.Join(t.config.DataDir, "mercury_relay"),
		t.config.HiddenServicePort,
	)

	return os.WriteFile(path, []byte(config), 0600)
}

func (t *TorTransport) waitForHiddenService() {
	// Wait for hidden service to be ready
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			// Check if hostname file exists
			hostnamePath := filepath.Join(t.config.DataDir, "mercury_relay", "hostname")
			if data, err := os.ReadFile(hostnamePath); err == nil {
				t.address = strings.TrimSpace(string(data))
				t.healthy = true
				log.Printf("Tor hidden service ready: %s", t.address)
				return
			}
			time.Sleep(time.Second)
		}
	}
}

func (t *TorTransport) Stop() error {
	t.cancel()
	if t.process != nil {
		return t.process.Process.Kill()
	}
	return nil
}

func (t *TorTransport) GetAddress() string {
	return t.address
}

func (t *TorTransport) IsHealthy() bool {
	return t.healthy
}

func (t *TorTransport) GetSocksProxy() string {
	return fmt.Sprintf("127.0.0.1:%d", t.config.SocksPort)
}
