package transport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/test/helpers"
)

func TestSSHTransport(t *testing.T) {
	// Create unique test directory for this test
	testDir := fmt.Sprintf("./test-ssh-keys-%d", time.Now().UnixNano())

	// Create test SSH config
	sshConfig := config.SSHConfig{
		Enabled: true,
		KeyStorage: config.SSHKeyStorage{
			KeyDir:        testDir,
			PrivateKeyExt: ".pem",
			PublicKeyExt:  ".pub",
			KeySize:       2048,
			KeyType:       "rsa",
		},
		Connection: config.SSHConnection{
			Host:        "localhost",
			Port:        22,
			Username:    "testuser",
			Timeout:     30 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxRetries:  3,
			RetryDelay:  5 * time.Second,
			Compression: false,
		},
		TerminalInterface: config.TerminalInterface{
			Enabled:     false, // Disable for testing
			Port:        2222,
			Host:        "localhost",
			Interactive: false,
			LogLevel:    "info",
		},
	}

	// Clean up test directory before and after
	os.RemoveAll(testDir)
	defer func() {
		os.RemoveAll(testDir)
	}()

	t.Run("Create SSH transport", func(t *testing.T) {
		transport := NewSSHTransport(sshConfig)
		helpers.AssertNotNil(t, transport)
		helpers.AssertNotNil(t, transport.keyManager)
		helpers.AssertBoolEqual(t, false, transport.IsHealthy())
	})

	t.Run("Initialize key manager", func(t *testing.T) {
		transport := NewSSHTransport(sshConfig)

		// Create test directory
		err := os.MkdirAll(sshConfig.KeyStorage.KeyDir, 0700)
		helpers.AssertNoError(t, err)

		// Initialize key manager
		err = transport.keyManager.Initialize()
		helpers.AssertNoError(t, err)
	})

	t.Run("Generate SSH key", func(t *testing.T) {
		transport := NewSSHTransport(sshConfig)

		// Create test directory
		err := os.MkdirAll(sshConfig.KeyStorage.KeyDir, 0700)
		helpers.AssertNoError(t, err)

		// Initialize key manager
		err = transport.keyManager.Initialize()
		helpers.AssertNoError(t, err)

		// Generate a test key
		key, err := transport.keyManager.GenerateKey("test-key", "test@mercury-relay")
		helpers.AssertNoError(t, err)
		helpers.AssertNotNil(t, key)
		helpers.AssertStringEqual(t, "test-key", key.Name)
		helpers.AssertStringEqual(t, "test@mercury-relay", key.Comment)
		helpers.AssertNotNil(t, key.PrivateKey)
		helpers.AssertNotNil(t, key.PublicKey)
	})

	t.Run("List SSH keys", func(t *testing.T) {
		// Create a unique directory for this specific test
		listTestDir := fmt.Sprintf("./test-ssh-keys-list-%d", time.Now().UnixNano())
		listSSHConfig := sshConfig
		listSSHConfig.KeyStorage.KeyDir = listTestDir

		// Create a fresh transport for this test
		transport := NewSSHTransport(listSSHConfig)

		// Clean up after test
		defer os.RemoveAll(listTestDir)

		// Create test directory
		err := os.MkdirAll(listSSHConfig.KeyStorage.KeyDir, 0700)
		helpers.AssertNoError(t, err)

		// Initialize key manager
		err = transport.keyManager.Initialize()
		helpers.AssertNoError(t, err)

		// Generate test keys
		_, err = transport.keyManager.GenerateKey("key1", "key1@mercury-relay")
		helpers.AssertNoError(t, err)

		_, err = transport.keyManager.GenerateKey("key2", "key2@mercury-relay")
		helpers.AssertNoError(t, err)

		// List keys
		keys := transport.keyManager.ListKeys()
		t.Logf("Found %d keys: %v", len(keys), keys)
		helpers.AssertIntEqual(t, 2, len(keys))
	})

	t.Run("Remove SSH key", func(t *testing.T) {
		// Create a unique directory for this specific test
		removeTestDir := fmt.Sprintf("./test-ssh-keys-remove-%d", time.Now().UnixNano())
		removeSSHConfig := sshConfig
		removeSSHConfig.KeyStorage.KeyDir = removeTestDir

		// Create a fresh transport for this test
		transport := NewSSHTransport(removeSSHConfig)

		// Clean up after test
		defer os.RemoveAll(removeTestDir)

		// Create test directory
		err := os.MkdirAll(removeSSHConfig.KeyStorage.KeyDir, 0700)
		helpers.AssertNoError(t, err)

		// Initialize key manager
		err = transport.keyManager.Initialize()
		helpers.AssertNoError(t, err)

		// Generate a test key
		_, err = transport.keyManager.GenerateKey("test-key", "test@mercury-relay")
		helpers.AssertNoError(t, err)

		// Verify key exists
		keys := transport.keyManager.ListKeys()
		helpers.AssertIntEqual(t, 1, len(keys))

		// Remove key
		err = transport.keyManager.RemoveKey("test-key")
		helpers.AssertNoError(t, err)

		// Verify key is removed
		keys = transport.keyManager.ListKeys()
		helpers.AssertIntEqual(t, 0, len(keys))
	})

	t.Run("Get authentication methods", func(t *testing.T) {
		// Create a unique directory for this specific test
		authTestDir := fmt.Sprintf("./test-ssh-keys-auth-%d", time.Now().UnixNano())
		authSSHConfig := sshConfig
		authSSHConfig.KeyStorage.KeyDir = authTestDir

		// Create a fresh transport for this test
		transport := NewSSHTransport(authSSHConfig)

		// Clean up after test
		defer os.RemoveAll(authTestDir)

		// Create test directory
		err := os.MkdirAll(authSSHConfig.KeyStorage.KeyDir, 0700)
		helpers.AssertNoError(t, err)

		// Initialize key manager
		err = transport.keyManager.Initialize()
		helpers.AssertNoError(t, err)

		// Generate a test key
		_, err = transport.keyManager.GenerateKey("test-key", "test@mercury-relay")
		helpers.AssertNoError(t, err)

		// Get auth methods
		authMethods := transport.keyManager.GetAuthMethods()
		helpers.AssertIntEqual(t, 1, len(authMethods))
	})

	t.Run("Get address", func(t *testing.T) {
		transport := NewSSHTransport(sshConfig)
		address := transport.GetAddress()
		helpers.AssertStringEqual(t, "localhost:22", address)
	})

	t.Run("Start transport without SSH server", func(t *testing.T) {
		transport := NewSSHTransport(sshConfig)

		// Create test directory
		err := os.MkdirAll(sshConfig.KeyStorage.KeyDir, 0700)
		helpers.AssertNoError(t, err)

		// This should fail because there's no SSH server running
		err = transport.Start(context.Background())
		// We expect this to fail since there's no SSH server
		helpers.AssertError(t, err)
		helpers.AssertBoolEqual(t, false, transport.IsHealthy())
	})

	t.Run("Stop transport", func(t *testing.T) {
		transport := NewSSHTransport(sshConfig)

		err := transport.Stop()
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, false, transport.IsHealthy())
	})
}

