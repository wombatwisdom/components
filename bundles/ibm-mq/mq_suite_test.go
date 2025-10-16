package ibm_mq_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	ibmMQContainer testcontainers.Container
	mqHost         string
	mqPort         string
)

func TestMQ(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		ctx := context.Background()

		// Start IBM MQ container
		req := testcontainers.ContainerRequest{
			Image:        "icr.io/ibm-messaging/mq:latest",
			ExposedPorts: []string{"1414/tcp", "9443/tcp"},
			Env: map[string]string{
				"LICENSE":         "accept",
				"MQ_QMGR_NAME":    "QM1",
				"MQ_DEV":          "true",     // Enable developer defaults
				"MQ_APP_PASSWORD": "passw0rd", // Set a password for the app user
			},
			WaitingFor: wait.ForLog("Started web server").WithStartupTimeout(2 * time.Minute),
		}

		var err error
		ibmMQContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		Expect(err).ToNot(HaveOccurred())

		// Get the host and port for connecting
		mqHost, err = ibmMQContainer.Host(ctx)
		Expect(err).ToNot(HaveOccurred())

		mappedPort, err := ibmMQContainer.MappedPort(ctx, "1414")
		Expect(err).ToNot(HaveOccurred())
		mqPort = mappedPort.Port()

		// Set MQSERVER for connection
		// DEV.APP.SVRCONN is the default channel in developer mode
		mqserver := fmt.Sprintf("DEV.APP.SVRCONN/TCP/%s(%s)", mqHost, mqPort)
		err = os.Setenv("MQSERVER", mqserver)
		Expect(err).ToNot(HaveOccurred())

		// Set authentication for the app user
		err = os.Setenv("MQSAMP_USER_ID", "app")
		Expect(err).ToNot(HaveOccurred())

		GinkgoLogr.Info("IBM MQ container started", "host", mqHost, "port", mqPort, "MQSERVER", mqserver)
	})

	AfterSuite(func() {
		if ibmMQContainer != nil {
			ctx := context.Background()
			_ = ibmMQContainer.Terminate(ctx)
		}
	})

	RunSpecs(t, "MQ Suite")
}
