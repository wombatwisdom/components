package test

import (
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/onsi/gomega"
	"github.com/wombatwisdom/components/spec"
	"log/slog"
	"net"
)

type environment struct {
	spec.Logger
	spec.DynamicFieldFactory
}

func (e *environment) GetString(key string) string {
	return ""
}

func (e *environment) GetInt(key string) int {
	return 0
}

func (e *environment) GetBool(key string) bool {
	return false
}

func TestEnvironment() spec.Environment {
	env, err := cel.NewEnv(
		cel.Variable("this", cel.AnyType),
	)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return &environment{
		Logger:              &logger{slog.Default()},
		DynamicFieldFactory: &dynamicFieldFactory{env: env},
	}
}

// RandomPort returns an unused random port on this machine.
func RandomPort() (int, error) {
	var err error
	var listener net.Listener

	for {
		listener, err = net.Listen("tcp", ":0")
		if err == nil {
			break
		}

		slog.Default().Warn(fmt.Sprintf("Failed to listen on random port: %v", err))
	}

	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}
