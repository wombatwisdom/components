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

var _ = Describe("Input", func() {
	var input *ibm_mq.Input
	var ctx spec.ComponentContext

	BeforeEach(func() {
		ctx = test.NewMockComponentContext()

		input = ibm_mq.NewInput(test.TestEnvironment(), ibm_mq.InputConfig{
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
})
