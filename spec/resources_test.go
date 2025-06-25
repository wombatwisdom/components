package spec_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/spec"
)

var _ = Describe("ResourceManager", func() {
	var (
		ctx    context.Context
		logger *mockLogger
		rm     spec.ResourceManager
		mockSys *mockSystem
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = &mockLogger{}
		rm = spec.NewResourceManager(ctx, logger)
		mockSys = &mockSystem{}
	})

	Describe("Logger", func() {
		It("should return the provided logger", func() {
			Expect(rm.Logger()).To(Equal(logger))
		})
	})

	Describe("Context", func() {
		It("should return the provided context", func() {
			Expect(rm.Context()).To(Equal(ctx))
		})
	})

	Describe("System management", func() {
		It("should register and retrieve systems", func() {
			err := rm.RegisterSystem("test-system", mockSys)
			Expect(err).NotTo(HaveOccurred())

			retrievedSystem, err := rm.System("test-system")
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedSystem).To(Equal(mockSys))
		})

		It("should return error for non-existent system", func() {
			_, err := rm.System("non-existent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`system "non-existent" not found`))
		})

		It("should prevent duplicate system registration", func() {
			err := rm.RegisterSystem("duplicate", mockSys)
			Expect(err).NotTo(HaveOccurred())

			err = rm.RegisterSystem("duplicate", mockSys)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`system "duplicate" already registered`))
		})

		It("should allow multiple different systems", func() {
			system1 := &mockSystem{}
			system2 := &mockSystem{}

			err := rm.RegisterSystem("system1", system1)
			Expect(err).NotTo(HaveOccurred())

			err = rm.RegisterSystem("system2", system2)
			Expect(err).NotTo(HaveOccurred())

			retrieved1, err := rm.System("system1")
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved1).To(Equal(system1))

			retrieved2, err := rm.System("system2")
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved2).To(Equal(system2))
		})
	})

	Describe("Metrics", func() {
		It("should return metrics interface", func() {
			metrics := rm.Metrics()
			Expect(metrics).NotTo(BeNil())
		})

		It("should provide counter interface", func() {
			metrics := rm.Metrics()
			counter := metrics.Counter("test-counter")
			Expect(counter).NotTo(BeNil())

			// Should not panic
			counter.Inc(1)
			counter.Inc(10)
		})

		It("should provide gauge interface", func() {
			metrics := rm.Metrics()
			gauge := metrics.Gauge("test-gauge")
			Expect(gauge).NotTo(BeNil())

			// Should not panic
			gauge.Set(42.5)
			gauge.Set(0.0)
		})

		It("should provide timer interface", func() {
			metrics := rm.Metrics()
			timer := metrics.Timer("test-timer")
			Expect(timer).NotTo(BeNil())

			// Should not panic
			timer.Record(123.45)
			timer.Record(0.1)
		})
	})
})

// Mock implementations for testing
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *mockLogger) Debugf(format string, args ...interface{}) {
	m.debugMessages = append(m.debugMessages, format)
}

func (m *mockLogger) Infof(format string, args ...interface{}) {
	m.infoMessages = append(m.infoMessages, format)
}

func (m *mockLogger) Warnf(format string, args ...interface{}) {
	m.warnMessages = append(m.warnMessages, format)
}

func (m *mockLogger) Errorf(format string, args ...interface{}) {
	m.errorMessages = append(m.errorMessages, format)
}

type mockSystem struct {
	connected bool
	client    any
}

func (m *mockSystem) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *mockSystem) Close(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockSystem) Client() any {
	return m.client
}