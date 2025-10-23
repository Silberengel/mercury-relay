package transport

import (
	"context"
	"fmt"
	"log"

	"mercury-relay/internal/config"
)

type Manager struct {
	torConfig config.TorConfig
	i2pConfig config.I2PConfig
	sshConfig config.SSHConfig
	tor       *TorTransport
	i2p       *I2PTransport
	ssh       *SSHTransport
}

func NewManager(torConfig config.TorConfig, i2pConfig config.I2PConfig, sshConfig config.SSHConfig) *Manager {
	return &Manager{
		torConfig: torConfig,
		i2pConfig: i2pConfig,
		sshConfig: sshConfig,
	}
}

func (m *Manager) Start(ctx context.Context) error {
	var errors []error

	// Start Tor if enabled
	if m.torConfig.Enabled {
		m.tor = NewTorTransport(m.torConfig)
		if err := m.tor.Start(ctx); err != nil {
			log.Printf("Failed to start Tor transport: %v", err)
			errors = append(errors, err)
		} else {
			log.Println("Tor transport started successfully")
		}
	}

	// Start I2P if enabled
	if m.i2pConfig.Enabled {
		m.i2p = NewI2PTransport(m.i2pConfig)
		if err := m.i2p.Start(ctx); err != nil {
			log.Printf("Failed to start I2P transport: %v", err)
			errors = append(errors, err)
		} else {
			log.Println("I2P transport started successfully")
		}
	}

	// Start SSH if enabled
	if m.sshConfig.Enabled {
		m.ssh = NewSSHTransport(m.sshConfig)
		if err := m.ssh.Start(ctx); err != nil {
			log.Printf("Failed to start SSH transport: %v", err)
			errors = append(errors, err)
		} else {
			log.Println("SSH transport started successfully")
		}
	}

	// Return error if all transports failed
	if len(errors) > 0 && m.tor == nil && m.i2p == nil && m.ssh == nil {
		return fmt.Errorf("all transports failed to start: %v", errors)
	}

	return nil
}

func (m *Manager) Stop() error {
	var errors []error

	if m.tor != nil {
		if err := m.tor.Stop(); err != nil {
			errors = append(errors, err)
		}
	}

	if m.i2p != nil {
		if err := m.i2p.Stop(); err != nil {
			errors = append(errors, err)
		}
	}

	if m.ssh != nil {
		if err := m.ssh.Stop(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("transport stop errors: %v", errors)
	}

	return nil
}

func (m *Manager) GetTorAddress() string {
	if m.tor != nil {
		return m.tor.GetAddress()
	}
	return ""
}

func (m *Manager) GetI2PAddress() string {
	if m.i2p != nil {
		return m.i2p.GetAddress()
	}
	return ""
}

func (m *Manager) IsTorHealthy() bool {
	if m.tor != nil {
		return m.tor.IsHealthy()
	}
	return false
}

func (m *Manager) IsI2PHealthy() bool {
	if m.i2p != nil {
		return m.i2p.IsHealthy()
	}
	return false
}

func (m *Manager) GetSSHAddress() string {
	if m.ssh != nil {
		return m.ssh.GetAddress()
	}
	return ""
}

func (m *Manager) IsSSHHealthy() bool {
	if m.ssh != nil {
		return m.ssh.IsHealthy()
	}
	return false
}

func (m *Manager) GetSSHTransport() *SSHTransport {
	return m.ssh
}
