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

# Start the development environment using the dev Docker Compose file
dev:
	echo "Generating Swagger docs"
	swag init --parseDependency
	echo "Starting development environment"
	docker compose -f docker-compose.dev.yml up

# Tear down the development environment, removing orphan containers
dev-down:
	echo "Stopping and removing development containers"
	docker compose -f docker-compose.dev.yml down --remove-orphans
	echo "To remove all containers, volumes, and networks, use --volumes"
