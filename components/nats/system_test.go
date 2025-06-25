package nats_test

import (
	"context"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	natscomp "github.com/wombatwisdom/components/components/nats"
	"github.com/wombatwisdom/components/framework/spec"
)

var _ = Describe("System", func() {
	When("an invalid configuration format is provided", func() {
		It("should return an error", func() {
			config := spec.NewYamlConfig(`
invalid yaml: [
auth:
  jwt: ##jwt##
  seed: ##seed##
`)

			system, err := natscomp.NewSystemFromConfig(config)
			Expect(err).To(HaveOccurred())
			Expect(system).To(BeNil())
		})
	})

	When("invalid credentials are provided", func() {
		It("should return an error when connecting", func() {
			config := spec.NewYamlConfig(`
url: ##url##
auth:
  jwt: ##jwt##
  seed: ##seed##
`,
				"##url##", srv.ClientURL(),
				"##jwt##", "invalid_seed",
				"##seed##", "invalid_jwt")

			system, err := natscomp.NewSystemFromConfig(config)
			Expect(err).ToNot(HaveOccurred())
			Expect(system).ToNot(BeNil())

			err = system.Connect(context.Background())
			Expect(err).To(HaveOccurred())
		})
	})

	When("a valid configuration is provided", func() {
		It("should return a system which can be connected to", func() {
			jwt, seed := acc.Creds()
			config := spec.NewYamlConfig(`
url: ##url##
auth:
  jwt: ##jwt##
  seed: ##seed##
`, "##url##", srv.ClientURL(), "##jwt##", jwt, "##seed##", string(seed))

			system, err := natscomp.NewSystemFromConfig(config)
			Expect(err).ToNot(HaveOccurred())
			Expect(system).ToNot(BeNil())

			err = system.Connect(context.Background())
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = system.Close(context.Background())
				Expect(err).ToNot(HaveOccurred())
			}()

			nc, ok := system.Client().(*nats.Conn)
			Expect(ok).To(BeTrue())
			Expect(nc).ToNot(BeNil())
			defer nc.Close()
		})
	})
})
