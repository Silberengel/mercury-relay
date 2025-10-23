# 🔐 Nostr SSH Key Authentication System

## Overview

The Mercury Relay now implements **Nostr-only authentication** for SSH key management, ensuring that only authenticated Nostr users can manage SSH keys. This provides secure, identity-based access control using NIP-42 authentication.

## 🔑 Authentication Methods

### 1. Browser Authentication (Recommended)

**For Web Interface:**
- Uses Nostr browser extensions (Alby, Nos2x, etc.)
- Automatic challenge generation and signing
- Seamless user experience

**Usage:**
1. Visit `http://localhost:8082/ssh-keys`
2. Click "🔑 Authenticate with Nostr"
3. Browser extension handles signing automatically
4. Access SSH key management interface

### 2. Command Line Authentication

**For Terminal Interface:**
- Uses `$MERCURY_PRIVATE_KEY` environment variable
- Manual NIP-42 authentication flow
- Full terminal control

**Usage:**
```bash
# Set your Nostr private key
export MERCURY_PRIVATE_KEY="nsec1your-private-key"

# Run the authenticated SSH key manager
./nostr-ssh-manager

# Or with custom relay URL
./nostr-ssh-manager -relay http://your-relay:8082
```

## 🛡️ Security Features

### Authentication Requirements

- ✅ **Nostr-only authentication** - No other auth methods allowed
- ✅ **NIP-42 compliance** - Standard Nostr authentication
- ✅ **Challenge-based security** - Cryptographically secure challenges
- ✅ **Time-limited sessions** - Authentication expires after 24 hours
- ✅ **Ownership validation** - Users can only manage their own keys

### SSH Key Association

- ✅ **Identity-based ownership** - Each SSH key tied to specific npub
- ✅ **No cross-user access** - Users cannot see others' keys
- ✅ **Audit trail** - All operations logged with npub
- ✅ **Secure isolation** - Complete user separation

## 🚀 Usage Examples

### Browser Interface

```html
<!-- Automatic Nostr authentication -->
<button onclick="authenticateWithNostr()">
  🔑 Authenticate with Nostr
</button>

<!-- After authentication, manage SSH keys -->
<form action="/ssh-keys" method="POST">
  <input name="key_name" placeholder="Key name" required>
  <textarea name="private_key" placeholder="Private key (PEM)"></textarea>
  <button type="submit">Upload SSH Key</button>
</form>
```

### Command Line Interface

```bash
# Authenticate and manage SSH keys
export MERCURY_PRIVATE_KEY="nsec1your-private-key"
./nostr-ssh-manager

# Interactive commands
nostr-ssh> list
nostr-ssh> add
nostr-ssh> remove my-server-key
nostr-ssh> help
```

### API Usage

```bash
# 1. Get challenge
curl -X GET http://localhost:8082/api/v1/nostr/challenge

# 2. Sign with your Nostr private key (kind 22242 event)
# 3. Submit authentication
curl -X POST http://localhost:8082/api/v1/nostr/auth \
  -H "Content-Type: application/json" \
  -d '{"event": {...}}'

# 4. Use authenticated session
curl -H "X-Nostr-Pubkey: npub1your-pubkey" \
  http://localhost:8082/api/v1/ssh-keys
```

## 🔧 Configuration

### Environment Variables

```bash
# Required for terminal authentication
export MERCURY_PRIVATE_KEY="nsec1your-private-key"

# Optional relay configuration
export MERCURY_RELAY_URL="http://localhost:8082"
```

### Authorized Pubkeys

```yaml
ssh:
  authentication:
    authorized_pubkeys:
      - "npub1user1-pubkey"
      - "npub1user2-pubkey"
      - "npub1admin-pubkey"
```

## 📋 API Endpoints

### Authentication

- `GET /api/v1/nostr/challenge` - Get authentication challenge
- `POST /api/v1/nostr/auth` - Submit signed authentication event

