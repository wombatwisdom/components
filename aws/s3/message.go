package s3

import (
	"io"
	"iter"
	"maps"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/wombatwisdom/components/spec"
)

func NewObjectResponseMessage(resp *s3.GetObjectOutput) spec.Message {
	return &ObjectResponseMessage{
		resp:     resp,
		meta:     &ObjectResponseMetadata{resp: resp},
		metadata: make(map[string]any),
	}
}

type ObjectResponseMessage struct {
	resp     *s3.GetObjectOutput
	meta     *ObjectResponseMetadata
	metadata map[string]any
}

func (o *ObjectResponseMessage) SetMetadata(key string, value any) {
	o.metadata[key] = value
}

func (o *ObjectResponseMessage) SetRaw(b []byte) {
	// S3 objects are immutable, so this is a no-op
	// In a real implementation, you might store this separately
}

func (o *ObjectResponseMessage) Raw() ([]byte, error) {
	if o.resp.Body == nil {
		return nil, nil
	}

	// This is a simplified implementation
	// In practice, you'd want to cache the body or use a different approach
	return io.ReadAll(o.resp.Body)
}

func (o *ObjectResponseMessage) Metadata() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		// Add S3-specific metadata
		for k, v := range o.resp.Metadata {
			if !yield(k, v) {
				return
			}
		}

		// Add any custom metadata
		for k, v := range o.metadata {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Legacy methods for backward compatibility
func (o *ObjectResponseMessage) Meta() spec.Metadata {
	return o.meta
}

func (o *ObjectResponseMessage) Ack() error {
	if o.resp.Body == nil {
		return nil
	}

	return o.resp.Body.Close()
}

func (o *ObjectResponseMessage) Nack() error {
	if o.resp.Body == nil {
		return nil
	}

	return o.resp.Body.Close()
}

type ObjectResponseMetadata struct {
	resp *s3.GetObjectOutput
}

func (o *ObjectResponseMetadata) Keys() iter.Seq[string] {
	return maps.Keys(o.resp.Metadata)
}

func (o *ObjectResponseMetadata) Get(key string) any {
	return o.resp.Metadata[key]
}
