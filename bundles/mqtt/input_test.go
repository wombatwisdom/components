package mqtt_test

import (
	"context"
	"fmt"
	"time"

	mqtt2 "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/bundles/mqtt"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var _ = Describe("Input", func() {
	var input *mqtt.Input

	var collector *test.ListCollector

	BeforeEach(func() {
		var err error
		input, err = mqtt.NewInput(env, mqtt.InputConfig{
			CommonMQTTConfig: mqtt.CommonMQTTConfig{
				Urls:     []string{url},
				ClientId: "SINK",
			},
			Filters: map[string]byte{
				"test": 1,
			},
		})
		Expect(err).ToNot(HaveOccurred())

		collector = test.NewListCollector()
		err = input.Connect(context.Background(), collector)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = input.Disconnect(context.Background())
		_ = collector.Disconnect()
	})

	When("sending a message to MQTT", func() {
		It("should receive the message on the input", func() {
			msg := spec.NewBytesMessage([]byte("hello, world"))
			b, err := msg.Raw()
			Expect(err).ToNot(HaveOccurred())

			tc := mqtt2.NewClient(mqtt2.NewClientOptions().AddBroker(url))
			tc.Connect().Wait()

			tc.Publish("test", 1, false, b).Wait()

			success := collector.WaitWithTimeout(10 * time.Second)
			Expect(success).To(BeTrue(), "Expected to receive message within timeout")

			Expect(collector.Messages()).To(HaveLen(1))
			GinkgoLogr.Info(fmt.Sprintf("Received messages: %v", collector.Messages()))
		})
	})
})
