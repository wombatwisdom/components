package spec

import "iter"

type MessageFactory interface {
	NewBatch(msg ...Message) Batch
	NewMessage() Message
}

type Message interface {
	SetMetadata(key string, value any)
	SetRaw(b []byte)

	Raw() ([]byte, error)
	Metadata() iter.Seq2[string, any]
}

// NewBytesMessage creates a simple message from bytes for testing
func NewBytesMessage(data []byte) Message {
	return &bytesMessage{
		data:     data,
		metadata: make(map[string]any),
	}
}

type bytesMessage struct {
	data     []byte
	metadata map[string]any
}

func (b *bytesMessage) SetMetadata(key string, value any) {
	b.metadata[key] = value
}

func (b *bytesMessage) SetRaw(data []byte) {
	b.data = data
}

func (b *bytesMessage) Raw() ([]byte, error) {
	return b.data, nil
}

func (b *bytesMessage) Metadata() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range b.metadata {
			if !yield(k, v) {
				return
			}
		}
	}
}
