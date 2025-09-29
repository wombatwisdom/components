package mqtt_test

import (
	mqtt3 "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/bundles/mqtt"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var _ = Describe("Output Topic Interpolation", func() {
	var ctx spec.ComponentContext

	BeforeEach(func() {
		ctx = test.NewMockComponentContext()
	})

	Describe("Topic Expression Parsing", func() {
		It("should support static quoted topics", func() {
			output, err := mqtt.NewOutput(env, mqtt.OutputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "test-static",
				},
				TopicExpr: `"test/mqtt/output"`,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ToNot(BeNil())

			err = output.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer output.Close(ctx)
		})

		It("should support expr-lang field references", func() {
			output, err := mqtt.NewOutput(env, mqtt.OutputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "test-bloblang",
				},
				TopicExpr: `"devices/" + json.device_id + "/data"`,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ToNot(BeNil())

			err = output.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer output.Close(ctx)
		})

		It("should handle Benthos interpolation in quoted strings", func() {
			// Test if ${!...} syntax works within quoted strings
			output, err := mqtt.NewOutput(env, mqtt.OutputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "test-benthos-interp",
				},
				TopicExpr: `"data/output/${!json(\"device_id\")}"`,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ToNot(BeNil())

			err = output.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer output.Close(ctx)
		})
	})

	Describe("Dynamic Topic Publishing", func() {
		type receivedMsg struct {
			topic   string
			message mqtt3.Message
		}
		var msgChan chan receivedMsg
		var client mqtt3.Client
		var ready chan struct{}

		BeforeEach(func() {
			msgChan = make(chan receivedMsg, 10)
			ready = make(chan struct{})

			// Set up MQTT client to subscribe to multiple topics
			opts := mqtt3.NewClientOptions().
				AddBroker(url).
				SetClientID("test-subscriber").
				SetOnConnectHandler(func(c mqtt3.Client) {
					// Subscribe to wildcard to catch all test messages
					tok := c.Subscribe("test/+/+", 1, func(client mqtt3.Client, msg mqtt3.Message) {
						msgChan <- receivedMsg{topic: msg.Topic(), message: msg}
					})
					tok.Wait()

					// Also subscribe to specific topics
					c.Subscribe("devices/+/data", 1, func(client mqtt3.Client, msg mqtt3.Message) {
						msgChan <- receivedMsg{topic: msg.Topic(), message: msg}
					})

					c.Subscribe("data/output/+", 1, func(client mqtt3.Client, msg mqtt3.Message) {
						msgChan <- receivedMsg{topic: msg.Topic(), message: msg}
					})

					close(ready)
				})

			client = mqtt3.NewClient(opts)
			tok := client.Connect()
			tok.Wait()
			Expect(tok.Error()).ToNot(HaveOccurred())
			<-ready
		})

		AfterEach(func() {
			client.Disconnect(250)
			close(msgChan) // Clean up the channel
		})

		It("should publish to dynamic topic based on message content", func() {
			output, err := mqtt.NewOutput(env, mqtt.OutputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "test-dynamic-content",
				},
				TopicExpr: `"devices/" + json.device_id + "/data"`,
			})
			Expect(err).ToNot(HaveOccurred())

			err = output.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer output.Close(ctx)

			// Create a message with device_id field
			msg := test.NewMockMessage([]byte(`{"device_id": "sensor123", "temperature": 25.5}`))

			err = output.Write(ctx, ctx.NewBatch(msg))
			Expect(err).ToNot(HaveOccurred())

			// Check if message was received on the expected topic
			Eventually(msgChan, "5s").Should(Receive(WithTransform(func(msg receivedMsg) string {
				return msg.topic
			}, Equal("devices/sensor123/data"))))
		})

		It("should handle metadata in topic expressions", func() {
			output, err := mqtt.NewOutput(env, mqtt.OutputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "test-metadata",
				},
				TopicExpr: `metadata.topic_prefix + "/" + json.type`,
			})
			Expect(err).ToNot(HaveOccurred())

			err = output.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer output.Close(ctx)

			// Create a message with metadata
			msg := test.NewMockMessage([]byte(`{"type": "temperature", "value": 22.3}`))
			msg.SetMetadata("topic_prefix", "test/sensors")

			err = output.Write(ctx, ctx.NewBatch(msg))
			Expect(err).ToNot(HaveOccurred())

			Eventually(msgChan, "5s").Should(Receive(WithTransform(func(msg receivedMsg) string {
				return msg.topic
			}, Equal("test/sensors/temperature"))))
		})
	})

	Describe("Error Handling", func() {
		It("should error on invalid Bloblang expressions", func() {
			output, err := mqtt.NewOutput(env, mqtt.OutputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "test-invalid",
				},
				TopicExpr: `this.nonexistent.field.deep`, // Invalid expression
			})
			Expect(err).ToNot(HaveOccurred())

			err = output.Init(ctx)

			if err == nil {
				defer output.Close(ctx)

				msg := test.NewMockMessage([]byte(`{"foo": "bar"}`))
				err = output.Write(ctx, ctx.NewBatch(msg))
				// Should error because the field doesn't exist
				Expect(err).To(HaveOccurred())
			}
		})

		It("should handle unquoted literals correctly", func() {
			output, err := mqtt.NewOutput(env, mqtt.OutputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "test-unquoted",
				},
				TopicExpr: `test/mqtt/output`, // Unquoted - will be interpreted as division
			})
			Expect(err).ToNot(HaveOccurred())

			err = output.Init(ctx)
			if err == nil {
				defer output.Close(ctx)

				msg := test.NewMockMessage([]byte(`{"test": "value"}`))
				err = output.Write(ctx, ctx.NewBatch(msg))
				// This should error because it's trying to divide
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid operation"))
			}
		})
	})
})
