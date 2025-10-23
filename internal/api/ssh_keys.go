package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"mercury-relay/internal/auth"
	"mercury-relay/internal/config"
	"mercury-relay/internal/transport"

	"github.com/nbd-wtf/go-nostr"
)

// SSHKeyManager handles SSH key operations via REST API
type SSHKeyManager struct {
	keyManager *transport.SSHKeyManager
	config     config.SSHConfig
	nostrAuth  *auth.NostrAuthenticator
}

// NewSSHKeyManager creates a new SSH key manager for REST API
func NewSSHKeyManager(sshConfig config.SSHConfig, relayURL string) *SSHKeyManager {
	keyManager := transport.NewSSHKeyManager(sshConfig.KeyStorage)
	nostrAuth := auth.NewNostrAuthenticator(relayURL, sshConfig.Authentication.AuthorizedPubkeys)
	return &SSHKeyManager{
		keyManager: keyManager,
		config:     sshConfig,
		nostrAuth:  nostrAuth,
	}
}

// SSHKeyRequest represents a request to upload an SSH key
type SSHKeyRequest struct {
	Name        string `json:"name"`
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key,omitempty"`
	KeyType     string `json:"key_type,omitempty"`
	Description string `json:"description,omitempty"`
}

// SSHKeyResponse represents the response for SSH key operations
type SSHKeyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	KeyName string `json:"key_name,omitempty"`
	KeyPath string `json:"key_path,omitempty"`
}

// SSHKeyListResponse represents the response for listing SSH keys
type SSHKeyListResponse struct {
	Success bool                   `json:"success"`
	Keys    []transport.SSHKeyInfo `json:"keys"`
	Count   int                    `json:"count"`
}

