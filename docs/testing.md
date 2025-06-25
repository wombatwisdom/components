# Testing Guide

Comprehensive testing strategies for WombatWisdom Components using Ginkgo v2 and Gomega.

## Testing Philosophy

- **BDD Style**: Use Ginkgo's descriptive syntax for clear test intentions
- **Isolated Units**: Test components in isolation with mocked dependencies
- **Integration**: Test real interactions with external services
- **Contract Testing**: Ensure interface compliance across implementations

## Test Structure

### Project Layout

```
component/
├── system.go
├── system_test.go          # System unit tests
├── input.go
├── input_test.go           # Input unit tests
├── output.go
├── output_test.go          # Output unit tests
├── integration_test.go     # Integration tests
└── component_suite_test.go # Test suite setup
```

### Test Suite Setup

```go
// component_suite_test.go
package component_test

import (
    "testing"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestComponent(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Component Suite")
}
```

## Unit Testing Patterns

### System Testing

```go
var _ = Describe("System", func() {
    var (
        system *component.System
        ctx    context.Context
    )
    
    BeforeEach(func() {
        ctx = context.Background()
    })
    
    Describe("Configuration", func() {
        When("invalid configuration is provided", func() {
            It("should return validation error", func() {
                config := spec.NewYamlConfig(`
