//go:build mqclient

package mq

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/wombatwisdom/components/framework/spec"
)

const (
	InputComponentName = "mq"
)

// NewInput creates a new MQ input component
func NewInput(sys spec.System, rawConfig spec.Config) (*Input, error) {
	var cfg InputConfig
	if err := rawConfig.Decode(&cfg); err != nil {
		return nil, err
	}

	return &Input{
		sys: sys,
		cfg: cfg,
	}, nil
}

// NewInputFromConfig creates an input from a spec.Config interface
func NewInputFromConfig(sys spec.System, config spec.Config) (*Input, error) {
	return NewInput(sys, config)
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
	sys spec.System
	cfg InputConfig

	queueConnections []*queueConnection
	msgChan          chan asyncMessage
	shutdownOnce     sync.Once
	shutdownChan     chan struct{}
	wg               sync.WaitGroup
}

type queueConnection struct {
	qmgr    *ibmmq.MQQueueManager
	qObject ibmmq.MQObject
	mutex   sync.Mutex
}

type asyncMessage struct {
	batch spec.Batch
	ackFn spec.ProcessedCallback
}

func (i *Input) Init(ctx spec.ComponentContext) error {
	client, ok := i.sys.Client().(*ibmmq.MQQueueManager)
	if !ok {
		return fmt.Errorf("mq client is not of type *ibmmq.MQQueueManager")
	}

	// Initialize channels
	i.msgChan = make(chan asyncMessage, i.cfg.NumThreads)
	i.shutdownChan = make(chan struct{})

	// Create queue connections based on num_threads
	i.queueConnections = make([]*queueConnection, i.cfg.NumThreads)

	for i := range i.queueConnections {
		// Create a new queue manager connection for each thread
		// This provides better parallelism and avoids contention
		qmgr, ok := i.sys.Client().(*ibmmq.MQQueueManager)
		if !ok {
			return fmt.Errorf("failed to get queue manager for thread %d", i)
		}

		// Open the queue for input
		mqod := ibmmq.NewMQOD()
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = i.cfg.QueueName

		openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF + ibmmq.MQOO_FAIL_IF_QUIESCING
		qObject, err := qmgr.Open(mqod, openOptions)
		if err != nil {
			return fmt.Errorf("failed to open queue %s: %w", i.cfg.QueueName, err)
		}

		i.queueConnections[i] = &queueConnection{
			qmgr:    qmgr,
			qObject: qObject,
		}

		// Start a goroutine for this connection to read messages
		i.wg.Add(1)
		go i.processMessages(ctx, i.queueConnections[i])
	}

	return nil
}

func (i *Input) Close(ctx spec.ComponentContext) error {
	i.shutdownOnce.Do(func() {
		close(i.shutdownChan)
	})

	// Wait for all goroutines to finish
	i.wg.Wait()

	// Close all queue connections
	for _, conn := range i.queueConnections {
		if conn != nil {
			if err := conn.qObject.Close(0); err != nil {
				// Log error but continue cleanup
				// Note: We should use a proper logger here
			}
		}
	}

	return nil
}

func (i *Input) Read(ctx spec.ComponentContext) (spec.Batch, spec.ProcessedCallback, error) {
	select {
	case msg := <-i.msgChan:
		return msg.batch, msg.ackFn, nil
	case <-i.shutdownChan:
		return nil, nil, spec.ErrEndOfInput
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

func (i *Input) processMessages(ctx spec.ComponentContext, conn *queueConnection) {
	defer i.wg.Done()

	waitTime, err := time.ParseDuration(i.cfg.WaitTime)
	if err != nil {
		waitTime = 5 * time.Second // fallback default
	}

	for {
		select {
		case <-i.shutdownChan:
			return
		default:
			if err := i.readBatch(ctx, conn, waitTime); err != nil {
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

func (i *Input) readBatch(ctx spec.ComponentContext, conn *queueConnection, waitTime time.Duration) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	batch := ctx.NewBatch()
	var messages []*ibmmq.MQMessage

	// Read up to batch_count messages
	for j := 0; j < i.cfg.BatchCount; j++ {
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
		datalen, err := conn.qObject.Get(mqmd, gmo, buffer)

		if err != nil {
			var mqret *ibmmq.MQReturn
			if errors.As(err, &mqret) {
				// No message available is not an error for batching
				if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
					break
				}
			}
			// Rollback any messages read so far
			if len(messages) > 0 {
				conn.qmgr.Back()
			}
			return fmt.Errorf("failed to get message from queue: %w", err)
		}

		// Store the message for processing
		msg := &ibmmq.MQMessage{
			Data: buffer[:datalen],
			MQMD: mqmd,
		}
		messages = append(messages, msg)

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
	}

	// Only send batch if we have messages
	if batch.Len() > 0 {
		// Create acknowledgment function
		ackFn := func(ackErr error) error {
			conn.mutex.Lock()
			defer conn.mutex.Unlock()

			if ackErr != nil {
				// Rollback transaction
				return conn.qmgr.Back()
			} else {
				// Commit transaction
				return conn.qmgr.Cmit()
			}
		}

		// Send batch to channel
		select {
		case i.msgChan <- asyncMessage{batch: batch, ackFn: ackFn}:
		case <-i.shutdownChan:
			// Rollback if we're shutting down
			conn.qmgr.Back()
			return nil
		}
	}

	return nil
}
