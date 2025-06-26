package eventbridge_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/eventbridge"
)

var _ = Describe("TriggerInputConfig", func() {
	Describe("DefaultTriggerInputConfig", func() {
		It("should return valid defaults", func() {
			config := eventbridge.DefaultTriggerInputConfig()
			
			Expect(string(config.Mode)).To(Equal("sqs"))
			Expect(config.EventBusName).To(Equal("default"))
			Expect(config.MaxBatchSize).To(Equal(10))
			Expect(config.EnableDeadLetter).To(BeFalse())
			Expect(config.Region).To(Equal("us-east-1"))
			Expect(config.ForcePathStyleURLs).To(BeFalse())
			Expect(config.SQSMaxMessages).To(Equal(int32(10)))
			Expect(config.SQSWaitTimeSeconds).To(Equal(int32(20)))
			Expect(config.SQSVisibilityTimeout).To(Equal(int32(30)))
			Expect(config.PipeBatchSize).To(Equal(int32(10)))
		})
	})

	Describe("Validate", func() {
		var config eventbridge.TriggerInputConfig

		BeforeEach(func() {
			config = eventbridge.DefaultTriggerInputConfig()
			config.Mode = "simulation" // Use simulation mode for basic tests
			config.EventSource = "aws.s3"
		})

		It("should pass validation with all required fields", func() {
			err := config.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail when EventBusName is missing", func() {
			config.EventBusName = ""
			err := config.Validate()
			Expect(err).To(MatchError(eventbridge.ErrMissingEventBusName))
		})

		It("should not require RuleName for simulation mode", func() {
			config.RuleName = ""
			err := config.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail when EventSource is missing", func() {
			config.EventSource = ""
			err := config.Validate()
			Expect(err).To(MatchError(eventbridge.ErrMissingEventSource))
		})

		It("should set default MaxBatchSize when zero", func() {
			config.MaxBatchSize = 0
			err := config.Validate()
			Expect(err).ToNot(HaveOccurred())
			Expect(config.MaxBatchSize).To(Equal(10))
		})

		It("should set default MaxBatchSize when negative", func() {
			config.MaxBatchSize = -5
			err := config.Validate()
			Expect(err).ToNot(HaveOccurred())
			Expect(config.MaxBatchSize).To(Equal(10))
		})

		It("should preserve valid MaxBatchSize", func() {
			config.MaxBatchSize = 25
			err := config.Validate()
			Expect(err).ToNot(HaveOccurred())
			Expect(config.MaxBatchSize).To(Equal(25))
		})

		Context("SQS Mode", func() {
			BeforeEach(func() {
				config.Mode = "sqs"
				config.RuleName = "test-rule"
				config.SQSQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
			})

			It("should pass validation with SQS configuration", func() {
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should fail when SQS queue URL is missing", func() {
				config.SQSQueueURL = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("sqs_queue_url is required"))
			})

			It("should fail when rule name is missing", func() {
				config.RuleName = ""
				err := config.Validate()
				Expect(err).To(MatchError(eventbridge.ErrMissingRuleName))
			})
		})

		Context("Pipes Mode", func() {
			BeforeEach(func() {
				config.Mode = "pipes"
				config.PipeName = "test-pipe"
				config.PipeSourceARN = "arn:aws:events:us-east-1:123456789012:event-bus/default"
				config.PipeTargetARN = "arn:aws:sqs:us-east-1:123456789012:test-queue"
			})

			It("should pass validation with Pipes configuration", func() {
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should fail when pipe name is missing", func() {
				config.PipeName = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("pipe_name is required"))
			})

			It("should fail when pipe source ARN is missing", func() {
				config.PipeSourceARN = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("pipe_source_arn is required"))
			})

			It("should fail when pipe target ARN is missing", func() {
				config.PipeTargetARN = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("pipe_target_arn is required"))
			})
		})

		Context("Simulation Mode", func() {
			BeforeEach(func() {
				config.Mode = "simulation"
			})

			It("should pass validation without additional requirements", func() {
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("should fail with invalid integration mode", func() {
			config.Mode = "invalid"
			err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid integration mode"))
		})
	})
})