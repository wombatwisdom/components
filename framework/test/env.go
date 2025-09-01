package test

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/wombatwisdom/components/framework/spec"
)

type environment struct {
	spec.Logger
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
	return &environment{
		Logger: &logger{slog.Default()},
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

	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}
