package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mercury-relay/internal/config"

	"golang.org/x/crypto/ssh"
)

type SSHTransport struct {
	config     config.SSHConfig
	client     *ssh.Client
	keyManager *SSHKeyManager
	mu         sync.RWMutex
	healthy    bool
}

type SSHKeyManager struct {
	config config.SSHKeyStorage
	keys   map[string]*SSHKey
	mu     sync.RWMutex
}

// NewSSHKeyManager creates a new SSH key manager
func NewSSHKeyManager(config config.SSHKeyStorage) *SSHKeyManager {
	return &SSHKeyManager{
		config: config,
		keys:   make(map[string]*SSHKey),
	}
}

type SSHKey struct {
	Name       string
	PrivateKey *rsa.PrivateKey
	PublicKey  ssh.PublicKey
	CreatedAt  time.Time
	Comment    string
	OwnerNpub  string // Nostr pubkey of the owner
}

// SSHKeyInfo represents public information about an SSH key for API responses
type SSHKeyInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Comment   string `json:"comment"`
	OwnerNpub string `json:"owner_npub"`
}

type SSHConnection struct {
	Client     *ssh.Client
	Session    *ssh.Session
	LocalAddr  string
	RemoteAddr string
	CreatedAt  time.Time
	LastUsed   time.Time
}

func NewSSHTransport(config config.SSHConfig) *SSHTransport {
	keyManager := &SSHKeyManager{
		config: config.KeyStorage,
		keys:   make(map[string]*SSHKey),
	}

	return &SSHTransport{
		config:     config,
		keyManager: keyManager,
		healthy:    false,
	}
}

func (s *SSHTransport) Start(ctx context.Context) error {
	log.Println("Starting SSH transport...")

	// Initialize key manager
	if err := s.keyManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	// Start terminal interface if enabled
	if s.config.TerminalInterface.Enabled {
		go s.startTerminalInterface(ctx)
	}

	// Test connection
	if err := s.testConnection(ctx); err != nil {
		log.Printf("SSH connection test failed: %v", err)
		s.healthy = false
		return err
	}

	s.healthy = true
	log.Println("SSH transport started successfully")
	return nil
}

func (s *SSHTransport) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		s.client.Close()
		s.client = nil
	}

	s.healthy = false
	log.Println("SSH transport stopped")
	return nil
}

func (s *SSHTransport) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy
}

func (s *SSHTransport) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.config.Connection.Host, s.config.Connection.Port)
}

func (s *SSHTransport) testConnection(ctx context.Context) error {
	// Create SSH client config
	config := &ssh.ClientConfig{
		User:            s.config.Connection.Username,
		Timeout:         s.config.Connection.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For testing - should be configured properly in production
	}

	// Add authentication methods
	authMethods := s.keyManager.GetAuthMethods()
	config.Auth = authMethods

	// Connect to SSH server
	client, err := ssh.Dial("tcp", s.GetAddress(), config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	// Test connection with a simple command
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Run a simple test command
	output, err := session.CombinedOutput("echo 'SSH connection test'")
	if err != nil {
		client.Close()
		return fmt.Errorf("SSH test command failed: %w", err)
	}

	log.Printf("SSH connection test successful: %s", string(output))
	client.Close()
	return nil
}

func (s *SSHTransport) startTerminalInterface(ctx context.Context) {
	addr := fmt.Sprintf("%s:%d", s.config.TerminalInterface.Host, s.config.TerminalInterface.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to start SSH terminal interface: %v", err)
		return
	}
	defer listener.Close()

	log.Printf("SSH terminal interface listening on %s", addr)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept terminal connection: %v", err)
				continue
			}

			go s.handleTerminalConnection(conn)
		}
	}
}

func (s *SSHTransport) handleTerminalConnection(conn net.Conn) {
	defer conn.Close()

	// Simple terminal interface for key management
	conn.Write([]byte("Mercury Relay SSH Key Manager\n"))
	conn.Write([]byte("Commands: list, add, remove, help, quit\n"))

	buffer := make([]byte, 1024)
	for {
		conn.Write([]byte("ssh> "))
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Terminal connection error: %v", err)
			}
			return
		}

		command := string(buffer[:n])
		command = command[:len(command)-1] // Remove newline

		switch command {
		case "list":
			s.handleListKeys(conn)
		case "add":
			s.handleAddKey(conn)
		case "remove":
			s.handleRemoveKey(conn)
		case "help":
			s.handleHelp(conn)
		case "quit":
			conn.Write([]byte("Goodbye!\n"))
			return
		default:
			conn.Write([]byte("Unknown command. Type 'help' for available commands.\n"))
		}
	}
}

func (s *SSHTransport) handleListKeys(conn net.Conn) {
	conn.Write([]byte("❌ Authentication required for SSH key management.\n"))
	conn.Write([]byte("Please use the Nostr-authenticated SSH key manager:\n"))
	conn.Write([]byte("  export MERCURY_PRIVATE_KEY=\"nsec1your-private-key\"\n"))
	conn.Write([]byte("  ./nostr-ssh-manager\n"))
}

