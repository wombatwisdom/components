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
				Password:         "passw0rd", // testcontainer default
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
			csp.Password = "passw0rd" // testcontainer default
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
	})
})
