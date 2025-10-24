package test

import (
	"context"
	"iter"
	"regexp"

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

func (m *mockComponentContext) BuildMetadataFilter(patterns []string, invert bool) (spec.MetadataFilter, error) {
	regexes := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		regexes = append(regexes, re)
	}

	return &mockMetadataFilter{
		patterns: regexes,
		invert:   invert,
	}, nil
}

func (m *mockComponentContext) NewBatch(msgs ...spec.Message) spec.Batch {
	return &mockBatch{messages: msgs}
}

func (m *mockComponentContext) NewMessage() spec.Message {
	return &mockMessage{
		raw:      make([]byte, 0),
		metadata: make(map[string]any),
	}
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

// mockMetadataFilter implements spec.MetadataFilter for testing
type mockMetadataFilter struct {
	patterns []*regexp.Regexp
	invert   bool
}

func (f *mockMetadataFilter) Include(key string) bool {
	matched := false
	for _, re := range f.patterns {
		if re.MatchString(key) {
			matched = true
			break
		}
	}

	if f.invert {
		return !matched
	}
	return matched
}
