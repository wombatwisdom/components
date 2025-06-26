package eventbridge

import "errors"

var (
	ErrMissingEventBusName = errors.New("event_bus_name is required")
	ErrMissingRuleName     = errors.New("rule_name is required") 
	ErrMissingEventSource  = errors.New("event_source is required")
	ErrInvalidMaxBatchSize = errors.New("max_batch_size must be greater than 0")
	ErrEventBridgeClient   = errors.New("failed to create EventBridge client")
	ErrRuleNotFound        = errors.New("EventBridge rule not found")
	ErrInvalidEventFormat  = errors.New("invalid event format received")
)