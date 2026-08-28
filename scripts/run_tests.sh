#!/bin/bash
set -e

echo "=== Running EKA ID Backend Automated Tests ==="
docker run --rm -v "$(pwd)/services/api:/app" -w /app golang:1.23-alpine go test -v ./...

echo "=== All Tests Succeeded ==="