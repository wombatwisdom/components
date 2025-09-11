package spec_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

// MockTriggerInput for testing trigger-retrieval workflow
type mockTriggerInput struct {
	triggers []spec.TriggerEvent
	closed   bool
}

func NewMockTriggerInput(triggers []spec.TriggerEvent) *mockTriggerInput {
	return &mockTriggerInput{triggers: triggers}
}

func (m *mockTriggerInput) Init(ctx spec.ComponentContext) error {
	return nil
}

func (m *mockTriggerInput) Close(ctx spec.ComponentContext) error {
	m.closed = true
	return nil
}

func (m *mockTriggerInput) ReadTriggers(ctx spec.ComponentContext) (spec.TriggerBatch, spec.ProcessedCallback, error) {
	if len(m.triggers) == 0 {
		time.Sleep(10 * time.Millisecond) // Simulate polling delay
		return nil, nil, spec.ErrNoData
	}

	batch := test.NewMockTriggerBatch()
	for _, trigger := range m.triggers {
		batch.Append(trigger)
	}

	// Clear triggers after reading (simulate consumption)
	m.triggers = nil

	callback := func(ctx context.Context, err error) error {
		// In real implementation, this would ack the underlying event
		return nil
	}

	return batch, callback, nil
}

// MockRetrievalProcessor for testing trigger-retrieval workflow
type mockRetrievalProcessor struct {
	retrievedData map[string][]byte
	closed        bool
}

func NewMockRetrievalProcessor(data map[string][]byte) *mockRetrievalProcessor {
	return &mockRetrievalProcessor{retrievedData: data}
}

func (m *mockRetrievalProcessor) Init(ctx spec.ComponentContext) error {
	return nil
}

func (m *mockRetrievalProcessor) Close(ctx spec.ComponentContext) error {
	m.closed = true
	return nil
}

func (m *mockRetrievalProcessor) Retrieve(ctx spec.ComponentContext, triggers spec.TriggerBatch) (spec.Batch, spec.ProcessedCallback, error) {
	batch := test.NewMockBatch()

	for _, trigger := range triggers.Triggers() {
		if data, exists := m.retrievedData[trigger.Reference()]; exists {
			msg := test.NewMockMessage(data)
			msg.SetRaw(data)
			msg.SetMetadata("trigger_source", trigger.Source())
			msg.SetMetadata("trigger_reference", trigger.Reference())
			msg.SetMetadata("trigger_timestamp", trigger.Timestamp())
			batch.Append(msg)
		}
	}

	callback := func(ctx context.Context, err error) error {
		// In real implementation, this would handle retrieval acknowledgment
		return nil
	}

	return batch, callback, nil
}

// MockTriggerEvent for testing
type mockTriggerEvent struct {
	source    string
	reference string
	metadata  map[string]any
	timestamp int64
}

func NewMockTriggerEvent(source, reference string) *mockTriggerEvent {
	return &mockTriggerEvent{
		source:    source,
		reference: reference,
		metadata:  make(map[string]any),
		timestamp: time.Now().Unix(),
	}
}

func (m *mockTriggerEvent) Source() string           { return m.source }
func (m *mockTriggerEvent) Reference() string        { return m.reference }
func (m *mockTriggerEvent) Metadata() map[string]any { return m.metadata }
func (m *mockTriggerEvent) Timestamp() int64         { return m.timestamp }

