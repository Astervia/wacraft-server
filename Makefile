.DEFAULT_GOAL := build

.PHONY: help build compile prod prod-down dev dev-distributed dev-scaled dev-down dev-migrate-down dev-migrate-to dev-migrate-status test test-memory test-distributed

DC_PROD := docker compose
DC_DEV := docker compose -f docker-compose.dev.yml
TEST_REDIS_IMG := redis:7-alpine
TEST_PG_IMG := postgres:17-alpine
GO_TEST_FLAGS :=

ifeq ($(NO_CACHE),1)
GO_TEST_FLAGS += -count=1
endif

help: ## Show this interactive help map
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the project for the current OS and architecture (typically for local development)
	@printf "\033[36m==> Generating Swagger docs\033[0m\n"
	swag init --parseDependency
	@printf "\033[36m==> Compiling project for your local OS/architecture\033[0m\n"
	go build -o ./bin/wacraft-server

compile: ## Cross-compile the project for multiple OS/architecture targets
	@printf "\033[36m==> Generating Swagger docs\033[0m\n"
	swag init --parseDependency
	@printf "\033[36m==> Compiling project for Linux ARM\033[0m\n"
	GOOS=linux GOARCH=arm go build -o ./bin/wacraft-server-linux-arm
	@printf "\033[36m==> Compiling project for Linux ARM64\033[0m\n"
	GOOS=linux GOARCH=arm64 go build -o ./bin/wacraft-server-linux-arm64
	@printf "\033[36m==> Compiling project for Windows 32-bit\033[0m\n"
	GOOS=windows GOARCH=386 go build -o ./bin/wacraft-server-windows-386
	@printf "\033[36m==> Compiling project for Windows ARM64\033[0m\n"
	GOOS=windows GOARCH=arm64 go build -o ./bin/wacraft-server-windows-arm64

prod: ## Start the production environment using Docker Compose
	@printf "\033[36m==> Starting production Docker containers\033[0m\n"
	$(DC_PROD) up

prod-down: ## Tear down the production Docker environment, removing orphan containers
	@printf "\033[36m==> Stopping and removing production containers\033[0m\n"
	$(DC_PROD) down --remove-orphans
	@printf "\033[36m==> To remove all containers, volumes, and networks, use --volumes\033[0m\n"

# Start the development environment using the dev Docker Compose file.
# Usage:
#   make dev                                        # memory mode, 1 instance
#   make dev PROFILE=distributed                    # Redis mode, 1 instance
#   make dev PROFILE=distributed REPLICAS=3         # Redis mode, 3 instances behind nginx
dev: ## Start the development environment using the dev Docker Compose file
	@printf "\033[36m==> Generating Swagger docs\033[0m\n"
	swag init --parseDependency
	@printf "\033[36m==> Starting development environment (replicas=$(or $(REPLICAS),1))\033[0m\n"
	APP_REPLICAS=$(or $(REPLICAS),1) SYNC_BACKEND=$${SYNC_BACKEND:-memory} $(DC_DEV) $(if $(PROFILE),--profile $(PROFILE)) up

# Shorthand for distributed mode with optional replica count.
# Usage:
#   make dev-distributed                # 1 instance
#   make dev-distributed REPLICAS=3    # 3 instances behind nginx
dev-distributed: ## Shorthand for distributed mode with optional replica count
	SYNC_BACKEND=redis $(MAKE) dev PROFILE=distributed REPLICAS=$(or $(REPLICAS),1)

# Shorthand for running multiple scaled replicas in distributed mode.
# Usage:
#   make dev-scaled            # 3 instances (default)
#   make dev-scaled REPLICAS=5 # 5 instances
dev-scaled: ## Shorthand for running multiple scaled replicas in distributed mode
	$(MAKE) dev-distributed REPLICAS=$(or $(REPLICAS),3)

dev-down: ## Tear down the development environment, removing orphan containers
	@printf "\033[36m==> Stopping and removing development containers\033[0m\n"
	$(DC_DEV) down --remove-orphans
	@printf "\033[36m==> To remove all containers, volumes, and networks, use --volumes\033[0m\n"

dev-migrate-down: ## Roll back the last migration in development
	@printf "\033[36m==> Rolling back last migration\033[0m\n"
	$(DC_DEV) exec app go run main.go migrate:down

dev-migrate-to: ## Roll back migrations to a specific version in development
	@if [ -z "$(VERSION)" ]; then \
		printf "\033[31m==> Error: VERSION is required. Usage: make dev-migrate-to VERSION=20240625233555\033[0m\n"; \
		exit 1; \
	fi
	@printf "\033[36m==> Rolling back to migration version $(VERSION)\033[0m\n"
	$(DC_DEV) exec app go run main.go migrate:down-to $(VERSION)

dev-migrate-status: ## Check migration status in development
	@printf "\033[36m==> Checking migration status\033[0m\n"
	$(DC_DEV) exec app go run main.go migrate:status

test: ## Run unit tests (no external dependencies)
	go test ./... -v -p 1 $(GO_TEST_FLAGS)

test-memory: ## Run tests in memory mode (Postgres only, no Redis)
	@printf "\033[36m==> Starting ephemeral PostgreSQL container...\033[0m\n"
	docker run -d --name wacraft-test-postgres-mem -p 15433:5432 \
		-e POSTGRES_DB=postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		$(TEST_PG_IMG) > /dev/null
	@printf "\033[36m==> Waiting for PostgreSQL to be ready...\033[0m\n"
	until docker exec wacraft-test-postgres-mem pg_isready -U postgres -q 2>/dev/null; do sleep 0.1; done
	@printf "\033[36m==> Running tests (memory mode)...\033[0m\n"
	SYNC_BACKEND=memory \
		REDIS_URL="" \
		DATABASE_URL="postgres://postgres:postgres@localhost:15433/postgres?sslmode=disable" \
		go test ./... -v -race -p 1 $(GO_TEST_FLAGS); \
		EXIT=$$?; \
		printf "\033[36m==> Removing PostgreSQL container...\033[0m\n"; \
		docker rm -f wacraft-test-postgres-mem > /dev/null; \
		exit $$EXIT

test-distributed: ## Run the full test suite with both Redis and PostgreSQL
	@printf "\033[36m==> Starting ephemeral Redis container...\033[0m\n"
	docker run -d --name wacraft-test-redis -p 16379:6379 $(TEST_REDIS_IMG) > /dev/null
	@printf "\033[36m==> Starting ephemeral PostgreSQL container...\033[0m\n"
	docker run -d --name wacraft-test-postgres -p 15432:5432 \
		-e POSTGRES_DB=postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		$(TEST_PG_IMG) > /dev/null
	@printf "\033[36m==> Waiting for Redis to be ready...\033[0m\n"
	until docker exec wacraft-test-redis redis-cli ping 2>/dev/null | grep -q PONG; do sleep 0.1; done
	@printf "\033[36m==> Waiting for PostgreSQL to be ready...\033[0m\n"
	until docker exec wacraft-test-postgres pg_isready -U postgres -q 2>/dev/null; do sleep 0.1; done
	@printf "\033[36m==> Running tests (distributed mode)...\033[0m\n"
	SYNC_BACKEND=redis \
		REDIS_URL=redis://localhost:16379 \
		DATABASE_URL="postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable" \
		go test ./... -v -race -p 1 $(GO_TEST_FLAGS); \
		EXIT=$$?; \
		printf "\033[36m==> Removing containers...\033[0m\n"; \
		docker rm -f wacraft-test-redis wacraft-test-postgres > /dev/null; \
		exit $$EXIT
