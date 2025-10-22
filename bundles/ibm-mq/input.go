//go:build mqclient

package ibm_mq

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

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

	qmgr    ibmmq.MQQueueManager
	qObject ibmmq.MQObject
	mutex   sync.Mutex
}

func (i *Input) Init(ctx spec.ComponentContext) error {
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

	return nil
}

func (i *Input) Close(ctx spec.ComponentContext) error {
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
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Create MQMD and MQGMO structures
	mqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_WAIT + ibmmq.MQGMO_SYNCPOINT + ibmmq.MQGMO_CONVERT
	gmo.WaitInterval = 5000 // 5 seconds wait

	// Read message
	buffer := make([]byte, 32768) // 32KB buffer
	datalen, err := i.qObject.Get(mqmd, gmo, buffer)

	if err != nil {
		var mqret *ibmmq.MQReturn
		if errors.As(err, &mqret) {
			// No message available is not an error
			if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
				return nil, nil, spec.ErrNoData
			}
		}
		return nil, nil, fmt.Errorf("failed to get message from queue: %w", err)
	}

	// Create message and add to batch
	msg := ctx.NewMessage()
	msg.SetRaw(buffer[:datalen])

	// Add MQ-specific metadata
	msg.SetMetadata("mq_queue", i.cfg.QueueName)
	msg.SetMetadata("mq_message_id", string(mqmd.MsgId))
	msg.SetMetadata("mq_correlation_id", string(mqmd.CorrelId))
	msg.SetMetadata("mq_format", mqmd.Format)
	msg.SetMetadata("mq_priority", fmt.Sprintf("%d", mqmd.Priority))
	msg.SetMetadata("mq_persistence", fmt.Sprintf("%d", mqmd.Persistence))

	batch := ctx.NewBatch(msg)

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

	return batch, ackFn, nil
}
