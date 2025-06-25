# Component Development Guide

Learn how to create new components for the WombatWisdom Components ecosystem.

## Quick Start

Generate a new component using the built-in generator:

```bash
task generate:component redis
```

Follow the interactive prompts:
- **Service name**: Redis
- **Description**: Redis pub/sub and caching component
- **Client type**: `*redis.Client`
- **Configuration examples**: URL, auth details

This creates a complete component structure with:
- System implementation
- Input/Output components  
- JSON schemas
- Tests
- Taskfile automation

## Component Structure

```
redis/
├── system.go              # System implementation
├── input.go               # Input component
├── output.go              # Output component  
├── system_test.go         # System tests
├── redis_suite_test.go    # Test suite
├── Taskfile.yml           # Component tasks
├── system_config.schema.yaml
├── input_config.schema.yaml
└── output_config.schema.yaml
```

## Implementation Steps

### 1. System Implementation

The system manages the connection lifecycle:

```go
package redis

import (
    "context"
    "github.com/go-redis/redis/v8"
    "github.com/wombatwisdom/components/spec"
)

type System struct {
    cfg    SystemConfig
    client *redis.Client
}

func NewSystemFromConfig(config spec.Config) (*System, error) {
    var cfg SystemConfig
    if err := config.Decode(&cfg); err != nil {
        return nil, err
    }
    
    return &System{cfg: cfg}, nil
}

func (s *System) Connect(ctx context.Context) error {
    opts := &redis.Options{
        Addr:     s.cfg.Addr,
        Password: s.cfg.Password,
        DB:       s.cfg.DB,
    }
    
    s.client = redis.NewClient(opts)
    
    // Test connection
    return s.client.Ping(ctx).Err()
}

func (s *System) Close(ctx context.Context) error {
    if s.client != nil {
        return s.client.Close()
    }
    return nil
}

func (s *System) Client() any {
    return s.client
}
```

### 2. Input Component

Reads data from the service:

```go
type Input struct {
    system *System
    config InputConfig
    log    spec.Logger
}

func NewInput(system *System, env spec.Environment, config InputConfig) (*Input, error) {
    return &Input{
        system: system,
        config: config,
        log:    env,
    }, nil
}

func (i *Input) Connect(ctx context.Context) error {
    // Input-specific connection setup
    return nil
}

func (i *Input) Read(ctx context.Context) (spec.Batch, error) {
    client := i.system.Client().(*redis.Client)
    
    // Example: BLPOP from Redis list
    result, err := client.BLPop(ctx, i.config.Timeout, i.config.Key).Result()
    if err != nil {
        return nil, err
    }
    
    // Create message from result
    msg := spec.NewBytesMessage([]byte(result[1]))
    msg.SetMetadata("redis.key", result[0])
    
    // Return batch with single message
    batch := spec.NewBatch()
    batch.Add(msg)
    return batch, nil
}
```

### 3. Output Component

Writes data to the service:

```go
type Output struct {
    system *System
    config OutputConfig
    log    spec.Logger
}

func NewOutput(system *System, env spec.Environment, config OutputConfig) (*Output, error) {
    return &Output{
        system: system,
        config: config,
        log:    env,
    }, nil
}

func (o *Output) Write(ctx context.Context, batch spec.Batch) error {
    client := o.system.Client().(*redis.Client)
    
    pipe := client.Pipeline()
    
    for _, msg := range batch.Messages() {
        data, err := msg.Raw()
        if err != nil {
            return err
        }
        
        // Example: RPUSH to Redis list
        pipe.RPush(ctx, o.config.Key, data)
    }
    
    _, err := pipe.Exec(ctx)
    return err
}
```

### 4. Configuration Schemas

Define JSON schemas for validation:

```yaml
# system_config.schema.yaml
$schema: "https://json-schema.org/draft/2020-12/schema"
title: "Redis System Configuration"
type: object
properties:
  addr:
    type: string
    description: "Redis server address"
    default: "localhost:6379"
  password:
    type: string
    description: "Redis password"
  db:
    type: integer
    description: "Redis database number"
    default: 0
  pool_size:
    type: integer
    description: "Connection pool size"
    default: 10
required: [addr]
```

```yaml
# input_config.schema.yaml  
$schema: "https://json-schema.org/draft/2020-12/schema"
title: "Redis Input Configuration"
type: object
properties:
  key:
    type: string
    description: "Redis key to read from"
  timeout:
    type: string
    description: "Blocking timeout duration"
    default: "5s"
required: [key]
```

