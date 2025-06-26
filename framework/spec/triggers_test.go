package spec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/framework/spec"
)

var _ = Describe("Trigger-Retrieval Pattern", func() {
	Describe("TriggerEvent", func() {
		It("should create trigger event with metadata", func() {
			metadata := map[string]any{
				spec.MetadataBucket: "my-bucket",
				spec.MetadataKey:    "path/to/object.json",
				spec.MetadataSize:   1024,
			}

			trigger := spec.NewTriggerEvent(
				spec.TriggerSourceEventBridge,
				"s3://my-bucket/path/to/object.json",
				metadata,
			)

			Expect(trigger.Source()).To(Equal(spec.TriggerSourceEventBridge))
			Expect(trigger.Reference()).To(Equal("s3://my-bucket/path/to/object.json"))
			Expect(trigger.Metadata()[spec.MetadataBucket]).To(Equal("my-bucket"))
			Expect(trigger.Metadata()[spec.MetadataKey]).To(Equal("path/to/object.json"))
			Expect(trigger.Metadata()[spec.MetadataSize]).To(Equal(1024))
			Expect(trigger.Timestamp()).To(BeNumerically(">", 0))
		})

		It("should handle nil metadata", func() {
			trigger := spec.NewTriggerEvent(
				spec.TriggerSourceGenerate,
				"timer-1",
				nil,
			)

			Expect(trigger.Metadata()).ToNot(BeNil())
			Expect(len(trigger.Metadata())).To(Equal(0))
		})
	})

	Describe("TriggerBatch", func() {
		It("should collect trigger events", func() {
			batch := spec.NewTriggerBatch()

			trigger1 := spec.NewTriggerEvent(
				spec.TriggerSourceS3Polling,
				"s3://bucket1/file1.txt",
				map[string]any{spec.MetadataBucket: "bucket1"},
			)

			trigger2 := spec.NewTriggerEvent(
				spec.TriggerSourceEventBridge,
				"s3://bucket2/file2.txt",
				map[string]any{spec.MetadataBucket: "bucket2"},
			)

			batch.Append(trigger1)
			batch.Append(trigger2)

			triggers := batch.Triggers()
			Expect(len(triggers)).To(Equal(2))
			Expect(triggers[0].Source()).To(Equal(spec.TriggerSourceS3Polling))
			Expect(triggers[1].Source()).To(Equal(spec.TriggerSourceEventBridge))
		})

		It("should start empty", func() {
			batch := spec.NewTriggerBatch()
			Expect(len(batch.Triggers())).To(Equal(0))
		})
	})

	Describe("Trigger Sources", func() {
		It("should have defined constants", func() {
			Expect(spec.TriggerSourceEventBridge).To(Equal("eventbridge"))
			Expect(spec.TriggerSourceS3Polling).To(Equal("s3-polling"))
			Expect(spec.TriggerSourceGenerate).To(Equal("generate"))
			Expect(spec.TriggerSourceSQS).To(Equal("sqs"))
			Expect(spec.TriggerSourceFile).To(Equal("file"))
		})
	})

	Describe("Metadata Keys", func() {
		It("should have standard constants", func() {
			Expect(spec.MetadataBucket).To(Equal("bucket"))
			Expect(spec.MetadataKey).To(Equal("key"))
			Expect(spec.MetadataEventName).To(Equal("event_name"))
			Expect(spec.MetadataRegion).To(Equal("region"))
			Expect(spec.MetadataTimestamp).To(Equal("timestamp"))
			Expect(spec.MetadataSize).To(Equal("size"))
			Expect(spec.MetadataETag).To(Equal("etag"))
		})
	})
})
