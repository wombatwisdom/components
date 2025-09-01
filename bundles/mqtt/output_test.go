package mqtt_test

import (
	mqtt2 "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/bundles/mqtt"
	"github.com/wombatwisdom/components/framework/spec"
)

var _ = Describe("Output", func() {
	var output *mqtt.Output
	var ctx spec.ComponentContext

	BeforeEach(func() {
		var err error
		output, err = mqtt.NewOutput(env, mqtt.OutputConfig{
			CommonMQTTConfig: mqtt.CommonMQTTConfig{
				Urls:     []string{url},
				ClientId: "SINK",
			},
			TopicExpr: "test",
		})
		Expect(err).ToNot(HaveOccurred())

		_ = output.Init(ctx)
	})

	AfterEach(func() {
		_ = output.Close(ctx)
	})

	When("sending a message using the output", func() {
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
				Expect(output.Write(ctx, ctx.NewBatch(msg))).To(Succeed())
				Eventually(recv).Should(Receive())
			case msg := <-recv:
				Expect(msg.Payload()).To(Equal(b))
			}
		})
	})
})
