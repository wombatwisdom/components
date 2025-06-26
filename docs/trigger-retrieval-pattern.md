# Trigger-Retrieval Pattern

## Overview

This document defines the foundational pattern for separating event triggering from data retrieval in streaming components.

## Problem Statement

Current components like S3 input mix two concerns:
1. **Triggering**: Detecting when something has changed (listing S3 objects)
2. **Retrieval**: Fetching the actual data (downloading S3 objects)

This coupling leads to:
- Inefficient processing (fetching data that might not be needed)
- Poor scaling (can't independently scale trigger vs retrieval)
- Limited flexibility (can't swap trigger mechanisms)

## Pattern Design

### Three Component Types

#### 1. TriggerInput
**Purpose**: Detect events and emit lightweight trigger signals
**Characteristics**:
- Fast, low-resource operations
- Emits references/pointers to data, not data itself
- Multiple trigger mechanisms for same data source
- High frequency, low payload

**Examples**:
- EventBridge events (S3 object created)
- SQS polling (object notifications) 
- Generate (periodic triggers)
- File system watches

#### 2. RetrievalProcessor  
**Purpose**: Fetch actual data based on trigger events
**Characteristics**:
- Heavy operations (network calls, file I/O)
- Can filter triggers before expensive operations
- Stateless, idempotent
- Low frequency, high payload

**Examples**:
- S3 object fetcher
- HTTP API caller
- Database query executor
- File reader

#### 3. SelfContainedInput
**Purpose**: Trigger + data in one component (existing pattern)
**Characteristics**:
- Real-time data streams where trigger IS the data
- No separation needed
- Direct message consumption

**Examples**:
- NATS (message = trigger + data)
- MQTT (message = trigger + data)
- Kafka (message = trigger + data)

## Flow Patterns

### Pattern 1: Event-Driven Retrieval
```
TriggerInput → RetrievalProcessor → Output
  │              │
  │              └─ Filters triggers before retrieval
  └─ Emits object references
```

### Pattern 2: Self-Contained Streaming  
```
SelfContainedInput → Output
  │
  └─ Message contains all needed data
```

### Pattern 3: Multi-Trigger Retrieval
```
EventBridgeInput ─┐
SQSInput ────────┼─→ RetrievalProcessor → Output
GenerateInput ───┘
```

## Message Flow Design

### Trigger Messages
```go
type TriggerMessage struct {
    Source     string            // "s3", "eventbridge", "generate"
    Reference  string            // Object key, event ID, timer ID
    Metadata   map[string]any    // Bucket, timestamp, etc.
    // No actual data payload
}
```

### Data Messages  
```go
type DataMessage struct {
    Source     string            // Original trigger source
    Reference  string            // Original reference
    Data       []byte            // Actual content
    Metadata   map[string]any    // Enhanced metadata
}
```

## Benefits

1. **Efficiency**: Only fetch data that passes trigger filters
2. **Scalability**: Independent scaling of trigger detection vs data retrieval
3. **Flexibility**: Multiple trigger mechanisms for same data source
4. **Testability**: Easier to test triggers vs retrieval separately
5. **Cost Optimization**: Reduce expensive operations (S3 API calls)

## Implementation Strategy

1. **Phase 1**: Define new interfaces in framework/spec
2. **Phase 2**: Implement EventBridge trigger input
3. **Phase 3**: Refactor S3 component to retrieval processor
4. **Phase 4**: Create example flows and documentation
5. **Phase 5**: Migration guide for existing components