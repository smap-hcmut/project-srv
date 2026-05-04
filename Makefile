.PHONY: help run models swagger test clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

models: ## Generate models
	@echo "Generating models"
	@sqlboiler psql

swagger: ## Generate swagger docs
	@echo "Generating swagger"
	@swag init -g cmd/server/main.go --parseVendor
	@echo "Fixing swagger docs (removing deprecated LeftDelim/RightDelim)..."
	@sed -i '' '/LeftDelim:/d' docs/docs.go
	@sed -i '' '/RightDelim:/d' docs/docs.go

run: swagger ## Run the project service
	@echo "Running the application"
	@go run cmd/server/main.go

test: ## Run tests with coverage
	@echo "Running tests..."
	@go test -mod=readonly -coverprofile=coverage.out -failfast -timeout 5m ./internal/... ./pkg/...
	@grep -v 'mock_' coverage.out | grep -v 'internal/sqlboiler' | grep -v 'internal/httpserver' | grep -v 'internal/consumer' > c.out
	@GOFLAGS=-mod=readonly go tool cover -func=c.out
	@rm -f *.out

.DEFAULT_GOAL := help
