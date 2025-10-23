# Mercury Relay API Documentation

This document describes the REST API endpoints provided by Mercury Relay, including the new kind-based filtering endpoints.

## Base URL

- **Local Development**: `http://localhost:8082`
- **Production**: `https://your-relay.com`

## Authentication

Most endpoints require Nostr authentication using NIP-42. Include the authentication header:

```
Authorization: Nostr <base64-encoded-event>
```

## Health and Status

### Health Check
```http
GET /api/v1/health
```

**Description**: Check if the relay is running and healthy.

**Authentication**: None required

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h30m15s"
}
```

### Relay Statistics
```http
GET /api/v1/stats
```

**Description**: Get general relay statistics.

**Authentication**: Required

**Response**:
```json
{
  "total_events": 15000,
  "total_authors": 500,
  "events_per_second": 2.5,
  "uptime": "2h30m15s",
  "memory_usage": "45MB",
  "active_connections": 25
}
```

## Event Management

### Get Events
```http
GET /api/v1/events
POST /api/v1/events
```

**Description**: Query or publish events.

**Authentication**: Required for publishing

**Query Parameters** (GET):
- `authors`: Comma-separated list of author pubkeys
- `kinds`: Comma-separated list of event kinds
- `since`: Unix timestamp (start time)
- `until`: Unix timestamp (end time)
- `limit`: Maximum number of events to return

**Request Body** (POST):
```json
{
  "id": "event_id",
  "pubkey": "author_pubkey",
  "created_at": 1700000000,
  "kind": 1,
  "tags": [["e", "referenced_event_id"]],
  "content": "Hello, Nostr!",
  "sig": "signature"
}
```

**Response**:
```json
[
  {
    "id": "event_id",
    "pubkey": "author_pubkey",
    "created_at": 1700000000,
    "kind": 1,
    "tags": [["e", "referenced_event_id"]],
    "content": "Hello, Nostr!",
    "sig": "signature"
  }
]
```

### Publish Event
```http
POST /api/v1/publish
```

**Description**: Publish a new event to the relay.

**Authentication**: Required

**Request Body**:
```json
{
  "id": "event_id",
  "pubkey": "author_pubkey",
  "created_at": 1700000000,
  "kind": 1,
  "tags": [],
  "content": "Hello, Nostr!",
  "sig": "signature"
}
```

**Response**:
```json
{
  "status": "success",
  "event_id": "event_id",
  "message": "Event published successfully"
}
```

## Kind-Based Filtering

### Get Events by Kind
```http
GET /api/v1/kind/{kind}/events
```

**Description**: Retrieve events from a specific kind topic.

**Authentication**: Required

**Parameters**:
- `kind`: Event kind number (e.g., 0, 1, 7, 10002)
- `limit`: Maximum number of events (default: 100)
- `since`: Unix timestamp (start time)
- `until`: Unix timestamp (end time)

**Response**:
```json
{
  "kind": 1,
  "description": "Text Note",
  "total_events": 150,
  "events": [
    {
      "id": "event_id",
      "pubkey": "author_pubkey",
      "created_at": 1700000000,
      "kind": 1,
      "tags": [],
      "content": "Hello, Nostr!",
      "sig": "signature"
    }
  ]
}
```

### Get Kind Statistics
```http
GET /api/v1/kind/{kind}/stats
```

**Description**: Get statistics for a specific kind topic.

**Authentication**: Required

**Parameters**:
- `kind`: Event kind number

**Response**:
```json
{
  "kind": 1,
  "description": "Text Note",
  "queue_name": "nostr_kind_1",
  "message_count": 150,
  "consumer_count": 1,
  "last_updated": "2024-01-15T10:30:00Z"
}
```

### Get All Kind Statistics
```http
GET /api/v1/kind/stats
```

**Description**: Get statistics for all kind topics.

**Authentication**: Required

**Response**:
```json
{
  "kind_0": {
    "kind": 0,
    "description": "Profile metadata",
    "count": 50,
    "queue_name": "nostr_kind_0"
  },
  "kind_1": {
    "kind": 1,
    "description": "Short notes/text posts",
    "count": 2500,
    "queue_name": "nostr_kind_1"
  },
  "kind_3": {
    "kind": 3,
    "description": "Contacts/follow list",
    "count": 200,
    "queue_name": "nostr_kind_3"
  },
  "kind_7": {
    "kind": 7,
    "description": "Reactions",
    "count": 500,
    "queue_name": "nostr_kind_7"
  },
  "kind_10002": {
    "kind": 10002,
    "description": "Relay lists",
    "count": 75,
    "queue_name": "nostr_kind_10002"
  },
  "kind_undefined": {
    "kind": -1,
    "description": "Undefined/Unknown kinds",
    "count": 25,
    "queue_name": "nostr_kind_undefined"
  },
  "kind_moderation": {
    "kind": -2,
    "description": "Moderation/Invalid events",
    "count": 5,
    "queue_name": "nostr_kind_moderation"
  }
}
```

## Ebook Management

### Get Ebooks
```http
GET /api/v1/ebooks
```

**Description**: Retrieve available ebooks.

**Authentication**: Required

**Query Parameters**:
- `author`: Filter by author pubkey
- `format`: Filter by format (epub, pdf, etc.)
- `limit`: Maximum number of ebooks

**Response**:
```json
[
  {
    "id": "ebook_id",
    "title": "Book Title",
    "author": "author_pubkey",
    "format": "epub",
    "size": 1024000,
    "created_at": 1700000000
  }
]
```

### Get Ebook Content
```http
GET /api/v1/ebooks/{id}
```

**Description**: Retrieve ebook content.

**Authentication**: Required

**Response**: Binary content (ebook file)

### Generate EPUB
```http
GET /api/v1/ebooks/{id}/epub
```

**Description**: Generate EPUB format from ebook.

**Authentication**: Required

**Response**: Binary content (EPUB file)

## Error Responses

### Standard Error Format
```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": "Additional error details",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `INVALID_REQUEST` | 400 | Invalid request format |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Internal server error |

