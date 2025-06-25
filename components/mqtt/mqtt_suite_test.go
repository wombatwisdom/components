package mqtt_test

import (
	"fmt"
	"testing"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var url string
var env spec.Environment
var server *mochi.Server

func TestMqtt(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		var err error

		server = mochi.New(nil)
		// Allow all connections.
		_ = server.AddHook(new(auth.AllowHook), nil)

		// generate a random port
		port, err := test.RandomPort()
		Expect(err).ToNot(HaveOccurred())

		// Create a TCP listener on a standard port.
		tcp := listeners.NewTCP(listeners.Config{ID: "t1", Address: fmt.Sprintf(":%d", port)})
		err = server.AddListener(tcp)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			err := server.Serve()
			if err != nil {
				Expect(err).ToNot(HaveOccurred())
			}
		}()

		url = fmt.Sprintf("tcp://localhost:%d", port)
		env = test.TestEnvironment()

		GinkgoLogr.Info("MQTT URL: %s\n", url)
	})

	AfterSuite(func() {
		if server != nil {
			_ = server.Close()
		}
	})

	RunSpecs(t, "Mqtt Suite")
}
