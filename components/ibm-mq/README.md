# IBM MQ Component for WombatWisdom

This component provides IBM MQ input and output capabilities.

## Overview

The IBM MQ component implements the WombatWisdom architecture pattern with:

- **System**: Manages IBM MQ queue manager connections with TLS and authentication support
- **Input**: Reads messages from IBM MQ queues with batching and parallel processing
- **Output**: Writes messages to IBM MQ queues with transaction support and metadata handling

## Prerequisites

### IBM MQ Client Libraries

The IBM MQ Go client requires the IBM MQ client libraries to be installed on your system. This includes:

- IBM MQ client runtime libraries
- IBM MQ client development headers (cmqc.h, etc.)

**Installation Options:**

1. **IBM MQ Client (Redistributable)**: Download from IBM website
2. **IBM MQ Server**: Full server installation includes client libraries
3. **Container Images**: Use IBM MQ container images with client libraries pre-installed

### System Dependencies

- Go 1.24.1 or later
- CGO enabled (required for IBM MQ Go client)
- IBM MQ client libraries (version 9.0+)

## Configuration

### System Configuration

```yaml
queue_manager_name: QM1              # Required: Queue manager name
channel_name: USER.CHANNEL           # Optional: Channel name (default: SYSTEM.DEF.SVRCONN)
connection_name: localhost(1414)     # Optional: Connection string (default: localhost(1414))
user_id: myuser                     # Optional: Username for authentication
password: mypassword                # Optional: Password for authentication
application_name: MyApp             # Optional: Application identifier

# Optional TLS configuration
tls:
  enabled: true
  cipher_spec: ANY_TLS12_OR_HIGHER
  key_repository: /path/to/keystore
  key_repository_password: keystorepass
  certificate_label: mycert
```

### Input Configuration

```yaml
queue_name: INPUT_QUEUE              # Required: Queue to read from
batch_count: 10                     # Optional: Messages per batch (default: 1)
num_threads: 4                      # Optional: Parallel connections (default: 1)
wait_time: 5s                       # Optional: Wait time for messages (default: 5s)
sleep_time_before_exit_after_failure: 2m  # Optional: Shutdown delay (default: 2m)
auto_retry_nacks: true              # Optional: Retry failed messages (default: true)
```

### Output Configuration

```yaml
queue_name: OUTPUT_QUEUE             # Required: Queue to write to
num_threads: 4                      # Optional: Parallel connections (default: 1)
ccsid: "1208"                       # Optional: Character set (default: "1208")
encoding: "546"                     # Optional: Data encoding (default: "546")
format: MQSTR                       # Optional: Message format (default: MQSTR)

# Optional metadata filtering
metadata:
  patterns: ["^app_.*"]             # Only include metadata starting with "app_"
  invert: false                     # false = include matching, true = exclude matching
```

## Usage Example

```go
package main

import (
    "context"
    "github.com/wombatwisdom/components/framework/spec"
    "github.com/wombatwisdom/components/mq"
)

func main() {
    // Create system
    systemConfig := spec.NewYamlConfig(`
queue_manager_name: QM1
channel_name: USER.CHANNEL
connection_name: localhost(1414)
user_id: myuser
password: mypassword
`)
    
    system, err := mq.NewSystemFromConfig(systemConfig)
    if err != nil {
        panic(err)
    }
    
    // Connect
    ctx := context.Background()
    if err := system.Connect(ctx); err != nil {
        panic(err)
    }
    defer system.Close(ctx)
    
    // Create input
    inputConfig := spec.NewYamlConfig(`
queue_name: TEST_INPUT
batch_count: 5
`)
    
    input, err := mq.NewInputFromConfig(system, inputConfig)
    if err != nil {
        panic(err)
    }
    
    // Initialize and use...
}
```

## Features

### Connection Management
- Persistent connections with automatic reconnection
- TLS encryption support with certificate management  
- Username/password and certificate-based authentication
- Connection pooling for high-throughput scenarios

### Message Processing
- Transactional message processing (MQGET/MQPUT with syncpoint)
- Configurable batch sizes for performance optimization
- Parallel processing with multiple queue manager connections
- Automatic message acknowledgment and rollback on errors

### Metadata Support
- MQ message properties mapped to WombatWisdom metadata
- Support for message ID, correlation ID, priority, persistence
- Configurable metadata filtering for outputs
- Format, CCSID, and encoding handling

### Error Handling
- Graceful connection failure handling
- Configurable retry behavior for failed messages
- Proper resource cleanup on shutdown
- Backout queue support (configured on queue manager)

## Development

### Building

Note: Building requires IBM MQ client libraries to be installed.

```bash
task build
```

### Testing

```bash
task test
```

### Schema Generation

Configuration structs are generated from JSON schemas:

```bash
task models:generate
```

## Limitations

- Requires IBM MQ client libraries to be installed
- CGO dependency means cross-compilation requires target system libraries
- Queue manager must be configured with appropriate permissions
- TLS configuration requires proper certificate setup

## Contributing

When contributing to this component:

1. Follow the existing code patterns from other WombatWisdom components
2. Update JSON schemas when adding new configuration options
3. Regenerate configuration structs after schema changes
4. Add appropriate tests for new functionality
5. Update this README for significant changes