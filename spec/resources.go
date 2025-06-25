package spec

import (
	"context"
	"fmt"
)

// ResourceManager provides access to shared resources and services.
// This is equivalent to benthos's service.Resources but designed for the System-first architecture.
type ResourceManager interface {
	Logger() Logger
	
	// System returns a shared system instance by name
	System(name string) (System, error)
	
	// RegisterSystem registers a system instance for sharing
	RegisterSystem(name string, sys System) error
	
	// Context returns the base context for operations
	Context() context.Context
	
	// Metrics returns a metrics interface (placeholder for future implementation)
	Metrics() Metrics
}

// Metrics provides telemetry collection interface.
// This is a placeholder for future metrics implementation.
type Metrics interface {
	Counter(name string) Counter
	Gauge(name string) Gauge
	Timer(name string) Timer
}

// Counter represents a monotonically increasing counter metric.
type Counter interface {
	Inc(delta int64)
}

// Gauge represents a gauge metric that can go up and down.
type Gauge interface {
	Set(value float64)
}

// Timer represents a timer metric for measuring durations.
type Timer interface {
	Record(duration float64)
}

// NewResourceManager creates a new resource manager instance.
func NewResourceManager(ctx context.Context, logger Logger) ResourceManager {
	return &resourceManager{
		ctx:     ctx,
		logger:  logger,
		systems: make(map[string]System),
		metrics: &noopMetrics{},
	}
}

type resourceManager struct {
	ctx     context.Context
	logger  Logger
	systems map[string]System
	metrics Metrics
}

func (r *resourceManager) Logger() Logger {
	return r.logger
}

func (r *resourceManager) System(name string) (System, error) {
	sys, exists := r.systems[name]
	if !exists {
		return nil, fmt.Errorf("system %q not found", name)
	}
	return sys, nil
}

func (r *resourceManager) RegisterSystem(name string, sys System) error {
	if _, exists := r.systems[name]; exists {
		return fmt.Errorf("system %q already registered", name)
	}
	r.systems[name] = sys
	return nil
}

func (r *resourceManager) Context() context.Context {
	return r.ctx
}

func (r *resourceManager) Metrics() Metrics {
	return r.metrics
}

// Noop implementations for metrics
type noopMetrics struct{}

func (n *noopMetrics) Counter(name string) Counter { return &noopCounter{} }
func (n *noopMetrics) Gauge(name string) Gauge     { return &noopGauge{} }
func (n *noopMetrics) Timer(name string) Timer     { return &noopTimer{} }

type noopCounter struct{}
func (n *noopCounter) Inc(delta int64) {}

type noopGauge struct{}
func (n *noopGauge) Set(value float64) {}

type noopTimer struct{}
func (n *noopTimer) Record(duration float64) {}