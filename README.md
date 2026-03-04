# wacraft-server

This is the backend server for the **[wacraft project](https://github.com/Astervia/wacraft)** — a development platform for the WhatsApp Cloud API.

With **wacraft**, you can send and receive WhatsApp messages, handle webhooks, and perform a wide variety of operations using a consistent and extensible API.

For details on client usage, see:

- 🔗 [wacraft repository](https://github.com/Astervia/wacraft): full-featured open-source platform.

> ℹ️ Note: The `wacraft-lite` repository is now legacy, as all features are fully open-source in the main repository.

This `README.md` focuses on the server (this repo).

## ✨ What's New in v0.2.x

wacraft v0.2.x introduces major architectural upgrades:

- **Multitenancy & Workspaces**: The server now securely isolates data across multiple tenants. Within each tenant, you can create workspaces to organize your teams or projects seamlessly.
- **Multiple Phone Number Configurations**: A single deployment can now manage and route messages for multiple WhatsApp Cloud API phone numbers simultaneously, rather than being limited to just one.
- **n8n Integrations**: Built-in support for sending WhatsApp Cloud API events directly to n8n, enabling powerful visual workflow automation. For integration nodes, check out [n8n-nodes-wacraft](https://github.com/Astervia/n8n-nodes-wacraft).

*Note: Due to these changes, WhatsApp API credentials and associated webhooks are no longer set via global environment variables. They are now configured per-workspace via our API or client interface, providing maximum flexibility.*

## 🧪 Getting Started

### 📦 Environment Variables

Create your `.env` file:

```bash
cp .env.local .env
```

Fill in your database connection, server secrets, and other required values. Descriptions for each variable are included in `.env.local`.

> ⚠️ **Note for v0.2.x:** WhatsApp Cloud API credentials are now configured per-workspace via the API, rather than globally in the `.env` file.
> 
> ⚠️ **Don't skip variables or remove them unless you're sure.**

### 🐳 Running with Docker (Recommended)

Use the `Makefile`:

```bash
make dev  # Development mode (with live reload)
make prod # Production mode
```

- `docker-compose.dev.yml`: optimized for development, includes hot-reloading.
- `docker-compose.yml`: for production, always pulls the latest `wacraft-server`.

> ℹ️ Tip: Check the `Makefile`, `Dockerfile`, and `docker-compose` files to customize the behavior as needed.

> ℹ️ Tip: If you get any SSH errors when running the compose files, hit

    ```bash
    eval "$(ssh-agent -s)" && ssh-add ~/.ssh/id_rsa
    ```
    or the equivalent for your SSH key.

### 🔁 Running with Live Reload

Install [air](https://github.com/cosmtrek/air) and run:

```bash
air
```

### 🏃 Running with `go run`

```bash
go run main.go
```

### ⚙️ Running the Compiled Executable

```bash
make build
./bin/wacraft-server
```

## 📘 OpenAPI Documentation

API documentation is automatically updated when you run:

```bash
make build
```

Or manually via:

```bash
swag init --parseDependency
```

> 🛠 You’ll need [`swaggo`](https://github.com/swaggo/swag) installed.

## 🧬 Database Migrations

We use GORM for automatic schema generation and structure migrations in two stages:

### 1. **Before GORM Auto-Migrations**

```bash
goose -dir src/database/migrations-before create migration_name go   # Go-based
goose -dir src/database/migrations-before create migration_name sql  # SQL-based
```

### 2. **After GORM Auto-Migrations**

```bash
goose -dir src/database/migrations create migration_name go   # Go-based
goose -dir src/database/migrations create migration_name sql  # SQL-based
```

> 🐤 Migration tool: [`pressly/goose`](https://github.com/pressly/goose)

## 🐋 Docker Image

### 🔨 Build Instructions

To support private modules when building the Docker image, use SSH forwarding:

```bash
docker build --ssh default -t wacraft-server:latest -f Dockerfile .
```

## Meta authentication

### Verification Requests

Set the environment variable `WEBHOOK_VERIFY_TOKEN` and configure your webhook URL in the Meta developer console with this variable as the verify token. If you don't provide this environment variable, no authentication will be applied at Meta's Verification Requests.

### Event Notifications

We authenticate event notification webhooks from Meta using the App Secret and the `X-Hub-Signature-256` header. If you set the environment variable `META_APP_SECRET`, the server will verify the signature of incoming webhooks. If the signature is invalid, the server will return a 403 Forbidden response. If you don't set the environment variable, the server will not verify the signature and will accept all incoming webhooks which is not recommended for production environments.

> ℹ️ Tip: Make sure that you are passing the correct headers if using a reverse proxy like Nginx or Traefik or an AWS Load Balancer. The `X-Hub-Signature-256` header must be passed to the backend server.

## 🚀 Legacy Lite Version

We previously maintained a separate repository for a [lite version of wacraft-server](https://github.com/Astervia/wacraft-server-lite). **This is now legacy.** The full backend, including all formerly premium-only features, is now completely open-source and available in this repository.
