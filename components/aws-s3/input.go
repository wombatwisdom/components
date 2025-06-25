package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/wombatwisdom/components/framework/spec"
)

type InputConfig struct {
	aws.Config

	Bucket  string
	Prefix  string
	MaxKeys int32

	ForcePathStyleURLs bool
	EndpointURL        *string
}

func NewInput(env spec.Environment, config InputConfig) (*Input, error) {
	return &Input{
		config: config,
		log:    env,
	}, nil
}

type Input struct {
	config InputConfig

	s3        *s3.Client
	pageToken *string

	log spec.Logger
}

func (i *Input) Connect(ctx context.Context) error {
	if i.s3 != nil {
		return nil
	}

	i.s3 = s3.NewFromConfig(i.config.Config, func(o *s3.Options) {
		o.UsePathStyle = i.config.ForcePathStyleURLs
		if i.config.EndpointURL != nil {
			o.BaseEndpoint = i.config.EndpointURL
		}
	})

	return nil
}

func (i *Input) Disconnect(ctx context.Context) error {
	return nil
}

func (i *Input) Read(ctx context.Context, collector spec.Collector) error {
	// -- list the objects and get the keys
	resp, err := i.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:            &i.config.Bucket,
		Prefix:            &i.config.Prefix,
		MaxKeys:           &i.config.MaxKeys,
		ContinuationToken: i.pageToken,
	})
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	for _, obj := range resp.Contents {
		// -- get the object
		objResp, err := i.s3.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &i.config.Bucket,
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to get object: %w", err)
		}

		// -- create the message
		msg := NewObjectResponseMessage(objResp)

		// -- write the message
		if err := collector.Write(msg); err != nil {
			// Note: Ack/Nack are legacy methods, now handled by message implementation
			return fmt.Errorf("failed to write message: %w", err)
		}
	}

	if resp.IsTruncated != nil && !*resp.IsTruncated {
		if err := collector.Disconnect(); err != nil {
			return fmt.Errorf("failed to disconnect collector: %w", err)
		}
	} else {
		// update the pointer to the next page
		i.pageToken = resp.NextContinuationToken
	}

	return nil
}
