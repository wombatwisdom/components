//go:build !mqclient

package mq

import (
	"context"
	"fmt"

	"github.com/wombatwisdom/components/framework/spec"
)

// Stub implementations when IBM MQ client libraries are not available
// This allows the component to compile for development and testing

// System stub implementation
type System struct {
	cfg SystemConfig
}

func NewSystem(rawConfig string) (*System, error) {
	return nil, fmt.Errorf("IBM MQ client libraries not available - build with -tags mqclient")
}

func NewSystemFromConfig(config spec.Config) (*System, error) {
	return nil, fmt.Errorf("IBM MQ client libraries not available - build with -tags mqclient")
}

func (s *System) Connect(ctx context.Context) error {
	return fmt.Errorf("IBM MQ client libraries not available")
}

func (s *System) Client() any {
	return nil
}

func (s *System) Close(ctx context.Context) error {
	return nil
}

// Input stub implementation
type Input struct {
	sys spec.System
	cfg InputConfig
}

func NewInput(sys spec.System, rawConfig spec.Config) (*Input, error) {
	return nil, fmt.Errorf("IBM MQ client libraries not available - build with -tags mqclient")
}

func NewInputFromConfig(sys spec.System, config spec.Config) (*Input, error) {
	return nil, fmt.Errorf("IBM MQ client libraries not available - build with -tags mqclient")
}

func (i *Input) Init(ctx spec.ComponentContext) error {
	return fmt.Errorf("IBM MQ client libraries not available")
}

func (i *Input) Close(ctx spec.ComponentContext) error {
	return nil
}

func (i *Input) Read(ctx spec.ComponentContext) (spec.Batch, spec.ProcessedCallback, error) {
	return nil, nil, fmt.Errorf("IBM MQ client libraries not available")
}

// Output stub implementation
type Output struct {
	sys spec.System
	cfg OutputConfig
}

func NewOutput(sys spec.System, cfg OutputConfig) *Output {
	return &Output{sys: sys, cfg: cfg}
}

func NewOutputFromConfig(sys spec.System, config spec.Config) (*Output, error) {
	return nil, fmt.Errorf("IBM MQ client libraries not available - build with -tags mqclient")
}

func (o *Output) Init(ctx spec.ComponentContext) error {
	return fmt.Errorf("IBM MQ client libraries not available")
}

func (o *Output) Close(ctx spec.ComponentContext) error {
	return nil
}

func (o *Output) Write(ctx spec.ComponentContext, batch spec.Batch) error {
	return fmt.Errorf("IBM MQ client libraries not available")
}

func (o *Output) WriteMessage(ctx spec.ComponentContext, message spec.Message) error {
	return fmt.Errorf("IBM MQ client libraries not available")
}
