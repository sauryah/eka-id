# Contributing to EKA ID

We welcome contributions to the EKA ID universal digital identity platform.

## Development Workflow
1. Run automated backend tests:
   ```bash
   docker run --rm -v "${PWD}/services/api:/app" -w /app golang:1.23-alpine go test -v ./...
   ```
2. Run frontend lint & build:
   ```bash
   cd apps/web && npm run build
   ```
3. Submit Pull Requests with unit test coverage.