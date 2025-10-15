//go:build mqclient

package ibm_mq

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/wombatwisdom/components/framework/spec"
)

const (
	InputComponentName = "mq"
)

// NewInput creates a new MQ input component
func NewInput(env spec.Environment, config InputConfig) *Input {
	return &Input{
		env: env,
		cfg: config,
	}
}

// Input receives messages from an IBM MQ queue.
//
// The MQ input creates one or more queue connections to read messages
// from the specified queue. It supports batching to improve throughput
// and can handle multiple parallel connections for high-volume queues.
//
// Messages are read transactionally to ensure exactly-once processing
// semantics where possible. Failed messages can be automatically retried
// or sent to a backout queue if configured on the queue manager.
type Input struct {
	env spec.Environment
	cfg InputConfig

	qmgr         ibmmq.MQQueueManager
	qObject      ibmmq.MQObject
	msgChan      chan asyncMessage
	shutdownOnce sync.Once
	shutdownChan chan struct{}
	wg           sync.WaitGroup
	mutex        sync.Mutex
}

type asyncMessage struct {
	batch spec.Batch
	ackFn spec.ProcessedCallback
}

func (i *Input) Init(ctx spec.ComponentContext) error {
	// Create connection to IBM MQ
	cno := ibmmq.NewMQCNO()
	cd := ibmmq.NewMQCD()

	// Fill in required fields in the MQCD channel definition structure
	channelName := i.cfg.ChannelName
	connectionName := i.cfg.ConnectionName

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
	cno.ApplName = "WombatWisdom MQ Input"

	// Configure authentication if provided
	if i.cfg.UserId != "" {
		csp := ibmmq.NewMQCSP()
		csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
		csp.UserId = i.cfg.UserId
		if i.cfg.Password != "" {
			csp.Password = i.cfg.Password
		}
		cno.SecurityParms = csp
	}

	// Connect to the queue manager
	qmgr, err := ibmmq.Connx(i.cfg.QueueManagerName, cno)
	if err != nil {
		return fmt.Errorf("failed to connect to queue manager %s: %w", i.cfg.QueueManagerName, err)
	}
	i.qmgr = qmgr

	// Initialize channels
	i.msgChan = make(chan asyncMessage)
	i.shutdownChan = make(chan struct{})

	// Open the queue for input
	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = i.cfg.QueueName

	openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF + ibmmq.MQOO_FAIL_IF_QUIESCING
	qObject, err := (&i.qmgr).Open(mqod, openOptions)
	if err != nil {
		return fmt.Errorf("failed to open queue %s: %w", i.cfg.QueueName, err)
	}
	i.qObject = qObject

	// Start a goroutine to read messages
	i.wg.Add(1)
	go i.processMessages(ctx)

	return nil
}

func (i *Input) Close(ctx spec.ComponentContext) error {
	i.shutdownOnce.Do(func() {
		close(i.shutdownChan)
	})

	// Wait for all goroutines to finish
	i.wg.Wait()

	// Close the queue
	if err := i.qObject.Close(0); err != nil {
		// Log error but continue cleanup
		i.env.Errorf("Failed to close queue: %v", err)
	}

	// Disconnect from queue manager
	if err := i.qmgr.Disc(); err != nil {
		// Log error but continue cleanup
		i.env.Errorf("Failed to disconnect from queue manager: %v", err)
	}

	return nil
}

func (i *Input) Read(ctx spec.ComponentContext) (spec.Batch, spec.ProcessedCallback, error) {
	select {
	case msg := <-i.msgChan:
		return msg.batch, msg.ackFn, nil
	case <-i.shutdownChan:
		return nil, nil, spec.ErrNoData // TODO: check if this is the correct error mapping
	case <-ctx.Context().Done():
		return nil, nil, ctx.Context().Err()
	}
}

func (i *Input) processMessages(ctx spec.ComponentContext) {
	defer i.wg.Done()

	waitTime := 5 * time.Second // default wait time
	if i.cfg.WaitTime != "" {
		if parsed, err := time.ParseDuration(i.cfg.WaitTime); err == nil {
			waitTime = parsed
		}
	}

	for {
		select {
		case <-i.shutdownChan:
			return
		default:
			if err := i.readBatch(ctx, waitTime); err != nil {
				// Handle error - in production we should log this
				// For now, just wait a bit before retrying
				select {
				case <-time.After(waitTime):
				case <-i.shutdownChan:
					return
				}
			}
		}
	}
}

func (i *Input) readBatch(ctx spec.ComponentContext, waitTime time.Duration) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	batch := ctx.NewBatch()
	hasMessages := false

	// Read up to batch_count messages
	batchCount := 1
	if i.cfg.BatchCount > 0 {
		batchCount = i.cfg.BatchCount
	}

	for j := 0; j < batchCount; j++ {
		// Create MQMD and MQGMO structures
		mqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		// Set wait time for first message, no wait for subsequent messages in batch
		if j == 0 {
			gmo.Options = ibmmq.MQGMO_WAIT + ibmmq.MQGMO_SYNCPOINT + ibmmq.MQGMO_CONVERT
			gmo.WaitInterval = int32(waitTime.Milliseconds())
		} else {
			gmo.Options = ibmmq.MQGMO_NO_WAIT + ibmmq.MQGMO_SYNCPOINT + ibmmq.MQGMO_CONVERT
		}

		// Read message
		buffer := make([]byte, 32768) // 32KB buffer
		datalen, err := i.qObject.Get(mqmd, gmo, buffer)

		if err != nil {
			var mqret *ibmmq.MQReturn
			if errors.As(err, &mqret) {
				// No message available is not an error for batching
				if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
					break
				}
			}
			// Rollback any messages read so far
			if hasMessages {
				i.qmgr.Back()
			}
			return fmt.Errorf("failed to get message from queue: %w", err)
		}

		// Convert to WombatWisdom message and add to batch
		wmsg := ctx.NewMessage()
		wmsg.SetRaw(buffer[:datalen])

		// Add MQ-specific metadata
		wmsg.SetMetadata("mq_queue", i.cfg.QueueName)
		wmsg.SetMetadata("mq_message_id", string(mqmd.MsgId))
		wmsg.SetMetadata("mq_correlation_id", string(mqmd.CorrelId))
		wmsg.SetMetadata("mq_format", mqmd.Format)
		wmsg.SetMetadata("mq_priority", fmt.Sprintf("%d", mqmd.Priority))
		wmsg.SetMetadata("mq_persistence", fmt.Sprintf("%d", mqmd.Persistence))

		batch.Append(wmsg)
		hasMessages = true
	}

	// Only send batch if we have messages
	if hasMessages {
		// Create acknowledgment function
		ackFn := func(ctx context.Context, ackErr error) error {
			i.mutex.Lock()
			defer i.mutex.Unlock()

			if ackErr != nil {
				// Rollback transaction
				return i.qmgr.Back()
			} else {
				// Commit transaction
				return i.qmgr.Cmit()
			}
		}

		// Send batch to channel
		select {
		case i.msgChan <- asyncMessage{batch: batch, ackFn: ackFn}:
		case <-i.shutdownChan:
			// Rollback if we're shutting down
			i.qmgr.Back()
			return nil
		}
	}

	return nil
}
