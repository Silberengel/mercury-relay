# Kind-Based Event Filtering System

Mercury Relay implements a sophisticated kind-based filtering system that automatically routes Nostr events to appropriate topics based on their kind and quality. This system provides better organization, quality control, and scalability for handling diverse event types.

## Overview

The kind-based filtering system consists of three main components:

1. **Dynamic Kind Loading**: Automatically discovers event kinds from YAML configuration files
2. **Quality Control Filtering**: Validates events and routes invalid ones to moderation
3. **Topic-Based Routing**: Routes valid events to appropriate kind-specific topics

## Architecture

### Event Flow

```
Event Received
     ↓
Basic Validation
     ↓
┌─────────────────┬─────────────────┐
│   Invalid       │     Valid       │
│     ↓           │       ↓         │
│ Moderation      │ Kind Check      │
│   Topic         │     ↓           │
└─────────────────┴─────────────────┘
                        ↓
                ┌───────────────┐
                │ Known Kind?   │
                └───────────────┘
                        ↓
                ┌───────────────┐
                │ Yes    │ No   │
                │   ↓    │  ↓   │
                │Kind.X  │Undef │
                └───────────────┘
```

### Topics Created

The system automatically creates the following RabbitMQ topics:

- **`nostr_kind_0`**: User metadata events
- **`nostr_kind_1`**: Text note events  
- **`nostr_kind_3`**: Follow list events
- **`nostr_kind_7`**: Reaction events
- **`nostr_kind_9`**: Chat message events
- **`nostr_kind_30`**: Internal citation events
- **`nostr_kind_31`**: External web citation events
- **`nostr_kind_10002`**: Relay list events
- **`nostr_kind_30023`**: Long-form content events
- **`nostr_kind_30040`**: Publication index events
- **`nostr_kind_30041`**: Publication content events
- **`nostr_kind_30042`**: Drive events
- **`nostr_kind_30043`**: Traceback events
- **`nostr_kind_undefined`**: Valid events with unknown kinds
- **`nostr_kind_moderation`**: Invalid events requiring review

## Configuration

### Individual Kind Configuration Files

Each event kind is configured in its own YAML file located in `configs/kinds/`. The filename format is `{kind_number}.yml`.

#### Configuration Structure

```yaml
# Kind {number}: {name}
name: "Human-readable name"
description: "Description of the event kind"
required_tags: ["tag1", "tag2"]  # Required tags for this kind
optional_tags: ["tag3", "tag4"]  # Optional tags for this kind
content_validation:
  type: "text|json|empty"        # Content type validation
  max_length: 10000              # Maximum content length
  min_length: 1                  # Minimum content length
  required_fields: ["field1"]    # Required JSON fields (for json type)
  optional_fields: ["field2"]    # Optional JSON fields (for json type)
quality_rules:
  - name: "rule_name"
    weight: 1.0                  # Rule weight (0.0-1.0)
    description: "Rule description"
replaceable: true|false          # Whether events of this kind are replaceable
ephemeral: true|false            # Whether events of this kind are ephemeral
```

#### Example Configuration

**`configs/kinds/1.yml`** (Text Notes):
```yaml
# Kind 1: Text Note
name: "Text Note"
description: "Regular social media post"
required_tags: []
optional_tags: ["e", "p", "a", "t", "d", "r", "g", "alt"]
content_validation:
  type: "text"
  max_length: 10000
  min_length: 1
quality_rules:
  - name: "not_spam"
    weight: 1.0
    description: "Content should not be spam"
  - name: "meaningful_content"
    weight: 0.8
    description: "Content should have meaningful text"
  - name: "proper_encoding"
    weight: 0.9
    description: "Content should be properly encoded"
replaceable: false
ephemeral: false
```

**`configs/kinds/0.yml`** (User Metadata):
```yaml
# Kind 0: User Metadata
name: "User Metadata"
description: "User profile information"
required_tags: []
optional_tags: ["d"]
content_validation:
  type: "json"
  required_fields: ["name"]
  optional_fields: ["about", "picture", "banner", "website", "lud16", "nip05"]
  max_length: 1000
quality_rules:
  - name: "valid_json"
    weight: 1.0
    description: "Content must be valid JSON"
  - name: "has_name"
    weight: 0.8
    description: "Must have a name field"
  - name: "reasonable_length"
    weight: 0.6
    description: "Content length should be reasonable"
replaceable: true
ephemeral: false
```

## API Endpoints

The system provides several REST API endpoints for monitoring and managing kind-based filtering:

### Get Events by Kind
```
GET /api/v1/kind/{kind}/events
```
Retrieves events from a specific kind topic.

### Get Kind Statistics
```
GET /api/v1/kind/{kind}/stats
```
Returns statistics for a specific kind topic.

### Get All Kind Statistics
```
GET /api/v1/kind/stats
```
Returns statistics for all kind topics, including:
- Known kinds with their descriptions
- Undefined kinds count
- Moderation queue count

