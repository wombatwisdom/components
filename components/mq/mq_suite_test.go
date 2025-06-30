package mq_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMQ(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MQ Suite")
}
