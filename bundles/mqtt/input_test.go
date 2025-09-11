package mqtt_test

import (
	"context"
	"fmt"
	"maps"
	"time"

	mqtt2 "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/bundles/mqtt"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

func waitForSubscription(input *mqtt.Input) {
	// Small delay to ensure subscription handlers are fully set up
	// This is more reliable than checking connection state alone
	time.Sleep(200 * time.Millisecond)
}

var _ = Describe("Input", func() {
	var input *mqtt.Input
	var ctx spec.ComponentContext

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

		ctx = test.NewMockComponentContext()
		err = input.Init(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = input.Close(ctx)
	})

	When("sending a message to MQTT", func() {
		It("should receive the message on the input", func() {
			msg := spec.NewBytesMessage([]byte("hello, world"))
			b, err := msg.Raw()
			Expect(err).ToNot(HaveOccurred())

			tc := mqtt2.NewClient(mqtt2.NewClientOptions().AddBroker(url))
			tc.Connect().Wait()

			received := make(chan spec.Batch)
			defer close(received)
			receivedErr := make(chan error)
			defer close(receivedErr)

			// start reading messages
			go func() {
				batch, cb, err := input.Read(ctx)
				if err != nil {
					receivedErr <- err
					return
				}

				if cb != nil {
					_ = cb(ctx.Context(), nil)
				}

				received <- batch
			}()

			tc.Publish("test", 1, false, b).Wait()

			select {
			case <-time.After(5 * time.Second):
				Fail("Did not receive message within timeout")
			case err := <-receivedErr:
				Fail(fmt.Sprintf("Error reading message: %v", err))
			case batch := <-received:
				Expect(batch).ToNot(BeNil())
				msgs := maps.Collect(batch.Messages())
				Expect(msgs).To(HaveLen(1))
			}
		})
	})
})

var _ = Describe("Input ACK behavior", func() {
	var input *mqtt.Input
	var ctx spec.ComponentContext
	var publisher mqtt2.Client

	BeforeEach(func() {
		// Set up publisher
		publisher = mqtt2.NewClient(mqtt2.NewClientOptions().
			AddBroker(url).
			SetClientID("ACK_TEST_PUBLISHER"))
		token := publisher.Connect()
		token.Wait()
		Expect(token.Error()).ToNot(HaveOccurred())

		ctx = test.NewMockComponentContext()
	})

	AfterEach(func() {
		if input != nil {
			_ = input.Close(ctx)
		}
		if publisher != nil && publisher.IsConnected() {
			publisher.Disconnect(250)
		}
	})

	When("testing ACK behavior", func() {
		It("should NOT ACK message when context is cancelled", func() {
			var err error
			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_SUBSCRIBER_2",
				},
				Filters: map[string]byte{
					"ack-test/cancel": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			waitForSubscription(input)

			pubToken := publisher.Publish("ack-test/cancel", 1, false, []byte("test-no-ack"))
			pubToken.Wait()
			Expect(pubToken.Error()).ToNot(HaveOccurred())

			_, callback, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(callback).ToNot(BeNil())

			cancelCtx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel context immediately

			// cancelled context - should NOT ACK
			// we're not checking that we don't ack here.
			// Which is fine, we're verifying that it gets redelivered later.
			err = callback(cancelCtx, nil)
			Expect(err).ToNot(HaveOccurred()) // No error expected, just no ACK

			_ = input.Close(ctx)

			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_SUBSCRIBER_2",
				},
				Filters: map[string]byte{
					"ack-test/cancel": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Should receive the message again as it wasn't ACKed
			batch, _, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			msgs := maps.Collect(batch.Messages())
			Expect(msgs).To(HaveLen(1))
			raw, _ := msgs[0].Raw()
			Expect(raw).To(Equal([]byte("test-no-ack")))
		})

		It("should handle ACK when client disconnects with valid context", func() {
			var err error
			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_DISCONNECT",
				},
				Filters: map[string]byte{
					"ack-test/disconnect": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false, // Manual ACK to test disconnect behavior
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			waitForSubscription(input)

			pubToken := publisher.Publish("ack-test/disconnect", 1, false, []byte("test-disconnect"))
			pubToken.Wait()
			Expect(pubToken.Error()).ToNot(HaveOccurred())

			// Read the message and get the callback
			_, callback, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(callback).ToNot(BeNil())

			// Forcibly disconnect the client to simulate network failure
			_ = input.Close(ctx) // disconnects the client

			// Call callback with a VALID context (not cancelled)
			validCtx := context.Background()

			// This should NOT panic and should log about skipping ACK
			err = callback(validCtx, nil)
			Expect(err).ToNot(HaveOccurred()) // Should handle gracefully, no panic
		})

		It("should auto-ACK when EnableAutoAck is true", func() {
			var err error
			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_SUBSCRIBER_3",
				},
				Filters: map[string]byte{
					"ack-test/auto": 1,
				},
				CleanSession:  false,
				EnableAutoAck: true, // Auto-ACK enabled
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			waitForSubscription(input)

			pubToken := publisher.Publish("ack-test/auto", 1, false, []byte("test-auto-ack"))
			pubToken.Wait()
			Expect(pubToken.Error()).ToNot(HaveOccurred())

			_, callback, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(callback).ToNot(BeNil())

			// Even with cancelled context, message should already be ACKed
			cancelCtx, cancel := context.WithCancel(context.Background())
			cancel()

			err = callback(cancelCtx, nil)
			Expect(err).ToNot(HaveOccurred())

			_ = input.Close(ctx)

			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_SUBSCRIBER_3_VERIFY",
				},
				Filters: map[string]byte{
					"ack-test/auto": 1,
				},
				CleanSession:  false,
				EnableAutoAck: true,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Should NOT receive the message again as it was auto-ACKed
			readDone := make(chan bool)
			go func() {
				_, _, _ = input.Read(ctx)
				readDone <- true
			}()

			select {
			case <-readDone:
				Fail("Should not receive any message - it should have been auto-ACKed")
			case <-time.After(1 * time.Second):
				// Success - no message redelivered
			}
		})
	})

})