### SSH Key Management

- `GET /api/v1/ssh-keys` - List your SSH keys (requires auth)
- `POST /api/v1/ssh-keys` - Upload SSH key (requires auth)
- `DELETE /api/v1/ssh-keys/{name}` - Delete SSH key (requires auth)

### Web Interface

- `GET /ssh-keys` - SSH key management form (requires auth)

## 🔒 Security Considerations

### Authentication Flow

1. **Challenge Generation**: Server creates cryptographically secure challenge
2. **Event Signing**: Client signs NIP-42 auth event with private key
3. **Signature Verification**: Server verifies signature and validates event
4. **Session Creation**: Authenticated session established for 24 hours
5. **Operation Authorization**: All SSH key operations require valid session

### Key Management

- **Private Key Security**: Store `MERCURY_PRIVATE_KEY` securely
- **Session Expiration**: Authentication expires after 24 hours
- **Challenge Expiry**: Challenges expire after 10 minutes
- **Ownership Validation**: Users can only access their own keys

## 🚫 Restricted Access

### Terminal Interface

The original SSH terminal interface now requires authentication:

```bash
# Old terminal interface shows authentication required message
ssh-key-manager> list
❌ Authentication required for SSH key management.
Please use the Nostr-authenticated SSH key manager:
  export MERCURY_PRIVATE_KEY="nsec1your-private-key"
  ./nostr-ssh-manager
```

### API Access

All SSH key management endpoints require Nostr authentication:

```bash
# Without authentication
curl http://localhost:8082/api/v1/ssh-keys
# Returns: 401 Unauthorized: SSH key management requires authentication

# With authentication
curl -H "X-Nostr-Pubkey: npub1your-pubkey" \
  http://localhost:8082/api/v1/ssh-keys
# Returns: Your SSH keys
```

## 🎯 Benefits

### Security

- **Identity-based access** - SSH keys tied to Nostr identities
- **No shared credentials** - Each user manages their own keys
- **Cryptographic security** - NIP-42 standard authentication
- **Audit trail** - All operations logged with npub

### User Experience

- **Familiar authentication** - Uses existing Nostr infrastructure
- **Browser integration** - Works with Nostr browser extensions
- **Terminal control** - Full command-line interface
- **Cross-platform** - Works on any system with Nostr support

### Administration

- **Centralized management** - All SSH keys in one place
- **User isolation** - Complete separation between users
- **Easy deployment** - Standard Nostr authentication
- **Scalable** - Supports unlimited users

## 🔧 Troubleshooting

### Common Issues

1. **"MERCURY_PRIVATE_KEY not set"**
   ```bash
   export MERCURY_PRIVATE_KEY="nsec1your-private-key"
   ```

2. **"Authentication failed"**
   - Check private key format (nsec1...)
   - Verify relay URL is correct
   - Ensure challenge hasn't expired

3. **"Forbidden: You can only delete your own SSH keys"**
   - Only the key owner can delete keys
   - Check you're using the correct npub

### Debug Mode

Enable debug logging to see authentication details:

```yaml
logging:
  level: debug
```

## 🚀 Getting Started

### 1. Set Up Authentication

```bash
# Get your Nostr private key (nsec1...)
export MERCURY_PRIVATE_KEY="nsec1your-private-key"
```

### 2. Browser Access

1. Visit `http://localhost:8082/ssh-keys`
2. Click "🔑 Authenticate with Nostr"
3. Use your Nostr browser extension
4. Manage SSH keys through the web interface

### 3. Terminal Access

```bash
# Run the authenticated SSH key manager
./nostr-ssh-manager

# List your SSH keys
nostr-ssh> list

# Add a new SSH key
nostr-ssh> add

# Remove an SSH key
nostr-ssh> remove my-key
```

The Mercury Relay now provides **secure, Nostr-authenticated SSH key management** with complete user isolation and identity-based access control! 🔐✨
