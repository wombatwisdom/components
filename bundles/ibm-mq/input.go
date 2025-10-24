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
func NewInput(env spec.Environment, config InputConfig) (*Input, error) {
	return &Input{
		env: env,
		cfg: config,
	}, nil
}

// Input receives messages from an IBM MQ queue.
type Input struct {
	env spec.Environment
	cfg InputConfig

	qmgr    ibmmq.MQQueueManager
	qObject ibmmq.MQObject
	mqLock  sync.Mutex

	initialized bool
}

func (i *Input) Init(ctx spec.ComponentContext) error {
	if i.initialized {
		return spec.ErrAlreadyConnected
	}

	cno := ibmmq.NewMQCNO()
	cd := ibmmq.NewMQCD()

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

	if i.cfg.TLS != nil && i.cfg.TLS.Enabled {
		if i.cfg.TLS.CipherSpec != "" {
			cd.SSLCipherSpec = i.cfg.TLS.CipherSpec
		}

		// SSL Configuration Options
		sco := ibmmq.NewMQSCO()

		if i.cfg.TLS.KeyRepository != "" {
			sco.KeyRepository = i.cfg.TLS.KeyRepository
		}

		if i.cfg.TLS.KeyRepositoryPassword != "" {
			sco.KeyRepoPassword = i.cfg.TLS.KeyRepositoryPassword
		}

		if i.cfg.TLS.CertificateLabel != "" {
			sco.CertificateLabel = i.cfg.TLS.CertificateLabel
		}

		if i.cfg.TLS.FipsRequired {
			sco.FipsRequired = true
		}

		cno.SSLConfig = sco

		if i.cfg.TLS.SSLPeerName != "" {
			cd.SSLPeerName = i.cfg.TLS.SSLPeerName
		}
	}

	// Reference the CD structure from the CNO and indicate client connection
	cno.ClientConn = cd
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING + ibmmq.MQCNO_RECONNECT + ibmmq.MQCNO_HANDLE_SHARE_BLOCK

	hostname, _ := os.Hostname()
	maxHostLen := 16
	if len(hostname) > maxHostLen {
		hostname = hostname[:maxHostLen]
	}
	cno.ApplName = fmt.Sprintf("WW MQ Input %s", hostname)

	// auth
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
	i.initialized = true

	return nil
}

func (i *Input) Close(ctx spec.ComponentContext) error {
	if !i.initialized {
		return nil
	}

	i.mqLock.Lock()
	if err := i.qmgr.Back(); err != nil {
		i.env.Errorf("Failed to rollback transaction: %v", err)
	}
	i.mqLock.Unlock()

	if err := i.qObject.Close(0); err != nil {
		i.env.Errorf("Failed to close queue: %v", err)
	}

	if err := i.qmgr.Disc(); err != nil {
		i.env.Errorf("Failed to disconnect from queue manager: %v", err)
	}

	i.initialized = false
	return nil
}

func (i *Input) Read(ctx spec.ComponentContext) (spec.Batch, spec.ProcessedCallback, error) {
	i.mqLock.Lock()
	defer i.mqLock.Unlock()

	if !i.initialized {
		return nil, nil, spec.ErrNotConnected
	}

	// Default batch size to 1 if not set
	batchSize := i.cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	// Parse batch wait time, default to 5 seconds
	waitInterval := int32(5000) // milliseconds
	if i.cfg.BatchWaitTime != "" {
		if duration, err := time.ParseDuration(i.cfg.BatchWaitTime); err == nil {
			waitInterval = int32(duration.Milliseconds())
			if waitInterval <= 0 {
				waitInterval = 5000
			}
		}
	}

	// Collect messages for the batch
	var messages []spec.Message
	buffer := make([]byte, 32768)

	for j := 0; j < batchSize; j++ {
		mqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()
		gmo.Options = ibmmq.MQGMO_SYNCPOINT + ibmmq.MQGMO_CONVERT

		// Only wait on first message, subsequent messages should be available immediately
		if j == 0 {
			gmo.Options |= ibmmq.MQGMO_WAIT
			gmo.WaitInterval = waitInterval
		} else {
			// For subsequent messages, don't wait - return partial batch if no more messages
			gmo.Options |= ibmmq.MQGMO_NO_WAIT
		}

		datalen, err := i.qObject.Get(mqmd, gmo, buffer)

		if err != nil {
			var mqret *ibmmq.MQReturn
			if errors.As(err, &mqret) {
				if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
					// If this is the first message and no messages available, return error
					if j == 0 {
						return nil, nil, spec.ErrNoData
					}
					// Otherwise return partial batch
					break
				}
			}
			// Any other error, rollback and return
			if j > 0 {
				// We have partial messages, need to rollback
				if rollbackErr := i.qmgr.Back(); rollbackErr != nil {
					i.env.Errorf("Failed to rollback partial batch: %v", rollbackErr)
				}
			}
			return nil, nil, fmt.Errorf("failed to get message from queue: %w", err)
		}

		// Create message with data and metadata
		msg := ctx.NewMessage()

		// Copy the data to avoid reuse issues
		msgData := make([]byte, datalen)
		copy(msgData, buffer[:datalen])
		msg.SetRaw(msgData)

		msg.SetMetadata("mq_queue", i.cfg.QueueName)
		msg.SetMetadata("mq_message_id", string(mqmd.MsgId))
		msg.SetMetadata("mq_correlation_id", string(mqmd.CorrelId))
		msg.SetMetadata("mq_format", mqmd.Format)
		msg.SetMetadata("mq_priority", fmt.Sprintf("%d", mqmd.Priority))
		msg.SetMetadata("mq_persistence", fmt.Sprintf("%d", mqmd.Persistence))

		messages = append(messages, msg)
	}

	// Create batch with all collected messages
	batch := ctx.NewBatch(messages...)

	// Acknowledgment function handles commit/rollback for entire batch
	ackFn := func(ackCtx context.Context, ackErr error) error {
		i.mqLock.Lock()
		defer i.mqLock.Unlock()

		if ackCtx.Err() != nil || ctx.Context().Err() != nil {
			return i.qmgr.Back()
		}

		if ackErr != nil {
			return i.qmgr.Back()
		}
		return i.qmgr.Cmit()
	}

	return batch, ackFn, nil
}
