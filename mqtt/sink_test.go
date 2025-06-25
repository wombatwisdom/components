package mqtt_test

import (
	"context"

	mqtt2 "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/mqtt"
	"github.com/wombatwisdom/components/spec"
)

var _ = Describe("Sink", func() {
	var snk *mqtt.Sink

	BeforeEach(func() {
		var err error
		snk, err = mqtt.NewSink(env, mqtt.SinkConfig{
			CommonMQTTConfig: mqtt.CommonMQTTConfig{
				Urls:     []string{url},
				ClientId: "SINK",
			},
			TopicExpr: "test",
		})
		Expect(err).ToNot(HaveOccurred())

		_ = snk.Connect(context.Background())
	})

	AfterEach(func() {
		_ = snk.Disconnect(context.Background())
	})

	When("sending a message using the sink", func() {
		It("should put the message on the MQTT server", func() {
			msg := spec.NewBytesMessage([]byte("hello, world"))
			b, err := msg.Raw()
			Expect(err).ToNot(HaveOccurred())

			recv := make(chan mqtt2.Message)
			ready := make(chan struct{})
			tc := mqtt2.NewClient(mqtt2.NewClientOptions().AddBroker(url).SetOnConnectHandler(func(c mqtt2.Client) {
				tok := c.Subscribe("test", 1, func(client mqtt2.Client, msg mqtt2.Message) {
					recv <- msg
				})
				tok.Wait()
				close(ready)
			}))
			tok := tc.Connect()
			tok.Wait()
			Expect(tok.Error()).ToNot(HaveOccurred())

			select {
			case <-ready:
				Expect(snk.Write(context.Background(), msg)).To(Succeed())
				Eventually(recv).Should(Receive())
			case msg := <-recv:
				Expect(msg.Payload()).To(Equal(b))
			}
		})
	})
})
