//go:build mqclient

package ibm_mq_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ibm_mq "github.com/wombatwisdom/components/bundles/ibm-mq"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var _ = Describe("Roundtrip", func() {
	var input *ibm_mq.Input
	var output *ibm_mq.Output
	var ctx spec.ComponentContext

	BeforeEach(func() {
		ctx = test.NewMockComponentContext()
		env := test.TestEnvironment()

		// Create output
		queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
		Expect(err).ToNot(HaveOccurred())

		output, _ = ibm_mq.NewOutput(env, ibm_mq.OutputConfig{
			CommonMQConfig: ibm_mq.CommonMQConfig{
				QueueManagerName: "QM1",
				ConnectionName:   "",         // fallback to MQSERVER env var
				UserId:           "app",      // testcontainer default
				Password:         "passw0rd", // testcontainer default
			},
			QueueExpr: queueName,
		})

		err = output.Init(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Create input
		input, _ = ibm_mq.NewInput(env, ibm_mq.InputConfig{
			CommonMQConfig: ibm_mq.CommonMQConfig{
				QueueManagerName: "QM1",
				ConnectionName:   "",         // fallback to MQSERVER env var
				UserId:           "app",      // testcontainer default
				Password:         "passw0rd", // testcontainer default
			},
			QueueName: "DEV.QUEUE.1",
		})

		err = input.Init(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = input.Close(ctx)
		_ = output.Close(ctx)
	})

	When("sending a message through output and reading through input", func() {
		It("should successfully roundtrip the message", func() {
			testData := []byte("roundtrip test message")

			// Create and send message through output
			msg := ctx.NewMessage()
			msg.SetRaw(testData)
			batch := ctx.NewBatch(msg)

			err := output.Write(ctx, batch)
			Expect(err).ToNot(HaveOccurred())

			// Read message through input
			receivedBatch, ackFn, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(receivedBatch).ToNot(BeNil())
			Expect(ackFn).ToNot(BeNil())

			// Verify the message content
			var messageCount int
			var receivedData []byte
			for _, msg := range receivedBatch.Messages() {
				messageCount++
				receivedData, err = msg.Raw()
				Expect(err).ToNot(HaveOccurred())
			}
			Expect(messageCount).To(Equal(1))
			Expect(receivedData).To(Equal(testData))

			// Acknowledge the message
			err = ackFn(ctx.Context(), nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("sending multiple messages", func() {
		It("should successfully roundtrip all messages", func() {
			testData1 := []byte("first message")
			testData2 := []byte("second message")
			testData3 := []byte("third message")

			// Send three messages
			for _, data := range [][]byte{testData1, testData2, testData3} {
				msg := ctx.NewMessage()
				msg.SetRaw(data)
				batch := ctx.NewBatch(msg)

				err := output.Write(ctx, batch)
				Expect(err).ToNot(HaveOccurred())
			}

			// Read three messages
			receivedMessages := make([][]byte, 0, 3)
			for i := 0; i < 3; i++ {
				receivedBatch, ackFn, err := input.Read(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(receivedBatch).ToNot(BeNil())
				Expect(ackFn).ToNot(BeNil())

				for _, msg := range receivedBatch.Messages() {
					data, err := msg.Raw()
					Expect(err).ToNot(HaveOccurred())
					receivedMessages = append(receivedMessages, data)
				}

				// Acknowledge the message
				err = ackFn(ctx.Context(), nil)
				Expect(err).ToNot(HaveOccurred())
			}

			// Verify all messages were received
			Expect(receivedMessages).To(HaveLen(3))
			Expect(receivedMessages).To(ContainElements(testData1, testData2, testData3))
		})
	})

	When("sending a batch of messages", func() {
		It("should successfully roundtrip the batch", func() {
			testData1 := []byte("batch message 1")
			testData2 := []byte("batch message 2")

			// Create a batch with multiple messages
			msg1 := ctx.NewMessage()
			msg1.SetRaw(testData1)
			msg2 := ctx.NewMessage()
			msg2.SetRaw(testData2)
			batch := ctx.NewBatch(msg1, msg2)

			err := output.Write(ctx, batch)
			Expect(err).ToNot(HaveOccurred())

			// Read messages (may come as separate reads since we're not batching on input by default)
			receivedMessages := make([][]byte, 0, 2)
			for i := 0; i < 2; i++ {
				receivedBatch, ackFn, err := input.Read(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(receivedBatch).ToNot(BeNil())
				Expect(ackFn).ToNot(BeNil())

				for _, msg := range receivedBatch.Messages() {
					data, err := msg.Raw()
					Expect(err).ToNot(HaveOccurred())
					receivedMessages = append(receivedMessages, data)
				}

				// Acknowledge the message
				err = ackFn(ctx.Context(), nil)
				Expect(err).ToNot(HaveOccurred())
			}

			// Verify both messages were received
			Expect(receivedMessages).To(HaveLen(2))
			Expect(receivedMessages).To(ContainElements(testData1, testData2))
		})
	})
})
