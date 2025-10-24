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

var env = test.TestEnvironment()

var _ = Describe("Output", func() {
	var output *ibm_mq.Output
	var ctx spec.ComponentContext

	BeforeEach(func() {
		var err error
		ctx = test.NewMockComponentContext()

		queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
		Expect(err).ToNot(HaveOccurred())

		cfg := ibm_mq.OutputConfig{
			CommonMQConfig: ibm_mq.CommonMQConfig{
				QueueManagerName: "QM1",
				ConnectionName:   "",         // fallback to MQSERVER env var
				UserId:           "app",      // testcontainer default
				Password:         "passw0rd", // #nosec G101 - testcontainer default credential
			},
			QueueExpr: queueName,
		}

		output, _ = ibm_mq.NewOutput(env, cfg)

		err = output.Init(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = output.Close(ctx)
	})

	When("sending a message using the output", func() {
		It("should put the message on the queue", func() {
			// Clear queue before test
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
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			// Configure filter to only include mq_ prefixed metadata
			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
				Metadata: &ibm_mq.MetadataConfig{
					Patterns: []string{"^mq_.*"}, // Only include mq_ prefixed metadata
					Invert:   false,
				},
			}

			filteredOutput, _ := ibm_mq.NewOutput(env, cfg)
			err = filteredOutput.Init(ctx)
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
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			// Configure filter to only include custom_ prefixed metadata
			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
				Metadata: &ibm_mq.MetadataConfig{
					Patterns: []string{"^custom_.*"}, // Only include custom_ prefixed metadata
					Invert:   false,
				},
			}

			filteredOutput, _ := ibm_mq.NewOutput(env, cfg)
			err = filteredOutput.Init(ctx)
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

		It("should apply message format configuration", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			// Test with MQSTR format
			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
				Format:    "MQSTR",
			}

			formattedOutput, _ := ibm_mq.NewOutput(env, cfg)
			err = formattedOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer formattedOutput.Close(ctx)

			msg := spec.NewBytesMessage([]byte("format test message"))

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
			Expect(formattedOutput.Write(ctx, ctx.NewBatch(msg))).To(Succeed())

			var receivedMQMD *ibmmq.MQMD
			Eventually(recv).Should(Receive(&receivedMQMD))

			// Verify format is set correctly
			// IBM MQ library trims spaces when retrieving the format
			Expect(receivedMQMD.Format).To(Equal("MQSTR"))
		})

		It("should apply CCSID configuration", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())
			// Test with ISO-8859-1 CCSID (non-default value)
			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
				Ccsid:     "819", // ISO-8859-1 (non-default to ensure we're actually setting it)
			}

			ccsidOutput, _ := ibm_mq.NewOutput(env, cfg)
			err = ccsidOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer ccsidOutput.Close(ctx)

			msg := spec.NewBytesMessage([]byte("ccsid test message"))

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
			Expect(ccsidOutput.Write(ctx, ctx.NewBatch(msg))).To(Succeed())

			var receivedMQMD *ibmmq.MQMD
			Eventually(recv).Should(Receive(&receivedMQMD))

			// Verify CCSID is set to ISO-8859-1 (819), not the default
			Expect(receivedMQMD.CodedCharSetId).To(Equal(int32(819)))
		})

		It("should apply encoding configuration", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			// Test with big-endian encoding (non-default value)
			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
				Encoding:  "273", // Big-endian (non-default to ensure we're actually setting it)
			}

			encodingOutput, _ := ibm_mq.NewOutput(env, cfg)
			err = encodingOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer encodingOutput.Close(ctx)

			msg := spec.NewBytesMessage([]byte("encoding test message"))

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
			Expect(encodingOutput.Write(ctx, ctx.NewBatch(msg))).To(Succeed())

			var receivedMQMD *ibmmq.MQMD
			Eventually(recv).Should(Receive(&receivedMQMD))

			// Verify encoding is set to big-endian (273), not the default
			Expect(receivedMQMD.Encoding).To(Equal(int32(273)))
		})

		It("should use defaults when format/encoding not specified", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			// Create output without format/encoding configuration
			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}

			defaultOutput, _ := ibm_mq.NewOutput(env, cfg)
			err = defaultOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer defaultOutput.Close(ctx)

			msg := spec.NewBytesMessage([]byte("default format test"))

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
			Expect(defaultOutput.Write(ctx, ctx.NewBatch(msg))).To(Succeed())

			var receivedMQMD *ibmmq.MQMD
			Eventually(recv).Should(Receive(&receivedMQMD))

			// Verify defaults are applied
			// IBM MQ library trims spaces when retrieving the format
			Expect(receivedMQMD.Format).To(Equal("MQSTR"))
			Expect(receivedMQMD.CodedCharSetId).To(Equal(int32(1208))) // Our UTF-8 default
			Expect(receivedMQMD.Encoding).To(Equal(int32(546)))        // Our little-endian default
		})
	})

	Context("when using TLS configuration", func() {
		It("should fail to connect with TLS to non-existent TLS channel", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "SYSTEM.TLS.SVRCONN", // This channel doesn't exist
					ConnectionName:   "localhost(1414)",
					TLS: &ibm_mq.TLSConfig{
						Enabled:               true,
						CipherSpec:            "TLS_RSA_WITH_AES_128_CBC_SHA256",
						KeyRepository:         "/opt/mqm/ssl/key",
						KeyRepositoryPassword: "password123",
						CertificateLabel:      "ibmwebspheremqapp",
					},
				},
				QueueExpr: queueName,
			}
			tlsOutput, _ := ibm_mq.NewOutput(env, cfg)

			// This should fail because SYSTEM.TLS.SVRCONN doesn't exist
			// and the key repository likely doesn't exist either
			err = tlsOutput.Init(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to connect"))

			tlsOutput.Close(ctx)
		})

		It("should work without TLS when disabled", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "DEV.APP.SVRCONN",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
					TLS: &ibm_mq.TLSConfig{
						Enabled: false,
					},
				},
				QueueExpr: queueName,
			}
			nonTlsOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = nonTlsOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			nonTlsOutput.Close(ctx)
		})

		It("should handle nil TLS config as disabled", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "DEV.APP.SVRCONN",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
					TLS:              nil,
				},
				QueueExpr: queueName,
			}
			nilTlsOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = nilTlsOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			nilTlsOutput.Close(ctx)
		})

		It("should apply cipher spec when TLS is enabled", func() {
			Skip("Cannot setup client with gskit for client cert handling.")
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ChannelName:      "SYSTEM.TLS.SVRCONN",
					ConnectionName:   "localhost(1414)",
					TLS: &ibm_mq.TLSConfig{
						Enabled:       true,
						CipherSpec:    "TLS_RSA_WITH_AES_256_CBC_SHA256",
						KeyRepository: "/opt/mqm/ssl/key",
					},
				},
				QueueExpr: queueName}

			tlsOutput, _ := ibm_mq.NewOutput(env, cfg)

			_ = tlsOutput.Init(ctx)

		})
	})

	Context("when using batch writes", func() {
		It("should write multiple messages in a single batch", func() {
			// Clear queue before test
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}
			batchOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = batchOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchOutput.Close(ctx)

			// Create a batch with multiple messages
			messages := []spec.Message{
				spec.NewBytesMessage([]byte("batch message 1")),
				spec.NewBytesMessage([]byte("batch message 2")),
				spec.NewBytesMessage([]byte("batch message 3")),
			}
			batch := ctx.NewBatch(messages...)

			// Write the batch
			err = batchOutput.Write(ctx, batch)
			Expect(err).ToNot(HaveOccurred())

			// Verify messages were written
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
			openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Read back messages to verify
			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
			buffer := make([]byte, 1024)

			for i := 0; i < 3; i++ {
				datalen, err := qObj.Get(ibmmq.NewMQMD(), gmo, buffer)
				Expect(err).ToNot(HaveOccurred())
				expectedMsg := fmt.Sprintf("batch message %d", i+1)
				Expect(string(buffer[:datalen])).To(Equal(expectedMsg))
			}
		})

		It("should handle batch with different message metadata", func() {
			// Clear queue before test
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}
			batchOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = batchOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchOutput.Close(ctx)

			// Create messages with different metadata
			msg1 := spec.NewBytesMessage([]byte("high priority message"))
			msg1.SetMetadata("mq_priority", "9")

			msg2 := spec.NewBytesMessage([]byte("normal priority message"))
			msg2.SetMetadata("mq_priority", "5")

			msg3 := spec.NewBytesMessage([]byte("low priority message"))
			msg3.SetMetadata("mq_priority", "1")

			batch := ctx.NewBatch(msg1, msg2, msg3)

			// Write the batch
			err = batchOutput.Write(ctx, batch)
			Expect(err).ToNot(HaveOccurred())

			// Verify messages with correct priorities
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
			openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF | ibmmq.MQOO_BROWSE

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Browse messages (don't consume) to check priorities
			expectedPriorities := []int32{9, 5, 1}
			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_BROWSE_FIRST
			buffer := make([]byte, 1024)

			for i, expectedPri := range expectedPriorities {
				mqmd := ibmmq.NewMQMD()
				_, err := qObj.Get(mqmd, gmo, buffer)
				Expect(err).ToNot(HaveOccurred())
				Expect(mqmd.Priority).To(Equal(expectedPri), fmt.Sprintf("Message %d should have priority %d", i+1, expectedPri))
				gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_BROWSE_NEXT
			}

			// Now consume the messages to clean up the queue
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
			for i := 0; i < 3; i++ {
				_, _ = qObj.Get(ibmmq.NewMQMD(), gmo, buffer)
			}
		})

		It("should write large batches efficiently", func() {

			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}
			batchOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = batchOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchOutput.Close(ctx)

			// Create a large batch
			var messages []spec.Message
			for i := 0; i < 100; i++ {
				msg := spec.NewBytesMessage([]byte(fmt.Sprintf("large batch msg %d", i)))
				messages = append(messages, msg)
			}
			batch := ctx.NewBatch(messages...)

			// Time the batch write
			start := time.Now()
			err = batchOutput.Write(ctx, batch)
			duration := time.Since(start)

			Expect(err).ToNot(HaveOccurred())
			Expect(duration).To(BeNumerically("<", 10*time.Second), "Large batch write should be efficient")

			// Verify all messages were written
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
			// Check queue depth
			openOptions := ibmmq.MQOO_INQUIRE

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			depth := getQueueDepth(qObj)
			Expect(depth).To(BeNumerically(">=", 100), "All 100 messages should be in queue")

			// Clean up: consume all 100 messages to prevent test bleed
			qObj.Close(ibmmq.MQCO_NONE)

			// Reopen for input to consume messages
			openOptions = ibmmq.MQOO_INPUT_AS_Q_DEF
			qObj, err = qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
			buffer := make([]byte, 1024)

			for i := 0; i < 100; i++ {
				_, err := qObj.Get(ibmmq.NewMQMD(), gmo, buffer)
				if err != nil {
					break // Stop if no more messages
				}
			}
		})

		It("should handle empty batches gracefully", func() {

			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}
			batchOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = batchOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchOutput.Close(ctx)

			// Create empty batch
			batch := ctx.NewBatch()

			// Write empty batch should succeed without errors
			err = batchOutput.Write(ctx, batch)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle batch with single message (default)", func() {
			// Clear queue before test
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}
			batchOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = batchOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchOutput.Close(ctx)

			// Create batch with single message
			msg := spec.NewBytesMessage([]byte("single message in batch"))
			batch := ctx.NewBatch(msg)

			// Write single message batch
			err = batchOutput.Write(ctx, batch)
			Expect(err).ToNot(HaveOccurred())

			// Verify message was written
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
			openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
			buffer := make([]byte, 1024)

			datalen, err := qObj.Get(ibmmq.NewMQMD(), gmo, buffer)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buffer[:datalen])).To(Equal("single message in batch"))
		})

		It("should maintain message order in batch", func() {
			// Clear queue before test
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}
			batchOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = batchOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer batchOutput.Close(ctx)

			// Create ordered messages
			var messages []spec.Message
			for i := 0; i < 10; i++ {
				msg := spec.NewBytesMessage([]byte(fmt.Sprintf("ordered message %02d", i)))
				messages = append(messages, msg)
			}
			batch := ctx.NewBatch(messages...)

			// Write the batch
			err = batchOutput.Write(ctx, batch)
			Expect(err).ToNot(HaveOccurred())

			// Read back and verify order
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
			openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
			buffer := make([]byte, 1024)

			for i := 0; i < 10; i++ {
				datalen, err := qObj.Get(ibmmq.NewMQMD(), gmo, buffer)
				Expect(err).ToNot(HaveOccurred())
				expectedMsg := fmt.Sprintf("ordered message %02d", i)
				Expect(string(buffer[:datalen])).To(Equal(expectedMsg), "Messages should maintain order")
			}
		})
	})

	Context("when using transactional writes", func() {
		It("should rollback all messages on batch failure", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}
			transactionalOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = transactionalOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer transactionalOutput.Close(ctx)

			// Create a batch with multiple messages
			msg1 := spec.NewBytesMessage([]byte("transaction test message 1"))
			msg2 := spec.NewBytesMessage([]byte("transaction test message 2"))

			// Create a third message with invalid priority that will cause failure
			msg3 := spec.NewBytesMessage([]byte("transaction test message 3"))
			msg3.SetMetadata("mq_priority", "999") // Invalid priority - must be 0-9

			batch := ctx.NewBatch(msg1, msg2, msg3)

			// Create a direct connection to check queue state
			cno := ibmmq.NewMQCNO()
			csp := ibmmq.NewMQCSP()
			csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
			csp.UserId = "app"
			csp.Password = "passw0rd" // #nosec G101 - testcontainer default credential
			cno.SecurityParms = csp

			qMgr, err := ibmmq.Connx("QM1", cno)
			Expect(err).ToNot(HaveOccurred())
			defer qMgr.Disc()

			// Open queue for checking
			mqod := ibmmq.NewMQOD()
			mqod.ObjectType = ibmmq.MQOT_Q
			mqod.ObjectName = "DEV.QUEUE.1"
			openOptions := ibmmq.MQOO_BROWSE + ibmmq.MQOO_FAIL_IF_QUIESCING

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Get initial queue depth
			initialDepth := getQueueDepth(qObj)

			// Write the batch - should fail on third message due to invalid priority
			err = transactionalOutput.Write(ctx, batch)
			Expect(err).To(HaveOccurred(), "Write should fail due to invalid priority")

			// Check that NO messages were committed (all rolled back)
			finalDepth := getQueueDepth(qObj)
			Expect(finalDepth).To(Equal(initialDepth), "No messages should be committed after rollback")
		})

		It("should commit messages only after successful batch write", func() {
			queueName, err := spec.NewExprLangExpression("${!\"DEV.QUEUE.1\"}")
			Expect(err).ToNot(HaveOccurred())

			cfg := ibm_mq.OutputConfig{
				CommonMQConfig: ibm_mq.CommonMQConfig{
					QueueManagerName: "QM1",
					ConnectionName:   "",
					UserId:           "app",
					Password:         "passw0rd", // #nosec G101 - testcontainer default credential
				},
				QueueExpr: queueName,
			}
			transactionalOutput, _ := ibm_mq.NewOutput(env, cfg)

			err = transactionalOutput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer transactionalOutput.Close(ctx)

			// Create a message
			msg := spec.NewBytesMessage([]byte("commit test message"))
			batch := ctx.NewBatch(msg)

			// Create a direct connection to monitor queue
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
			openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF + ibmmq.MQOO_FAIL_IF_QUIESCING

			qObj, err := qMgr.Open(mqod, openOptions)
			Expect(err).ToNot(HaveOccurred())
			defer qObj.Close(ibmmq.MQCO_NONE)

			// Write the message (with proper transaction handling, it should be committed)
			err = transactionalOutput.Write(ctx, batch)
			Expect(err).ToNot(HaveOccurred())

			// Verify message was committed by trying to read it
			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT + ibmmq.MQGMO_WAIT
			gmo.WaitInterval = 1000 // 1 second timeout

			buffer := make([]byte, 1024)
			datalen, err := qObj.Get(ibmmq.NewMQMD(), gmo, buffer)
			Expect(err).ToNot(HaveOccurred(), "Should be able to read the committed message")
			Expect(datalen).To(BeNumerically(">", 0))
			Expect(string(buffer[:datalen])).To(Equal("commit test message"))
		})
	})
})

// Helper function to get queue depth
func getQueueDepth(qObj ibmmq.MQObject) int32 {
	// Get queue attributes to check depth
	selectors := []int32{ibmmq.MQIA_CURRENT_Q_DEPTH}
	attrs, err := qObj.Inq(selectors)
	if err != nil {
		return -1
	}
	if depth, ok := attrs[ibmmq.MQIA_CURRENT_Q_DEPTH].(int32); ok {
		return depth
	}
	return -1
}