### Example Error Responses

**Authentication Required**:
```json
{
  "error": "UNAUTHORIZED",
  "message": "Nostr authentication required",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Invalid Kind**:
```json
{
  "error": "INVALID_REQUEST",
  "message": "Invalid kind number",
  "details": "Kind must be a positive integer",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Rate Limited**:
```json
{
  "error": "RATE_LIMITED",
  "message": "Too many requests",
  "details": "Rate limit: 10 requests per minute",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Rate Limiting

### Limits

- **General API**: 100 requests per minute per IP
- **Event Publishing**: 10 events per minute per user
- **Kind Queries**: 50 requests per minute per user
- **Admin Operations**: 20 requests per minute per admin

### Headers

Rate limit information is included in response headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1700003600
```

## CORS Support

The API supports Cross-Origin Resource Sharing (CORS) for web applications:

### Preflight Request
```http
OPTIONS /api/v1/events
```

**Response Headers**:
```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Max-Age: 86400
```

## WebSocket API

### Connection
```javascript
const ws = new WebSocket('ws://localhost:8080');
```

### Subscribe to Events
```javascript
// Subscribe to all events
ws.send(JSON.stringify([
  "REQ",
  "subscription_id",
  {
    "kinds": [1, 7],
    "authors": ["author_pubkey"],
    "since": 1700000000
  }
]));

// Listen for events
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data[0] === "EVENT") {
    console.log("Received event:", data[2]);
  }
};
```

### Publish Event
```javascript
ws.send(JSON.stringify([
  "EVENT",
  {
    "id": "event_id",
    "pubkey": "author_pubkey",
    "created_at": 1700000000,
    "kind": 1,
    "tags": [],
    "content": "Hello, Nostr!",
    "sig": "signature"
  }
]));
```

## Examples

### Complete Workflow

1. **Check Health**:
```bash
curl http://localhost:8082/api/v1/health
```

2. **Get Kind Statistics**:
```bash
curl -H "Authorization: Nostr <auth_token>" \
     http://localhost:8082/api/v1/kind/stats
```

3. **Get Text Notes**:
```bash
curl -H "Authorization: Nostr <auth_token>" \
     "http://localhost:8082/api/v1/kind/1/events?limit=10"
```

4. **Publish Event**:
```bash
curl -X POST \
     -H "Authorization: Nostr <auth_token>" \
     -H "Content-Type: application/json" \
     -d '{"id":"event_id","pubkey":"author_pubkey","created_at":1700000000,"kind":1,"tags":[],"content":"Hello, Nostr!","sig":"signature"}' \
     http://localhost:8082/api/v1/publish
```

### JavaScript Client

```javascript
class MercuryClient {
  constructor(baseUrl, authToken) {
    this.baseUrl = baseUrl;
    this.authToken = authToken;
  }

  async getKindStats(kind) {
    const response = await fetch(`${this.baseUrl}/api/v1/kind/${kind}/stats`, {
      headers: {
        'Authorization': `Nostr ${this.authToken}`
      }
    });
    return response.json();
  }

  async getKindEvents(kind, limit = 100) {
    const response = await fetch(`${this.baseUrl}/api/v1/kind/${kind}/events?limit=${limit}`, {
      headers: {
        'Authorization': `Nostr ${this.authToken}`
      }
    });
    return response.json();
  }

  async publishEvent(event) {
    const response = await fetch(`${this.baseUrl}/api/v1/publish`, {
      method: 'POST',
      headers: {
        'Authorization': `Nostr ${this.authToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(event)
    });
    return response.json();
  }
}

