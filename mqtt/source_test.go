package mqtt_test

import (
	"context"
	"fmt"
	mqtt2 "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/mqtt"
	"github.com/wombatwisdom/components/spec"
	"github.com/wombatwisdom/components/test"
	"io"
)

var _ = Describe("Source", func() {
	var src *mqtt.Source

	var collector *test.ListCollector

	BeforeEach(func() {
		var err error
		src, err = mqtt.NewSource(env, mqtt.SourceConfig{
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
		err = src.Connect(context.Background(), collector)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		src.Disconnect(context.Background())
		collector.Disconnect(context.Background())
	})

	When("sending a message to MQTT", func() {
		It("should receive the message on the source", func() {
			msg := spec.NewBytesMessage([]byte("hello, world"), nil)
			r, err := msg.Data()
			Expect(err).ToNot(HaveOccurred())
			b, err := io.ReadAll(r)
			Expect(err).ToNot(HaveOccurred())

			tc := mqtt2.NewClient(mqtt2.NewClientOptions().AddBroker(url))
			tc.Connect().Wait()

			tc.Publish("test", 1, false, b).Wait()
			collector.Wait()

			Expect(collector.Messages()).To(HaveLen(1))
			GinkgoLogr.Info(fmt.Sprintf("Received messages: %v", collector.Messages()))
		})
	})
})
