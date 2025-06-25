package spec

// Environment provides environment variable and dynamic field access.
// This interface maintains compatibility with existing components.
type Environment interface {
	Logger
	DynamicFieldFactory
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
}

// DynamicField represents a field that can be evaluated dynamically.
// This maintains compatibility with existing component patterns.
type DynamicField interface {
	String() string
	Int() int
	Bool() bool
	// Legacy methods for backward compatibility
	AsString(msg Message) (string, error)
	AsBool(msg Message) (bool, error)
}

// DynamicFieldFactory creates dynamic fields from expressions.
type DynamicFieldFactory interface {
	NewDynamicField(expr string) DynamicField
}

// Collector collects messages for batch processing.
// This maintains compatibility with existing input patterns.
type Collector interface {
	Collect(msg Message) error
	Flush() (Batch, error)
	// Legacy methods for backward compatibility
	Write(msg Message) error
	Disconnect() error
}

// NewSimpleEnvironment creates a basic environment implementation.
func NewSimpleEnvironment() Environment {
	return &simpleEnvironment{
		values: make(map[string]string),
	}
}

type simpleEnvironment struct {
	values map[string]string
}

func (e *simpleEnvironment) GetString(key string) string {
	return e.values[key]
}

func (e *simpleEnvironment) GetInt(key string) int {
	// Simple implementation - would need proper parsing in real use
	return 0
}

func (e *simpleEnvironment) GetBool(key string) bool {
	return e.values[key] == "true"
}

// Logger methods
func (e *simpleEnvironment) Debugf(format string, args ...interface{}) {
	// Simple no-op implementation
}

func (e *simpleEnvironment) Infof(format string, args ...interface{}) {
	// Simple no-op implementation
}

func (e *simpleEnvironment) Warnf(format string, args ...interface{}) {
	// Simple no-op implementation
}

func (e *simpleEnvironment) Errorf(format string, args ...interface{}) {
	// Simple no-op implementation
}

func (e *simpleEnvironment) NewDynamicField(expr string) DynamicField {
	return &simpleDynamicField{expr: expr}
}

type simpleDynamicField struct {
	expr string
}

func (f *simpleDynamicField) String() string {
	return f.expr
}

func (f *simpleDynamicField) Int() int {
	return 0
}

func (f *simpleDynamicField) Bool() bool {
	return false
}

func (f *simpleDynamicField) AsString(msg Message) (string, error) {
	return f.expr, nil
}

func (f *simpleDynamicField) AsBool(msg Message) (bool, error) {
	return false, nil
}

// NewSimpleCollector creates a basic collector implementation.
func NewSimpleCollector() Collector {
	return &simpleCollector{
		messages: make([]Message, 0),
	}
}

type simpleCollector struct {
	messages []Message
}

func (c *simpleCollector) Collect(msg Message) error {
	c.messages = append(c.messages, msg)
	return nil
}

func (c *simpleCollector) Flush() (Batch, error) {
	// This would need a proper batch implementation
	// For now, return nil to avoid compile errors
	return nil, nil
}

func (c *simpleCollector) Write(msg Message) error {
	return c.Collect(msg)
}

func (c *simpleCollector) Disconnect() error {
	// Clear collected messages
	c.messages = c.messages[:0]
	return nil
}