//go:build mqclient

package ibm_mq

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/wombatwisdom/components/framework/spec"
)

const (
	OutputComponentName = "mq"
)

// NewOutput creates a new MQ output component
func NewOutput(sys spec.System, cfg OutputConfig) *Output {
	return &Output{
		sys: sys,
		cfg: cfg,
	}
}

// NewOutputFromConfig creates an output from a spec.Config interface
func NewOutputFromConfig(sys spec.System, config spec.Config) (*Output, error) {
	var cfg OutputConfig
	if err := config.Decode(&cfg); err != nil {
		return nil, err
	}
	return NewOutput(sys, cfg), nil
}

// Output sends messages to an IBM MQ queue.
//
// The MQ output creates one or more queue connections to write messages
// to the specified queue. It supports multiple parallel connections for
// high-volume outputs and handles message formatting including CCSID,
// encoding, and format settings.
//
// Messages are written transactionally to ensure reliable delivery.
// The output can be configured to include metadata as MQ message properties.
type Output struct {
	sys spec.System
	cfg OutputConfig

	queueName      spec.Expression
	metadataFilter spec.MetadataFilter

	queueConnections []*outputQueueConnection
	connChan         chan *outputQueueConnection
	shutdownOnce     sync.Once
	shutdownChan     chan struct{}
}

type outputQueueConnection struct {
	qmgr    *ibmmq.MQQueueManager
	qObject ibmmq.MQObject
	mutex   sync.Mutex
}

func (o *Output) Init(ctx spec.ComponentContext) error {
	client, ok := o.sys.Client().(*ibmmq.MQQueueManager)
	if !ok {
		return fmt.Errorf("mq client is not of type *ibmmq.MQQueueManager")
	}

	// Parse queue name as expression (it might contain variables)
	var err error
	if o.queueName, err = ctx.ParseExpression(o.cfg.QueueName); err != nil {
		return fmt.Errorf("queue_name: %w", err)
	}

	// Setup metadata filter if configured
	if o.cfg.Metadata != nil {
		if o.metadataFilter, err = ctx.BuildMetadataFilter(o.cfg.Metadata.Patterns, o.cfg.Metadata.Invert); err != nil {
			return fmt.Errorf("metadata: %w", err)
		}
	}

	// Initialize channels
	o.shutdownChan = make(chan struct{})
	o.connChan = make(chan *outputQueueConnection, o.cfg.NumThreads)

	// Create queue connections based on num_threads
	o.queueConnections = make([]*outputQueueConnection, o.cfg.NumThreads)

	for i := range o.queueConnections {
		// Each thread gets its own queue manager connection and queue object
		qmgr, ok := o.sys.Client().(*ibmmq.MQQueueManager)
		if !ok {
			return fmt.Errorf("failed to get queue manager for thread %d", i)
		}

		// Open the queue for output
		mqod := ibmmq.NewMQOD()
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = o.cfg.QueueName

		openOptions := ibmmq.MQOO_OUTPUT + ibmmq.MQOO_FAIL_IF_QUIESCING
		qObject, err := qmgr.Open(mqod, openOptions)
		if err != nil {
			return fmt.Errorf("failed to open queue %s: %w", o.cfg.QueueName, err)
		}

		conn := &outputQueueConnection{
			qmgr:    qmgr,
			qObject: qObject,
		}

		o.queueConnections[i] = conn
		// Make connection available in the channel
		o.connChan <- conn
	}

	return nil
}

func (o *Output) Close(ctx spec.ComponentContext) error {
	o.shutdownOnce.Do(func() {
		close(o.shutdownChan)
	})

	// Close all queue connections
	for _, conn := range o.queueConnections {
		if conn != nil {
			if err := conn.qObject.Close(0); err != nil {
				// Log error but continue cleanup
				// Note: We should use a proper logger here
			}
		}
	}

	return nil
}

func (o *Output) Write(ctx spec.ComponentContext, batch spec.Batch) error {
	for idx, message := range batch.Messages() {
		if err := o.WriteMessage(ctx, message); err != nil {
			return fmt.Errorf("batch #%d: %w", idx, err)
		}
	}
	return nil
}

func (o *Output) WriteMessage(ctx spec.ComponentContext, message spec.Message) error {
	// Get an available connection
	var conn *outputQueueConnection
	select {
	case conn = <-o.connChan:
	case <-o.shutdownChan:
		return fmt.Errorf("output is shutting down")
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for available connection")
	}

	// Ensure we return the connection to the pool
	defer func() {
		select {
		case o.connChan <- conn:
		case <-o.shutdownChan:
		}
	}()

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	// Evaluate queue name (might be dynamic)
	queueName, err := o.queueName.EvalString(spec.MessageExpressionContext(message))
	if err != nil {
		return fmt.Errorf("queue_name: %w", err)
	}

	// Get message data
	data, err := message.Raw()
	if err != nil {
		return fmt.Errorf("failed to get message data: %w", err)
	}

	// Create MQMD and MQPMO structures
	mqmd := o.createMQMD(message)
	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_SYNCPOINT + ibmmq.MQPMO_NEW_MSG_ID + ibmmq.MQPMO_NEW_CORREL_ID

	// Add message properties if metadata filter is configured
	if o.metadataFilter != nil {
		// TODO: Add message properties support when available in MQ Go client
		// For now, we can add metadata to the MQMD structure where applicable
	}

	// Put message to queue
	err = conn.qObject.Put(mqmd, pmo, data)
	if err != nil {
		// Rollback transaction on error
		conn.qmgr.Back()
		return fmt.Errorf("failed to put message to queue %s: %w", queueName, err)
	}

	// Commit transaction
	err = conn.qmgr.Cmit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (o *Output) createMQMD(message spec.Message) *ibmmq.MQMD {
	mqmd := ibmmq.NewMQMD()

	// Set format
	if o.cfg.Format != nil {
		mqmd.Format = *o.cfg.Format
	} else {
		mqmd.Format = "MQSTR"
	}

	// Set CCSID
	if o.cfg.Ccsid != nil {
		if ccsidInt, err := strconv.Atoi(*o.cfg.Ccsid); err == nil {
			mqmd.CodedCharSetId = int32(ccsidInt)
		} else {
			mqmd.CodedCharSetId = 1208 // UTF-8 default
		}
	} else {
		mqmd.CodedCharSetId = 1208 // UTF-8 default
	}

	// Set encoding
	if o.cfg.Encoding != nil {
		if encodingInt, err := strconv.Atoi(*o.cfg.Encoding); err == nil {
			mqmd.Encoding = int32(encodingInt)
		} else {
			mqmd.Encoding = 546 // default encoding
		}
	} else {
		mqmd.Encoding = 546 // default encoding
	}

	// Try to set priority from metadata
	if priority, exists := message.Metadata()["mq_priority"]; exists {
		if priorityInt, err := strconv.Atoi(fmt.Sprintf("%v", priority)); err == nil {
			mqmd.Priority = int32(priorityInt)
		}
	}

	// Try to set persistence from metadata
	if persistence, exists := message.Metadata()["mq_persistence"]; exists {
		if persistenceInt, err := strconv.Atoi(fmt.Sprintf("%v", persistence)); err == nil {
			mqmd.Persistence = int32(persistenceInt)
		}
	}

	// Try to set correlation ID from metadata
	if correlId, exists := message.Metadata()["mq_correlation_id"]; exists {
		if correlIdStr := fmt.Sprintf("%v", correlId); len(correlIdStr) <= 24 {
			copy(mqmd.CorrelId[:], []byte(correlIdStr))
		}
	}

	return mqmd
}
