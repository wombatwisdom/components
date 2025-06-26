package test

import (
	"context"
	"iter"

	"github.com/wombatwisdom/components/framework/spec"
)

// NewMockComponentContext creates a mock ComponentContext for testing
func NewMockComponentContext() spec.ComponentContext {
	return &mockComponentContext{
		env: TestEnvironment(),
		ctx: context.Background(),
	}
}

type mockComponentContext struct {
	env spec.Environment
	ctx context.Context
}

func (m *mockComponentContext) Context() context.Context {
	return m.ctx
}

func (m *mockComponentContext) Logger() spec.Logger {
	return m.env
}

func (m *mockComponentContext) Debugf(format string, args ...interface{}) {
	m.env.Debugf(format, args...)
}

func (m *mockComponentContext) Infof(format string, args ...interface{}) {
	m.env.Infof(format, args...)
}

func (m *mockComponentContext) Warnf(format string, args ...interface{}) {
	m.env.Warnf(format, args...)
}

func (m *mockComponentContext) Errorf(format string, args ...interface{}) {
	m.env.Errorf(format, args...)
}

func (m *mockComponentContext) ParseExpression(expr string) (spec.Expression, error) {
	return nil, nil // Mock implementation
}

func (m *mockComponentContext) BuildMetadataFilter(patterns []string, invert bool) (spec.MetadataFilter, error) {
	return nil, nil // Mock implementation
}

func (m *mockComponentContext) NewBatch() spec.Batch {
	return &mockBatch{messages: make([]spec.Message, 0)}
}

func (m *mockComponentContext) NewMessage() spec.Message {
	return &mockMessage{
		raw:      make([]byte, 0),
		metadata: make(map[string]any),
	}
}

func (m *mockComponentContext) Resources() spec.ResourceManager {
	return &mockResourceManager{ctx: m.ctx}
}

func (m *mockComponentContext) Input(name string) (spec.Input, error) {
	return nil, spec.ErrNotConnected
}

func (m *mockComponentContext) Output(name string) (spec.Output, error) {
	return nil, spec.ErrNotConnected
}

func (m *mockComponentContext) System(name string) (spec.System, error) {
	return nil, spec.ErrNotConnected
}

// Mock implementations for testing

type mockBatch struct {
	messages []spec.Message
}

func (b *mockBatch) Messages() iter.Seq2[int, spec.Message] {
	return func(yield func(int, spec.Message) bool) {
		for i, msg := range b.messages {
			if !yield(i, msg) {
				return
			}
		}
	}
}

func (b *mockBatch) Append(msg spec.Message) {
	b.messages = append(b.messages, msg)
}

type mockMessage struct {
	raw      []byte
	metadata map[string]any
}

func (m *mockMessage) SetMetadata(key string, value any) {
	m.metadata[key] = value
}

func (m *mockMessage) SetRaw(b []byte) {
	m.raw = make([]byte, len(b))
	copy(m.raw, b)
}

func (m *mockMessage) Raw() ([]byte, error) {
	return m.raw, nil
}

func (m *mockMessage) Metadata() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range m.metadata {
			if !yield(k, v) {
				return
			}
		}
	}
}

type mockResourceManager struct {
	ctx context.Context
}

func (rm *mockResourceManager) Logger() spec.Logger {
	return TestEnvironment()
}

func (rm *mockResourceManager) System(name string) (spec.System, error) {
	return nil, spec.ErrNotConnected
}

func (rm *mockResourceManager) RegisterSystem(name string, sys spec.System) error {
	return nil
}

func (rm *mockResourceManager) Context() context.Context {
	return rm.ctx
}

func (rm *mockResourceManager) Metrics() spec.Metrics {
	return &mockMetrics{} // Mock implementation
}

type mockMetrics struct{}

func (m *mockMetrics) Counter(name string) spec.Counter { return &mockCounter{} }
func (m *mockMetrics) Gauge(name string) spec.Gauge     { return &mockGauge{} }
func (m *mockMetrics) Timer(name string) spec.Timer     { return &mockTimer{} }

type mockCounter struct{}

func (c *mockCounter) Inc(delta int64) {}

type mockGauge struct{}

func (g *mockGauge) Set(value float64) {}

type mockTimer struct{}

func (t *mockTimer) Record(duration float64) {}
