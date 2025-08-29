#!/bin/bash
set -e

echo "🚀 Running CI checks locally..."
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    echo -e "${YELLOW}==== $1 ====${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if we're in the right directory
if [ ! -f "Taskfile.yml" ]; then
    print_error "This script must be run from the project root directory (where Taskfile.yml exists)"
    exit 1
fi

# Check if Task is installed
if ! command -v task &> /dev/null; then
    print_error "Task is not installed. Please install Task from https://taskfile.dev/installation/"
    exit 1
fi

print_status "Step 1: Tidy module dependencies"
go mod tidy
print_success "Dependencies tidied"

print_status "Step 2: Download and verify dependencies"
go mod download
go mod download -C tools

go mod verify
go mod verify -C tools
print_success "Dependencies verified"

print_status "Step 3: Run tests"
task test
print_success "Tests passed"

print_status "Step 4: Run tests with race detector"
go test -C framework -race ./...
# Race tests are now handled by task commands
print_success "Race detector tests passed"

print_status "Step 5: Run linting"
task vet
print_success "Go vet passed"

print_status "Step 6: Check formatting"
task format
# Check for formatting changes (excluding go.mod/go.sum dependency updates and .claude settings)
if [ -n "$(git diff --exit-code --name-only | grep -v 'go\.mod$\|go\.sum$\|\.claude/')" ]; then
    print_error "Code formatting issues found:"
    git diff --name-only | grep -v 'go\.mod$\|go\.sum$\|\.claude/' | xargs git diff
    exit 1
fi
print_success "Code formatting is correct"

print_status "Step 7: Build all modules"
task build
task build:check
print_success "All modules built successfully"

echo ""
echo -e "${GREEN}🎉 All CI checks passed locally!${NC}"
echo ""
echo "Your code is ready for CI. The following commands were run:"
echo "  1. go mod tidy"
echo "  2. go mod download/verify on all modules"
echo "  3. task test (tests all workspace modules)"
echo "  4. Race detector tests on all modules"
echo "  5. task vet (go vet on all modules)"
echo "  6. task format + formatting check"
echo "  7. task build + task build:check"
echo ""
echo "💡 Tip: Run this script before committing to catch CI issues early!"