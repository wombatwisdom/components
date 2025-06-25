#!/bin/bash

# WombatWisdom Components Test Runner
# Comprehensive testing script for CI/CD and local development

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VERBOSE=${VERBOSE:-false}
INTEGRATION=${INTEGRATION:-false}
COVERAGE=${COVERAGE:-false}
RACE=${RACE:-false}
BENCHMARK=${BENCHMARK:-false}

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    if ! command -v task &> /dev/null; then
        log_warning "Task is not installed. Installing..."
        go install github.com/go-task/task/v3/cmd/task@latest
    fi
    
    log_success "Dependencies check passed"
}

run_unit_tests() {
    log_info "Running unit tests..."
    
    local args=""
    if [[ "$VERBOSE" == "true" ]]; then
        args="$args -v"
    fi
    
    if [[ "$RACE" == "true" ]]; then
        args="$args -race"
    fi
    
    if [[ "$COVERAGE" == "true" ]]; then
        args="$args -coverprofile=coverage.out"
    fi
    
    # Run core package tests
    go test $args ./spec/... ./nats/core/... ./mqtt/... ./test/...
    
    log_success "Unit tests passed"
}

run_integration_tests() {
    if [[ "$INTEGRATION" != "true" ]]; then
        log_info "Skipping integration tests (set INTEGRATION=true to enable)"
        return 0
    fi
    
    log_info "Running integration tests..."
    
    # Check if docker is available for test containers
    if ! command -v docker &> /dev/null; then
        log_warning "Docker not available, skipping integration tests"
        return 0
    fi
    
    # Run integration tests with longer timeout
    go test -v -timeout=30m -tags=integration ./...
    
    log_success "Integration tests passed"
}

run_benchmarks() {
    if [[ "$BENCHMARK" != "true" ]]; then
        log_info "Skipping benchmarks (set BENCHMARK=true to enable)"
        return 0
    fi
    
    log_info "Running benchmarks..."
    
    go test -bench=. -benchmem ./...
    
    log_success "Benchmarks completed"
}

check_code_quality() {
    log_info "Checking code quality..."
    
    # Format check
    if ! go fmt ./... | grep -q .; then
        log_success "Code formatting is correct"
    else
        log_error "Code is not formatted. Run 'go fmt ./...' or 'task format'"
        return 1
    fi
    
    # Vet check
    if go vet ./...; then
        log_success "Go vet passed"
    else
        log_error "Go vet failed"
        return 1
    fi
    
    # Lint check (if golangci-lint is available)
    if command -v golangci-lint &> /dev/null; then
        if golangci-lint run ./...; then
            log_success "Linting passed"
        else
            log_error "Linting failed"
            return 1
        fi
    else
        log_warning "golangci-lint not available, skipping lint check"
    fi
}

generate_coverage_report() {
    if [[ "$COVERAGE" != "true" ]] || [[ ! -f "coverage.out" ]]; then
        return 0
    fi
    
    log_info "Generating coverage report..."
    
    # Generate HTML report
    go tool cover -html=coverage.out -o coverage.html
    
    # Show coverage summary
    go tool cover -func=coverage.out | tail -1
    
    log_success "Coverage report generated: coverage.html"
}

print_summary() {
    log_info "Test Summary:"
    echo "  - Unit tests: ✅"
    
    if [[ "$INTEGRATION" == "true" ]]; then
        echo "  - Integration tests: ✅"
    else
        echo "  - Integration tests: ⏭️ (skipped)"
    fi
    
    if [[ "$BENCHMARK" == "true" ]]; then
        echo "  - Benchmarks: ✅"
    else
        echo "  - Benchmarks: ⏭️ (skipped)"
    fi
    
    echo "  - Code quality: ✅"
    
    if [[ "$COVERAGE" == "true" ]]; then
        echo "  - Coverage report: ✅"
    fi
}

show_help() {
    echo "WombatWisdom Components Test Runner"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help        Show this help message"
    echo "  -v, --verbose     Enable verbose output"
    echo "  -i, --integration Enable integration tests"
    echo "  -c, --coverage    Generate coverage report"
    echo "  -r, --race        Enable race detection"
    echo "  -b, --benchmark   Run benchmarks"
    echo "  --ci              CI mode (integration + coverage + race)"
    echo ""
    echo "Environment Variables:"
    echo "  VERBOSE           Enable verbose output (true/false)"
    echo "  INTEGRATION       Enable integration tests (true/false)"
    echo "  COVERAGE          Generate coverage report (true/false)"
    echo "  RACE              Enable race detection (true/false)"
    echo "  BENCHMARK         Run benchmarks (true/false)"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run basic unit tests"
    echo "  $0 -v -c             # Verbose mode with coverage"
    echo "  $0 --ci              # Full CI pipeline"
    echo "  INTEGRATION=true $0  # Run with integration tests"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -i|--integration)
            INTEGRATION=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -r|--race)
            RACE=true
            shift
            ;;
        -b|--benchmark)
            BENCHMARK=true
            shift
            ;;
        --ci)
            INTEGRATION=true
            COVERAGE=true
            RACE=true
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Main execution
main() {
    log_info "Starting WombatWisdom Components test suite..."
    
    check_dependencies
    
    # Clean up any previous artifacts
    rm -f coverage.out coverage.html
    
    # Run tests
    run_unit_tests
    run_integration_tests
    run_benchmarks
    check_code_quality
    generate_coverage_report
    
    print_summary
    log_success "All tests completed successfully!"
}

# Trap to clean up on exit
cleanup() {
    if [[ -f "coverage.out" ]] && [[ "$COVERAGE" != "true" ]]; then
        rm -f coverage.out
    fi
}
trap cleanup EXIT

# Run main function
main "$@"