// HandleUploadSSHKey handles SSH key upload via POST request
func (s *SSHKeyManager) HandleUploadSSHKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	if !s.authenticateRequest(r) {
		http.Error(w, "Unauthorized: SSH key management requires authentication", http.StatusUnauthorized)
		return
	}

	// Parse JSON request
	var req SSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.PrivateKey == "" {
		http.Error(w, "Name and private_key are required", http.StatusBadRequest)
		return
	}

	// Validate key name (alphanumeric, hyphens, underscores only)
	if !isValidKeyName(req.Name) {
		http.Error(w, "Invalid key name. Use only alphanumeric characters, hyphens, and underscores", http.StatusBadRequest)
		return
	}

	// Initialize key manager if not already done
	if err := s.keyManager.Initialize(); err != nil {
		log.Printf("Failed to initialize SSH key manager: %v", err)
		http.Error(w, "Failed to initialize key manager", http.StatusInternalServerError)
		return
	}

	// Get authenticated user's npub
	ownerNpub := s.getAuthenticatedNpub(r)
	if ownerNpub == "" {
		http.Error(w, "Authentication required: Nostr pubkey not found or not authenticated", http.StatusUnauthorized)
		return
	}

	// Save the private key
	privateKeyPath := filepath.Join(s.keyManager.GetKeyDir(), req.Name+".pem")
	if err := s.keyManager.SaveKey(req.Name, []byte(req.PrivateKey), []byte(req.PublicKey), ownerNpub); err != nil {
		log.Printf("Failed to save SSH key: %v", err)
		http.Error(w, "Failed to save SSH key", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := SSHKeyResponse{
		Success: true,
		Message: "SSH key uploaded successfully",
		KeyName: req.Name,
		KeyPath: privateKeyPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleListSSHKeys handles listing SSH keys via GET request
func (s *SSHKeyManager) HandleListSSHKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	if !s.authenticateRequest(r) {
		http.Error(w, "Unauthorized: SSH key management requires authentication", http.StatusUnauthorized)
		return
	}

	// Initialize key manager if not already done
	if err := s.keyManager.Initialize(); err != nil {
		log.Printf("Failed to initialize SSH key manager: %v", err)
		http.Error(w, "Failed to initialize key manager", http.StatusInternalServerError)
		return
	}

	// Get authenticated user's npub
	ownerNpub := s.getAuthenticatedNpub(r)
	if ownerNpub == "" {
		http.Error(w, "Authentication required: Nostr pubkey not found or not authenticated", http.StatusUnauthorized)
		return
	}

	// Get SSH keys owned by the authenticated user
	keys := s.keyManager.ListKeysByOwner(ownerNpub)

	// Return success response
	response := SSHKeyListResponse{
		Success: true,
		Keys:    keys,
		Count:   len(keys),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDeleteSSHKey handles SSH key deletion via DELETE request
func (s *SSHKeyManager) HandleDeleteSSHKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	if !s.authenticateRequest(r) {
		http.Error(w, "Unauthorized: SSH key management requires authentication", http.StatusUnauthorized)
		return
	}

	// Get key name from URL path
	keyName := strings.TrimPrefix(r.URL.Path, "/api/v1/ssh-keys/")
	if keyName == "" {
		http.Error(w, "Key name is required", http.StatusBadRequest)
		return
	}

	// Get authenticated user's npub
	ownerNpub := s.getAuthenticatedNpub(r)
	if ownerNpub == "" {
		http.Error(w, "Authentication required: Nostr pubkey not found or not authenticated", http.StatusUnauthorized)
		return
	}

	// Check if user owns this key
	if !s.keyManager.IsOwner(keyName, ownerNpub) {
		http.Error(w, "Forbidden: You can only delete your own SSH keys", http.StatusForbidden)
		return
	}

	// Initialize key manager if not already done
	if err := s.keyManager.Initialize(); err != nil {
		log.Printf("Failed to initialize SSH key manager: %v", err)
		http.Error(w, "Failed to initialize key manager", http.StatusInternalServerError)
		return
	}

	// Remove the key
	if err := s.keyManager.RemoveKey(keyName); err != nil {
		log.Printf("Failed to remove SSH key: %v", err)
		http.Error(w, "Failed to remove SSH key", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := SSHKeyResponse{
		Success: true,
		Message: "SSH key removed successfully",
		KeyName: keyName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleSSHKeyForm handles SSH key upload via HTML form
func (s *SSHKeyManager) HandleSSHKeyForm(w http.ResponseWriter, r *http.Request) {
	// Check authentication for both GET and POST
	if !s.authenticateRequest(r) {
		// Return a simple login form for unauthorized users
		loginHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>SSH Key Management - Nostr Authentication Required</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; background: #f8f9fa; }
        .login-form { background: #fff; padding: 30px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        .nostr-section { background: #e3f2fd; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .auth-button { background: #8e24aa; color: white; padding: 12px 24px; border: none; border-radius: 6px; cursor: pointer; font-size: 16px; margin: 10px 0; }
        .auth-button:hover { background: #7b1fa2; }
        .code-block { background: #f5f5f5; padding: 15px; border-radius: 4px; font-family: monospace; margin: 10px 0; }
        .step { margin: 15px 0; padding: 10px; background: #f8f9fa; border-left: 4px solid #8e24aa; }
    </style>
</head>
<body>
    <h1>🔐 SSH Key Management</h1>
    <div class="login-form">
        <h2>Nostr Authentication Required</h2>
        <p>SSH key management requires Nostr authentication using NIP-42. This ensures secure, identity-based access to your SSH keys.</p>
        
        <div class="nostr-section">
            <h3>🌐 Browser Authentication (Recommended)</h3>
            <p>Use your Nostr browser extension to authenticate:</p>
            <button class="auth-button" onclick="authenticateWithNostr()">🔑 Authenticate with Nostr</button>
            <div id="auth-status"></div>
        </div>

        <div class="step">
            <h3>📱 Manual Authentication</h3>
            <p>If you don't have a Nostr browser extension, you can authenticate manually:</p>
            <ol>
                <li>Get a challenge: <code>GET /api/v1/nostr/challenge</code></li>
                <li>Sign a NIP-42 auth event with your Nostr private key</li>
                <li>Submit the signed event: <code>POST /api/v1/nostr/auth</code></li>
                <li>Use your npub in the <code>X-Nostr-Pubkey</code> header</li>
            </ol>
        </div>

        <div class="step">
            <h3>💻 Command Line Authentication</h3>
            <p>For terminal access, set your Nostr private key:</p>
            <div class="code-block">export MERCURY_PRIVATE_KEY="nsec1your-private-key"</div>
            <p>Then use the SSH key manager terminal interface.</p>
        </div>
    </div>

    <script>
        async function authenticateWithNostr() {
            const statusDiv = document.getElementById('auth-status');
            statusDiv.innerHTML = '<p>🔄 Checking for Nostr extension...</p>';
            
            try {
                // Check if Nostr extension is available
                if (typeof window.nostr === 'undefined') {
                    statusDiv.innerHTML = '<p>❌ Nostr extension not found. Please install a Nostr browser extension.</p>';
                    return;
                }

                // Get challenge
                const challengeResponse = await fetch('/api/v1/nostr/challenge');
                const { challenge, relay } = await challengeResponse.json();
                
                statusDiv.innerHTML = '<p>🔑 Signing authentication event...</p>';
                
                // Create auth event
                const authEvent = {
                    kind: 22242,
                    created_at: Math.floor(Date.now() / 1000),
                    tags: [
                        ['relay', relay],
                        ['challenge', challenge]
                    ],
                    content: ''
                };
                
                // Sign with Nostr extension
                const signedEvent = await window.nostr.signEvent(authEvent);
                
                // Submit authentication
                const authResponse = await fetch('/api/v1/nostr/auth', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ event: signedEvent })
                });
                
                if (authResponse.ok) {
                    statusDiv.innerHTML = '<p>✅ Authentication successful! You can now manage SSH keys.</p>';
                    // Reload the page to show the SSH key management interface
                    setTimeout(() => window.location.reload(), 1000);
                } else {
                    statusDiv.innerHTML = '<p>❌ Authentication failed. Please try again.</p>';
                }
            } catch (error) {
                statusDiv.innerHTML = '<p>❌ Error: ' + error.message + '</p>';
            }
        }
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(loginHTML))
		return
	}

	if r.Method == "GET" {
		// Serve HTML form
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>SSH Key Upload</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, textarea { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
        textarea { height: 150px; font-family: monospace; }
        button { background: #007cba; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background: #005a87; }
        .success { color: green; margin-top: 10px; }
        .error { color: red; margin-top: 10px; }
        .key-list { margin-top: 30px; }
        .key-item { background: #f5f5f5; padding: 10px; margin: 5px 0; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>SSH Key Management</h1>
    
    <h2>Upload SSH Key</h2>
    <form id="uploadForm">
        <div class="form-group">
            <label for="name">Key Name:</label>
            <input type="text" id="name" name="name" required placeholder="my-key">
        </div>
        <div class="form-group">
            <label for="private_key">Private Key (PEM format):</label>
            <textarea id="private_key" name="private_key" required placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"></textarea>
        </div>
        <div class="form-group">
            <label for="public_key">Public Key (optional):</label>
            <textarea id="public_key" name="public_key" placeholder="ssh-rsa AAAAB3NzaC1yc2E..."></textarea>
        </div>
        <div class="form-group">
            <label for="description">Description (optional):</label>
            <input type="text" id="description" name="description" placeholder="My laptop key">
        </div>
        <button type="submit">Upload Key</button>
    </form>
    
    <div id="result"></div>
    
    <div class="key-list">
        <h2>Existing Keys</h2>
        <button onclick="loadKeys()">Refresh Key List</button>
        <div id="keyList"></div>
    </div>

    <script>
        document.getElementById('uploadForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const formData = new FormData(e.target);
            const data = {
                name: formData.get('name'),
                private_key: formData.get('private_key'),
                public_key: formData.get('public_key'),
                description: formData.get('description')
            };
            
            try {
                const response = await fetch('/api/v1/ssh-keys', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });
                
                const result = await response.json();
                const resultDiv = document.getElementById('result');
                
                if (result.success) {
                    resultDiv.innerHTML = '<div class="success">✓ ' + result.message + '</div>';
                    loadKeys(); // Refresh key list
                } else {
                    resultDiv.innerHTML = '<div class="error">✗ ' + result.message + '</div>';
                }
            } catch (error) {
                document.getElementById('result').innerHTML = '<div class="error">✗ Error: ' + error.message + '</div>';
            }
        });
        
        async function loadKeys() {
            try {
                const response = await fetch('/api/v1/ssh-keys');
                const result = await response.json();
                
                const keyListDiv = document.getElementById('keyList');
                if (result.success && result.keys.length > 0) {
                    keyListDiv.innerHTML = result.keys.map(key => 
                        '<div class="key-item">' +
                        '<strong>' + key.Name + '</strong> (' + key.Type + ')<br>' +
                        '<small>Created: ' + key.CreatedAt + '</small><br>' +
                        '<button onclick="deleteKey(\'' + key.Name + '\')">Delete</button>' +
                        '</div>'
                    ).join('');
                } else {
                    keyListDiv.innerHTML = '<p>No SSH keys found.</p>';
                }
            } catch (error) {
                document.getElementById('keyList').innerHTML = '<div class="error">Error loading keys: ' + error.message + '</div>';
            }
        }
        
        async function deleteKey(keyName) {
            if (confirm('Are you sure you want to delete key "' + keyName + '"?')) {
                try {
                    const response = await fetch('/api/v1/ssh-keys/' + keyName, {
                        method: 'DELETE'
                    });
                    
                    const result = await response.json();
                    if (result.success) {
                        loadKeys(); // Refresh key list
                    } else {
                        alert('Error: ' + result.message);
                    }
                } catch (error) {
                    alert('Error: ' + error.message);
                }
            }
        }
        
        // Load keys on page load
        loadKeys();
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
		return
	}

	// Handle POST request for form upload
	if r.Method == "POST" {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Create request from form data
		req := SSHKeyRequest{
			Name:        r.FormValue("name"),
			PrivateKey:  r.FormValue("private_key"),
			PublicKey:   r.FormValue("public_key"),
			Description: r.FormValue("description"),
		}

		// Validate required fields
		if req.Name == "" || req.PrivateKey == "" {
			http.Error(w, "Name and private_key are required", http.StatusBadRequest)
			return
		}

		// Initialize key manager if not already done
		if err := s.keyManager.Initialize(); err != nil {
			log.Printf("Failed to initialize SSH key manager: %v", err)
			http.Error(w, "Failed to initialize key manager", http.StatusInternalServerError)
			return
		}

		// Get authenticated user's npub
		ownerNpub := s.getAuthenticatedNpub(r)
		if ownerNpub == "" {
			http.Error(w, "Authentication required: Nostr pubkey not found or not authenticated", http.StatusUnauthorized)
			return
		}

		// Save the key
		if err := s.keyManager.SaveKey(req.Name, []byte(req.PrivateKey), []byte(req.PublicKey), ownerNpub); err != nil {
			log.Printf("Failed to save SSH key: %v", err)
			http.Error(w, "Failed to save SSH key", http.StatusInternalServerError)
			return
		}

		// Return success response
		response := SSHKeyResponse{
			Success: true,
			Message: "SSH key uploaded successfully",
			KeyName: req.Name,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// isValidKeyName validates SSH key names
func isValidKeyName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}

	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// HandleNostrChallenge handles Nostr authentication challenge generation
func (s *SSHKeyManager) HandleNostrChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	challenge, err := s.nostrAuth.GenerateChallenge()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate challenge: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"challenge": challenge,
		"relay":     s.nostrAuth.RelayURL,
		"message":   "Use this challenge to authenticate with NIP-42",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleNostrAuth handles Nostr authentication
func (s *SSHKeyManager) HandleNostrAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Event map[string]interface{} `json:"event"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Parse the Nostr event
	eventJSON, err := json.Marshal(req.Event)
	if err != nil {
		http.Error(w, "Invalid event format", http.StatusBadRequest)
		return
	}

	// Parse as Nostr event
	var nostrEvent nostr.Event
	if err := json.Unmarshal(eventJSON, &nostrEvent); err != nil {
		http.Error(w, "Invalid Nostr event format", http.StatusBadRequest)
		return
	}

	// Verify the authentication event
	if err := s.nostrAuth.VerifyAuthentication(&nostrEvent); err != nil {
		response := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Authentication failed: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Authentication successful
	response := map[string]interface{}{
		"success": true,
		"message": "Authentication successful",
		"pubkey":  nostrEvent.PubKey,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// authenticateRequest checks if the request is authorized to manage SSH keys
// Only allows Nostr authentication (NIP-42)
func (s *SSHKeyManager) authenticateRequest(r *http.Request) bool {
	// Check if SSH key management is enabled
	if !s.config.Enabled {
		return false
	}

	// Only allow Nostr authentication (NIP-42)
	npub := r.Header.Get("X-Nostr-Pubkey")
	if npub != "" {
		// Check if this pubkey is authenticated via NIP-42
		if s.nostrAuth.IsAuthenticated(npub) {
			return true
		}
	}

	return false
}

// getAuthenticatedNpub extracts the authenticated Nostr pubkey from the request
func (s *SSHKeyManager) getAuthenticatedNpub(r *http.Request) string {
	// Check for Nostr pubkey in header
	npub := r.Header.Get("X-Nostr-Pubkey")
	if npub != "" && s.nostrAuth.IsAuthenticated(npub) {
		return npub
	}
	return ""
}

// requireAuthentication middleware for SSH key operations
func (s *SSHKeyManager) requireAuthentication(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticateRequest(r) {
			http.Error(w, "Unauthorized: SSH key management requires authentication", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
