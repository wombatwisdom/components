package core

import "iter"

type MessageFactory interface {
	NewBatch() Batch
	NewMessage() Message
}

type Message interface {
	SetMetadata(key string, value any)
	SetRaw(b []byte)

	Raw() ([]byte, error)
	Metadata() iter.Seq2[string, any]
}
