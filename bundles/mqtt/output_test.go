package mqtt_test

import (
	mqtt3 "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	wwmqtt "github.com/wombatwisdom/components/bundles/mqtt"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var _ = Describe("MQTTOutput", func() {
	var output *wwmqtt.Output
	var ctx spec.ComponentContext

	BeforeEach(func() {
		var err error
		output, err = wwmqtt.NewOutput(env, wwmqtt.Config{
			Mqtt: wwmqtt.MqttConfig{
				Urls:     []string{url},
				ClientId: "SINK",
				Topic:    `"test"`,
			},
		})
		Expect(err).ToNot(HaveOccurred())

		ctx = test.NewMockComponentContext()
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

			recv := make(chan mqtt3.Message)
			ready := make(chan struct{})
			tc := mqtt3.NewClient(mqtt3.NewClientOptions().AddBroker(url).SetOnConnectHandler(func(c mqtt3.Client) {
				tok := c.Subscribe("test", 1, func(client mqtt3.Client, msg mqtt3.Message) {
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