// Usage
const client = new MercuryClient('http://localhost:8082', 'your_auth_token');

// Get all kind statistics
const stats = await client.getKindStats('all');

// Get text notes
const notes = await client.getKindEvents(1, 50);

// Publish a reaction
const reaction = {
  id: 'reaction_id',
  pubkey: 'your_pubkey',
  created_at: Math.floor(Date.now() / 1000),
  kind: 7,
  tags: [['e', 'target_event_id']],
  content: '+',
  sig: 'your_signature'
};
await client.publishEvent(reaction);
```

## SDKs and Libraries

### JavaScript/TypeScript
```bash
npm install @mercury-relay/client
```

```typescript
import { MercuryRelay } from '@mercury-relay/client';

const relay = new MercuryRelay({
  baseUrl: 'http://localhost:8082',
  authToken: 'your_auth_token'
});

// Get kind statistics
const stats = await relay.getKindStats();

// Subscribe to events
relay.subscribe({ kinds: [1, 7] }, (event) => {
  console.log('Received event:', event);
});
```

### Python
```bash
pip install mercury-relay
```

```python
from mercury_relay import MercuryRelay

relay = MercuryRelay(
    base_url='http://localhost:8082',
    auth_token='your_auth_token'
)

# Get kind statistics
stats = relay.get_kind_stats()

# Get events by kind
events = relay.get_kind_events(kind=1, limit=100)

# Publish event
relay.publish_event({
    'id': 'event_id',
    'pubkey': 'author_pubkey',
    'created_at': 1700000000,
    'kind': 1,
    'tags': [],
    'content': 'Hello, Nostr!',
    'sig': 'signature'
})
```

## Monitoring and Debugging

### Health Monitoring
```bash
# Check relay health
curl http://localhost:8082/api/v1/health

# Get detailed statistics
curl -H "Authorization: Nostr <auth_token>" \
     http://localhost:8082/api/v1/stats
```

### Kind-Based Monitoring
```bash
# Monitor all kind topics
curl -H "Authorization: Nostr <auth_token>" \
     http://localhost:8082/api/v1/kind/stats

# Check moderation queue
curl -H "Authorization: Nostr <auth_token>" \
     http://localhost:8082/api/v1/kind/-2/stats
```

### Log Analysis
```bash
# View relay logs
docker-compose logs -f mercury-relay

# Filter for kind-related logs
docker-compose logs mercury-relay | grep "kind"

# Monitor API requests
docker-compose logs mercury-relay | grep "api/v1"
```
