//go:build mqclient

package ibm_mq

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/wombatwisdom/components/framework/spec"
)

const (
	OutputComponentName = "mq"
)

// NewOutput creates a new MQ output component
func NewOutput(env spec.Environment, cfg OutputConfig) *Output {
	return &Output{
		env: env,
		cfg: cfg,
	}
}

//// NewOutputFromConfig creates an output from a spec.Config interface
//func NewOutputFromConfig(sys spec.System, config spec.Config) (*Output, error) {
//	var cfg OutputConfig
//	if err := config.Decode(&cfg); err != nil {
//		return nil, err
//	}
//	return NewOutput(sys, cfg), nil
//}

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
	env spec.Environment
	cfg OutputConfig

	metadataFilter spec.MetadataFilter

	qmgr        ibmmq.MQQueueManager
	queues      map[string]ibmmq.MQObject
	queuesMutex sync.RWMutex
}

func (o *Output) Init(ctx spec.ComponentContext) error {
	// Create connection to IBM MQ
	cno := ibmmq.NewMQCNO()
	cd := ibmmq.NewMQCD()

	// Fill in required fields in the MQCD channel definition structure
	channelName := o.cfg.ChannelName
	connectionName := o.cfg.ConnectionName

	// If ConnectionName is empty, check for MQSERVER environment variable
	// MQSERVER format: CHANNEL/TCP/HOST(PORT)
	if connectionName == "" {
		if mqserver := os.Getenv("MQSERVER"); mqserver != "" {
			parts := strings.Split(mqserver, "/")
			if len(parts) >= 3 {
				channelName = parts[0]
				connectionName = parts[2] // HOST(PORT) part
			}
		}
	}

	cd.ChannelName = channelName
	cd.ConnectionName = connectionName

	// Reference the CD structure from the CNO and indicate client connection
	cno.ClientConn = cd
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING + ibmmq.MQCNO_RECONNECT + ibmmq.MQCNO_HANDLE_SHARE_BLOCK
	cno.ApplName = "WombatWisdom MQ Output"

	// Configure authentication if provided
	if o.cfg.UserId != "" {
		csp := ibmmq.NewMQCSP()
		csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
		csp.UserId = o.cfg.UserId
		if o.cfg.Password != "" {
			csp.Password = o.cfg.Password
		}
		cno.SecurityParms = csp
	}

	// Connect to the queue manager
	qmgr, err := ibmmq.Connx(o.cfg.QueueManagerName, cno)
	if err != nil {
		return fmt.Errorf("failed to connect to queue manager %s: %w", o.cfg.QueueManagerName, err)
	}
	o.qmgr = qmgr

	// Setup metadata filter if configured
	if o.cfg.Metadata != nil {
		if o.metadataFilter, err = ctx.BuildMetadataFilter(o.cfg.Metadata.Patterns, o.cfg.Metadata.Invert); err != nil {
			return fmt.Errorf("metadata: %w", err)
		}
	}

	// Initialize queue cache
	o.queues = make(map[string]ibmmq.MQObject)

	return nil
}

func (o *Output) getOrOpenQueue(queueName string) (ibmmq.MQObject, error) {
	// Try to get from cache with read lock
	o.queuesMutex.RLock()
	queue, exists := o.queues[queueName]
	o.queuesMutex.RUnlock()

	if exists {
		return queue, nil
	}

	// Not in cache, need to open it
	o.queuesMutex.Lock()
	defer o.queuesMutex.Unlock()

	// Double-check in case another goroutine added it
	queue, exists = o.queues[queueName]
	if exists {
		return queue, nil
	}

	// Open the queue
	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queueName

	openOptions := ibmmq.MQOO_OUTPUT + ibmmq.MQOO_FAIL_IF_QUIESCING
	queue, err := (&o.qmgr).Open(mqod, openOptions)
	if err != nil {
		return ibmmq.MQObject{}, fmt.Errorf("failed to open queue %s: %w", queueName, err)
	}

	// Cache it
	o.queues[queueName] = queue
	return queue, nil
}

