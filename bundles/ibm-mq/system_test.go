package ibm_mq_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	mq "github.com/wombatwisdom/components/bundles/ibm-mq"
	"github.com/wombatwisdom/components/framework/spec"
)

var _ = Describe("System", func() {
	When("an invalid configuration format is provided", func() {
		It("should return an error", func() {
			config := spec.NewYamlConfig(`
invalid yaml: [
queue_manager_name: QM1
channel_name: USER.CHANNEL
`)

			system, err := mq.NewSystemFromConfig(config)
			Expect(err).To(HaveOccurred())
			Expect(system).To(BeNil())
		})
	})

	When("missing required fields", func() {
		It("should return an error when queue_manager_name is missing", func() {
			config := spec.NewYamlConfig(`
channel_name: USER.CHANNEL
connection_name: localhost(1414)
`)

			system, err := mq.NewSystemFromConfig(config)
			Expect(err).To(HaveOccurred())
			Expect(system).To(BeNil())
		})
	})

	When("a valid configuration is provided", func() {
		It("should create a system successfully", func() {
			config := spec.NewYamlConfig(`
queue_manager_name: QM1
channel_name: USER.CHANNEL
connection_name: localhost(1414)
application_name: Test Application
`)

			system, err := mq.NewSystemFromConfig(config)
			if err != nil && err.Error() == "IBM MQ client libraries not available - build with -tags mqclient" {
				Skip("IBM MQ client libraries not available")
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(system).ToNot(BeNil())
		})

		It("should handle TLS configuration", func() {
			config := spec.NewYamlConfig(`
queue_manager_name: QM1
channel_name: USER.CHANNEL
connection_name: localhost(1414)
tls:
  enabled: true
  cipher_spec: ANY_TLS12_OR_HIGHER
  key_repository: /path/to/keystore
`)

			system, err := mq.NewSystemFromConfig(config)
			if err != nil && err.Error() == "IBM MQ client libraries not available - build with -tags mqclient" {
				Skip("IBM MQ client libraries not available")
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(system).ToNot(BeNil())
		})

		It("should handle authentication configuration", func() {
			config := spec.NewYamlConfig(`
queue_manager_name: QM1
channel_name: USER.CHANNEL
connection_name: localhost(1414)
user_id: testuser
password: testpass
`)

			system, err := mq.NewSystemFromConfig(config)
			if err != nil && err.Error() == "IBM MQ client libraries not available - build with -tags mqclient" {
				Skip("IBM MQ client libraries not available")
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(system).ToNot(BeNil())
		})
	})

	When("using default values", func() {
		It("should apply correct defaults", func() {
			config := spec.NewYamlConfig(`
queue_manager_name: QM1
`)

			system, err := mq.NewSystemFromConfig(config)
			if err != nil && err.Error() == "IBM MQ client libraries not available - build with -tags mqclient" {
				Skip("IBM MQ client libraries not available")
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(system).ToNot(BeNil())
		})
	})
})
