//go:build mqclient

package ibm_mq_test

import (
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ibm_mq "github.com/wombatwisdom/components/bundles/ibm-mq"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var env = test.TestEnvironment()

var _ = Describe("Output", func() {
	var output *ibm_mq.Output
	var ctx spec.ComponentContext

	BeforeEach(func() {
		var err error
		ctx = test.NewMockComponentContext()

		queueExpr, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
		Expect(err).ToNot(HaveOccurred())

		cfg := ibm_mq.OutputConfig{
			CommonMQConfig: ibm_mq.CommonMQConfig{
				QueueManagerName: "QM1",
				ConnectionName:   "",         // fallback to MQSERVER env var
				UserId:           "app",      // testcontainer default
				Password:         "passw0rd", // #nosec G101 - testcontainer default credential
			},
			QueueName: "DEV.QUEUE.1", // testcontainer default developer queue
			QueueExpr: queueExpr,
		}

		output = ibm_mq.NewOutput(env, cfg)

		err = output.Init(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = output.Close(ctx)
	})

	When("sending a message using the output", func() {
		It("should put the message on the queue", func() {
			msg := spec.NewBytesMessage([]byte("hello, world"))
			b, err := msg.Raw()
			Expect(err).ToNot(HaveOccurred())

			recv := make(chan []byte)
			ready := make(chan struct{})

			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"        // testcontainer default
			csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())

			defer qMgr.Disc()

			// Open the queue for reading
			mqod := ibmmq.NewMQOD()
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1" // testcontainer default developer queue
			openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			close(ready)

			// poll read
			go func() {
				defer close(recv)
				getmqmd := ibmmq.NewMQMD()
				gmo := ibmmq.NewMQGMO()
				gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_WAIT
				gmo.WaitInterval = 3000

				buffer := make([]byte, 1024)
				datalen, err := qObj.Get(getmqmd, gmo, buffer)
				if err == nil {
					recv <- buffer[:datalen]
				}
			}()

			<-ready
			Expect(output.Write(ctx, ctx.NewBatch(msg))).To(Succeed())

			var received []byte
			Eventually(recv).Should(Receive(&received))
			Expect(received).To(Equal(b))
		})

		It("should apply metadata to the MQ message properties", func() {
			// Create message with metadata
			msg := spec.NewBytesMessage([]byte("metadata test"))
			msg.SetMetadata("mq_priority", "7")
			msg.SetMetadata("mq_persistence", "1") // MQPER_PERSISTENT
			msg.SetMetadata("mq_correlation_id", "CORR123")
			msg.SetMetadata("custom_header", "should_be_filtered")

			recv := make(chan *ibmmq.MQMD)
			ready := make(chan struct{})

			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"        // testcontainer default
			csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())
			defer qMgr.Disc()

			// Open the queue for reading
			mqod := ibmmq.NewMQOD()
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1"
			openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			close(ready)

			// Read message and capture MQMD
			go func() {
				defer close(recv)
				getmqmd := ibmmq.NewMQMD()
				gmo := ibmmq.NewMQGMO()
				gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_WAIT
				gmo.WaitInterval = 3000

				buffer := make([]byte, 1024)
				_, err := qObj.Get(getmqmd, gmo, buffer)
				if err == nil {
					recv <- getmqmd
				}
			}()

			<-ready
			Expect(output.Write(ctx, ctx.NewBatch(msg))).To(Succeed())

			var receivedMQMD *ibmmq.MQMD
			Eventually(recv).Should(Receive(&receivedMQMD))

			// Verify metadata was applied
			Expect(receivedMQMD.Priority).To(Equal(int32(7)))
			Expect(receivedMQMD.Persistence).To(Equal(int32(1)))
			// CorrelId is a fixed-size byte array, check the beginning matches
			correlId := string(receivedMQMD.CorrelId[:7])
			Expect(correlId).To(Equal("CORR123"))
		})

		It("should include metadata that matches filter patterns", func() {
			// Configure filter to only include mq_ prefixed metadata
			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueName: "DEV.QUEUE.1",
				Metadata: &ibm_mq.MetadataConfig{
					Patterns: []string{"^mq_.*"}, // Only include mq_ prefixed metadata
					Invert:   false,
				},
			}

			filteredOutput := ibm_mq.NewOutput(env, cfg)
			err := filteredOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer filteredOutput.Close(ctx)

			// Create message with mq_priority that should pass the filter
			msg := spec.NewBytesMessage([]byte("include filtered metadata test"))
			msg.SetMetadata("mq_priority", "5")

			recv := make(chan *ibmmq.MQMD)
			ready := make(chan struct{})

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
			openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			close(ready)

			go func() {
				defer close(recv)
				getmqmd := ibmmq.NewMQMD()
				gmo := ibmmq.NewMQGMO()
				gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_WAIT
				gmo.WaitInterval = 3000

				buffer := make([]byte, 1024)
				_, err := qObj.Get(getmqmd, gmo, buffer)
				if err == nil {
					recv <- getmqmd
				}
			}()

			<-ready
			Expect(filteredOutput.Write(ctx, ctx.NewBatch(msg))).To(Succeed())

			var receivedMQMD *ibmmq.MQMD
			Eventually(recv).Should(Receive(&receivedMQMD))

			// Verify mq_priority was applied (matches filter pattern)
			Expect(receivedMQMD.Priority).To(Equal(int32(5)))
		})

		It("should exclude metadata that doesn't match filter patterns", func() {
			// Configure filter to only include custom_ prefixed metadata
			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueName: "DEV.QUEUE.1",
				Metadata: &ibm_mq.MetadataConfig{
					Patterns: []string{"^custom_.*"}, // Only include custom_ prefixed metadata
					Invert:   false,
				},
			}

			filteredOutput := ibm_mq.NewOutput(env, cfg)
			err := filteredOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer filteredOutput.Close(ctx)

			// Create message with mq_priority that should NOT pass the filter
			msg := spec.NewBytesMessage([]byte("exclude filtered metadata test"))
			msg.SetMetadata("mq_priority", "7") // This doesn't match ^custom_.* pattern

			recv := make(chan *ibmmq.MQMD)
			ready := make(chan struct{})

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
			openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			close(ready)

			go func() {
				defer close(recv)
				getmqmd := ibmmq.NewMQMD()
				gmo := ibmmq.NewMQGMO()
				gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_WAIT
				gmo.WaitInterval = 3000

				buffer := make([]byte, 1024)
				_, err := qObj.Get(getmqmd, gmo, buffer)
				if err == nil {
					recv <- getmqmd
				}
			}()

			<-ready
			Expect(filteredOutput.Write(ctx, ctx.NewBatch(msg))).To(Succeed())

			var receivedMQMD *ibmmq.MQMD
			Eventually(recv).Should(Receive(&receivedMQMD))

			// Verify mq_priority was NOT applied (doesn't match filter pattern)
			// Default priority should be 0
			Expect(receivedMQMD.Priority).To(Equal(int32(0)))
		})
	})
})