func TestSSHKeyManager(t *testing.T) {
	keyDir := "./test-ssh-keys"
	os.RemoveAll(keyDir) // Clean up before test
	defer os.RemoveAll(keyDir)

	keyStorage := config.SSHKeyStorage{
		KeyDir:        keyDir,
		PrivateKeyExt: ".pem",
		PublicKeyExt:  ".pub",
		KeySize:       2048,
		KeyType:       "rsa",
	}

	t.Run("Initialize key manager", func(t *testing.T) {
		km := &SSHKeyManager{
			config: keyStorage,
			keys:   make(map[string]*SSHKey),
		}

		err := km.Initialize()
		helpers.AssertNoError(t, err)

		// Check that directory was created
		_, err = os.Stat(keyDir)
		helpers.AssertNoError(t, err)
	})

	t.Run("Generate and save key", func(t *testing.T) {
		km := &SSHKeyManager{
			config: keyStorage,
			keys:   make(map[string]*SSHKey),
		}

		err := km.Initialize()
		helpers.AssertNoError(t, err)

		key, err := km.GenerateKey("test-key", "test@mercury-relay")
		helpers.AssertNoError(t, err)
		helpers.AssertNotNil(t, key)
		helpers.AssertStringEqual(t, "test-key", key.Name)

		// Check that files were created
		privateKeyPath := filepath.Join(keyDir, "test-key.pem")
		publicKeyPath := filepath.Join(keyDir, "test-key.pub")

		_, err = os.Stat(privateKeyPath)
		helpers.AssertNoError(t, err)

		_, err = os.Stat(publicKeyPath)
		helpers.AssertNoError(t, err)
	})

	t.Run("Load existing key", func(t *testing.T) {
		km := &SSHKeyManager{
			config: keyStorage,
			keys:   make(map[string]*SSHKey),
		}

		err := km.Initialize()
		helpers.AssertNoError(t, err)

		// Generate a key
		_, err = km.GenerateKey("existing-key", "existing@mercury-relay")
		helpers.AssertNoError(t, err)

		// Create new key manager and load existing keys
		km2 := &SSHKeyManager{
			config: keyStorage,
			keys:   make(map[string]*SSHKey),
		}

		err = km2.Initialize()
		helpers.AssertNoError(t, err)

		// Check that key was loaded
		key, exists := km2.GetKey("existing-key")
		helpers.AssertBoolEqual(t, true, exists)
		helpers.AssertNotNil(t, key)
		helpers.AssertStringEqual(t, "existing-key", key.Name)
	})

	t.Run("Remove key", func(t *testing.T) {
		km := &SSHKeyManager{
			config: keyStorage,
			keys:   make(map[string]*SSHKey),
		}

		err := km.Initialize()
		helpers.AssertNoError(t, err)

		// Generate a key
		_, err = km.GenerateKey("remove-key", "remove@mercury-relay")
		helpers.AssertNoError(t, err)

		// Remove key
		err = km.RemoveKey("remove-key")
		helpers.AssertNoError(t, err)

		// Check that key is gone
		_, exists := km.GetKey("remove-key")
		helpers.AssertBoolEqual(t, false, exists)

		// Check that files are gone
		privateKeyPath := filepath.Join(keyDir, "remove-key.pem")
		publicKeyPath := filepath.Join(keyDir, "remove-key.pub")

		_, err = os.Stat(privateKeyPath)
		helpers.AssertError(t, err)

		_, err = os.Stat(publicKeyPath)
		helpers.AssertError(t, err)
	})
}

