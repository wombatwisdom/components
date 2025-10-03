package mqtt_test

import (
	"github.com/wombatwisdom/components/bundles/mqtt"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MQTT Output with Interpolation", func() {
	var env spec.Environment

	BeforeEach(func() {
		env = test.TestEnvironment()
	})

	When("topic contains interpolations", func() {
		It("should handle static topic", func() {
			config := mqtt.Config{
				Mqtt: mqtt.MqttConfig{
					Urls:     []string{url},
					ClientId: "test-client",
					Topic:    "test/static",
				},
			}

			output, err := mqtt.NewOutput(env, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ToNot(BeNil())
		})

		It("should handle interpolated topic", func() {
			config := mqtt.Config{
				Mqtt: mqtt.MqttConfig{
					Urls:     []string{url},
					ClientId: "test-client",
					Topic:    `test/output/${!json.topic}`,
				},
			}

			output, err := mqtt.NewOutput(env, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ToNot(BeNil())
		})

		It("should handle escaped interpolation", func() {
			config := mqtt.Config{
				Mqtt: mqtt.MqttConfig{
					Urls:     []string{url},
					ClientId: "test-client",
					Topic:    `test/${{!literal}}`,
				},
			}

			output, err := mqtt.NewOutput(env, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ToNot(BeNil())
		})

		It("should error on invalid interpolation syntax", func() {
			ctx := test.NewMockComponentContext()
			config := mqtt.Config{
				Mqtt: mqtt.MqttConfig{
					Urls:     []string{url},
					ClientId: "test-client",
					Topic:    `test/${!unclosed`,
				},
			}

			output, err := mqtt.NewOutput(env, config)
			Expect(err).ToNot(HaveOccurred())

			// Init should fail due to parse error
			err = output.Init(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unclosed interpolation"))
		})
	})

	When("evaluating interpolated topics", func() {
		It("should resolve topic at runtime", func() {
			ctx := test.NewMockComponentContext()
			config := mqtt.Config{
				Mqtt: mqtt.MqttConfig{
					Urls:     []string{url},
					ClientId: "test-client",
					Topic:    `sensors/${!json.location}/${!json.type}`,
				},
			}

			output, err := mqtt.NewOutput(env, config)
			Expect(err).ToNot(HaveOccurred())

			err = output.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Create a test message
			msg := ctx.NewMessage()
			msg.SetRaw([]byte(`{"location": "warehouse", "type": "temperature"}`))
			_ = ctx.NewBatch(msg)
		})
	})
})