var _ = Describe("Trigger-Retrieval Integration", func() {
	var (
		ctx                spec.ComponentContext
		triggerInput       *mockTriggerInput
		retrievalProcessor *mockRetrievalProcessor
	)

	BeforeEach(func() {
		ctx = test.NewMockComponentContext()

		// Setup test data
		triggers := []spec.TriggerEvent{
			NewMockTriggerEvent("s3", "bucket/file1.json"),
			NewMockTriggerEvent("s3", "bucket/file2.json"),
			NewMockTriggerEvent("s3", "bucket/missing.json"), // This file doesn't exist
		}

		retrievalData := map[string][]byte{
			"bucket/file1.json": []byte(`{"id": 1, "name": "test1"}`),
			"bucket/file2.json": []byte(`{"id": 2, "name": "test2"}`),
			// Note: bucket/missing.json is intentionally not included
		}

		triggerInput = NewMockTriggerInput(triggers)
		retrievalProcessor = NewMockRetrievalProcessor(retrievalData)
	})

	AfterEach(func() {
		if triggerInput != nil {
			_ = triggerInput.Close(ctx)
		}
		if retrievalProcessor != nil {
			_ = retrievalProcessor.Close(ctx)
		}
	})

	Describe("End-to-End Trigger-Retrieval Workflow", func() {
		It("should successfully process triggers and retrieve data", func() {
			// Initialize components
			err := triggerInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			err = retrievalProcessor.Init(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Step 1: Read triggers from input
			triggerBatch, triggerCallback, err := triggerInput.ReadTriggers(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(triggerBatch).ToNot(BeNil())
			Expect(triggerBatch.Triggers()).To(HaveLen(3))

			// Verify trigger content
			triggers := triggerBatch.Triggers()
			Expect(triggers[0].Source()).To(Equal("s3"))
			Expect(triggers[0].Reference()).To(Equal("bucket/file1.json"))
			Expect(triggers[1].Reference()).To(Equal("bucket/file2.json"))
			Expect(triggers[2].Reference()).To(Equal("bucket/missing.json"))

			// Step 2: Retrieve data based on triggers
			dataBatch, retrievalCallback, err := retrievalProcessor.Retrieve(ctx, triggerBatch)
			Expect(err).ToNot(HaveOccurred())
			Expect(dataBatch).ToNot(BeNil())

			// Should only retrieve data for files that exist
			messagesList := make([]spec.Message, 0)
			for _, msg := range dataBatch.Messages() {
				messagesList = append(messagesList, msg)
			}
			Expect(messagesList).To(HaveLen(2)) // Only file1.json and file2.json

			// Verify retrieved data
			msg1Raw, err := messagesList[0].Raw()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(msg1Raw)).To(ContainSubstring("test1"))

			msg2Raw, err := messagesList[1].Raw()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(msg2Raw)).To(ContainSubstring("test2"))

			// Step 3: Acknowledge processing
			ctx := context.Background()
			err = retrievalCallback(ctx, nil)
			Expect(err).ToNot(HaveOccurred())

			err = triggerCallback(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle empty trigger batches gracefully", func() {
			// Create input with no triggers
			emptyInput := NewMockTriggerInput([]spec.TriggerEvent{})
			err := emptyInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = emptyInput.Close(ctx) }()

			// Should return no data error
			triggerBatch, _, err := emptyInput.ReadTriggers(ctx)
			Expect(err).To(Equal(spec.ErrNoData))
			Expect(triggerBatch).To(BeNil())
		})

		It("should handle filtered triggers correctly", func() {
			// Create triggers
			triggers := []spec.TriggerEvent{
				NewMockTriggerEvent("s3", "bucket/important.json"),
				NewMockTriggerEvent("s3", "bucket/temp.tmp"),
			}

			// Only have data for important file
			retrievalData := map[string][]byte{
				"bucket/important.json": []byte(`{"important": true}`),
			}

			testInput := NewMockTriggerInput(triggers)
			testProcessor := NewMockRetrievalProcessor(retrievalData)

			err := testInput.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = testInput.Close(ctx) }()

			err = testProcessor.Init(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = testProcessor.Close(ctx) }()

			// Read triggers
			triggerBatch, _, err := testInput.ReadTriggers(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(triggerBatch.Triggers()).To(HaveLen(2))

			// Retrieve data (processor filters out unavailable data)
			dataBatch, _, err := testProcessor.Retrieve(ctx, triggerBatch)
			Expect(err).ToNot(HaveOccurred())

			// Should only contain data for the available file
			messagesList := make([]spec.Message, 0)
			for _, msg := range dataBatch.Messages() {
				messagesList = append(messagesList, msg)
			}
			Expect(messagesList).To(HaveLen(1))

			raw, err := messagesList[0].Raw()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(raw)).To(ContainSubstring("important"))
		})
	})
})
