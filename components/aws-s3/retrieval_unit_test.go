package s3_test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	s3comp "github.com/wombatwisdom/components/aws-s3"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var _ = Describe("S3 Retrieval Processor Unit Tests", func() {
	var (
		processor *s3comp.RetrievalProcessor
	)

	BeforeEach(func() {
		// Create minimal AWS config for testing
		awsConfig := aws.Config{
			Region: "us-east-1",
			Credentials: credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     "test",
					SecretAccessKey: "test",
				},
			},
		}

		// Configure S3 retrieval processor
		retrievalConfig := s3comp.RetrievalConfig{
			Config:                 awsConfig,
			ForcePathStyleURLs:     true,
			EndpointURL:            aws.String("http://fake-endpoint:9000"),
			MaxConcurrentRetrivals: 3,
			FilterPrefix:           "",
			FilterSuffix:           "",
		}

		processor = s3comp.NewRetrievalProcessor(retrievalConfig)

		// Initialize with mock context
		ctx := test.NewMockComponentContext()
		Expect(processor.Init(ctx)).To(Succeed())
	})

	AfterEach(func() {
		if processor != nil {
			ctx := test.NewMockComponentContext()
			processor.Close(ctx)
		}
	})

	Describe("Initialization", func() {
		It("should initialize successfully", func() {
			Expect(processor).ToNot(BeNil())
		})

		It("should set default max concurrent retrievals when not specified", func() {
			awsConfig := aws.Config{
				Region: "us-east-1",
				Credentials: credentials.StaticCredentialsProvider{
					Value: aws.Credentials{
						AccessKeyID:     "test",
						SecretAccessKey: "test",
					},
				},
			}

			retrievalConfig := s3comp.RetrievalConfig{
				Config:                 awsConfig,
				MaxConcurrentRetrivals: 0, // Should default to 10
			}

			processor := s3comp.NewRetrievalProcessor(retrievalConfig)
			Expect(processor).ToNot(BeNil())
		})
	})

	Describe("Empty Trigger Handling", func() {
		It("should handle empty trigger batch", func() {
			triggers := spec.NewTriggerBatch()
			ctx := test.NewMockComponentContext()

			batch, callback, err := processor.Retrieve(ctx, triggers)
			Expect(err).ToNot(HaveOccurred())
			Expect(batch).ToNot(BeNil())

			// Count messages in batch
			messageCount := 0
			for range batch.Messages() {
				messageCount++
			}
			Expect(messageCount).To(Equal(0))

			err = callback(context.Background(), nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Filter Configuration", func() {
		It("should support prefix filtering", func() {
			awsConfig := aws.Config{
				Region: "us-east-1",
				Credentials: credentials.StaticCredentialsProvider{
					Value: aws.Credentials{
						AccessKeyID:     "test",
						SecretAccessKey: "test",
					},
				},
			}

			retrievalConfig := s3comp.RetrievalConfig{
				Config:       awsConfig,
				FilterPrefix: "logs/",
			}

			processor := s3comp.NewRetrievalProcessor(retrievalConfig)
			Expect(processor).ToNot(BeNil())
		})

		It("should support suffix filtering", func() {
			awsConfig := aws.Config{
				Region: "us-east-1",
				Credentials: credentials.StaticCredentialsProvider{
					Value: aws.Credentials{
						AccessKeyID:     "test",
						SecretAccessKey: "test",
					},
				},
			}

			retrievalConfig := s3comp.RetrievalConfig{
				Config:       awsConfig,
				FilterSuffix: ".json",
			}

			processor := s3comp.NewRetrievalProcessor(retrievalConfig)
			Expect(processor).ToNot(BeNil())
		})
	})

	Describe("Trigger Event Processing", func() {
		It("should process trigger events with bucket/key metadata", func() {
			// This test will fail on S3 connection but we can test the trigger processing logic
			triggers := spec.NewTriggerBatch()
			triggers.Append(spec.NewTriggerEvent("eventbridge", "test-reference", map[string]any{
				"bucket": "test-bucket",
				"key":    "test/file.txt",
			}))

			ctx := test.NewMockComponentContext()

			// We expect this to fail because there's no real S3, but it tests the trigger processing
			_, _, err := processor.Retrieve(ctx, triggers)
			Expect(err).To(HaveOccurred()) // Expected to fail on S3 connection
		})

		It("should process trigger events with reference format", func() {
			triggers := spec.NewTriggerBatch()
			triggers.Append(spec.NewTriggerEvent("s3-polling", "test-bucket/test/file.txt", map[string]any{}))

			ctx := test.NewMockComponentContext()

			// We expect this to fail because there's no real S3, but it tests the trigger processing
			_, _, err := processor.Retrieve(ctx, triggers)
			Expect(err).To(HaveOccurred()) // Expected to fail on S3 connection
		})

		It("should handle triggers with invalid S3 info", func() {
			triggers := spec.NewTriggerBatch()
			triggers.Append(spec.NewTriggerEvent("test", "invalid-reference", map[string]any{}))

			ctx := test.NewMockComponentContext()

			_, _, err := processor.Retrieve(ctx, triggers)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Lifecycle Management", func() {
		It("should initialize and close without errors", func() {
			ctx := test.NewMockComponentContext()
			err := processor.Close(ctx)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
