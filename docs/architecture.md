# Architecture Guide

## System-First Design Philosophy

WombatWisdom Components implements a **System-first architecture** that addresses a key limitation in traditional component designs: connection proliferation.

### Traditional Component Architecture Problems

```go
// Traditional approach - each component creates its own connection
input1 := nats.NewInput(config1)    // Connection 1
input2 := nats.NewInput(config2)    // Connection 2  
output := nats.NewOutput(config3)   // Connection 3
cache := nats.NewCache(config4)     // Connection 4
// Result: 4 connections to the same NATS server!
```

**Issues:**
- Resource waste (multiple connections to same service)
- Configuration duplication
- Connection management complexity
- Harder to monitor and debug

### System-First Solution

```go
// System-first approach - shared connection management
system, err := nats.NewSystem(systemConfig)
system.Connect(ctx)

// All components share the same underlying connection
input1 := nats.NewInput(system, env, inputConfig1)
input2 := nats.NewInput(system, env, inputConfig2) 
output := nats.NewOutput(system, env, outputConfig)
cache := nats.NewCache(system, env, cacheConfig)
// Result: 1 shared connection, better resource usage
```

**Benefits:**
- **Resource Efficiency**: Single connection per service
- **Simplified Configuration**: Connection details in one place
- **Better Observability**: Centralized connection monitoring
- **Easier Testing**: Mock the system, not individual connections

## Core Interfaces

### System Interface

The `System` interface manages the lifecycle of connections:

```go
type System interface {
    Connect(ctx context.Context) error
    Close(ctx context.Context) error
    Client() any // Returns underlying client (e.g., *nats.Conn)
}
```

**Implementation Pattern:**
```go
type MySystem struct {
    cfg    SystemConfig
    client *service.Client
}

func (s *MySystem) Connect(ctx context.Context) error {
    client, err := service.Connect(s.cfg.URL, s.cfg.Options...)
    if err != nil {
        return err
    }
    s.client = client
    return nil
}

func (s *MySystem) Client() any {
    return s.client
}
```

### Component Interfaces

Components depend on a `System` rather than managing connections:

```go
// Input components read data
type Input interface {
    Connect(ctx context.Context) error
    Read(ctx context.Context) (Batch, error)
    Disconnect(ctx context.Context) error
}

// Output components write data  
type Output interface {
    Connect(ctx context.Context) error
    Write(ctx context.Context, batch Batch) error
    Disconnect(ctx context.Context) error
}
```

**Component Constructor Pattern:**
```go
func NewInput(system *MySystem, env Environment, config InputConfig) *Input {
    return &Input{
        system: system,
        config: config,
        log:    env,
    }
}
```

### Modern Message Interface

Uses Go 1.21+ `iter.Seq2` for efficient metadata iteration:

```go
type Message interface {
    // Core data access
    SetMetadata(key string, value any)
    SetRaw(b []byte)
    Raw() ([]byte, error)
    
    // Modern metadata iteration
    Metadata() iter.Seq2[string, any]
}
```

**Usage Example:**
```go
// Iterate over all metadata efficiently
for key, value := range message.Metadata() {
    fmt.Printf("%s: %v\n", key, value)
}
```

## Configuration Architecture

### Hierarchical Configuration

```
System Configuration (connection details)
├── Input Configuration (input-specific settings)
├── Output Configuration (output-specific settings)  
└── Cache Configuration (cache-specific settings)
```

**Example:**
```yaml
# System level - shared connection
system:
  servers: ["nats://localhost:4222"]
  auth:
    jwt: ${JWT_TOKEN}
    seed: ${NKEY_SEED}

# Component level - specific behavior  
input:
  subject: "orders.*"
  queue_group: "processors"
  
output:
  subject: "processed.orders"
  timeout: 5s
```

### Schema-Driven Configuration

Each component defines JSON schemas for validation:

```yaml
# system_config.schema.yaml
$schema: "https://json-schema.org/draft/2020-12/schema"
title: "NATS System Configuration"
type: object
properties:
  servers:
    type: array
    items:
      type: string
      format: uri
  auth:
    type: object
    properties:
      jwt:
        type: string
      seed:
        type: string
required: [servers]
```

