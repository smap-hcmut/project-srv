.PHONY: help dev-up dev-down dev-logs dev-clean build run test migrate-up migrate-down

# Colors
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)

help: ## Show this help
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${WHITE}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

## Development
dev-up: ## Start all development dependencies (PostgreSQL, Redis, Kafka)
	@echo "${GREEN}Starting development dependencies...${RESET}"
	docker-compose up -d
	@echo "${GREEN}Waiting for services to be healthy...${RESET}"
	@sleep 5
	@echo "${GREEN}Services started successfully!${RESET}"
	@echo "${YELLOW}PostgreSQL:${RESET} localhost:5432"
	@echo "${YELLOW}Redis:${RESET} localhost:6379"
	@echo "${YELLOW}Kafka:${RESET} localhost:9092"
	@echo "${YELLOW}Kafka UI:${RESET} http://localhost:8090"
	@echo "${YELLOW}Redis Commander:${RESET} http://localhost:8081"

dev-down: ## Stop all development dependencies
	@echo "${GREEN}Stopping development dependencies...${RESET}"
	docker-compose down

dev-logs: ## Show logs from all services
	docker-compose logs -f

dev-clean: ## Stop and remove all containers, volumes, and networks
	@echo "${YELLOW}Warning: This will remove all data!${RESET}"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		echo "${GREEN}Cleaned up successfully!${RESET}"; \
	fi

dev-restart: dev-down dev-up ## Restart all development dependencies

## Database
migrate-up: ## Run database migrations
	@echo "${GREEN}Running migrations...${RESET}"
	@if [ -f migration/init_schema.sql ]; then \
		PGPASSWORD=postgres psql -h localhost -U postgres -d smap -f migration/init_schema.sql; \
		echo "${GREEN}Migrations completed!${RESET}"; \
	else \
		echo "${YELLOW}No migration files found${RESET}"; \
	fi

migrate-down: ## Rollback database migrations
	@echo "${YELLOW}Rolling back migrations...${RESET}"
	@echo "Not implemented yet"

db-shell: ## Connect to PostgreSQL shell
	PGPASSWORD=postgres psql -h localhost -U postgres -d smap

## Build & Run
build: ## Build the application
	@echo "${GREEN}Building application...${RESET}"
	go build -o bin/api cmd/api/main.go
	go build -o bin/consumer cmd/consumer/main.go
	@echo "${GREEN}Build completed!${RESET}"

run-api: ## Run API server
# 	@echo "${GREEN}Starting API server...${RESET}"
	go run cmd/api/main.go

run-consumer: ## Run consumer service
	@echo "${GREEN}Starting consumer service...${RESET}"
	go run cmd/consumer/main.go

## Testing
test: ## Run tests
	@echo "${GREEN}Running tests...${RESET}"
	go test -v -race -coverprofile=coverage.out ./...
	@echo "${GREEN}Tests completed!${RESET}"

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out

## Code Quality
lint: ## Run linter
	@echo "${GREEN}Running linter...${RESET}"
	golangci-lint run

fmt: ## Format code
	@echo "${GREEN}Formatting code...${RESET}"
	go fmt ./...
	goimports -w .

## Dependencies
deps: ## Download dependencies
	@echo "${GREEN}Downloading dependencies...${RESET}"
	go mod download
	go mod tidy

deps-update: ## Update dependencies
	@echo "${GREEN}Updating dependencies...${RESET}"
	go get -u ./...
	go mod tidy

## Docker
docker-build: ## Build Docker image
	@echo "${GREEN}Building Docker image...${RESET}"
	docker build -t project-srv:latest -f cmd/api/Dockerfile .

docker-run: ## Run Docker container
	@echo "${GREEN}Running Docker container...${RESET}"
	docker run -p 8080:8080 \
		-e POSTGRES_HOST=localhost \
		-e POSTGRES_PASSWORD=postgres \
		-e REDIS_HOST=localhost \
		-e JWT_SECRET_KEY=your-secret-key \
		project-srv:latest

## Kafka
kafka-topics: ## List Kafka topics
	docker exec -it project-kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-create-topic: ## Create Kafka topic (usage: make kafka-create-topic TOPIC=my-topic)
	docker exec -it project-kafka kafka-topics --bootstrap-server localhost:9092 --create --topic $(TOPIC) --partitions 3 --replication-factor 1

kafka-consume: ## Consume messages from Kafka topic (usage: make kafka-consume TOPIC=project.events)
	docker exec -it project-kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic $(TOPIC) --from-beginning

kafka-produce: ## Produce messages to Kafka topic (usage: make kafka-produce TOPIC=project.events)
	docker exec -it project-kafka kafka-console-producer --bootstrap-server localhost:9092 --topic $(TOPIC)

## Redis
redis-cli: ## Connect to Redis CLI
	docker exec -it project-redis redis-cli -a redis_password

redis-flush: ## Flush all Redis data
	@echo "${YELLOW}Warning: This will delete all Redis data!${RESET}"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -it project-redis redis-cli -a redis_password FLUSHALL; \
		echo "${GREEN}Redis flushed!${RESET}"; \
	fi

## Monitoring
health: ## Check health of all services
	@echo "${GREEN}Checking service health...${RESET}"
	@echo "${YELLOW}PostgreSQL:${RESET}"
	@docker exec project-postgres pg_isready -U postgres || echo "Not ready"
	@echo "${YELLOW}Redis:${RESET}"
	@docker exec project-redis redis-cli -a redis_password ping || echo "Not ready"
	@echo "${YELLOW}Kafka:${RESET}"
	@docker exec project-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1 && echo "PONG" || echo "Not ready"

logs-api: ## Show API logs
	docker logs -f project-api 2>/dev/null || echo "API container not running"

logs-consumer: ## Show consumer logs
	docker logs -f project-consumer 2>/dev/null || echo "Consumer container not running"

## Cleanup
clean: ## Clean build artifacts
	@echo "${GREEN}Cleaning build artifacts...${RESET}"
	rm -rf bin/
	rm -f coverage.out
	@echo "${GREEN}Cleaned!${RESET}"

clean-all: clean dev-clean ## Clean everything (artifacts + Docker)

.DEFAULT_GOAL := help
