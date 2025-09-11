package test

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nkeys"
	"github.com/onsi/gomega"
)

type DecentralizedServer struct {
	tmpDir       string
	accountJwts  map[string]string
	operatorKp   nkeys.KeyPair
	operatorJwt  string
	sysAccountKp nkeys.KeyPair
	sysAccount   string
}

func NewDecentralizedServer() *DecentralizedServer {
	tmpDir, err := os.MkdirTemp("", "vent-test")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// Operator
	op, _ := nkeys.CreateOperator()
	opPk, _ := op.PublicKey()
	opClaim := jwt.NewOperatorClaims(opPk)
	opJwt, err := opClaim.Encode(op)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// System account
	sys, _ := nkeys.CreateAccount()
	sysPk, _ := sys.PublicKey()
	sysClaim := jwt.NewAccountClaims(sysPk)
	sysJwt, err := sysClaim.Encode(op)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return &DecentralizedServer{
		tmpDir:       tmpDir,
		operatorJwt:  opJwt,
		operatorKp:   op,
		sysAccount:   sysPk,
		sysAccountKp: sys,
		accountJwts: map[string]string{
			sysPk: sysJwt,
		},
	}
}

func (s *DecentralizedServer) WithAccount(acc Acc) *DecentralizedServer {
	accJwt, err := acc.Claims.Encode(s.operatorKp)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	s.accountJwts[acc.Id] = accJwt

	return s
}

func (s *DecentralizedServer) Run() *server.Server {
	var resolvers []string
	for acc, j := range s.accountJwts {
		resolvers = append(resolvers, fmt.Sprintf("%s : %s", acc, j))
	}

	confContent := []byte(fmt.Sprintf(`
		listen: 127.0.0.1:-1
		jetstream: {max_mem_store: 10Mb, max_file_store: 10Mb, store_dir: "%s"}
		operator: %s
		system_account: %s
		resolver = MEMORY
		resolver_preload = {
			%s
		}
    `, s.tmpDir, s.operatorJwt, s.sysAccount, strings.Join(resolvers, "\n\t")))
	conf := path.Join(s.tmpDir, "server.conf")
	err := os.WriteFile(conf, confContent, os.FileMode(0640))
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	srv, _ := test.RunServerWithConfig(conf)
	return srv
}