### 5. Generate Configuration Structs

```bash
cd redis
task models:generate
```

This generates Go structs from your schemas:

```go
// system_config.go (generated)
type SystemConfig struct {
    Addr     string `json:"addr"`
    Password string `json:"password,omitempty"`
    DB       int    `json:"db,omitempty"`
    PoolSize int    `json:"pool_size,omitempty"`
}
```

### 6. Write Tests

Use Ginkgo/Gomega for BDD-style tests:

```go
var _ = Describe("Redis System", func() {
    var (
        system *redis.System
        ctx    context.Context
    )
    
    BeforeEach(func() {
        ctx = context.Background()
    })
    
    When("valid configuration is provided", func() {
        BeforeEach(func() {
            config := spec.NewYamlConfig(`
addr: localhost:6379
db: 0
`)
            var err error
            system, err = redis.NewSystemFromConfig(config)
            Expect(err).ToNot(HaveOccurred())
        })
        
        It("should connect successfully", func() {
            err := system.Connect(ctx)
            Expect(err).ToNot(HaveOccurred())
            
            defer func() {
                err := system.Close(ctx)
                Expect(err).ToNot(HaveOccurred())
            }()
            
            client := system.Client().(*goredis.Client)
            Expect(client).ToNot(BeNil())
        })
    })
})
```

## Best Practices

### Error Handling

```go
// Define component-specific errors
type RedisError struct {
    Operation string
    Key       string
    Err       error
}

func (e *RedisError) Error() string {
    return fmt.Sprintf("redis %s failed for key %s: %v", e.Operation, e.Key, e.Err)
}

// Use in components
func (i *Input) Read(ctx context.Context) (spec.Batch, error) {
    result, err := client.BLPop(ctx, timeout, key).Result()
    if err != nil {
        if err == redis.Nil {
            return spec.EmptyBatch(), nil // No data available
        }
        return nil, &RedisError{
            Operation: "blpop",
            Key:       key,
            Err:       err,
        }
    }
    // ...
}
```

### Configuration Validation

```go
func (c *InputConfig) Validate() error {
    if c.Key == "" {
        return errors.New("key is required")
    }
    
    if c.Timeout <= 0 {
        return errors.New("timeout must be positive")
    }
    
    return nil
}

func NewInput(system *System, env spec.Environment, config InputConfig) (*Input, error) {
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("invalid input config: %w", err)
    }
    
    return &Input{...}, nil
}
```

### Metrics and Logging

```go
type Input struct {
    system  *System
    config  InputConfig
    log     spec.Logger
    metrics spec.Metrics
}

func (i *Input) Read(ctx context.Context) (spec.Batch, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        i.metrics.Histogram("redis.input.read.duration").Observe(duration)
    }()
    
    i.metrics.Counter("redis.input.read.total").Inc()
    i.log.Debugf("Reading from Redis key: %s", i.config.Key)
    
    // Implementation...
    
    i.metrics.Counter("redis.input.messages.total").Add(float64(batch.Len()))
    i.log.Infof("Read %d messages from Redis", batch.Len())
    
    return batch, nil
}
```

### Resource Management

```go
func (s *System) Connect(ctx context.Context) error {
    opts := &redis.Options{
        Addr:         s.cfg.Addr,
        PoolSize:     s.cfg.PoolSize,
        MinIdleConns: s.cfg.PoolSize / 4,
        MaxRetries:   3,
        
        // Proper timeouts
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        
        // Connection health checks
        PoolTimeout: 4 * time.Second,
        IdleTimeout: 300 * time.Second,
    }
    
    s.client = redis.NewClient(opts)
    
    // Test connection with timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    return s.client.Ping(ctx).Err()
}
```

## Testing Strategies

### Unit Testing

```go
var _ = Describe("Input", func() {
    var (
        mockSystem *MockRedisSystem
        input      *Input
    )
    
    BeforeEach(func() {
        mockSystem = NewMockRedisSystem()
        input = NewInput(mockSystem, env, InputConfig{
            Key:     "test-key",
            Timeout: time.Second,
        })
    })
    
    When("Redis returns data", func() {
        BeforeEach(func() {
            mockSystem.SetBLPopResult([]string{"test-key", "test-data"})
        })
        
        It("should return a batch with the message", func() {
            batch, err := input.Read(ctx)
            Expect(err).ToNot(HaveOccurred())
            Expect(batch.Len()).To(Equal(1))
            
            msg := batch.Messages()[0]
            data, err := msg.Raw()
            Expect(err).ToNot(HaveOccurred())
            Expect(string(data)).To(Equal("test-data"))
        })
    })
})
```

