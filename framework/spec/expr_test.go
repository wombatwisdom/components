package spec_test

import (
	"iter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/framework/spec"
)

var _ = Describe("MessageExpressionContext", func() {
	var mockMsg *mockMessage

	BeforeEach(func() {
		mockMsg = &mockMessage{
			raw:      []byte(`{"name": "test", "value": 42}`),
			metadata: map[string]any{"source": "test", "timestamp": "2023-01-01"},
		}
	})

	It("should create context with content as string", func() {
		ctx := spec.MessageExpressionContext(mockMsg)

		Expect(ctx["content"]).To(Equal(`{"name": "test", "value": 42}`))
	})

	It("should parse JSON content", func() {
		ctx := spec.MessageExpressionContext(mockMsg)

		json, ok := ctx["json"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(json["name"]).To(Equal("test"))
		Expect(json["value"]).To(Equal(float64(42))) // JSON unmarshals numbers as float64
	})

	It("should include metadata", func() {
		ctx := spec.MessageExpressionContext(mockMsg)

		metadata, ok := ctx["metadata"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(metadata["source"]).To(Equal("test"))
		Expect(metadata["timestamp"]).To(Equal("2023-01-01"))
	})

	It("should handle invalid JSON gracefully", func() {
		mockMsg.raw = []byte(`invalid json`)

		ctx := spec.MessageExpressionContext(mockMsg)

		Expect(ctx["content"]).To(Equal("invalid json"))
		json, ok := ctx["json"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(json).To(BeEmpty()) // Should be empty map for invalid JSON
	})

	It("should handle message raw error gracefully", func() {
		mockMsg.rawError = true

		ctx := spec.MessageExpressionContext(mockMsg)

		// Should not panic, but content and json won't be set
		Expect(ctx["content"]).To(BeNil())
		Expect(ctx["json"]).To(BeNil())

		// Metadata should still be present
		metadata, ok := ctx["metadata"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(metadata["source"]).To(Equal("test"))
	})
})

// Mock implementation of Message interface for testing
type mockMessage struct {
	raw      []byte
	metadata map[string]any
	rawError bool
}

func (m *mockMessage) SetMetadata(key string, value any) {
	if m.metadata == nil {
		m.metadata = make(map[string]any)
	}
	m.metadata[key] = value
}

func (m *mockMessage) SetRaw(b []byte) {
	m.raw = b
}

func (m *mockMessage) Raw() ([]byte, error) {
	if m.rawError {
		return nil, spec.ErrNotConnected // Using existing error
	}
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