## Benthos Integration

### Factory Pattern

Components implement factories for Benthos integration:

```go
type ComponentFactory interface {
    NewInput(conf Config, env Environment) (Input, error)
    NewOutput(conf Config, env Environment) (Output, error)
    NewCache(conf Config, env Environment) (Cache, error)
}

type SystemFactory interface {
    NewSystem(conf Config) (System, error)
}
```

### Resource Management

```go
type ResourceManager interface {
    // Benthos service.Resources compatibility
    GetSystem(name string) (System, error)
    SetSystem(name string, system System)
    GetMetrics() Metrics
    GetLogger() Logger
}
```

## Error Handling Strategy

### Hierarchical Error Handling

1. **System Level**: Connection failures, authentication errors
2. **Component Level**: Configuration validation, operation failures  
3. **Message Level**: Individual message processing errors

```go
// Custom error types for different levels
type SystemError struct {
    Operation string
    Err       error
}

type ComponentError struct {
    Component string
    Config    string
    Err       error
}
```

### Retry and Circuit Breaking

```go
type RetryConfig struct {
    MaxAttempts int           `json:"max_attempts"`
    Backoff     time.Duration `json:"backoff"`
    MaxBackoff  time.Duration `json:"max_backoff"`
}

type CircuitConfig struct {
    Threshold   int           `json:"failure_threshold"`
    Timeout     time.Duration `json:"timeout"`
    MaxRequests int           `json:"max_requests"`
}
```

## Testing Architecture

### System Mocking

```go
type MockSystem struct {
    connected bool
    client    any
    errors    map[string]error
}

func (m *MockSystem) Connect(ctx context.Context) error {
    if err := m.errors["connect"]; err != nil {
        return err
    }
    m.connected = true
    return nil
}
```

### Component Testing

```go
var _ = Describe("Input", func() {
    var (
        system *MockSystem
        input  *Input
    )
    
    BeforeEach(func() {
        system = NewMockSystem()
        input = NewInput(system, env, config)
    })
    
    When("system is connected", func() {
        BeforeEach(func() {
            err := system.Connect(ctx)
            Expect(err).ToNot(HaveOccurred())
        })
        
        It("should read messages", func() {
            batch, err := input.Read(ctx)
            Expect(err).ToNot(HaveOccurred())
            Expect(batch).ToNot(BeNil())
        })
    })
})
```

## Performance Considerations

### Connection Pooling

```go
type PooledSystem struct {
    pool *sync.Pool
    cfg  SystemConfig
}

func (p *PooledSystem) GetClient() *Client {
    return p.pool.Get().(*Client)
}

func (p *PooledSystem) PutClient(client *Client) {
    p.pool.Put(client)
}
```

### Batch Processing

```go
type Batch interface {
    Messages() []Message
    Add(msg Message)
    Len() int
    Clear()
}

// Efficient batch processing
func (o *Output) WriteBatch(ctx context.Context, batch Batch) error {
    // Process multiple messages in a single operation
    return o.system.Client().PublishBatch(batch.Messages())
}
```

## Monitoring and Observability

### Metrics Integration

```go
type Metrics interface {
    Counter(name string) Counter
    Gauge(name string) Gauge  
    Histogram(name string) Histogram
}

// Usage in components
func (i *Input) Read(ctx context.Context) (Batch, error) {
    start := time.Now()
    defer i.metrics.Histogram("input.read.duration").Observe(time.Since(start))
    
    i.metrics.Counter("input.read.total").Inc()
    
    // ... implementation
}
```

### Structured Logging

```go
type Logger interface {
    Debugf(format string, args ...interface{})
    Infof(format string, args ...interface{})
    Warnf(format string, args ...interface{})
    Errorf(format string, args ...interface{})
}

// Context-aware logging
func (i *Input) processMessage(msg Message) error {
    i.log.Debugf("Processing message from %s", msg.Source())
    
    if err := i.validate(msg); err != nil {
        i.log.Warnf("Message validation failed: %v", err)
        return err
    }
    
    return nil
}
```

This architecture provides a solid foundation for building scalable, maintainable, and efficient data processing components while maintaining compatibility with Benthos and modern Go practices.