func TestWebSocketSSHTunnel(t *testing.T) {
	// Create test SSH config
	sshConfig := config.SSHConfig{
		Enabled: true,
		KeyStorage: config.SSHKeyStorage{
			KeyDir:        "./test-ssh-keys",
			PrivateKeyExt: ".pem",
			PublicKeyExt:  ".pub",
			KeySize:       2048,
			KeyType:       "rsa",
		},
		Connection: config.SSHConnection{
			Host:        "localhost",
			Port:        22,
			Username:    "testuser",
			Timeout:     30 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxRetries:  3,
			RetryDelay:  5 * time.Second,
			Compression: false,
		},
		TerminalInterface: config.TerminalInterface{
			Enabled:     false,
			Port:        2222,
			Host:        "localhost",
			Interactive: false,
			LogLevel:    "info",
		},
	}

	// Clean up test directory
	defer func() {
		os.RemoveAll("./test-ssh-keys")
	}()

	t.Run("Create WebSocket SSH tunnel", func(t *testing.T) {
		sshTransport := NewSSHTransport(sshConfig)
		tunnel := NewWebSocketSSHTunnel(sshTransport)

		helpers.AssertNotNil(t, tunnel)
		helpers.AssertNotNil(t, tunnel.sshTransport)
		helpers.AssertNotNil(t, tunnel.connections)
	})

	t.Run("Get connection stats", func(t *testing.T) {
		sshTransport := NewSSHTransport(sshConfig)
		tunnel := NewWebSocketSSHTunnel(sshTransport)

		stats := tunnel.GetConnectionStats()
		helpers.AssertNotNil(t, stats)
		helpers.AssertIntEqual(t, 0, stats["total_connections"].(int))
		helpers.AssertIntEqual(t, 0, stats["active_connections"].(int))
	})

	t.Run("Close all connections", func(t *testing.T) {
		sshTransport := NewSSHTransport(sshConfig)
		tunnel := NewWebSocketSSHTunnel(sshTransport)

		// This should not panic even with no connections
		tunnel.CloseAllConnections()

		stats := tunnel.GetConnectionStats()
		helpers.AssertIntEqual(t, 0, stats["total_connections"].(int))
	})
}
