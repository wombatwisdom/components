package eventbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/wombatwisdom/components/framework/spec"
)

// SQSIntegration implements EventIntegration for SQS-based event consumption
type SQSIntegration struct {
	config    TriggerInputConfig
	sqsClient *sqs.Client
	logger    spec.Logger
}

// NewSQSIntegration creates a new SQS integration
func NewSQSIntegration(config TriggerInputConfig, sqsClient *sqs.Client) *SQSIntegration {
	return &SQSIntegration{
		config:    config,
		sqsClient: sqsClient,
	}
}

// Init initializes the SQS integration
func (s *SQSIntegration) Init(ctx context.Context, logger spec.Logger) error {
	s.logger = logger
	
	// Verify SQS queue exists and is accessible
	input := &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(s.config.SQSQueueURL),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
		},
	}
	
	_, err := s.sqsClient.GetQueueAttributes(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to access SQS queue %s: %w", s.config.SQSQueueURL, err)
	}
	
	s.logger.Infof("SQS integration initialized for queue: %s", s.config.SQSQueueURL)
	return nil
}

// ReadEvents reads events from SQS queue
func (s *SQSIntegration) ReadEvents(ctx context.Context, maxEvents int, timeout time.Duration) ([]EventBridgeEvent, error) {
	// Adjust maxEvents to SQS limits
	maxMessages := int32(maxEvents)
	if maxMessages > s.config.SQSMaxMessages {
		maxMessages = s.config.SQSMaxMessages
	}
	if maxMessages > 10 { // AWS SQS limit
		maxMessages = 10
	}
	
	// Create SQS receive message input
	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(s.config.SQSQueueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     s.config.SQSWaitTimeSeconds,
		VisibilityTimeout:   s.config.SQSVisibilityTimeout,
		MessageAttributeNames: []string{"All"},
	}
	
	// Apply timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Receive messages from SQS
	result, err := s.sqsClient.ReceiveMessage(timeoutCtx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to receive messages from SQS: %w", err)
	}
	
	// Convert SQS messages to EventBridge events
	events := make([]EventBridgeEvent, 0, len(result.Messages))
	messagesToDelete := make([]types.DeleteMessageBatchRequestEntry, 0, len(result.Messages))
	
	for i, message := range result.Messages {
		event, err := s.parseEventBridgeMessage(message)
		if err != nil {
			s.logger.Warnf("Failed to parse SQS message: %v", err)
			continue
		}
		
		events = append(events, event)
		
		// Prepare for batch deletion
		messagesToDelete = append(messagesToDelete, types.DeleteMessageBatchRequestEntry{
			Id:            aws.String(fmt.Sprintf("msg_%d", i)),
			ReceiptHandle: message.ReceiptHandle,
		})
	}
	
	// Delete successfully processed messages
	if len(messagesToDelete) > 0 {
		deleteInput := &sqs.DeleteMessageBatchInput{
			QueueUrl: aws.String(s.config.SQSQueueURL),
			Entries:  messagesToDelete,
		}
		
		_, err := s.sqsClient.DeleteMessageBatch(ctx, deleteInput)
		if err != nil {
			s.logger.Warnf("Failed to delete processed messages: %v", err)
		}
	}
	
	s.logger.Debugf("Read %d events from SQS queue", len(events))
	return events, nil
}

// parseEventBridgeMessage converts an SQS message to an EventBridge event
func (s *SQSIntegration) parseEventBridgeMessage(message types.Message) (EventBridgeEvent, error) {
	if message.Body == nil {
		return EventBridgeEvent{}, fmt.Errorf("message body is nil")
	}
	
	// SQS messages from EventBridge are wrapped in a specific format
	var sqsEventBridgeWrapper struct {
		Type      string `json:"Type"`
		MessageId string `json:"MessageId"`
		Message   string `json:"Message"`
	}
	
	// First, try to parse as SNS/SQS wrapper (common EventBridge → SQS pattern)
	err := json.Unmarshal([]byte(*message.Body), &sqsEventBridgeWrapper)
	if err == nil && sqsEventBridgeWrapper.Message != "" {
		// Parse the inner EventBridge event
		var event EventBridgeEvent
		err = json.Unmarshal([]byte(sqsEventBridgeWrapper.Message), &event)
		if err != nil {
			return EventBridgeEvent{}, fmt.Errorf("failed to parse inner EventBridge event: %w", err)
		}
		return event, nil
	}
	
	// Otherwise, try to parse directly as EventBridge event
	var event EventBridgeEvent
	err = json.Unmarshal([]byte(*message.Body), &event)
	if err != nil {
		return EventBridgeEvent{}, fmt.Errorf("failed to parse EventBridge event: %w", err)
	}
	
	return event, nil
}

// Close shuts down the SQS integration
func (s *SQSIntegration) Close(ctx context.Context) error {
	s.logger.Infof("SQS integration closed")
	return nil
}