#!/bin/bash
set -e

echo "🔍 Running linting checks..."

echo "📝 Formatting Go code..."
go fmt -C framework ./...
go fmt -C components/aws-s3 ./...
go fmt -C components/mqtt ./...
go fmt -C components/nats ./...

echo "📦 Running goimports..."
goimports -w ./framework ./components

echo "🔬 Running go vet..."
go vet -C framework ./...
go vet -C components/aws-s3 ./...
go vet -C components/mqtt ./...
go vet -C components/nats ./...

echo "📋 Tidying modules..."
go mod tidy -C framework
go mod tidy -C components/aws-s3
go mod tidy -C components/mqtt
go mod tidy -C components/nats
go mod tidy -C tools

echo "✅ All linting checks passed!"