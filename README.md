# WombatWisdom Components

A collection of reusable, well-tested Go components for building data processing pipelines with a System-first architecture designed for [Benthos](https://github.com/redpanda-data/benthos) compatibility.

[![CI](https://github.com/wombatwisdom/components/actions/workflows/ci.yml/badge.svg)](https://github.com/wombatwisdom/components/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wombatwisdom/components)](https://goreportcard.com/report/github.com/wombatwisdom/components)
[![codecov](https://codecov.io/gh/wombatwisdom/components/branch/main/graph/badge.svg)](https://codecov.io/gh/wombatwisdom/components)

## 🚀 Quick Start

```bash
# Install Task (task runner)
brew install go-task/tap/go-task

# Clone and setup
git clone https://github.com/wombatwisdom/components.git
cd components
task setup

# Run tests
task test

# Check status
task status
```

## ✨ Features

- **System-first Architecture**: Shared connection management across components
- **Benthos Compatible**: Designed to integrate seamlessly with Benthos pipelines
- **Modern Go**: Uses Go 1.24+ features including `iter.Seq2` for metadata
- **Comprehensive Testing**: 30+ tests with Ginkgo v2 and Gomega
- **Developer Experience**: Component generators, CI/CD, and automation tools
- **Production Ready**: Includes monitoring, logging, and error handling

## 📦 Components

| Component | Status | Description |
|-----------|--------|-------------|
| **spec** | ✅ Ready | Core interfaces and contracts |
| **nats/core** | ✅ Ready | NATS messaging system |
| **mqtt** | ✅ Ready | MQTT pub/sub components |
| **test** | ✅ Ready | Testing utilities and helpers |
| **aws/s3** | ⚠️ Partial | S3 storage components |

## 🏗️ Architecture

### System-First Design

Unlike traditional component-per-connection approaches, WombatWisdom Components uses a **System-first architecture**:

```go
// Create shared system
system, err := nats.NewSystem(config)
system.Connect(ctx)

// Multiple components share the same connection
input := nats.NewInput(system, env, inputConfig)
output := nats.NewOutput(system, env, outputConfig)
cache := nats.NewCache(system, env, cacheConfig)
```

**Benefits:**
- Reduced connection overhead
- Better resource management  
- Simplified configuration
- Enhanced observability

### Core Interfaces

```go
// System manages connections and resources
type System interface {
    Connect(ctx context.Context) error
    Close(ctx context.Context) error
    Client() any
}

// Modern message interface with iter.Seq2
type Message interface {
    SetMetadata(key string, value any)
    SetRaw(b []byte)
    Raw() ([]byte, error)
    Metadata() iter.Seq2[string, any]
}
```

## 🛠️ Development

### Creating New Components

```bash
# Generate a new component
task generate:component redis

# Follow the prompts to configure:
# - Service name: Redis
# - Description: Redis pub/sub and caching
# - Client type: *redis.Client
# - Configuration examples

# Implement the generated TODOs
cd redis
task models:generate
task test
```

### Available Commands

```bash
# Development
task test              # Run core tests
task test:all          # Run all tests (may fail on infrastructure)
task ci:test           # Full CI pipeline
task build             # Build working packages
task lint              # Run linters
task format            # Format code

# Project Management  
task status            # Show component status
task setup             # Setup development environment
task clean             # Clean caches
task deps:tidy         # Tidy dependencies

# Component Tools
task generate:component <name>  # Generate new component
task nats:schema:generate      # Generate NATS schemas
```

### Setting up IBM MQ Client Libraries

To build and test with actual IBM MQ support (using the `mqclient` tag), you need the IBM MQ client libraries.

#### Option 1: Download IBM MQ Redistributable Client

1. Download the IBM MQ redistributable client from IBM Fix Central:
    - Visit [IBM Fix Central](https://www.ibm.com/support/fixcentral/)
    - Search for "IBM MQ redistributable client"
    - Download the appropriate version (e.g., `9.3.0.0-IBM-MQC-Redist-LinuxX64.tar.gz`)
    - Or use a direct link (linux): [9.4.1.0-IBM-MQC-Redist-LinuxX64.tar.gz](https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.4.1.0-IBM-MQC-Redist-LinuxX64.tar.gz)

More info can be found at [developer.ibm.com](https://developer.ibm.com/components/ibm-mq/)

2. Extract to a local directory:
```bash
mkdir -p ~/mqclient
tar -xzf 9.3.0.0-IBM-MQC-Redist-LinuxX64.tar.gz -C ~/mqclient/
```

3. Set environment variables:
```bash
export MQ_HOME="$HOME/mqclient"
export CGO_CFLAGS="-I${MQ_HOME}/inc"
export CGO_LDFLAGS="-L${MQ_HOME}/lib64 -Wl,-rpath=${MQ_HOME}/lib64"
```

#### Option 2: Use Docker Container for Testing

Run tests in a container with IBM MQ pre-installed: 
```bash
task bundles:ibm-mq:test_container
```

Note that this requires elevated permissions to support docker-in-docker.

### Building with IBM MQ support

**Without IBM MQ client (stub implementation):**
```bash
go build ./...
```
This is the default build mode and doesn't require any IBM MQ libraries.

**With IBM MQ client support:**
```bash
# Ensure environment variables are set (see setup instructions above)
export CGO_ENABLED=1
export CGO_CFLAGS="-I${MQ_HOME}/inc"
export CGO_LDFLAGS="-L${MQ_HOME}/lib64 -Wl,-rpath=${MQ_HOME}/lib64"

# Build with mqclient tag
go build -tags mqclient ./...
```

### Run IBM MQ tests 

```bash
# Ensure MQ libraries are set up (see above)
export CGO_ENABLED=1
export CGO_CFLAGS="-I${MQ_HOME}/inc"
export CGO_LDFLAGS="-L${MQ_HOME}/lib64 -Wl,-rpath=${MQ_HOME}/lib64"

# Run tests with mqclient tag
task test:all
```

# Set MQSERVER for tests
export MQSERVER='DEV.APP.SVRCONN/TCP/localhost(1414)'

# Run tests
go test -tags mqclient ./...

# Clean up
docker stop mq-test
```

## 📖 Usage Examples

### NATS Pub/Sub

```go
package main

import (
    "context"
    "github.com/wombatwisdom/components/nats/core"
    "github.com/wombatwisdom/components/spec"
)

func main() {
    // Create system
    config := spec.NewYamlConfig(`
servers: [nats://localhost:4222]
`)
    system, err := core.NewSystemFromConfig(config)
    if err != nil {
        panic(err)
    }
    
    defer system.Close(context.Background())
    
    // Connect
    if err := system.Connect(context.Background()); err != nil {
        panic(err)
    }
    
    // Create input and output sharing the same connection
    input := core.NewInput(system, env, core.InputConfig{
        Subject: "orders.*",
    })
    
    output := core.NewOutput(system, env, core.OutputConfig{
        Subject: "processed.orders",
    })
}
```

### MQTT Components

```go
// MQTT source
source, err := mqtt.NewSource(env, mqtt.SourceConfig{
    CommonMQTTConfig: mqtt.CommonMQTTConfig{
        Urls:     []string{"tcp://localhost:1883"},
        ClientId: "consumer",
    },
    Filters: map[string]byte{"sensors/+": 1},
})

// MQTT sink
sink, err := mqtt.NewSink(env, mqtt.SinkConfig{
    CommonMQTTConfig: mqtt.CommonMQTTConfig{
        Urls:     []string{"tcp://localhost:1883"},
        ClientId: "publisher", 
    },
    TopicExpr: "processed/{{.metadata.sensor_id}}",
})
```

## 🔧 Testing

The project uses [Ginkgo v2](https://github.com/onsi/ginkgo) for BDD-style testing:

```go
var _ = Describe("Component", func() {
    When("valid configuration is provided", func() {
        It("should connect successfully", func() {
            system, err := NewSystem(validConfig)
            Expect(err).ToNot(HaveOccurred())
            
            err = system.Connect(ctx)
            Expect(err).ToNot(HaveOccurred())
        })
    })
})
```

Run tests:
```bash
task test          # Core functionality
task test:coverage # With coverage report
task test:all      # All tests (may have infrastructure deps)
```

## 🚀 CI/CD

GitHub Actions workflows provide:

- **Continuous Integration**: Tests across Go 1.21, 1.22, 1.23
- **Code Quality**: Linting, formatting, security scanning
- **Dependency Management**: Automated Dependabot updates
- **Release Automation**: Semantic versioning and changelog generation

## 🤝 Contributing

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/amazing-feature`
3. **Generate** component if needed: `task generate:component myservice`
4. **Implement** your changes with tests
5. **Test** your changes: `task ci:test`  
6. **Commit** with conventional commits: `feat: add redis component`
7. **Push** and create a **Pull Request**

### Development Guidelines

- Follow the [System-first architecture](docs/architecture.md)
- Write comprehensive tests with Ginkgo/Gomega
- Use conventional commit messages
- Update documentation for new features
- Ensure CI passes before submitting PRs

## 📚 Documentation

- [Architecture Guide](docs/architecture.md) - System-first design principles
- [Component Development](docs/component-development.md) - Creating new components
- [Testing Guide](docs/testing.md) - Testing patterns and practices
- [Benthos Integration](docs/benthos-integration.md) - Using with Benthos pipelines

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Benthos](https://github.com/redpanda-data/benthos) for inspiration and compatibility
- [Ginkgo](https://github.com/onsi/ginkgo) and [Gomega](https://github.com/onsi/gomega) for excellent testing tools
- [Task](https://taskfile.dev) for powerful automation