invalid yaml: [
connection:
  url: not-a-url
`)
                
                system, err := component.NewSystemFromConfig(config)
                Expect(err).To(HaveOccurred())
                Expect(system).To(BeNil())
            })
        })
        
        When("valid configuration is provided", func() {
            BeforeEach(func() {
                config := spec.NewYamlConfig(`
connection:
  url: "service://localhost:1234"
  timeout: 5s
auth:
  username: test
  password: test
`)
                var err error
                system, err = component.NewSystemFromConfig(config)
                Expect(err).ToNot(HaveOccurred())
                Expect(system).ToNot(BeNil())
            })
            
            It("should parse configuration correctly", func() {
                // Test that configuration was parsed properly
                Expect(system.Config().Connection.URL).To(Equal("service://localhost:1234"))
                Expect(system.Config().Connection.Timeout).To(Equal(5 * time.Second))
            })
        })
    })
    
    Describe("Connection Lifecycle", func() {
        BeforeEach(func() {
            config := spec.NewYamlConfig(`
connection:
  url: "mock://localhost"
`)
            var err error
            system, err = component.NewSystemFromConfig(config)
            Expect(err).ToNot(HaveOccurred())
        })
        
        When("connecting to a healthy service", func() {
            It("should establish connection successfully", func() {
                err := system.Connect(ctx)
                Expect(err).ToNot(HaveOccurred())
                
                client := system.Client()
                Expect(client).ToNot(BeNil())
                
                // Always clean up
                defer func() {
                    err := system.Close(ctx)
                    Expect(err).ToNot(HaveOccurred())
                }()
            })
        })
        
        When("service is unavailable", func() {
            BeforeEach(func() {
                config := spec.NewYamlConfig(`
connection:
  url: "service://unreachable:9999"
  timeout: 100ms
`)
                var err error
                system, err = component.NewSystemFromConfig(config)
                Expect(err).ToNot(HaveOccurred())
            })
            
            It("should return connection error", func() {
                err := system.Connect(ctx)
                Expect(err).To(HaveOccurred())
                Expect(err.Error()).To(ContainSubstring("connection"))
            })
        })
    })
})
```

### Input Testing

```go
var _ = Describe("Input", func() {
    var (
        mockSystem *MockSystem
        input      *component.Input
        env        spec.Environment
    )
    
    BeforeEach(func() {
        mockSystem = NewMockSystem()
        env = test.NewEnvironment()
        
        config := component.InputConfig{
            Source:   "test-source",
            BatchSize: 10,
            Timeout:   time.Second,
        }
        
        var err error
        input, err = component.NewInput(mockSystem, env, config)
        Expect(err).ToNot(HaveOccurred())
    })
    
    When("reading from a populated source", func() {
        BeforeEach(func() {
            // Setup mock to return test data
            messages := []spec.Message{
                spec.NewBytesMessage([]byte("message 1")),
                spec.NewBytesMessage([]byte("message 2")),
                spec.NewBytesMessage([]byte("message 3")),
            }
            mockSystem.SetMessages(messages)
        })
        
        It("should return a batch with messages", func() {
            batch, err := input.Read(context.Background())
            Expect(err).ToNot(HaveOccurred())
            Expect(batch).ToNot(BeNil())
            Expect(batch.Len()).To(Equal(3))
            
            messages := batch.Messages()
            Expect(messages).To(HaveLen(3))
            
            data, err := messages[0].Raw()
            Expect(err).ToNot(HaveOccurred())
            Expect(string(data)).To(Equal("message 1"))
        })
    })
    
    When("source is empty", func() {
        BeforeEach(func() {
            mockSystem.SetMessages([]spec.Message{})
        })
        
        It("should return empty batch", func() {
            batch, err := input.Read(context.Background())
            Expect(err).ToNot(HaveOccurred())
            Expect(batch.Len()).To(Equal(0))
        })
    })
    
    When("system returns error", func() {
        BeforeEach(func() {
            mockSystem.SetError("read", errors.New("connection lost"))
        })
        
        It("should propagate the error", func() {
            batch, err := input.Read(context.Background())
            Expect(err).To(HaveOccurred())
            Expect(err.Error()).To(ContainSubstring("connection lost"))
            Expect(batch).To(BeNil())
        })
    })
})
```

### Output Testing

```go
var _ = Describe("Output", func() {
    var (
        mockSystem *MockSystem
        output     *component.Output
    )
    
    BeforeEach(func() {
        mockSystem = NewMockSystem()
        env := test.NewEnvironment()
        
        config := component.OutputConfig{
            Destination: "test-destination",
            BatchSize:   10,
        }
        
        var err error
        output, err = component.NewOutput(mockSystem, env, config)
        Expect(err).ToNot(HaveOccurred())
    })
    
    When("writing a valid batch", func() {
        It("should write all messages successfully", func() {
            batch := spec.NewBatch()
            batch.Add(spec.NewBytesMessage([]byte("message 1")))
            batch.Add(spec.NewBytesMessage([]byte("message 2")))
            
            err := output.Write(context.Background(), batch)
            Expect(err).ToNot(HaveOccurred())
            
            // Verify mock received the messages
            writtenMessages := mockSystem.GetWrittenMessages()
            Expect(writtenMessages).To(HaveLen(2))
        })
    })
    
    When("system write fails", func() {
        BeforeEach(func() {
            mockSystem.SetError("write", errors.New("write failed"))
        })
        
        It("should return the error", func() {
            batch := spec.NewBatch()
            batch.Add(spec.NewBytesMessage([]byte("message")))
            
            err := output.Write(context.Background(), batch)
            Expect(err).To(HaveOccurred())
            Expect(err.Error()).To(ContainSubstring("write failed"))
        })
    })
})
```

## Mock Implementation

### Mock System

```go
type MockSystem struct {
    connected bool
    client    any
    errors    map[string]error
    messages  []spec.Message
    written   []spec.Message
    mu        sync.RWMutex
}

func NewMockSystem() *MockSystem {
    return &MockSystem{
        errors:   make(map[string]error),
        messages: make([]spec.Message, 0),
        written:  make([]spec.Message, 0),
    }
}

func (m *MockSystem) Connect(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if err := m.errors["connect"]; err != nil {
        return err
    }
    
    m.connected = true
    m.client = &MockClient{system: m}
    return nil
}

func (m *MockSystem) Close(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if err := m.errors["close"]; err != nil {
        return err
    }
    
    m.connected = false
    m.client = nil
    return nil
}

func (m *MockSystem) Client() any {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.client
}

// Test helpers
func (m *MockSystem) SetError(operation string, err error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.errors[operation] = err
}

func (m *MockSystem) SetMessages(messages []spec.Message) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.messages = messages
}

func (m *MockSystem) GetWrittenMessages() []spec.Message {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return append([]spec.Message{}, m.written...)
}

func (m *MockSystem) AddWrittenMessage(msg spec.Message) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.written = append(m.written, msg)
}
```

### Mock Client

```go
type MockClient struct {
    system *MockSystem
}

func (c *MockClient) Read(ctx context.Context) ([]spec.Message, error) {
    c.system.mu.RLock()
    defer c.system.mu.RUnlock()
    
    if err := c.system.errors["read"]; err != nil {
        return nil, err
    }
    
    return c.system.messages, nil
}

func (c *MockClient) Write(ctx context.Context, messages []spec.Message) error {
    c.system.mu.Lock()
    defer c.system.mu.Unlock()
    
    if err := c.system.errors["write"]; err != nil {
        return err
    }
    
    for _, msg := range messages {
        c.system.written = append(c.system.written, msg)
    }
    
    return nil
}
```

## Integration Testing

### Test Containers

Use [testcontainers-go](https://golang.testcontainers.org/) for integration tests:

```go
var _ = Describe("Integration", func() {
    var (
        container testcontainers.Container
        system    *component.System
        ctx       context.Context
    )
    
    BeforeEach(func() {
        ctx = context.Background()
        
        // Start service container
        req := testcontainers.ContainerRequest{
            Image:        "redis:7-alpine",
            ExposedPorts: []string{"6379/tcp"},
            WaitingFor:   wait.ForLog("Ready to accept connections"),
        }
        
        var err error
        container, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
            ContainerRequest: req,
            Started:          true,
        })
        Expect(err).ToNot(HaveOccurred())
        
        // Get connection details
        host, err := container.Host(ctx)
        Expect(err).ToNot(HaveOccurred())
        
        port, err := container.MappedPort(ctx, "6379")
        Expect(err).ToNot(HaveOccurred())
        
        // Create system with real connection
        config := spec.NewYamlConfig(fmt.Sprintf(`
connection:
  url: "redis://%s:%s"
  timeout: 5s
`, host, port.Port()))
        
        system, err = component.NewSystemFromConfig(config)
        Expect(err).ToNot(HaveOccurred())
        
        err = system.Connect(ctx)
        Expect(err).ToNot(HaveOccurred())
    })
    
    AfterEach(func() {
        if system != nil {
            system.Close(ctx)
        }
        if container != nil {
            container.Terminate(ctx)
        }
    })
    
    It("should perform end-to-end operations", func() {
        // Create input and output
        input := component.NewInput(system, env, component.InputConfig{
            Source: "test-queue",
        })
        
        output := component.NewOutput(system, env, component.OutputConfig{
            Destination: "test-queue",
        })
        
        // Write data
        batch := spec.NewBatch()
        batch.Add(spec.NewBytesMessage([]byte("integration test message")))
        
        err := output.Write(ctx, batch)
        Expect(err).ToNot(HaveOccurred())
        
        // Read it back
        readBatch, err := input.Read(ctx)
        Expect(err).ToNot(HaveOccurred())
        Expect(readBatch.Len()).To(Equal(1))
        
        msg := readBatch.Messages()[0]
        data, err := msg.Raw()
        Expect(err).ToNot(HaveOccurred())
        Expect(string(data)).To(Equal("integration test message"))
    })
})
```

### Environment Variables

Use environment variables for integration test configuration:

```go
var _ = BeforeSuite(func() {
    // Skip integration tests if not configured
    if os.Getenv("INTEGRATION_TESTS") != "true" {
        Skip("Integration tests disabled. Set INTEGRATION_TESTS=true to enable.")
    }
    
    // Use external service if configured
    if serviceURL := os.Getenv("SERVICE_URL"); serviceURL != "" {
        // Use external service instead of test container
        externalServiceURL = serviceURL
    }
})
```

## Performance Testing

### Benchmark Tests

```go
func BenchmarkInputRead(b *testing.B) {
    system := setupBenchmarkSystem(b)
    defer system.Close(context.Background())
    
    input := component.NewInput(system, env, component.InputConfig{
        Source:    "benchmark-source",
        BatchSize: 100,
    })
    
    ctx := context.Background()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            batch, err := input.Read(ctx)
            if err != nil {
                b.Fatal(err)
            }
            if batch.Len() == 0 {
                b.Skip("No data available")
            }
        }
    })
}

func BenchmarkOutputWrite(b *testing.B) {
    system := setupBenchmarkSystem(b)
    defer system.Close(context.Background())
    
    output := component.NewOutput(system, env, component.OutputConfig{
        Destination: "benchmark-destination",
        BatchSize:   100,
    })
    
    // Prepare test data
    batch := spec.NewBatch()
    for i := 0; i < 100; i++ {
        batch.Add(spec.NewBytesMessage([]byte(fmt.Sprintf("message %d", i))))
    }
    
    ctx := context.Background()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            err := output.Write(ctx, batch)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

### Load Testing

```go
var _ = Describe("Load Testing", func() {
    When("handling high throughput", func() {
        It("should maintain performance under load", func() {
            const (
                numGoroutines = 10
                messagesPerGoroutine = 1000
            )
            
            var wg sync.WaitGroup
            errors := make(chan error, numGoroutines)
            
            for i := 0; i < numGoroutines; i++ {
                wg.Add(1)
                go func(routineID int) {
                    defer wg.Done()
                    
                    output := component.NewOutput(system, env, config)
                    
                    for j := 0; j < messagesPerGoroutine; j++ {
                        batch := spec.NewBatch()
                        batch.Add(spec.NewBytesMessage([]byte(fmt.Sprintf("routine:%d msg:%d", routineID, j))))
                        
                        if err := output.Write(ctx, batch); err != nil {
                            errors <- err
                            return
                        }
                    }
                }(i)
            }
            
            wg.Wait()
            close(errors)
            
            // Check for errors
            for err := range errors {
                Fail(fmt.Sprintf("Error during load test: %v", err))
            }
        })
    })
})
```

## Test Utilities

### Message Factories

```go
func CreateTestMessage(data string, metadata map[string]any) spec.Message {
    msg := spec.NewBytesMessage([]byte(data))
    for k, v := range metadata {
        msg.SetMetadata(k, v)
    }
    return msg
}

func CreateTestBatch(messages ...string) spec.Batch {
    batch := spec.NewBatch()
    for _, data := range messages {
        batch.Add(spec.NewBytesMessage([]byte(data)))
    }
    return batch
}
```

### Assertion Helpers

```go
func ExpectMessageData(msg spec.Message, expected string) {
    data, err := msg.Raw()
    Expect(err).ToNot(HaveOccurred())
    Expect(string(data)).To(Equal(expected))
}

func ExpectBatchSize(batch spec.Batch, size int) {
    Expect(batch).ToNot(BeNil())
    Expect(batch.Len()).To(Equal(size))
}
```

## Continuous Integration

### GitHub Actions Integration

```yaml
# .github/workflows/test.yml
- name: Run tests
  run: |
    task test
    task test:integration
  env:
    INTEGRATION_TESTS: true
    REDIS_URL: redis://localhost:6379
    NATS_URL: nats://localhost:4222
```

### Test Coverage

```bash
# Generate coverage report
task test:coverage

# View coverage
open coverage.html

# Upload to codecov
bash <(curl -s https://codecov.io/bash)
```

This comprehensive testing approach ensures high-quality, reliable components with excellent test coverage and clear behavioral specifications.