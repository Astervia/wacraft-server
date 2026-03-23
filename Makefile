# Build the project for the current OS and architecture (typically for local development)
build:
	echo "Generating Swagger docs"
	swag init --parseDependency
	echo "Compiling project for your local OS/architecture"
	go build -o ./bin/wacraft-server

# Cross-compile the project for multiple OS/architecture targets
compile:
	echo "Generating Swagger docs"
	swag init --parseDependency
	echo "Compiling project for Linux ARM"
	GOOS=linux GOARCH=arm go build -o ./bin/wacraft-server-linux-arm
	echo "Compiling project for Linux ARM64"
	GOOS=linux GOARCH=arm64 go build -o ./bin/wacraft-server-linux-arm64
	echo "Compiling project for Windows 32-bit"
	GOOS=windows GOARCH=386 go build -o ./bin/wacraft-server-windows-386
	echo "Compiling project for Windows ARM64"
	GOOS=windows GOARCH=arm64 go build -o ./bin/wacraft-server-windows-arm64

# Start the production environment using Docker Compose
prod:
	echo "Starting production Docker containers"
	docker compose up

# Tear down the production Docker environment, removing orphan containers
prod-down:
	echo "Stopping and removing production containers"
	docker compose down --remove-orphans
	echo "To remove all containers, volumes, and networks, use --volumes"

# Start the development environment using the dev Docker Compose file.
# Usage:
#   make dev                                        # memory mode, 1 instance
#   make dev PROFILE=distributed                    # Redis mode, 1 instance
#   make dev PROFILE=distributed REPLICAS=3         # Redis mode, 3 instances behind nginx
dev:
	echo "Generating Swagger docs"
	swag init --parseDependency
	echo "Starting development environment (replicas=$(or $(REPLICAS),1))"
	APP_REPLICAS=$(or $(REPLICAS),1) SYNC_BACKEND=$${SYNC_BACKEND:-memory} docker compose -f docker-compose.dev.yml $(if $(PROFILE),--profile $(PROFILE)) up

# Shorthand for distributed mode with optional replica count.
# Usage:
#   make dev-distributed                # 1 instance
#   make dev-distributed REPLICAS=3    # 3 instances behind nginx
dev-distributed:
	SYNC_BACKEND=redis $(MAKE) dev PROFILE=distributed REPLICAS=$(or $(REPLICAS),1)

# Shorthand for running multiple scaled replicas in distributed mode.
# Usage:
#   make dev-scaled            # 3 instances (default)
#   make dev-scaled REPLICAS=5 # 5 instances
dev-scaled:
	$(MAKE) dev-distributed REPLICAS=$(or $(REPLICAS),3)

# Tear down the development environment, removing orphan containers
dev-down:
	echo "Stopping and removing development containers"
	docker compose -f docker-compose.dev.yml down --remove-orphans
	echo "To remove all containers, volumes, and networks, use --volumes"

# Roll back the last migration in development
dev-migrate-down:
	echo "Rolling back last migration"
	docker compose -f docker-compose.dev.yml exec app go run main.go migrate:down

# Roll back migrations to a specific version in development (usage: make dev-migrate-to VERSION=20240625233555)
dev-migrate-to:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make dev-migrate-to VERSION=20240625233555"; \
		exit 1; \
	fi
	echo "Rolling back to migration version $(VERSION)"
	docker compose -f docker-compose.dev.yml exec app go run main.go migrate:down-to $(VERSION)

# Check migration status in development
dev-migrate-status:
	echo "Checking migration status"
	docker compose -f docker-compose.dev.yml exec app go run main.go migrate:status

# Run unit tests (no external dependencies)
test:
	go test ./... -v
	cd wacraft-core && go test ./... -v

# Run all tests including Redis integration tests (wacraft-core only, no Postgres).
# Starts an ephemeral Redis container, runs tests, then removes it.
test-redis:
	@echo "Starting ephemeral Redis container..."
	@docker run -d --name wacraft-test-redis -p 16379:6379 redis:7-alpine > /dev/null
	@echo "Waiting for Redis to be ready..."
	@until docker exec wacraft-test-redis redis-cli ping 2>/dev/null | grep -q PONG; do sleep 0.1; done
	@echo "Running tests..."
	@REDIS_URL=redis://localhost:16379 go test ./... -v -race; \
		CORE_EXIT=0; \
		cd wacraft-core && REDIS_URL=redis://localhost:16379 go test ./... -v -race || CORE_EXIT=$$?; \
		cd ..; \
		echo "Removing Redis container..."; \
		docker rm -f wacraft-test-redis > /dev/null; \
		exit $$CORE_EXIT

# Run tests in memory mode (Postgres only, no Redis).
# Mirrors the CI "Run tests (memory mode)" step.
# Starts an ephemeral PostgreSQL container, runs all tests (server + core), then removes it.
test-memory:
	@echo "Starting ephemeral PostgreSQL container..."
	@docker run -d --name wacraft-test-postgres-mem -p 15433:5432 \
		-e POSTGRES_DB=postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		postgres:17-alpine > /dev/null
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec wacraft-test-postgres-mem pg_isready -U postgres -q 2>/dev/null; do sleep 0.1; done
	@echo "Running tests (memory mode)..."
	@SERVER_EXIT=0; CORE_EXIT=0; \
		SYNC_BACKEND=memory \
		REDIS_URL="" \
		DATABASE_URL="postgres://postgres:postgres@localhost:15433/postgres?sslmode=disable" \
		go test ./... -v -race || SERVER_EXIT=$$?; \
		cd wacraft-core && REDIS_URL="" go test ./... -v -race || CORE_EXIT=$$?; \
		cd ..; \
		echo "Removing PostgreSQL container..."; \
		docker rm -f wacraft-test-postgres-mem > /dev/null; \
		exit $$(( SERVER_EXIT || CORE_EXIT ))

# Run the full test suite with both Redis and PostgreSQL.
# Starts ephemeral containers for both services, runs all tests (server + core), then removes them.
# Exit code is non-zero if any test fails in either module.
test-distributed:
	@echo "Starting ephemeral Redis container..."
	@docker run -d --name wacraft-test-redis -p 16379:6379 redis:7-alpine > /dev/null
	@echo "Starting ephemeral PostgreSQL container..."
	@docker run -d --name wacraft-test-postgres -p 15432:5432 \
		-e POSTGRES_DB=postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		postgres:17-alpine > /dev/null
	@echo "Waiting for Redis to be ready..."
	@until docker exec wacraft-test-redis redis-cli ping 2>/dev/null | grep -q PONG; do sleep 0.1; done
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec wacraft-test-postgres pg_isready -U postgres -q 2>/dev/null; do sleep 0.1; done
	@echo "Running tests (distributed mode)..."
	@SERVER_EXIT=0; CORE_EXIT=0; \
		SYNC_BACKEND=redis \
		REDIS_URL=redis://localhost:16379 \
		DATABASE_URL="postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable" \
		go test ./... -v -race || SERVER_EXIT=$$?; \
		cd wacraft-core && REDIS_URL=redis://localhost:16379 go test ./... -v -race || CORE_EXIT=$$?; \
		cd ..; \
		echo "Removing containers..."; \
		docker rm -f wacraft-test-redis wacraft-test-postgres > /dev/null; \
		exit $$(( SERVER_EXIT || CORE_EXIT ))
