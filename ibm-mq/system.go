//go:build mqclient

package ibm_mq

import (
	"context"
	"fmt"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/wombatwisdom/components/framework/spec"
)

// NewSystem creates a new MQ system from raw configuration
func NewSystem(rawConfig string) (*System, error) {
	var cfg SystemConfig
	if err := cfg.UnmarshalJSON([]byte(rawConfig)); err != nil {
		return nil, err
	}

	return &System{
		cfg: cfg,
	}, nil
}

// NewSystemFromConfig creates a system from a spec.Config interface
func NewSystemFromConfig(config spec.Config) (*System, error) {
	var cfg SystemConfig
	if err := config.Decode(&cfg); err != nil {
		return nil, err
	}

	return &System{
		cfg: cfg,
	}, nil
}

// System represents an IBM MQ system connection manager.
//
// The System handles connection lifecycle for IBM MQ queue managers,
// providing authenticated connections with optional TLS encryption.
// It supports both basic username/password authentication and TLS
// certificate-based authentication.
//
// Multiple components can share the same system instance to reuse
// connections efficiently.
type System struct {
	cfg  SystemConfig
	qmgr *ibmmq.MQQueueManager
}

// Connect establishes a connection to the IBM MQ queue manager
func (s *System) Connect(ctx context.Context) error {
	// Allocate the MQCNO and MQCD structures needed for the CONNX call
	cno := ibmmq.NewMQCNO()
	cd := ibmmq.NewMQCD()

	// Fill in required fields in the MQCD channel definition structure
	cd.ChannelName = s.cfg.ChannelName
	cd.ConnectionName = s.cfg.ConnectionName

	// Configure TLS if enabled
	if s.cfg.Tls != nil && s.cfg.Tls.Enabled {
		cd.SSLCipherSpec = s.cfg.Tls.CipherSpec
		sco := ibmmq.NewMQSCO()

		if s.cfg.Tls.KeyRepository != nil {
			sco.KeyRepository = *s.cfg.Tls.KeyRepository
		}
		if s.cfg.Tls.KeyRepositoryPassword != nil {
			sco.KeyRepoPassword = *s.cfg.Tls.KeyRepositoryPassword
		}
		if s.cfg.Tls.CertificateLabel != nil {
			sco.CertificateLabel = *s.cfg.Tls.CertificateLabel
		}

		cno.SSLConfig = sco
	}

	// Reference the CD structure from the CNO and indicate client connection
	cno.ClientConn = cd
	// Use client binding with reconnect capability and handle sharing
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING + ibmmq.MQCNO_RECONNECT + ibmmq.MQCNO_HANDLE_SHARE_BLOCK

	// Set application name if configured
	if s.cfg.ApplicationName != nil {
		cno.ApplName = *s.cfg.ApplicationName
	} else {
		cno.ApplName = "WombatWisdom MQ Component"
	}

	// Configure authentication if provided
	if s.cfg.UserId != nil && *s.cfg.UserId != "" {
		csp := ibmmq.NewMQCSP()
		csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
		csp.UserId = *s.cfg.UserId
		if s.cfg.Password != nil {
			csp.Password = *s.cfg.Password
		}
		cno.SecurityParms = csp
	}

	// Connect to the queue manager
	var err error
	s.qmgr, err = ibmmq.Connx(s.cfg.QueueManagerName, cno)
	if err != nil {
		return fmt.Errorf("failed to connect to queue manager %s: %w", s.cfg.QueueManagerName, err)
	}

	return nil
}

// Client returns the IBM MQ queue manager client
func (s *System) Client() any {
	return s.qmgr
}

// Close disconnects from the IBM MQ queue manager
func (s *System) Close(ctx context.Context) error {
	if s.qmgr != nil {
		err := s.qmgr.Disc()
		s.qmgr = nil
		if err != nil {
			return fmt.Errorf("failed to disconnect from queue manager: %w", err)
		}
	}
	return nil
}
