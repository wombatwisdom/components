package core_test

import (
    "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
    "github.com/wombatwisdom/components/nats/test"
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var acc test.Acc
var srv *server.Server
var nc *nats.Conn

func TestCore(t *testing.T) {
    RegisterFailHandler(Fail)

    BeforeSuite(func() {
        acc = test.Account("TEST_ACCOUNT")
        srv = test.NewDecentralizedServer().WithAccount(acc).Run()
        nc = acc.Connect(srv)
    })

    AfterSuite(func() {
        nc.Close()
        srv.Shutdown()
    })

    RunSpecs(t, "Core Suite")
}
