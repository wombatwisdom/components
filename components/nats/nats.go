// Package nats provides NATS messaging components including core NATS functionality
// and JetStream-based components for streams, key-value store, and object store.
package nats

import (
	"github.com/wombatwisdom/components/nats/core"
)

// Core NATS component functions for backward compatibility
var (
	NewSystemFromConfig = core.NewSystemFromConfig
	NewInputFromConfig  = core.NewInputFromConfig
	NewOutputFromConfig = core.NewOutputFromConfig
)

// System interface re-export for backward compatibility
type System = core.System
type Input = core.Input
type Output = core.Output

// Config re-exports for backward compatibility
type SystemConfig = core.SystemConfig
type InputConfig = core.InputConfig
type OutputConfig = core.OutputConfig
