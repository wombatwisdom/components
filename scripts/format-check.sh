#!/bin/bash

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# First run formatting to ensure everything is formatted
echo "Running code formatting..."
task format

# Check for formatting changes (excluding go.mod/go.sum dependency updates and .claude settings)
if [ -n "$(git diff --exit-code --name-only | grep -v 'go\.mod$\|go\.sum$\|\.claude/')" ]; then
    print_error "Code formatting issues found:"
    git diff --name-only | grep -v 'go\.mod$\|go\.sum$\|\.claude/' | xargs git diff
    exit 1
fi

print_success "Code formatting is correct"