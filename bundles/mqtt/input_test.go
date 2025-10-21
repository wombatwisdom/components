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

		It("should NOT ACK message when downstream processing fails", func() {
			var err error
			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_ERROR_HANDLER",
				},
				Filters: map[string]byte{
					"ack-test/error": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			waitForSubscription(input)

			// Publish a test message
			pubToken := publisher.Publish("ack-test/error", 1, false, []byte("test-error-handling"))
			pubToken.Wait()
			Expect(pubToken.Error()).ToNot(HaveOccurred())

			// Read the message
			batch, callback, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(callback).ToNot(BeNil())

			// Verify we got the message
			msgs := maps.Collect(batch.Messages())
			Expect(msgs).To(HaveLen(1))
			raw, _ := msgs[0].Raw()
			Expect(raw).To(Equal([]byte("test-error-handling")))

			// Simulate a downstream error (e.g., NATS timeout)
			downstreamErr := fmt.Errorf("nats: timeout")
			err = callback(context.Background(), downstreamErr)
			Expect(err).ToNot(HaveOccurred())

			// Close and reconnect to verify message redelivery
			_ = input.Close(ctx)

			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_ERROR_HANDLER",
				},
				Filters: map[string]byte{
					"ack-test/error": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Should receive the message again as it wasn't ACKed due to error
			batch, callback, err = input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			msgs = maps.Collect(batch.Messages())
			Expect(msgs).To(HaveLen(1))
			raw, _ = msgs[0].Raw()
			Expect(raw).To(Equal([]byte("test-error-handling")))

			// This time ACK it successfully
			err = callback(context.Background(), nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should disconnect on downstream error forcing pipeline restart for redelivery", func() {
			var err error
			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_DISCONNECT_ON_ERROR",
				},
				Filters: map[string]byte{
					"ack-test/disconnect-error": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			waitForSubscription(input)

			// Publish a test message
			pubToken := publisher.Publish("ack-test/disconnect-error", 1, false, []byte("test-disconnect-on-error"))
			pubToken.Wait()
			Expect(pubToken.Error()).ToNot(HaveOccurred())

			// Read the first message
			batch1, callback1, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(callback1).ToNot(BeNil())

			// Verify we got the message
			msgs1 := maps.Collect(batch1.Messages())
			Expect(msgs1).To(HaveLen(1))
			raw1, _ := msgs1[0].Raw()
			Expect(raw1).To(Equal([]byte("test-disconnect-on-error")))

			// Track if we get a disconnection error
			disconnectChan := make(chan error, 1)

			// Start reading in the background - should get ErrNotConnected after disconnect
			go func() {
				// This should fail with ErrNotConnected after the disconnect
				_, _, err2 := input.Read(ctx)
				disconnectChan <- err2
			}()

			// Give the background reader time to start waiting
			time.Sleep(100 * time.Millisecond)

			// Simulate a downstream error - this should trigger disconnect on line 198
			downstreamErr := fmt.Errorf("nats: timeout")
			err = callback1(context.Background(), downstreamErr)
			Expect(err).ToNot(HaveOccurred())

			// Should get ErrNotConnected from the Read operation
			select {
			case err := <-disconnectChan:
				// We expect ErrNotConnected after the disconnect
				Expect(err).To(Equal(spec.ErrNotConnected))
			case <-time.After(2 * time.Second):
				Fail("Did not receive disconnect notification")
			}

			// Now simulate a pipeline restart by closing and reinitializing
			_ = input.Close(ctx)

			// Create a new input with the same client ID to simulate pipeline restart
			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_DISCONNECT_ON_ERROR",
				},
				Filters: map[string]byte{
					"ack-test/disconnect-error": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Should receive the message again as it wasn't ACKed due to error
			batch2, callback2, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			msgs2 := maps.Collect(batch2.Messages())
			Expect(msgs2).To(HaveLen(1))
			raw2, _ := msgs2[0].Raw()
			Expect(raw2).To(Equal([]byte("test-disconnect-on-error")))

			// This time ACK it successfully
			err = callback2(context.Background(), nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should ACK message when downstream processing succeeds", func() {
			var err error
			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_SUCCESS_HANDLER",
				},
				Filters: map[string]byte{
					"ack-test/success": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			waitForSubscription(input)

			// Publish a test message
			pubToken := publisher.Publish("ack-test/success", 1, false, []byte("test-success-handling"))
			pubToken.Wait()
			Expect(pubToken.Error()).ToNot(HaveOccurred())

			// Read the message
			batch, callback, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(callback).ToNot(BeNil())

			// Verify we got the message
			msgs := maps.Collect(batch.Messages())
			Expect(msgs).To(HaveLen(1))
			raw, _ := msgs[0].Raw()
			Expect(raw).To(Equal([]byte("test-success-handling")))

			// Simulate successful processing
			err = callback(context.Background(), nil)
			Expect(err).ToNot(HaveOccurred())

			// Close and reconnect to verify message was ACKed
			_ = input.Close(ctx)

			input, err = mqtt.NewInput(env, mqtt.InputConfig{
				CommonMQTTConfig: mqtt.CommonMQTTConfig{
					Urls:     []string{url},
					ClientId: "ACK_TEST_SUCCESS_HANDLER",
				},
				Filters: map[string]byte{
					"ack-test/success": 1,
				},
				CleanSession:  false,
				EnableAutoAck: false,
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Should NOT receive the message again as it was successfully ACKed
			readDone := make(chan bool)
			go func() {
				_, _, _ = input.Read(ctx)
				readDone <- true
			}()

			select {
			case <-readDone:
				Fail("Should not receive any message - it should have been ACKed")
			case <-time.After(1 * time.Second):
				// Success - no message redelivered
			}
		})
	})

})
