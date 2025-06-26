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
			
			Expect(config.EventBusName).To(Equal("default"))
			Expect(config.MaxBatchSize).To(Equal(10))
			Expect(config.EnableDeadLetter).To(BeFalse())
			Expect(config.Region).To(Equal("us-east-1"))
			Expect(config.ForcePathStyleURLs).To(BeFalse())
		})
	})

	Describe("Validate", func() {
		var config eventbridge.TriggerInputConfig

		BeforeEach(func() {
			config = eventbridge.DefaultTriggerInputConfig()
			config.RuleName = "test-rule"
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

		It("should fail when RuleName is missing", func() {
			config.RuleName = ""
			err := config.Validate()
			Expect(err).To(MatchError(eventbridge.ErrMissingRuleName))
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
	})
})