func (s *SSHTransport) handleAddKey(conn net.Conn) {
	conn.Write([]byte("❌ Authentication required for SSH key management.\n"))
	conn.Write([]byte("Please use the Nostr-authenticated SSH key manager:\n"))
	conn.Write([]byte("  export MERCURY_PRIVATE_KEY=\"nsec1your-private-key\"\n"))
	conn.Write([]byte("  ./nostr-ssh-manager\n"))
}

func (s *SSHTransport) handleRemoveKey(conn net.Conn) {
	conn.Write([]byte("❌ Authentication required for SSH key management.\n"))
	conn.Write([]byte("Please use the Nostr-authenticated SSH key manager:\n"))
	conn.Write([]byte("  export MERCURY_PRIVATE_KEY=\"nsec1your-private-key\"\n"))
	conn.Write([]byte("  ./nostr-ssh-manager\n"))
}

func (s *SSHTransport) handleHelp(conn net.Conn) {
	conn.Write([]byte("🔐 SSH Key Management - Nostr Authentication Required\n"))
	conn.Write([]byte("==================================================\n"))
	conn.Write([]byte("❌ This terminal interface requires Nostr authentication.\n"))
	conn.Write([]byte("Please use the Nostr-authenticated SSH key manager:\n"))
	conn.Write([]byte("\n"))
	conn.Write([]byte("1. Set your Nostr private key:\n"))
	conn.Write([]byte("   export MERCURY_PRIVATE_KEY=\"nsec1your-private-key\"\n"))
	conn.Write([]byte("\n"))
	conn.Write([]byte("2. Run the authenticated SSH key manager:\n"))
	conn.Write([]byte("   ./nostr-ssh-manager\n"))
	conn.Write([]byte("\n"))
	conn.Write([]byte("Available commands in authenticated mode:\n"))
	conn.Write([]byte("  list   - List your SSH keys\n"))
	conn.Write([]byte("  add    - Add a new SSH key\n"))
	conn.Write([]byte("  remove - Remove an SSH key\n"))
	conn.Write([]byte("  help   - Show this help message\n"))
	conn.Write([]byte("  quit   - Exit the terminal\n"))
}

// SSHKeyManager methods

func (km *SSHKeyManager) Initialize() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Create key directory if it doesn't exist
	if err := os.MkdirAll(km.config.KeyDir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Load existing keys
	return km.loadExistingKeys()
}

func (km *SSHKeyManager) loadExistingKeys() error {
	// Scan key directory for existing keys
	entries, err := os.ReadDir(km.config.KeyDir)
	if err != nil {
		return fmt.Errorf("failed to read key directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if filepath.Ext(filename) == km.config.PrivateKeyExt {
			keyName := filename[:len(filename)-len(km.config.PrivateKeyExt)]
			if err := km.loadKey(keyName); err != nil {
				log.Printf("Failed to load key %s: %v", keyName, err)
			}
		}
	}

	return nil
}

func (km *SSHKeyManager) loadKey(name string) error {
	privateKeyPath := filepath.Join(km.config.KeyDir, name+km.config.PrivateKeyExt)

	// Read private key
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}

	// Create SSH key object
	sshKey := &SSHKey{
		Name:       name,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		CreatedAt:  time.Now(), // In practice, you'd get this from file metadata
		Comment:    fmt.Sprintf("%s@mercury-relay", name),
	}

	km.keys[name] = sshKey
	return nil
}

func (km *SSHKeyManager) GenerateKey(name, comment string) (*SSHKey, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, km.config.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate SSH public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	// Create SSH key object
	sshKey := &SSHKey{
		Name:       name,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		CreatedAt:  time.Now(),
		Comment:    comment,
	}

	// Save keys to disk
	if err := km.saveKey(sshKey); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	km.keys[name] = sshKey
	return sshKey, nil
}

