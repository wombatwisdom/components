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

This project uses [Task](https://taskfile.dev) for development workflow automation. Install with:
```bash
# macOS
brew install go-task/tap/go-task

# Linux/Windows - see https://taskfile.dev/installation/
```

### Quick Start
```bash
task                    # Show all available tasks
task status            # Show project status
task setup             # Setup development environment
task test              # Run working package tests (spec, nats/core)
task test:all          # Attempt all tests (will fail on legacy components)
task ci:test           # Run CI pipeline for working packages
```

### Testing Tasks
- `task test` - Run working package tests (spec, nats/core) 
- `task test:all` - Attempt all tests (will fail on legacy mqtt/s3)
- `task test:short` - Run working tests without verbose output
- `task test:spec` - Run spec package tests only
- `task test:nats` - Run NATS core tests only
- `task test:coverage` - Generate test coverage report

### Code Quality Tasks
- `task lint` - Run linting checks
- `task lint:fix` - Run linting with auto-fix
- `task format` - Format code (go fmt + goimports)
- `task vet` - Run go vet

### Build Tasks
- `task build` - Build all packages
- `task build:check` - Check that all packages compile

### Schema Generation
- `task schema:generate` - Generate schemas for all components
- `task nats:schema:generate` - Generate NATS schemas only

### Component Status
- `task mqtt:status` - Check MQTT component migration status
- `task s3:status` - Check S3 component migration status

### Dependencies
- `task deps:tidy` - Tidy module dependencies (`go mod tidy`)
- `task deps:update` - Update dependencies (`go get -u ./...`)
- `task tools:install` - Install development tools

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