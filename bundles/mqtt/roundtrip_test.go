package mqtt_test

import (
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/bundles/mqtt"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var _ = Describe("Roundtrip", func() {
	var input *mqtt.Input
	var output *mqtt.Output
	var ctx spec.ComponentContext

	BeforeEach(func() {
		var err error
		input, err = mqtt.NewInput(env, mqtt.InputConfig{
			CommonMQTTConfig: mqtt.CommonMQTTConfig{
				Urls:     []string{url},
				ClientId: uuid.New().String(),
			},
			Filters: map[string]byte{
				"test": 1,
			},
		})
		Expect(err).ToNot(HaveOccurred())

		ctx = test.NewMockComponentContext()

		topic, err := spec.NewExprLangExpression("${!\"test\"}")
		Expect(err).ToNot(HaveOccurred())

		output, err = mqtt.NewOutput(env, mqtt.OutputConfig{
			CommonMQTTConfig: mqtt.CommonMQTTConfig{
				Urls:     []string{url},
				ClientId: uuid.New().String(),
			},
			QOS:       1,
			TopicExpr: topic,
		})
		Expect(err).ToNot(HaveOccurred())

		err = output.Init(ctx)
		Expect(err).ToNot(HaveOccurred())

		err = input.Init(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Allow time for MQTT subscription to be fully established
		// The subscription happens asynchronously in the OnConnectHandler
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		_ = input.Close(ctx)
		_ = output.Close(ctx)
	})

	When("sending a message to the output", func() {
		It("should receive the message on the input", func() {
			msg := spec.NewBytesMessage([]byte("hello, world"))

			received := make(chan spec.Batch)
			defer close(received)
			receivedErr := make(chan error)
			defer close(receivedErr)

			// start reading messages
			go func() {
				batch, cb, err := input.Read(ctx)
				if err != nil {
					receivedErr <- err
					return
				}

				if cb != nil {
					_ = cb(ctx.Context(), nil)
				}

				received <- batch
			}()

			err := output.Write(ctx, ctx.NewBatch(msg))
			Expect(err).ToNot(HaveOccurred())

			select {
			case <-time.After(5 * time.Second):
				Fail("Did not receive message within timeout")
			case err := <-receivedErr:
				Fail(fmt.Sprintf("Error reading message: %v", err))
			case batch := <-received:
				Expect(batch).ToNot(BeNil())
				msgs := maps.Collect(batch.Messages())
				Expect(msgs).To(HaveLen(1))
			}
		})
	})
})
