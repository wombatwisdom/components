package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/framework/core"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

// Helper function to assert string results
func expectStringResult(result interface{}, expected string) {
	Expect(result).To(BeAssignableToTypeOf(""))
	Expect(result.(string)).To(Equal(expected))
}

// Helper function to assert integer results
func expectIntResult(result interface{}, expected int) {
	Expect(result).To(BeAssignableToTypeOf(0))
	Expect(result.(int)).To(Equal(expected))
}

// Helper function to assert boolean results
func expectBoolResult(result interface{}, expected bool) {
	Expect(result).To(BeAssignableToTypeOf(true))
	Expect(result.(bool)).To(Equal(expected))
}

var _ = Describe("FieldEvaluator", func() {
	var (
		evaluator   *core.FieldEvaluator
		err         error
		testBatch   spec.Batch
		singleBatch spec.Batch
		emptyBatch  spec.Batch
	)

	BeforeEach(func() {
		// Create test messages with different payloads and metadata
		msg1 := test.NewMockMessage([]byte(`{"name": "Alice", "score": 100, "level": 5}`))
		msg1.SetMetadata("user_id", "123")
		msg1.SetMetadata("timestamp", "2024-01-01T10:00:00Z")
		msg1.SetMetadata("source", "api")
		msg1.SetMetadata("priority", "high")

		msg2 := test.NewMockMessage([]byte(`{"name": "Bob", "score": 200, "level": 8}`))
		msg2.SetMetadata("user_id", "456")
		msg2.SetMetadata("timestamp", "2024-01-01T10:01:00Z")
		msg2.SetMetadata("source", "webhook")
		msg2.SetMetadata("priority", "medium")

		msg3 := test.NewMockMessage([]byte(`{"name": "Charlie", "score": 300, "level": 12}`))
		msg3.SetMetadata("user_id", "789")
		msg3.SetMetadata("timestamp", "2024-01-01T10:02:00Z")
		msg3.SetMetadata("source", "api")
		msg3.SetMetadata("priority", "low")

		testBatch = test.NewMockBatch()
		testBatch.Append(msg1)
		testBatch.Append(msg2)
		testBatch.Append(msg3)

		singleBatch = test.NewMockBatch()
		singleBatch.Append(msg1)

		emptyBatch = test.NewMockBatch()
	})

	Describe("NewFieldEvaluator", func() {
		Context("when creating evaluator with valid expressions", func() {
			It("should successfully compile simple literal text", func() {
				evaluator, err = core.NewFieldEvaluator("simple text")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator).ToNot(BeNil())
				Expect(evaluator.HasExpressions()).To(BeFalse())
				Expect(evaluator.GetExpressionCount()).To(Equal(0))
				Expect(evaluator.GetOriginal()).To(Equal("simple text"))
			})

			It("should successfully compile meta access expressions", func() {
				evaluator, err = core.NewFieldEvaluator("${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator).ToNot(BeNil())
				Expect(evaluator.HasExpressions()).To(BeTrue())
				Expect(evaluator.GetExpressionCount()).To(Equal(1))
			})

			It("should successfully compile payload access expressions", func() {
				evaluator, err = core.NewFieldEvaluator("${!payload}")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator).ToNot(BeNil())
				Expect(evaluator.HasExpressions()).To(BeTrue())
				Expect(evaluator.GetExpressionCount()).To(Equal(1))
			})

			It("should successfully compile payload bytes access expressions", func() {
				evaluator, err = core.NewFieldEvaluator("${!len(payloadBytes)}")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator).ToNot(BeNil())
				Expect(evaluator.HasExpressions()).To(BeTrue())
				Expect(evaluator.GetExpressionCount()).To(Equal(1))
			})

			It("should successfully compile batch access expressions", func() {
				evaluator, err = core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator).ToNot(BeNil())
				Expect(evaluator.HasExpressions()).To(BeTrue())
				Expect(evaluator.GetExpressionCount()).To(Equal(1))
			})

			It("should successfully compile index access expressions", func() {
				evaluator, err = core.NewFieldEvaluator("Position: ${!index}")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator).ToNot(BeNil())
				Expect(evaluator.HasExpressions()).To(BeTrue())
				Expect(evaluator.GetExpressionCount()).To(Equal(1))
			})

			It("should successfully compile multiple expressions", func() {
				evaluator, err = core.NewFieldEvaluator("user: ${!meta('user_id')}, batch size: ${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator).ToNot(BeNil())
				Expect(evaluator.HasExpressions()).To(BeTrue())
				Expect(evaluator.GetExpressionCount()).To(Equal(2))
			})
		})

		Context("when creating evaluator with invalid expressions", func() {
			It("should return error for unclosed expression", func() {
				evaluator, err = core.NewFieldEvaluator("${!meta('user_id')")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unclosed expression"))
				Expect(evaluator).To(BeNil())
			})

			It("should return error for invalid expression syntax", func() {
				evaluator, err = core.NewFieldEvaluator("${!invalid syntax here}")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to compile expression"))
				Expect(evaluator).To(BeNil())
			})
		})
	})

	Describe("Eval", func() {
		Context("with literal text only", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("literal text")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return the text verbatim as string", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "literal text")
			})
		})

		Context("with metadata access using meta function", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("User ID: ${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should evaluate metadata correctly for first message", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "User ID: 123")
			})

			It("should evaluate metadata correctly for second message", func() {
				result, err := evaluator.Eval(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "User ID: 456")
			})

			It("should evaluate metadata correctly for third message", func() {
				result, err := evaluator.Eval(testBatch, 2)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "User ID: 789")
			})
		})

		Context("with multiple metadata values using metas function", func() {
			BeforeEach(func() {
				// Create a message with multiple tags metadata
				msg := test.NewMockMessage([]byte(`{"name": "Alice", "score": 100, "level": 5}`))
				msg.SetMetadata("user_id", "123")
				msg.SetMetadata("timestamp", "2024-01-01T10:00:00Z")
				msg.SetMetadata("source", "api")
				msg.SetMetadata("priority", "high")
				msg.SetMetadata("tags", []string{"important", "urgent", "customer"})

				testBatch = test.NewMockBatch()
				testBatch.Append(msg)

				evaluator, err = core.NewFieldEvaluator("Tags: ${!len(meta('tags'))}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should access multiple metadata values and return string", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Tags: 3")
			})

			It("should return integer for single expression", func() {
				singleEvaluator, err := core.NewFieldEvaluator("${!len(metas['tags'])}")
				Expect(err).ToNot(HaveOccurred())

				result, err := singleEvaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 3)
			})
		})

		Context("with payload access", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("Payload: ${!payload}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should access message payload correctly in mixed content", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, `Payload: {"name": "Alice", "score": 100, "level": 5}`)
			})

			It("should access different payload for different index in mixed content", func() {
				result, err := evaluator.Eval(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, `Payload: {"name": "Bob", "score": 200, "level": 8}`)
			})

			It("should access payload as both string and bytes", func() {
				stringEvaluator, err := core.NewFieldEvaluator("String: ${!len(payload)} chars, Bytes: ${!len(payloadBytes)} bytes")
				Expect(err).ToNot(HaveOccurred())

				result, err := stringEvaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "String: 43 chars, Bytes: 43 bytes") // Same for ASCII content
			})

			It("should return raw payload string for single expression", func() {
				singleEvaluator, err := core.NewFieldEvaluator("${!payload}")
				Expect(err).ToNot(HaveOccurred())

				result, err := singleEvaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, `{"name": "Alice", "score": 100, "level": 5}`)
			})
		})

		Context("with batch access", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return batch length as integer", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 3)
			})

			It("should work with single message batch", func() {
				result, err := evaluator.Eval(singleBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 1)
			})
		})

		Context("with mixed content (literal + expression)", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("Batch size: ${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return concatenated string", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Batch size: 3")
			})
		})

		Context("with index access", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("${!index}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return correct index as integer for first message", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 0)
			})

			It("should return correct index as integer for middle message", func() {
				result, err := evaluator.Eval(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 1)
			})

			It("should return correct index as integer for last message", func() {
				result, err := evaluator.Eval(testBatch, 2)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 2)
			})
		})

		Context("with index access in mixed content", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("Message position: ${!index}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return string for mixed content", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Message position: 0")
			})
		})

		Context("with batch message access via batch array", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("First user: ${!batch[0].GetHeader('user_id')}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should access other messages in batch", func() {
				result, err := evaluator.Eval(testBatch, 1) // Evaluating from second message
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("First user: 123"))
			})
		})

		Context("with complex expressions", func() {
			var complexEvaluator *core.FieldEvaluator

			BeforeEach(func() {
				var evalErr error
				complexEvaluator, evalErr = core.NewFieldEvaluator(
					"Message ${!index + 1} of ${!len(batch)}: user ${!meta('user_id')} from ${!meta('source')}")
				Expect(evalErr).ToNot(HaveOccurred())
			})

			It("should combine multiple expressions with mathematical operations", func() {
				result, err := complexEvaluator.Eval(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("Message 2 of 3: user 456 from webhook"))
			})
		})

		Context("with conditional expressions", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator(
					"Source: ${!meta('source') == 'api' ? 'API Call' : 'Other'}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle conditional expressions for API source", func() {
				result, err := evaluator.Eval(testBatch, 0) // First message has source=api
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Source: API Call")
			})

			It("should handle conditional expressions for non-API source", func() {
				result, err := evaluator.Eval(testBatch, 1) // Second message has source=webhook
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Source: Other")
			})

			It("should return boolean for single conditional expression", func() {
				boolEvaluator, err := core.NewFieldEvaluator("${!meta('source') == 'api'}")
				Expect(err).ToNot(HaveOccurred())

				result, err := boolEvaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectBoolResult(result, true)
			})
		})

		Context("with priority-based conditional expressions", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator(
					"Priority Level: ${!meta('priority') == 'high' ? 1 : meta('priority') == 'medium' ? 2 : 3}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle nested conditional expressions for high priority", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Priority Level: 1")
			})

			It("should handle nested conditional expressions for medium priority", func() {
				result, err := evaluator.Eval(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Priority Level: 2")
			})

			It("should handle nested conditional expressions for low priority", func() {
				result, err := evaluator.Eval(testBatch, 2)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Priority Level: 3")
			})

			It("should return integer for single conditional expression", func() {
				singleEvaluator, err := core.NewFieldEvaluator("${!meta('priority') == 'high' ? 1 : meta('priority') == 'medium' ? 2 : 3}")
				Expect(err).ToNot(HaveOccurred())

				result, err := singleEvaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 1)
			})
		})

		Context("with single expression returning different types", func() {
			It("should return integer for length operations", func() {
				evaluator, err := core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 3)
			})

			It("should return boolean for comparison operations", func() {
				evaluator, err := core.NewFieldEvaluator("${!len(batch) > 2}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectBoolResult(result, true)
			})

			It("should return string for meta access", func() {
				evaluator, err := core.NewFieldEvaluator("${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "123")
			})

			It("should return float for mathematical operations", func() {
				evaluator, err := core.NewFieldEvaluator("${!(index + 1) * 100 / len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.Eval(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeAssignableToTypeOf(0.0))
				Expect(result.(float64)).To(BeNumerically("~", 66.66666666666667, 0.0001))
			})
		})

		Context("with mathematical operations on index and batch", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("Progress: ${!(index + 1) * 100 / len(batch)}%")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should perform complex mathematical operations in mixed content", func() {
				result, err := evaluator.Eval(testBatch, 1) // Second message (index 1)
				Expect(err).ToNot(HaveOccurred())
				// Mixed content returns string: "Progress: 66.66666666666667%"
				expectStringResult(result, "Progress: 66.66666666666667%")
			})
		})

		Context("with payload length calculations", func() {
			It("should return integer for single expression", func() {
				evaluator, err := core.NewFieldEvaluator("${!len(payloadBytes)}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectIntResult(result, 43) // Length of Alice's JSON
			})

			It("should return string for mixed content", func() {
				evaluator, err := core.NewFieldEvaluator("Payload size: ${!len(payloadBytes)} bytes")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Payload size: 43 bytes")
			})
		})

		Context("with error conditions", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("${!meta('nonexistent')}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle missing metadata gracefully", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("")) // meta function returns empty string for missing keys
			})
		})

		Context("with invalid batch index", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return error for out of bounds index", func() {
				result, err := evaluator.Eval(testBatch, 10) // Index out of bounds
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("index 10 is out of bounds for batch of size 3"))
				Expect(result).To(BeNil())
			})

			It("should return error for negative index", func() {
				result, err := evaluator.Eval(testBatch, -1) // Negative index
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("negative index -1 is not allowed"))
				Expect(result).To(BeNil())
			})
		})

		Context("with empty batch", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return error for empty batch", func() {
				result, err := evaluator.Eval(emptyBatch, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot evaluate expression on empty batch"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("Multiple batch evaluation", func() {
		var (
			batch1, batch2 spec.Batch
		)

		BeforeEach(func() {
			msg1 := test.NewMockMessage([]byte(`{"status": "active"}`))
			msg1.SetMetadata("env", "prod")
			msg1.SetMetadata("service", "auth")

			msg2 := test.NewMockMessage([]byte(`{"status": "inactive"}`))
			msg2.SetMetadata("env", "staging")
			msg2.SetMetadata("service", "billing")

			batch1 = test.NewMockBatch()
			batch1.Append(msg1)
			batch2 = test.NewMockBatch()
			batch2.Append(msg2)

			evaluator, err = core.NewFieldEvaluator("Environment: ${!meta('env')}, Service: ${!meta('service')}")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should evaluate correctly against multiple batches", func() {
			result1, err1 := evaluator.Eval(batch1, 0)
			result2, err2 := evaluator.Eval(batch2, 0)

			Expect(err1).ToNot(HaveOccurred())
			Expect(err2).ToNot(HaveOccurred())

			Expect(result1).To(Equal("Environment: prod, Service: auth"))
			Expect(result2).To(Equal("Environment: staging, Service: billing"))
		})
	})

	Describe("Edge cases", func() {
		Context("with empty expressions", func() {
			It("should handle empty expression content", func() {
				evaluator, err = core.NewFieldEvaluator("${!}")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to compile expression"))
			})
		})

		Context("with consecutive expressions", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("${!meta('user_id')}${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle consecutive expressions without separating text", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "1233")
			})
		})

		Context("with expressions accessing batch metadata via index", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("All sources: ${!batch[0].GetHeader('source')}, ${!batch[1].GetHeader('source')}, ${!batch[2].GetHeader('source')}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should access metadata from all batch messages", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "All sources: api, webhook, api")
			})
		})

		Context("with mixed access patterns", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("Current: ${!meta('source')}, First: ${!batch[0].GetHeader('source')}, Index: ${!index}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should handle mixed access patterns correctly", func() {
				result, err := evaluator.Eval(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "Current: webhook, First: api, Index: 1")
			})
		})

		Context("with only expression delimiters in text", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("This has ${ but not a complete expression")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should treat incomplete delimiters as literal text", func() {
				result, err := evaluator.Eval(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				expectStringResult(result, "This has ${ but not a complete expression")
			})
		})
	})

	Describe("Type-specific evaluation methods", func() {
		Context("EvalString", func() {
			It("should convert integer result to string", func() {
				evaluator, err := core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalString(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("3"))
			})

			It("should return string result as-is", func() {
				evaluator, err := core.NewFieldEvaluator("${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalString(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("123"))
			})

			It("should convert boolean result to string", func() {
				evaluator, err := core.NewFieldEvaluator("${!index == 0}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalString(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("true"))
			})

			It("should convert float result to string", func() {
				evaluator, err := core.NewFieldEvaluator("${!index * 1.5}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalString(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("1.5"))
			})
		})

		Context("EvalInt", func() {
			It("should return integer result directly", func() {
				evaluator, err := core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalInt(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(3))
			})

			It("should convert float to integer", func() {
				evaluator, err := core.NewFieldEvaluator("${!index * 2.7}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalInt(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(2)) // 1 * 2.7 = 2.7 -> 2
			})

			It("should convert string to integer", func() {
				evaluator, err := core.NewFieldEvaluator("${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalInt(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(123))
			})

			It("should convert boolean to integer", func() {
				evaluator, err := core.NewFieldEvaluator("${!index == 0}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalInt(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(1)) // true -> 1
			})

			It("should return error for invalid string conversion", func() {
				evaluator, err := core.NewFieldEvaluator("${!meta('source')}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalInt(testBatch, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot convert string 'api' to int"))
				Expect(result).To(Equal(0))
			})
		})

		Context("EvalBool", func() {
			It("should return boolean result directly", func() {
				evaluator, err := core.NewFieldEvaluator("${!index == 0}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalBool(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeTrue())
			})

			It("should convert integer to boolean", func() {
				evaluator, err := core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalBool(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeTrue()) // 3 -> true
			})

			It("should convert zero to false", func() {
				evaluator, err := core.NewFieldEvaluator("${!len(batch) - 3}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalBool(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeFalse()) // 0 -> false
			})

			It("should convert empty string to false", func() {
				evaluator, err := core.NewFieldEvaluator("${!meta('nonexistent')}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalBool(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeFalse())
			})

			It("should return error for invalid string conversion", func() {
				evaluator, err := core.NewFieldEvaluator("${!meta('source')}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalBool(testBatch, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot convert string 'api' to bool"))
				Expect(result).To(BeFalse())
			})
		})

		Context("EvalFloat64", func() {
			It("should return float result directly", func() {
				evaluator, err := core.NewFieldEvaluator("${!index * 1.5}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalFloat64(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(1.5))
			})

			It("should convert integer to float", func() {
				evaluator, err := core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalFloat64(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(3.0))
			})

			It("should convert string to float", func() {
				// Use user_id which contains a numeric string "123"
				evaluator, err := core.NewFieldEvaluator("${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalFloat64(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(123.0))
			})

			It("should convert boolean to float", func() {
				evaluator, err := core.NewFieldEvaluator("${!index == 0}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalFloat64(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(1.0)) // true -> 1.0
			})

			It("should return error for invalid string conversion", func() {
				evaluator, err := core.NewFieldEvaluator("${!meta('source')}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalFloat64(testBatch, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot convert string 'api' to float64"))
				Expect(result).To(Equal(0.0))
			})

			It("should handle complex mathematical expressions", func() {
				evaluator, err := core.NewFieldEvaluator("${!(index + 1) * 100 / len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				result, err := evaluator.EvalFloat64(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNumerically("~", 66.66666666666667, 0.0001))
			})
		})

		Context("Error propagation in utility methods", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should propagate errors in EvalString", func() {
				result, err := evaluator.EvalString(testBatch, 10) // Invalid index
				Expect(err).To(HaveOccurred())
				Expect(result).To(Equal(""))
			})

			It("should propagate errors in EvalInt", func() {
				result, err := evaluator.EvalInt(testBatch, 10) // Invalid index
				Expect(err).To(HaveOccurred())
				Expect(result).To(Equal(0))
			})

			It("should propagate errors in EvalBool", func() {
				result, err := evaluator.EvalBool(testBatch, 10) // Invalid index
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeFalse())
			})

			It("should propagate errors in EvalFloat64", func() {
				result, err := evaluator.EvalFloat64(testBatch, 10) // Invalid index
				Expect(err).To(HaveOccurred())
				Expect(result).To(Equal(0.0))
			})
		})

		Context("Practical usage examples", func() {
			It("should demonstrate type-safe usage patterns", func() {
				// Example 1: Count evaluation
				countEvaluator, err := core.NewFieldEvaluator("${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				count, err := countEvaluator.EvalInt(testBatch, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(3))

				// Example 2: Conditional check
				conditionEvaluator, err := core.NewFieldEvaluator("${!index < len(batch) - 1}")
				Expect(err).ToNot(HaveOccurred())

				hasNext, err := conditionEvaluator.EvalBool(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(hasNext).To(BeTrue())

				// Example 3: Percentage calculation
				percentEvaluator, err := core.NewFieldEvaluator("${!(index + 1) * 100.0 / len(batch)}")
				Expect(err).ToNot(HaveOccurred())

				percent, err := percentEvaluator.EvalFloat64(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(percent).To(BeNumerically("~", 66.66666666666667, 0.0001))

				// Example 4: Formatted message
				messageEvaluator, err := core.NewFieldEvaluator("Processing ${!index + 1}/${!len(batch)}: ${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())

				message, err := messageEvaluator.EvalString(testBatch, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(message).To(Equal("Processing 2/3: 456"))
			})
		})
	})

	Describe("Utility methods", func() {
		Context("GetOriginal", func() {
			BeforeEach(func() {
				evaluator, err = core.NewFieldEvaluator("test ${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return the original field string", func() {
				Expect(evaluator.GetOriginal()).To(Equal("test ${!meta('user_id')}"))
			})
		})

		Context("HasExpressions", func() {
			It("should return false for literal text", func() {
				evaluator, err = core.NewFieldEvaluator("literal text")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator.HasExpressions()).To(BeFalse())
			})

			It("should return true for fields with expressions", func() {
				evaluator, err = core.NewFieldEvaluator("${!meta('user_id')}")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator.HasExpressions()).To(BeTrue())
			})
		})

		Context("GetExpressionCount", func() {
			It("should return 0 for literal text", func() {
				evaluator, err = core.NewFieldEvaluator("literal text")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator.GetExpressionCount()).To(Equal(0))
			})

			It("should return correct count for multiple expressions", func() {
				evaluator, err = core.NewFieldEvaluator("${!meta('a')} ${!meta('b')} ${!len(batch)}")
				Expect(err).ToNot(HaveOccurred())
				Expect(evaluator.GetExpressionCount()).To(Equal(3))
			})
		})
	})
})