### Integration Testing

```go
var _ = Describe("Redis Integration", func() {
    var (
        redisContainer testcontainers.Container
        system         *System
    )
    
    BeforeEach(func() {
        // Start Redis container for testing
        ctx := context.Background()
        req := testcontainers.ContainerRequest{
            Image:        "redis:7-alpine",
            ExposedPorts: []string{"6379/tcp"},
        }
        
        var err error
        redisContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
            ContainerRequest: req,
            Started:          true,
        })
        Expect(err).ToNot(HaveOccurred())
        
        // Get container connection details
        host, err := redisContainer.Host(ctx)
        Expect(err).ToNot(HaveOccurred())
        
        port, err := redisContainer.MappedPort(ctx, "6379")
        Expect(err).ToNot(HaveOccurred())
        
        // Create system with container connection
        config := spec.NewYamlConfig(fmt.Sprintf(`
addr: %s:%s
db: 0
`, host, port.Port()))
        
        system, err = NewSystemFromConfig(config)
        Expect(err).ToNot(HaveOccurred())
        
        err = system.Connect(ctx)
        Expect(err).ToNot(HaveOccurred())
    })
    
    AfterEach(func() {
        if system != nil {
            system.Close(context.Background())
        }
        if redisContainer != nil {
            redisContainer.Terminate(context.Background())
        }
    })
    
    It("should perform roundtrip operations", func() {
        // Test actual Redis operations
    })
})
```

## Component Lifecycle

### Startup Sequence

1. **System Creation**: `NewSystemFromConfig()`
2. **System Connection**: `system.Connect(ctx)`  
3. **Component Creation**: `NewInput(system, env, config)`
4. **Component Connection**: `input.Connect(ctx)`
5. **Operation**: `input.Read(ctx)` / `output.Write(ctx, batch)`

### Shutdown Sequence

1. **Component Disconnect**: `input.Disconnect(ctx)`
2. **System Close**: `system.Close(ctx)`
3. **Resource Cleanup**: Connections, pools, etc.

### Error Recovery

```go
type ResilientInput struct {
    *Input
    retryConfig RetryConfig
}

func (r *ResilientInput) Read(ctx context.Context) (spec.Batch, error) {
    var lastErr error
    
    for attempt := 0; attempt < r.retryConfig.MaxAttempts; attempt++ {
        batch, err := r.Input.Read(ctx)
        if err == nil {
            return batch, nil
        }
        
        lastErr = err
        
        // Check if error is retryable
        if !isRetryableError(err) {
            break
        }
        
        // Exponential backoff
        backoff := r.retryConfig.BaseDelay * time.Duration(1<<attempt)
        if backoff > r.retryConfig.MaxDelay {
            backoff = r.retryConfig.MaxDelay
        }
        
        select {
        case <-time.After(backoff):
            continue
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    return nil, fmt.Errorf("failed after %d attempts: %w", r.retryConfig.MaxAttempts, lastErr)
}
```

## Performance Optimization

### Batch Processing

```go
func (o *Output) Write(ctx context.Context, batch spec.Batch) error {
    client := o.system.Client().(*redis.Client)
    
    // Use pipeline for batch operations
    pipe := client.Pipeline()
    
    for _, msg := range batch.Messages() {
        data, err := msg.Raw()
        if err != nil {
            return err
        }
        
        pipe.RPush(ctx, o.config.Key, data)
    }
    
    // Execute all commands in a single round-trip
    _, err := pipe.Exec(ctx)
    return err
}
```

### Connection Pooling

```go
type PooledSystem struct {
    System
    pool *redis.Ring // Use Redis Ring for horizontal scaling
}

func (p *PooledSystem) Connect(ctx context.Context) error {
    p.pool = redis.NewRing(&redis.RingOptions{
        Addrs: p.cfg.Addrs,
        // Pool configuration
        PoolSize:     p.cfg.PoolSize,
        MinIdleConns: p.cfg.MinIdleConns,
    })
    
    return p.pool.Ping(ctx).Err()
}
```

This guide provides a comprehensive foundation for developing high-quality, production-ready components in the WombatWisdom ecosystem.