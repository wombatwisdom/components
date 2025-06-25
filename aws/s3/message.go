package s3

import (
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/wombatwisdom/components/spec"
    "io"
    "iter"
    "maps"
)

func NewObjectResponseMessage(resp *s3.GetObjectOutput) spec.Message {
    return &ObjectResponseMessage{
        resp: resp,
        meta: &ObjectResponseMetadata{resp: resp},
    }
}

type ObjectResponseMessage struct {
    resp *s3.GetObjectOutput
    meta *ObjectResponseMetadata
}

func (o *ObjectResponseMessage) Meta() spec.Metadata {
    return o.meta
}

func (o *ObjectResponseMessage) Data() (io.Reader, error) {
    return o.resp.Body, nil
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