func (km *SSHKeyManager) saveKey(key *SSHKey) error {
	// Save private key
	privateKeyPath := filepath.Join(km.config.KeyDir, key.Name+km.config.PrivateKeyExt)
	privateKeyData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key.PrivateKey),
	})

	if err := os.WriteFile(privateKeyPath, privateKeyData, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key
	publicKeyPath := filepath.Join(km.config.KeyDir, key.Name+km.config.PublicKeyExt)
	publicKeyData := ssh.MarshalAuthorizedKey(key.PublicKey)

	if err := os.WriteFile(publicKeyPath, publicKeyData, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

func (km *SSHKeyManager) GetKey(name string) (*SSHKey, bool) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	key, exists := km.keys[name]
	return key, exists
}

func (km *SSHKeyManager) ListKeys() []SSHKeyInfo {
	km.mu.RLock()
	defer km.mu.RUnlock()

	keys := make([]SSHKeyInfo, 0, len(km.keys))
	for _, key := range km.keys {
		keys = append(keys, SSHKeyInfo{
			Name:      key.Name,
			Type:      "rsa", // Default type, could be determined from key
			CreatedAt: key.CreatedAt.Format("2006-01-02 15:04:05"),
			Comment:   key.Comment,
			OwnerNpub: key.OwnerNpub,
		})
	}
	return keys
}

// ListKeysByOwner returns SSH keys owned by a specific Nostr pubkey
func (km *SSHKeyManager) ListKeysByOwner(ownerNpub string) []SSHKeyInfo {
	km.mu.RLock()
	defer km.mu.RUnlock()

	keys := make([]SSHKeyInfo, 0)
	for _, key := range km.keys {
		if key.OwnerNpub == ownerNpub {
			keys = append(keys, SSHKeyInfo{
				Name:      key.Name,
				Type:      "rsa", // Default type, could be determined from key
				CreatedAt: key.CreatedAt.Format("2006-01-02 15:04:05"),
				Comment:   key.Comment,
				OwnerNpub: key.OwnerNpub,
			})
		}
	}
	return keys
}

// IsOwner checks if a Nostr pubkey owns a specific SSH key
func (km *SSHKeyManager) IsOwner(keyName, ownerNpub string) bool {
	km.mu.RLock()
	defer km.mu.RUnlock()

	key, exists := km.keys[keyName]
	if !exists {
		return false
	}
	return key.OwnerNpub == ownerNpub
}

func (km *SSHKeyManager) RemoveKey(name string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	_, exists := km.keys[name]
	if !exists {
		return fmt.Errorf("key %s not found", name)
	}

	// Remove files
	privateKeyPath := filepath.Join(km.config.KeyDir, name+km.config.PrivateKeyExt)
	publicKeyPath := filepath.Join(km.config.KeyDir, name+km.config.PublicKeyExt)

	if err := os.Remove(privateKeyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove private key: %w", err)
	}

	if err := os.Remove(publicKeyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove public key: %w", err)
	}

	// Remove from memory
	delete(km.keys, name)
	return nil
}

func (km *SSHKeyManager) SaveKey(name string, privateKeyData, publicKeyData []byte, ownerNpub string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Parse private key
	privateKey, err := parsePrivateKey(privateKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Parse or generate public key
	var publicKey ssh.PublicKey
	if len(publicKeyData) > 0 {
		publicKey, err = parsePublicKey(publicKeyData)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
	} else {
		// Generate public key from private key
		publicKey, err = createPublicKey(privateKey)
		if err != nil {
			return fmt.Errorf("failed to create public key: %w", err)
		}
	}

	// Create SSH key object
	sshKey := &SSHKey{
		Name:       name,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		CreatedAt:  time.Now(),
		Comment:    fmt.Sprintf("Uploaded key %s", name),
		OwnerNpub:  ownerNpub,
	}

	// Save private key to file
	privateKeyPath := filepath.Join(km.config.KeyDir, name+km.config.PrivateKeyExt)
	if err := os.WriteFile(privateKeyPath, privateKeyData, 0600); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	// Save public key to file
	publicKeyPath := filepath.Join(km.config.KeyDir, name+km.config.PublicKeyExt)
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	if err := os.WriteFile(publicKeyPath, publicKeyBytes, 0644); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	// Store in memory
	km.keys[name] = sshKey
	return nil
}

func (km *SSHKeyManager) GetKeyDir() string {
	return km.config.KeyDir
}

func (km *SSHKeyManager) GetAuthMethods() []ssh.AuthMethod {
	km.mu.RLock()
	defer km.mu.RUnlock()

	var authMethods []ssh.AuthMethod

	for _, key := range km.keys {
		signer, err := ssh.NewSignerFromKey(key.PrivateKey)
		if err != nil {
			log.Printf("Failed to create signer for key %s: %v", key.Name, err)
			continue
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	return authMethods
}

// CreateSSHConnection creates a new SSH connection for WebSocket tunneling
func (s *SSHTransport) CreateSSHConnection(ctx context.Context) (*SSHConnection, error) {
	// Create SSH client config
	config := &ssh.ClientConfig{
		User:            s.config.Connection.Username,
		Timeout:         s.config.Connection.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Should be configured properly in production
	}

	// Add authentication methods
	config.Auth = s.keyManager.GetAuthMethods()

	// Connect to SSH server
	client, err := ssh.Dial("tcp", s.GetAddress(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	// Create session
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Get connection info
	localAddr := client.LocalAddr().String()
	remoteAddr := client.RemoteAddr().String()

	connection := &SSHConnection{
		Client:     client,
		Session:    session,
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
	}

	return connection, nil
}

// CloseSSHConnection closes an SSH connection
func (s *SSHTransport) CloseSSHConnection(conn *SSHConnection) error {
	if conn.Session != nil {
		conn.Session.Close()
	}
	if conn.Client != nil {
		return conn.Client.Close()
	}
	return nil
}

// parsePrivateKey parses a private key from PEM data
func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		genericKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		rsaKey, ok := genericKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA private key")
		}
		return rsaKey, nil
	}

	return key, nil
}

// parsePublicKey parses a public key from SSH format
func parsePublicKey(data []byte) (ssh.PublicKey, error) {
	return ssh.ParsePublicKey(data)
}

// createPublicKey creates an SSH public key from an RSA private key
func createPublicKey(privateKey *rsa.PrivateKey) (ssh.PublicKey, error) {
	return ssh.NewPublicKey(&privateKey.PublicKey)
}
