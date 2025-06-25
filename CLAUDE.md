# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WombatWisdom Components is a Go library that provides messaging system components with a clear separation between clients and components. It aims to create consistent alternatives to redpanda-connect components while maintaining compatibility with the latest benthos version.

## Architecture

The codebase uses a layered architecture with clear separation of concerns:

- **Core Package** (`core/`): Defines the fundamental interfaces for the component system
- **Spec Package** (`spec/`): Extended specification interfaces that components implement  
- **System vs Component Pattern**: Key architectural distinction where:
  - `System` interface manages connection lifecycle and provides raw clients
  - `Component` interfaces (Input/Output) handle messaging operations using the system's client
  - Each component package implements both a System and Input/Output components

### Component Structure Pattern

Each messaging system follows this pattern:
```
packagename/
├── system.go          # Implements spec.System, manages connections
├── input.go           # Implements spec.Input, reads messages
├── output.go          # Implements spec.Output, writes messages  
├── *_config.go        # Generated from schema files
├── *.schema.yaml      # JSON schema definitions
└── *_test.go          # Tests using Ginkgo/Gomega
```

## Development Commands

### Testing
- Run all tests: `go test ./...`
- Run specific package tests: `go test ./nats/core` or `go test ./spec`
- Run tests with verbose output: `go test ./... -v`
- Tests use Ginkgo v2 framework with Gomega matchers
- All packages should have comprehensive test coverage

### Code Generation
For components with schema files (like `nats/core/`):
```bash
cd nats/core/
task models:generate
```
This generates Go structs from JSON schema files using go-jsonschema.

### Dependencies
- Install dependencies: `go mod tidy`
- Update dependencies: `go get -u ./...`

## Key Interfaces

- `spec.System`: Connection management with `Connect()`, `Close()`, and `Client()` methods
- `spec.Component`: Base interface with `Init()` and `Close()` methods
- `spec.Input`: Reads message batches via `Read()` method
- `spec.Output`: Writes message batches via `Write()` method
- `spec.ComponentContext`: Provides message creation, expression parsing, and configuration

## Testing Infrastructure

- Uses Ginkgo BDD testing framework with Gomega assertions
- Test suites are organized with `*_suite_test.go` files
- Integration tests use real servers (NATS server, S3 mocks, MQTT brokers)
- Test utilities in `test/` package provide common testing infrastructure

## Messaging Patterns

- Messages are processed in batches (`spec.Batch`)
- Each message contains raw data and metadata
- Components use expression evaluation for dynamic configuration (subjects, routing)
- Metadata filtering allows selective header propagation

## Design Memories

- Keep using the core benthos engine