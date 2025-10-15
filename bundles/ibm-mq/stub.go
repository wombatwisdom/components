//go:build !mqclient

package ibm_mq

import (
	"context"
	"fmt"

	"github.com/wombatwisdom/components/framework/spec"
)

// CommonMQConfig stub for non-mqclient builds
type CommonMQConfig struct {
	QueueManagerName string
	ChannelName      string
	ConnectionName   string
	UserId           string
	Password         string
	ApplicationName  string
}

// OutputConfig stub for non-mqclient builds
type OutputConfig struct {
	CommonMQConfig
	QueueName  string
	QueueExpr  spec.Expression
	NumThreads int
	Metadata   *MetadataConfig
}

// MetadataConfig stub for non-mqclient builds
type MetadataConfig struct {
	Patterns []string
	Invert   bool
}

// InputConfig stub for non-mqclient builds
type InputConfig struct {
	CommonMQConfig
	QueueName    string
	NumWorkers   int
	BatchSize    int
	PollInterval string
	NumThreads   int
	WaitTime     string
	BatchCount   int
}

// SystemConfig stub for non-mqclient builds
type SystemConfig struct {
	CommonMQConfig
	TLS *TLSConfig
}

// TLSConfig stub for non-mqclient builds
type TLSConfig struct {
	Enabled               bool
	CipherSpec            string
	KeyRepository         *string
	KeyRepositoryPassword *string
	CertificateLabel      *string
}

// Stub implementations when IBM MQ client libraries are not available
// This allows the component to compile for development and testing

// System stub implementation
type System struct {
	_ SystemConfig
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
	env spec.Environment
	cfg InputConfig
}

func NewInput(env spec.Environment, config InputConfig) *Input {
	return &Input{env: env, cfg: config}
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
	env spec.Environment
	cfg OutputConfig
}

func NewOutput(env spec.Environment, cfg OutputConfig) *Output {
	return &Output{env: env, cfg: cfg}
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
