package mqtt_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
	"github.com/wombatwisdom/components/mqtt"
)

var _ = Describe("Roundtrip", func() {
	var input *mqtt.Input
	var output *mqtt.Output

	var collector *test.ListCollector

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

		output, err = mqtt.NewOutput(env, mqtt.OutputConfig{
			CommonMQTTConfig: mqtt.CommonMQTTConfig{
				Urls:     []string{url},
				ClientId: uuid.New().String(),
			},
			QOS:       1,
			TopicExpr: "test",
		})
		Expect(err).ToNot(HaveOccurred())

		err = output.Connect(context.Background())
		Expect(err).ToNot(HaveOccurred())

		collector = test.NewListCollector()
		err = input.Connect(context.Background(), collector)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = input.Disconnect(context.Background())
		_ = output.Disconnect(context.Background())
		_ = collector.Disconnect()
	})

	When("sending a message to the output", func() {
		It("should receive the message on the input", func() {
			msg := spec.NewBytesMessage([]byte("hello, world"))

			err := output.Write(context.Background(), msg)
			Expect(err).ToNot(HaveOccurred())

			collector.Wait()
			Expect(collector.Messages()).To(HaveLen(1))
		})
	})
})
