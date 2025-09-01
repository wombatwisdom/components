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

	Context("SetAutoAckDisabled parameter", func() {
		Describe("default behavior", func() {
			It("should default SetAutoAckDisabled to true when not specified", func() {
				input, err := mqtt.NewInput(env, mqtt.InputConfig{
					CommonMQTTConfig: mqtt.CommonMQTTConfig{
						Urls:     []string{url},
						ClientId: "TEST_DEFAULT",
					},
					Filters: map[string]byte{"test/default": 1},
					// SetAutoAckDisabled not specified - should default to true
				})
				Expect(err).ToNot(HaveOccurred())

				// Validate that default value is applied correctly
				Expect(input.InputConfig.SetAutoAckDisabled).ToNot(BeNil(), "SetAutoAckDisabled should be set to default value")
				Expect(*input.InputConfig.SetAutoAckDisabled).To(BeTrue(), "SetAutoAckDisabled should default to true")
			})
		})

		Describe("explicit configuration", func() {
			It("should respect SetAutoAckDisabled: true", func() {
				trueValue := true
				input, err := mqtt.NewInput(env, mqtt.InputConfig{
					CommonMQTTConfig: mqtt.CommonMQTTConfig{
						Urls:     []string{url},
						ClientId: "TEST_TRUE",
					},
					Filters:            map[string]byte{"test/true": 1},
					SetAutoAckDisabled: &trueValue,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(input.InputConfig.SetAutoAckDisabled).ToNot(BeNil())
				Expect(*input.InputConfig.SetAutoAckDisabled).To(BeTrue())
			})

			It("should respect SetAutoAckDisabled: false", func() {
				falseValue := false
				input, err := mqtt.NewInput(env, mqtt.InputConfig{
					CommonMQTTConfig: mqtt.CommonMQTTConfig{
						Urls:     []string{url},
						ClientId: "TEST_FALSE",
					},
					Filters:            map[string]byte{"test/false": 1},
					SetAutoAckDisabled: &falseValue,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(input.InputConfig.SetAutoAckDisabled).ToNot(BeNil())
				Expect(*input.InputConfig.SetAutoAckDisabled).To(BeFalse())
			})
		})

		Describe("ManualAckCollector creation", func() {
			It("should create ManualAckCollector when SetAutoAckDisabled is true", func() {
				trueValue := true
				input, err := mqtt.NewInput(env, mqtt.InputConfig{
					CommonMQTTConfig: mqtt.CommonMQTTConfig{
						Urls:     []string{url},
						ClientId: "TEST_MANUAL_ACK",
					},
					Filters:            map[string]byte{"test/manual": 1},
					SetAutoAckDisabled: &trueValue,
					PrefetchCount:      5,
				})
				Expect(err).ToNot(HaveOccurred())

				collector := test.NewListCollector()
				err = input.Connect(context.Background(), collector)
				Expect(err).ToNot(HaveOccurred())

				// Validate that ManualAckCollector is created
				Expect(input.GetManualAckCollector()).ToNot(BeNil(), "ManualAckCollector should be created when SetAutoAckDisabled is true")

				_ = input.Disconnect(context.Background())
			})

			It("should not create ManualAckCollector when SetAutoAckDisabled is false", func() {
				falseValue := false
				input, err := mqtt.NewInput(env, mqtt.InputConfig{
					CommonMQTTConfig: mqtt.CommonMQTTConfig{
						Urls:     []string{url},
						ClientId: "TEST_AUTO_ACK",
					},
					Filters:            map[string]byte{"test/auto": 1},
					SetAutoAckDisabled: &falseValue,
				})
				Expect(err).ToNot(HaveOccurred())

				collector := test.NewListCollector()
				err = input.Connect(context.Background(), collector)
				Expect(err).ToNot(HaveOccurred())

				// Validate that ManualAckCollector is not created
				Expect(input.GetManualAckCollector()).To(BeNil(), "ManualAckCollector should not be created when SetAutoAckDisabled is false")

				_ = input.Disconnect(context.Background())
			})
		})

		Describe("PrefetchCount validation", func() {
			It("should use default PrefetchCount of 10 when not specified", func() {
				trueValue := true
				input, err := mqtt.NewInput(env, mqtt.InputConfig{
					CommonMQTTConfig: mqtt.CommonMQTTConfig{
						Urls:     []string{url},
						ClientId: "TEST_PREFETCH_DEFAULT",
					},
					Filters:            map[string]byte{"test/prefetch": 1},
					SetAutoAckDisabled: &trueValue,
					// PrefetchCount not specified
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(input.InputConfig.PrefetchCount).To(Equal(10), "PrefetchCount should default to 10")
			})

			It("should respect custom PrefetchCount", func() {
				trueValue := true
				input, err := mqtt.NewInput(env, mqtt.InputConfig{
					CommonMQTTConfig: mqtt.CommonMQTTConfig{
						Urls:     []string{url},
						ClientId: "TEST_PREFETCH_CUSTOM",
					},
					Filters:            map[string]byte{"test/prefetch_custom": 1},
					SetAutoAckDisabled: &trueValue,
					PrefetchCount:      25,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(input.InputConfig.PrefetchCount).To(Equal(25))
			})
		})
	})
})
