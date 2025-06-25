package mqtt_test

import (
	"context"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/mqtt"
	"github.com/wombatwisdom/components/spec"
	"github.com/wombatwisdom/components/test"
)

var _ = Describe("Roundtrip", func() {
	var src *mqtt.Source
	var snk *mqtt.Sink

	var collector *test.ListCollector

	BeforeEach(func() {
		var err error
		src, err = mqtt.NewSource(env, mqtt.SourceConfig{
			CommonMQTTConfig: mqtt.CommonMQTTConfig{
				Urls:     []string{url},
				ClientId: uuid.New().String(),
			},
			Filters: map[string]byte{
				"test": 1,
			},
		})
		Expect(err).ToNot(HaveOccurred())

		snk, err = mqtt.NewSink(env, mqtt.SinkConfig{
			CommonMQTTConfig: mqtt.CommonMQTTConfig{
				Urls:     []string{url},
				ClientId: uuid.New().String(),
			},
			QOS:       1,
			TopicExpr: "test",
		})
		Expect(err).ToNot(HaveOccurred())

		err = snk.Connect(context.Background())
		Expect(err).ToNot(HaveOccurred())

		collector = test.NewListCollector()
		err = src.Connect(context.Background(), collector)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		src.Disconnect(context.Background())
		snk.Disconnect(context.Background())
		collector.Disconnect()
	})

	When("sending a message to the sink", func() {
		It("should receive the message on the source", func() {
			msg := spec.NewBytesMessage([]byte("hello, world"))

			err := snk.Write(context.Background(), msg)
			Expect(err).ToNot(HaveOccurred())

			collector.Wait()
			Expect(collector.Messages()).To(HaveLen(1))
		})
	})
})
