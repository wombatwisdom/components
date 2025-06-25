package s3

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/wombatwisdom/components/spec"
)

type SourceConfig struct {
	aws.Config

	Bucket  string
	Prefix  string
	MaxKeys int32

	ForcePathStyleURLs bool
}

func NewSource(env spec.Environment, config SourceConfig) (*Source, error) {
	return &Source{
		config: config,
		log:    env,
	}, nil
}

type Source struct {
	config SourceConfig

	s3        *s3.Client
	pageToken *string

	log spec.Logger
}

func (s *Source) Connect(ctx context.Context) error {
	if s.s3 != nil {
		return nil
	}

	s.s3 = s3.NewFromConfig(s.config.Config, func(o *s3.Options) {
		o.UsePathStyle = s.config.ForcePathStyleURLs
	})

	return nil
}

func (s *Source) Disconnect(ctx context.Context) error {
	return nil
}

func (s *Source) Read(ctx context.Context, collector spec.Collector) error {
	// -- list the objects and get the keys
	resp, err := s.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:            &s.config.Bucket,
		Prefix:            &s.config.Prefix,
		MaxKeys:           &s.config.MaxKeys,
		ContinuationToken: s.pageToken,
	})
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	for _, obj := range resp.Contents {
		// -- get the object
		objResp, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &s.config.Bucket,
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to get object: %w", err)
		}

		// -- create the message
		msg := NewObjectResponseMessage(objResp)

		// -- write the message
		if err := collector.Write(ctx, msg); err != nil {
			msg.Nack()
			return fmt.Errorf("failed to write message: %w", err)
		}

		msg.Ack()
	}

	if resp.IsTruncated != nil && !*resp.IsTruncated {
		if err := collector.Disconnect(ctx); err != nil {
			return fmt.Errorf("failed to disconnect collector: %w", err)
		}
	} else {
		// update the pointer to the next page
		s.pageToken = resp.NextContinuationToken
	}

	return nil
}
