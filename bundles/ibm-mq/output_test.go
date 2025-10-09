package ibm_mq_test

import (
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/bundles/ibm-mq"
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

		queue, err := spec.NewExprLangExpression("${!\"test\"}")
		Expect(err).ToNot(HaveOccurred())

		cfg := ibm_mq.OutputConfig{
			QueueName: "DEV.QUEUE.1", // Using default developer queue
			QueueExpr: queue,
		}

		// TODO: check for err
		output = ibm_mq.NewOutput(env, cfg)

		_ = output.Init(ctx)
	})

	AfterEach(func() {
		_ = output.Close(ctx)
	})

	When("sending a message using the output", func() {
		It("should put the message on the IBM MQ queue", func() {
			// TODO: some repetition here
			msg := spec.NewBytesMessage([]byte("hello, world"))
			b, err := msg.Raw()
			Expect(err).ToNot(HaveOccurred())

			recv := make(chan []byte)
			ready := make(chan struct{})

			// Connect to queue manager with authentication
			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"
			csp.Password = "passw0rd" // Matches MQ_APP_PASSWORD in container setup
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())
			// TODO: When do we disconnect?
			defer qMgr.Disc()

			// Open the queue for reading
			mqod := ibmmq.NewMQOD() // object descriptor?
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1" // Using default developer queue
			openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE) // TODO: when?

			// polling
			go func() {
				defer close(recv)
				getmqmd := ibmmq.NewMQMD() // message descriptor?
				gmo := ibmmq.NewMQGMO()
				gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_WAIT
				gmo.WaitInterval = 3000 // 3 seconds

				buffer := make([]byte, 1024)
				datalen, err := qObj.Get(getmqmd, gmo, buffer)
				if err == nil {
					recv <- buffer[:datalen]
				}
				close(ready)
			}()

			select {
			case <-ready:
				Expect(output.Write(ctx, ctx.NewBatch(msg))).To(Succeed())
				Eventually(recv).Should(Receive())
			case msg := <-recv:
				Expect(msg).To(Equal(b))
			}
		})
	})
})
