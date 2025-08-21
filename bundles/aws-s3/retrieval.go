package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/wombatwisdom/components/framework/spec"
)

// RetrievalConfig defines configuration for S3 retrieval processor
type RetrievalConfig struct {
	aws.Config

	// S3 client configuration
	ForcePathStyleURLs bool
	EndpointURL        *string

	// Retrieval options
	MaxConcurrentRetrivals int    // Maximum concurrent S3 retrievals
	FilterPrefix           string // Only retrieve objects with this prefix
	FilterSuffix           string // Only retrieve objects with this suffix
}

// NewRetrievalProcessor creates a new S3 retrieval processor
func NewRetrievalProcessor(config RetrievalConfig) *RetrievalProcessor {
	if config.MaxConcurrentRetrivals <= 0 {
		config.MaxConcurrentRetrivals = 10 // Default to 10 concurrent retrievals
	}

	return &RetrievalProcessor{
		config: config,
	}
}

// RetrievalProcessor implements spec.RetrievalProcessor for S3 objects
type RetrievalProcessor struct {
	config RetrievalConfig
	s3     *s3.Client
	logger spec.Logger
}

// Init initializes the S3 retrieval processor
func (r *RetrievalProcessor) Init(ctx spec.ComponentContext) error {
	r.logger = ctx

	r.s3 = s3.NewFromConfig(r.config.Config, func(o *s3.Options) {
		o.UsePathStyle = r.config.ForcePathStyleURLs
		if r.config.EndpointURL != nil {
			o.BaseEndpoint = r.config.EndpointURL
		}
	})

	r.logger.Infof("S3 retrieval processor initialized")
	return nil
}

// Close cleans up the S3 retrieval processor
func (r *RetrievalProcessor) Close(ctx spec.ComponentContext) error {
	r.logger.Infof("S3 retrieval processor closed")
	return nil
}

// Retrieve fetches S3 objects based on trigger events
func (r *RetrievalProcessor) Retrieve(ctx spec.ComponentContext, triggers spec.TriggerBatch) (spec.Batch, spec.ProcessedCallback, error) {
	batch := ctx.NewBatch()
	triggerList := triggers.Triggers()

	if len(triggerList) == 0 {
		return batch, spec.NoopCallback, nil
	}

	r.logger.Infof("Retrieving S3 objects, count: %d", len(triggerList))

	// Process triggers with concurrency control
	semaphore := make(chan struct{}, r.config.MaxConcurrentRetrivals)
	results := make(chan retrievalResult, len(triggerList))

	for _, trigger := range triggerList {
		go func(t spec.TriggerEvent) {
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			result := r.retrieveSingleObject(ctx.Context(), t)
			results <- result
		}(trigger)
	}

	// Collect results
	var errors []error
	for i := 0; i < len(triggerList); i++ {
		result := <-results
		if result.err != nil {
			errors = append(errors, result.err)
			r.logger.Errorf("Failed to retrieve object %s: %v", result.reference, result.err)
		} else if result.message != nil {
			batch.Append(result.message)
		}
	}

	if len(errors) > 0 {
		return nil, nil, fmt.Errorf("failed to retrieve %d objects: %v", len(errors), errors[0])
	}

	callback := func(ctx context.Context, err error) error {
		r.logger.Debugf("S3 retrieval batch processed")
		return err
	}

	return batch, callback, nil
}

// retrievalResult holds the result of a single object retrieval
type retrievalResult struct {
	reference string
	message   spec.Message
	err       error
}

// retrieveSingleObject retrieves a single S3 object based on a trigger event
func (r *RetrievalProcessor) retrieveSingleObject(ctx context.Context, trigger spec.TriggerEvent) retrievalResult {
	// Extract S3 information from trigger
	s3Info, err := r.extractS3Info(trigger)
	if err != nil {
		return retrievalResult{
			reference: trigger.Reference(),
			err:       fmt.Errorf("failed to extract S3 info: %w", err),
		}
	}

	// Apply filters
	if !r.shouldRetrieve(s3Info.Key) {
		r.logger.Debugf("Skipping object due to filters: %s", s3Info.Key)
		return retrievalResult{reference: trigger.Reference()}
	}

	// Retrieve the object
	resp, err := r.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3Info.Bucket),
		Key:    aws.String(s3Info.Key),
	})
	if err != nil {
		return retrievalResult{
			reference: trigger.Reference(),
			err:       fmt.Errorf("failed to get S3 object %s/%s: %w", s3Info.Bucket, s3Info.Key, err),
		}
	}

	// Create message from S3 response
	message := NewObjectResponseMessage(resp)

	// Add trigger metadata to message
	message.SetMetadata("trigger_source", trigger.Source())
	message.SetMetadata("trigger_timestamp", trigger.Timestamp())
	for key, value := range trigger.Metadata() {
		message.SetMetadata("trigger_"+key, value)
	}

	r.logger.Debugf("Successfully retrieved S3 object %s/%s", s3Info.Bucket, s3Info.Key)

	return retrievalResult{
		reference: trigger.Reference(),
		message:   message,
	}
}

// s3ObjectInfo contains S3 bucket and key information
type s3ObjectInfo struct {
	Bucket string
	Key    string
}

// extractS3Info extracts S3 bucket and key from trigger event
func (r *RetrievalProcessor) extractS3Info(trigger spec.TriggerEvent) (s3ObjectInfo, error) {
	metadata := trigger.Metadata()

	// Try to get bucket and key from metadata first
	if bucket, ok := metadata["bucket"].(string); ok {
		if key, ok := metadata["key"].(string); ok {
			return s3ObjectInfo{Bucket: bucket, Key: key}, nil
		}
	}

	// Try to parse from reference (format: bucket/key)
	reference := trigger.Reference()
	parts := strings.SplitN(reference, "/", 2)
	if len(parts) == 2 {
		return s3ObjectInfo{Bucket: parts[0], Key: parts[1]}, nil
	}

	// Try EventBridge S3 event format
	if bucket, ok := metadata["bucket"].(map[string]interface{}); ok {
		if name, ok := bucket["name"].(string); ok {
			if object, ok := metadata["object"].(map[string]interface{}); ok {
				if key, ok := object["key"].(string); ok {
					return s3ObjectInfo{Bucket: name, Key: key}, nil
				}
			}
		}
	}

	return s3ObjectInfo{}, fmt.Errorf("unable to extract S3 bucket and key from trigger: %s", reference)
}

// shouldRetrieve checks if an object should be retrieved based on filters
func (r *RetrievalProcessor) shouldRetrieve(key string) bool {
	if r.config.FilterPrefix != "" && !strings.HasPrefix(key, r.config.FilterPrefix) {
		return false
	}

	if r.config.FilterSuffix != "" && !strings.HasSuffix(key, r.config.FilterSuffix) {
		return false
	}

	return true
}