#### Example Response
```json
{
  "kind_0": {
    "kind": 0,
    "description": "Profile metadata",
    "count": 150,
    "queue_name": "nostr_kind_0"
  },
  "kind_1": {
    "kind": 1,
    "description": "Short notes/text posts",
    "count": 2500,
    "queue_name": "nostr_kind_1"
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

## Quality Control

### Event Validation

All events undergo basic validation before routing:

1. **Required Fields**: ID, PubKey, Signature must be present
2. **Timestamp Validation**: Created timestamp must be within reasonable bounds
3. **Kind Validation**: Kind must be a valid integer (0-65535)
4. **Content Validation**: Content must meet kind-specific requirements

### Moderation Queue

Events that fail validation are routed to the `moderation` topic for manual review. This includes:

- Events with missing required fields
- Events with invalid timestamps (too old/future)
- Events with invalid kind numbers
- Events that fail kind-specific validation rules

## Adding New Event Kinds

### Step 1: Create Configuration File

Create a new YAML file in `configs/kinds/` with the kind number as the filename:

```bash
# Example: Adding support for kind 1337 (code snippets)
touch configs/kinds/1337.yml
```

### Step 2: Define Kind Configuration

Edit the file with appropriate configuration:

```yaml
# Kind 1337: Code Snippet
name: "Code Snippet"
description: "Code snippets with syntax highlighting"
required_tags: ["language"]
optional_tags: ["title", "description", "url"]
content_validation:
  type: "text"
  max_length: 50000
  min_length: 10
quality_rules:
  - name: "valid_syntax"
    weight: 0.8
    description: "Code should have valid syntax for the specified language"
  - name: "not_spam"
    weight: 1.0
    description: "Content should not be spam"
  - name: "meaningful_code"
    weight: 0.7
    description: "Code should be meaningful and not just random characters"
replaceable: false
ephemeral: false
```

### Step 3: Restart Relay

Restart the Mercury Relay to pick up the new configuration:

```bash
# If using Docker
docker-compose restart mercury-relay

# If running directly
./mercury-relay
```

The new kind will be automatically detected and a corresponding topic will be created.

## Monitoring and Debugging

### Viewing Kind Statistics

Use the REST API to monitor kind statistics:

```bash
# Get all kind statistics
curl -H "Authorization: Nostr <auth_token>" \
     http://localhost:8082/api/v1/kind/stats

# Get statistics for a specific kind
curl -H "Authorization: Nostr <auth_token>" \
     http://localhost:8082/api/v1/kind/1/stats
```

### RabbitMQ Management

Access the RabbitMQ management interface to view topic details:

1. Open `http://localhost:15672` in your browser
2. Login with `guest/guest`
3. Navigate to "Queues" tab
4. Look for queues starting with `nostr_kind_`

### Logs

Monitor relay logs for kind-based routing information:

```bash
# View logs
docker-compose logs -f mercury-relay

# Filter for kind-related logs
docker-compose logs mercury-relay | grep "kind"
```

## Best Practices

### Configuration Management

1. **Version Control**: Keep all kind configuration files in version control
2. **Documentation**: Document each kind's purpose and requirements
3. **Testing**: Test new kinds with sample events before deployment
4. **Backup**: Backup configuration files before making changes

### Performance Considerations

1. **Kind Limits**: Avoid creating too many kind topics (recommended max: 100)
2. **Memory Usage**: Each topic consumes memory; monitor usage
3. **Queue Depth**: Monitor queue depths to prevent memory issues
4. **Cleanup**: Regularly clean up old events from queues

### Security

1. **Validation**: Ensure all validation rules are properly configured
2. **Moderation**: Regularly review the moderation queue
3. **Access Control**: Secure API endpoints with proper authentication
4. **Monitoring**: Set up alerts for unusual activity in any topic

## Troubleshooting

### Common Issues

**New kind not detected:**
- Check file naming: must be `{kind_number}.yml`
- Verify YAML syntax is valid
- Restart the relay after adding the file

**Events not routing correctly:**
- Check event validation rules
- Verify kind number is correct
- Review logs for validation errors

**High moderation queue count:**
- Review validation rules for false positives
- Check for systematic issues with event structure
- Consider adjusting validation thresholds

### Debug Commands

```bash
# Check if kind configuration files exist
ls -la configs/kinds/

# Validate YAML syntax
yamllint configs/kinds/*.yml

# Check RabbitMQ topics
rabbitmqctl list_queues | grep nostr_kind

# View relay configuration
./mercury-relay --config-dump
```

## Future Enhancements

Planned improvements to the kind-based filtering system:

1. **Dynamic Reloading**: Hot-reload kind configurations without restart
2. **Advanced Validation**: More sophisticated content validation rules
3. **Analytics**: Detailed analytics for each kind topic
4. **Auto-scaling**: Automatic topic scaling based on load
5. **Integration**: Better integration with external moderation tools
