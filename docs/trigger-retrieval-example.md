# Trigger-Retrieval Pattern Example

This document demonstrates how the trigger-retrieval pattern works in practice.

## Example Scenario: S3 Object Processing

**Goal**: Process new JSON files uploaded to S3 bucket  
**Current Problem**: S3 input lists + fetches all objects (inefficient)  
**New Solution**: EventBridge triggers + S3 retrieval processor

## Component Flow

```
EventBridgeInput → S3RetrievalProcessor → JSONParserProcessor → Output
      │                    │                      │
      │                    │                      └─ Parse JSON content
      │                    └─ Fetch object content only when needed
      └─ Lightweight S3 event notifications
```

## Implementation Example

### 1. EventBridge Trigger Input

```go
type EventBridgeInput struct {
    // Listens to S3 events via EventBridge
    events chan aws.Event
}

func (e *EventBridgeInput) ReadTriggers(ctx ComponentContext) (TriggerBatch, ProcessedCallback, error) {
    batch := spec.NewTriggerBatch()
    
    select {
    case event := <-e.events:
        // Convert S3 event to trigger
        trigger := spec.NewTriggerEvent(
            spec.TriggerSourceEventBridge,
            fmt.Sprintf("s3://%s/%s", event.Bucket, event.Key),
            map[string]any{
                spec.MetadataBucket:    event.Bucket,
                spec.MetadataKey:       event.Key,
                spec.MetadataEventName: event.EventName,
                spec.MetadataSize:      event.Size,
                spec.MetadataETag:      event.ETag,
            },
        )
        batch.Append(trigger)
    default:
        // No events available
    }
    
    return batch, spec.NoopCallback, nil
}
```

### 2. S3 Retrieval Processor

```go
type S3RetrievalProcessor struct {
    s3Client *s3.Client
}

func (s *S3RetrievalProcessor) Retrieve(ctx ComponentContext, triggers TriggerBatch) (Batch, ProcessedCallback, error) {
    batch := ctx.NewBatch()
    
    for _, trigger := range triggers.Triggers() {
        // Filter: only process JSON files
        key := trigger.Metadata()[spec.MetadataKey].(string)
        if !strings.HasSuffix(key, ".json") {
            continue // Skip non-JSON files
        }
        
        // Filter: only process files > 100 bytes
        size := trigger.Metadata()[spec.MetadataSize].(int64)
        if size < 100 {
            continue // Skip tiny files
        }
        
        // Now fetch the actual object (expensive operation)
        bucket := trigger.Metadata()[spec.MetadataBucket].(string)
        
        resp, err := s.s3Client.GetObject(ctx.Context(), &s3.GetObjectInput{
            Bucket: &bucket,
            Key:    &key,
        })
        if err != nil {
            return nil, nil, fmt.Errorf("failed to get object %s/%s: %w", bucket, key, err)
        }
        
        // Create message with original trigger metadata
        msg := ctx.NewMessage()
        msg.SetRaw(resp.Body)
        msg.SetMetadata("trigger_source", trigger.Source())
        msg.SetMetadata("s3_bucket", bucket)
        msg.SetMetadata("s3_key", key)
        msg.SetMetadata("s3_size", size)
        
        batch.Append(msg)
    }
    
    return batch, spec.NoopCallback, nil
}
```

## Benefits Demonstrated

### 1. Efficiency Gains
```
Before (Current S3 Input):
- List 1000 objects: 1 API call
- Fetch 1000 objects: 1000 API calls  
- Total: 1001 API calls (all objects processed)

After (Trigger + Retrieval):
- EventBridge notifications: 0 API calls (AWS pushes to you)
- Filter 950 objects (wrong type/size): 0 API calls
- Fetch 50 relevant objects: 50 API calls
- Total: 50 API calls (95% reduction!)
```

### 2. Flexibility
```
Multiple Trigger Sources → Same Retrieval Logic:

EventBridge ──┐
SQS ─────────┼──→ S3RetrievalProcessor → Output
Generate ────┘     (same filtering & fetching logic)
```

### 3. Independent Scaling
```
Trigger Detection: High frequency, low CPU (scale horizontally)
Data Retrieval: Low frequency, high CPU+Network (scale vertically)
```

## Migration Path

### Phase 1: Current State
```
S3Input → Output
(lists + fetches in one component)
```

### Phase 2: Add Trigger Support  
```
S3Input → Output (legacy, unchanged)

EventBridgeInput → S3RetrievalProcessor → Output (new pattern)
```

### Phase 3: Deprecate Legacy
```
EventBridgeInput → S3RetrievalProcessor → Output (recommended)
GenerateInput → S3RetrievalProcessor → Output (polling fallback)
```

## Configuration Example

```yaml
pipeline:
  inputs:
    s3_events:
      type: eventbridge
      config:
        event_source: "aws.s3"
        event_bus: "default"
        filters:
          - bucket: "my-data-bucket"
            event_name: "ObjectCreated:*"
  
  processors:
    s3_fetch:
      type: s3_retrieval  
      config:
        filters:
          file_extensions: [".json", ".csv"]
          min_size_bytes: 100
          max_size_bytes: 10485760  # 10MB
        
    parse_json:
      type: json_parser
      
  outputs:
    processed_data:
      type: elasticsearch
```

This configuration creates an efficient pipeline that:
1. Receives S3 events via EventBridge (real-time, no polling)
2. Filters events before expensive S3 API calls
3. Only fetches objects that match criteria
4. Processes valid JSON data to Elasticsearch