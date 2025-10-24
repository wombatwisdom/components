//go:build mqclient

package ibm_mq_test

import (
	"fmt"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ibm_mq "github.com/wombatwisdom/components/bundles/ibm-mq"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var clearQueue = func() {
	cno := ibmmq.NewMQCNO()
	csp := ibmmq.NewMQCSP()
	csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
	csp.UserId = "app"
	csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
	cno.SecurityParms = csp

	qMgr, _ := ibmmq.Connx("QM1", cno)
	defer qMgr.Disc()

	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = "DEV.QUEUE.1"
	openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF

	qObj, _ := qMgr.Open(mqod, openOptions)
	defer qObj.Close(ibmmq.MQCO_NONE)

	// Consume all messages
	getmqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	buffer := make([]byte, 1024)

	for {
		_, err := qObj.Get(getmqmd, gmo, buffer)
		if err != nil {
			break
		}
	}
}

var _ = Describe("Input", func() {
	var input *ibm_mq.Input
	var ctx spec.ComponentContext

	BeforeEach(func() {
		ctx = test.NewMockComponentContext()

		input, _ = ibm_mq.NewInput(test.TestEnvironment(), ibm_mq.InputConfig{
			CommonMQConfig: ibm_mq.CommonMQConfig{
				QueueManagerName: "QM1",
				ConnectionName:   "",         // fallback to MQSERVER env var
				UserId:           "app",      // testcontainer default
				Password:         "passw0rd", // #nosec G101 - testcontainer default credential
			},
			QueueName: "DEV.QUEUE.1", // testcontainer default developer queue
		})

		err := input.Init(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = input.Close(ctx)
	})

	When("sending a message to IBM MQ", func() {
		It("should receive the message on the input", func() {
			testData := []byte("hello, world")

			// Connect to queue manager to send test message
			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"
			csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())
			defer qMgr.Disc()

			// Open the queue for writing
			mqod := ibmmq.NewMQOD()
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1"
			openOptions := ibmmq.MQOO_OUTPUT

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Put message to queue
			putmqmd := ibmmq.NewMQMD()
			pmo := ibmmq.NewMQPMO()
			pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

			err = qObj.Put(putmqmd, pmo, testData)
			Expect(err).ToNot(HaveOccurred())

			// Now read the message using the input
			batch, ackFn, err := input.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(batch).ToNot(BeNil())
			Expect(ackFn).ToNot(BeNil())

			// Verify the message content
			var messageCount int
			var receivedData []byte
			for _, msg := range batch.Messages() {
				messageCount++
				receivedData, err = msg.Raw()
				Expect(err).ToNot(HaveOccurred())
			}
			Expect(messageCount).To(Equal(1))
			Expect(receivedData).To(Equal(testData))

			// Acknowledge the message
			err = ackFn(ctx.Context(), nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when using batch processing", func() {
		It("should read multiple messages as a batch when batch_size > 1", func() {
			// Create input with batch_size = 3
			batchInput, _ := ibm_mq.NewInput(test.TestEnvironment(), ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueName: "DEV.QUEUE.1",
				BatchSize: 3,
			})

			err := batchInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchInput.Close(ctx)

			// Connect to queue manager to send test messages
			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"
			csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())
			defer qMgr.Disc()

			// Open the queue for writing
			mqod := ibmmq.NewMQOD()
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1"
			openOptions := ibmmq.MQOO_OUTPUT

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Put 3 messages to queue
			putmqmd := ibmmq.NewMQMD()
			pmo := ibmmq.NewMQPMO()
			pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

			testData := [][]byte{
				[]byte("batch message 1"),
				[]byte("batch message 2"),
				[]byte("batch message 3"),
			}

			for _, data := range testData {
				err = qObj.Put(putmqmd, pmo, data)
				Expect(err).ToNot(HaveOccurred())
			}

			// Read the batch
			batch, ackFn, err := batchInput.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(batch).ToNot(BeNil())
			Expect(ackFn).ToNot(BeNil())

			// Verify we got all 3 messages
			var collectedMessages []spec.Message
			for _, msg := range batch.Messages() {
				collectedMessages = append(collectedMessages, msg)
			}
			Expect(collectedMessages).To(HaveLen(3), "Should receive 3 messages in batch")

			// Verify message content
			for i, msg := range collectedMessages {
				receivedData, err := msg.Raw()
				Expect(err).ToNot(HaveOccurred())
				Expect(receivedData).To(Equal(testData[i]))
			}

			// Acknowledge the batch
			err = ackFn(ctx.Context(), nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return partial batch when fewer messages than batch_size are available", func() {
			// Create input with batch_size = 5
			batchInput, _ := ibm_mq.NewInput(test.TestEnvironment(), ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueName:     "DEV.QUEUE.1",
				BatchSize:     5,
				BatchWaitTime: "100ms", // Short wait time
			})

			err := batchInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchInput.Close(ctx)

			// Connect and send only 2 messages
			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"
			csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())
			defer qMgr.Disc()

			mqod := ibmmq.NewMQOD()
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1"
			openOptions := ibmmq.MQOO_OUTPUT

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Put only 2 messages
			putmqmd := ibmmq.NewMQMD()
			pmo := ibmmq.NewMQPMO()
			pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

			testData := [][]byte{
				[]byte("partial batch 1"),
				[]byte("partial batch 2"),
			}

			for _, data := range testData {
				err = qObj.Put(putmqmd, pmo, data)
				Expect(err).ToNot(HaveOccurred())
			}

			// Read should return partial batch
			batch, ackFn, err := batchInput.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(batch).ToNot(BeNil())

			// Should get 2 messages even though batch_size is 5
			var messages []spec.Message
			for _, msg := range batch.Messages() {
				messages = append(messages, msg)
			}
			Expect(messages).To(HaveLen(2), "Should receive partial batch of 2 messages")

			// Acknowledge the batch
			err = ackFn(ctx.Context(), nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle empty queue gracefully", func() {
			// Create input with batch_size = 3
			batchInput, _ := ibm_mq.NewInput(test.TestEnvironment(), ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueName:     "DEV.QUEUE.1",
				BatchSize:     3,
				BatchWaitTime: "100ms", // Short wait time
			})

			err := batchInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchInput.Close(ctx)

			// Ensure queue is empty by consuming any existing messages
			clearQueue()

			// Try to read from empty queue - should wait and then return appropriate response
			batch, ackFn, err := batchInput.Read(ctx)

			// Current implementation returns single empty batch or waits
			// This behavior should be documented
			if err == nil && batch != nil {
				var messages []spec.Message
				for _, msg := range batch.Messages() {
					messages = append(messages, msg)
				}
				Expect(messages).To(HaveLen(0), "Empty queue should return empty batch or wait")
				if ackFn != nil {
					ackFn(ctx.Context(), nil)
				}
			}
		})

		It("should rollback entire batch on acknowledgment failure", func() {
			// Create input with batch_size = 2
			batchInput, _ := ibm_mq.NewInput(test.TestEnvironment(), ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueName: "DEV.QUEUE.1",
				BatchSize: 2,
			})

			err := batchInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchInput.Close(ctx)

			// Send 2 test messages
			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"
			csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())
			defer qMgr.Disc()

			mqod := ibmmq.NewMQOD()
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1"
			openOptions := ibmmq.MQOO_OUTPUT | ibmmq.MQOO_BROWSE

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Put 2 messages
			putmqmd := ibmmq.NewMQMD()
			pmo := ibmmq.NewMQPMO()
			pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

			testData := [][]byte{
				[]byte("rollback test 1"),
				[]byte("rollback test 2"),
			}

			for _, data := range testData {
				err = qObj.Put(putmqmd, pmo, data)
				Expect(err).ToNot(HaveOccurred())
			}

			// Read the batch
			batch, ackFn, err := batchInput.Read(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(batch).ToNot(BeNil())

			var batchMessages []spec.Message
			for _, msg := range batch.Messages() {
				batchMessages = append(batchMessages, msg)
			}
			Expect(batchMessages).To(HaveLen(2))

			// Simulate failure by passing error to ack function
			err = ackFn(ctx.Context(), fmt.Errorf("simulated processing error"))
			Expect(err).ToNot(HaveOccurred(), "Rollback should succeed")

			// Messages should still be in queue after rollback
			// Try reading again to verify
			batch2, ackFn2, err := batchInput.Read(ctx)
			Expect(err).ToNot(HaveOccurred())

			var batch2Messages []spec.Message
			for _, msg := range batch2.Messages() {
				batch2Messages = append(batch2Messages, msg)
			}
			Expect(batch2Messages).To(HaveLen(2), "Messages should be available after rollback")

			// Properly acknowledge this time
			err = ackFn2(ctx.Context(), nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should process large batches efficiently", func() {
			// Create input with large batch_size
			batchInput, _ := ibm_mq.NewInput(test.TestEnvironment(), ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueName: "DEV.QUEUE.1",
				BatchSize: 50,
			})

			err := batchInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchInput.Close(ctx)

			// Send many messages
			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"
			csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())
			defer qMgr.Disc()

			mqod := ibmmq.NewMQOD()
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1"
			openOptions := ibmmq.MQOO_OUTPUT

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Put 50 messages
			putmqmd := ibmmq.NewMQMD()
			pmo := ibmmq.NewMQPMO()
			pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

			for i := 0; i < 50; i++ {
				data := []byte(fmt.Sprintf("large batch message %d", i))
				err = qObj.Put(putmqmd, pmo, data)
				Expect(err).ToNot(HaveOccurred())
			}

			// Time the batch read
			start := time.Now()
			batch, ackFn, err := batchInput.Read(ctx)
			duration := time.Since(start)

			Expect(err).ToNot(HaveOccurred())
			Expect(batch).ToNot(BeNil())

			// Should get all 50 messages
			var messages []spec.Message
			for _, msg := range batch.Messages() {
				messages = append(messages, msg)
			}
			Expect(messages).To(HaveLen(50), "Should receive full batch of 50 messages")

			// Performance check - should be reasonably fast
			Expect(duration).To(BeNumerically("<", 5*time.Second), "Large batch should be read efficiently")

			// Verify first and last message content
			if len(messages) > 0 {
				first, _ := messages[0].Raw()
				Expect(string(first)).To(Equal("large batch message 0"))
				last, _ := messages[49].Raw()
				Expect(string(last)).To(Equal("large batch message 49"))
			}

			// Acknowledge the batch
			err = ackFn(ctx.Context(), nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when using TLS configuration", func() {
		It("should configure TLS when enabled", func() {
			// Note: Tried to test this using test containers. Got the server to start with a self-signed TLS cert.
			// But could not get the client in the test to also use this cert. Seems like it expects an IBM utility
			// to be present on the system.
			Skip("Cannot setup client with gskit for cert handling.")
			cfg := ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "SYSTEM.TLS.SVRCONN",
					ConnectionName:   "localhost(1414)",
					TLS: &ibm_mq.TLSConfig{
						Enabled:               true,
						CipherSpec:            "TLS_RSA_WITH_AES_128_CBC_SHA256",
						KeyRepository:         "/opt/mqm/ssl/key",
						KeyRepositoryPassword: "password123",
						CertificateLabel:      "ibmwebspheremqapp",
					},
				},
				QueueName: "DEV.QUEUE.1",
			}
			tlsInput, _ := ibm_mq.NewInput(test.TestEnvironment(), cfg)

			err := tlsInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			tlsInput.Close(ctx)
		})

		It("should work without TLS when disabled", func() {
			cfg := ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "DEV.APP.SVRCONN",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - test container default credential
					TLS: &ibm_mq.TLSConfig{
						Enabled: false,
					},
				},
				QueueName: "DEV.QUEUE.1",
			}
			nonTlsInput, _ := ibm_mq.NewInput(test.TestEnvironment(), cfg)

			err := nonTlsInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			nonTlsInput.Close(ctx)
		})

		It("should handle nil TLS config as disabled", func() {
			cfg := ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "DEV.APP.SVRCONN",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - test container default credential
					TLS:              nil,
				},
				QueueName: "DEV.QUEUE.1",
			}
			nilTlsInput, _ := ibm_mq.NewInput(test.TestEnvironment(), cfg)

			err := nilTlsInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			nilTlsInput.Close(ctx)
		})

		It("should validate certificate label when provided", func() {
			Skip("Cannot setup client with gskit for cert handling.")
			cfg := ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "SYSTEM.TLS.SVRCONN",
					ConnectionName:   "localhost(1414)",
					TLS: &ibm_mq.TLSConfig{
						Enabled:          true,
						CipherSpec:       "TLS_RSA_WITH_AES_128_CBC_SHA256",
						KeyRepository:    "/opt/mqm/ssl/key",
						CertificateLabel: "mycertlabel",
					},
				},
				QueueName: "DEV.QUEUE.1",
			}

			tlsInput, _ := ibm_mq.NewInput(test.TestEnvironment(), cfg)
			// Note: This will fail to connect without a proper TLS server,
			_ = tlsInput.Init(ctx)

			// We should verify that:
			// - cd.SSLCipherSpec was set correctly
			// - sco.CertificateLabel was set to "mycertlabel"
		})

		It("should support FIPS mode when required", func() {
			Skip("Cannot setup client with gskit for cert handling.")
			cfg := ibm_mq.InputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "SYSTEM.TLS.SVRCONN",
					ConnectionName:   "localhost(1414)",
					TLS: &ibm_mq.TLSConfig{
						Enabled:       true,
						CipherSpec:    "TLS_RSA_WITH_AES_128_CBC_SHA256",
						KeyRepository: "/opt/mqm/ssl/key",
						FipsRequired:  true,
					},
				},
				QueueName: "DEV.QUEUE.1",
			}

			fipsInput, _ := ibm_mq.NewInput(test.TestEnvironment(), cfg)

			_ = fipsInput.Init(ctx)

			// We should verify that:
			// - sco.FipsRequired was set to true
		})
	})
})