func (o *Output) Close(ctx spec.ComponentContext) error {
	// Close all cached queues
	for queueName, queue := range o.queues {
		if err := queue.Close(0); err != nil {
			// Log error but continue cleanup
			o.env.Errorf("Failed to close queue %s: %v", queueName, err)
		}
	}

	// Disconnect from queue manager
	if err := o.qmgr.Disc(); err != nil {
		// Log error but continue cleanup
		o.env.Errorf("Failed to disconnect from queue manager: %v", err)
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
	// Determine queue name - use expression if set, otherwise static name
	queueName := o.cfg.QueueName
	if o.cfg.QueueExpr != nil {
		exprCtx := spec.MessageExpressionContext(message)
		var err error
		queueName, err = o.cfg.QueueExpr.Eval(exprCtx)
		if err != nil {
			return fmt.Errorf("queue_expr: %w", err)
		}
	}

	// Get or open the queue
	queue, err := o.getOrOpenQueue(queueName)
	if err != nil {
		return err
	}

	// Get message data
	data, err := message.Raw()
	if err != nil {
		return fmt.Errorf("failed to get message data: %w", err)
	}

	mqmd, hasCorrelId := o.createMQMD(message)
	pmo := ibmmq.NewMQPMO()

	pmoOptions := ibmmq.MQPMO_NO_SYNCPOINT + ibmmq.MQPMO_NEW_MSG_ID
	if !hasCorrelId {
		pmoOptions += ibmmq.MQPMO_NEW_CORREL_ID
	}
	pmo.Options = pmoOptions

	err = queue.Put(mqmd, pmo, data)
	if err != nil {
		return fmt.Errorf("failed to put message to queue %s: %w", queueName, err)
	}

	return nil
}

func (o *Output) createMQMD(message spec.Message) (*ibmmq.MQMD, bool) {
	mqmd := ibmmq.NewMQMD()
	hasCorrelId := false

	// Set format
	if o.cfg.Format != "" {
		mqmd.Format = o.cfg.Format
	} else {
		mqmd.Format = "MQSTR"
	}

	// Set CCSID
	if o.cfg.Ccsid != "" {
		if ccsidInt, err := strconv.Atoi(o.cfg.Ccsid); err == nil {
			mqmd.CodedCharSetId = int32(ccsidInt)
		} else {
			// If parsing fails, use UTF-8 default
			o.env.Warnf("Failed to parse CCSID '%s', using default 1208 (UTF-8): %v", o.cfg.Ccsid, err)
			mqmd.CodedCharSetId = 1208
		}
	} else {
		mqmd.CodedCharSetId = 1208 // UTF-8 default
	}

	// Set encoding
	if o.cfg.Encoding != "" {
		if encodingInt, err := strconv.Atoi(o.cfg.Encoding); err == nil {
			mqmd.Encoding = int32(encodingInt)
		} else {
			// If parsing fails, use little-endian default
			o.env.Warnf("Failed to parse encoding '%s', using default 546 (little-endian): %v", o.cfg.Encoding, err)
			mqmd.Encoding = 546
		}
	} else {
		mqmd.Encoding = 546 // default encoding (little-endian)
	}

	// Apply metadata to MQMD if it passes the filter
	metadataMap := make(map[string]any)
	for key, value := range message.Metadata() {
		metadataMap[key] = value
	}

	if o.shouldIncludeMetadata("mq_priority") {
		if priority, exists := metadataMap["mq_priority"]; exists {
			if priorityInt, err := strconv.Atoi(fmt.Sprintf("%v", priority)); err == nil {
				mqmd.Priority = int32(priorityInt)
			}
		}
	}

	if o.shouldIncludeMetadata("mq_persistence") {
		if persistence, exists := metadataMap["mq_persistence"]; exists {
			if persistenceInt, err := strconv.Atoi(fmt.Sprintf("%v", persistence)); err == nil {
				mqmd.Persistence = int32(persistenceInt)
			}
		}
	}

	if o.shouldIncludeMetadata("mq_correlation_id") {
		if correlId, exists := metadataMap["mq_correlation_id"]; exists {
			if correlIdStr := fmt.Sprintf("%v", correlId); len(correlIdStr) <= 24 {
				copy(mqmd.CorrelId[:], []byte(correlIdStr))
				hasCorrelId = true
			}
		}
	}

	return mqmd, hasCorrelId
}

// shouldIncludeMetadata checks if a metadata key should be included based on the filter
func (o *Output) shouldIncludeMetadata(key string) bool {
	// If no filter is configured, include all metadata
	if o.metadataFilter == nil {
		return true
	}

	// Use the metadata filter to check if the key should be included
	return o.metadataFilter.Include(key)
}
