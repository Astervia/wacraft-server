# 1️⃣ Base image for building Go application
FROM golang:1.23 AS builder

# Set working directory
WORKDIR /app

# Ensure necessary directories exist
RUN mkdir -p /app

# Enable SSH authentication for private repositories
ENV GOPRIVATE=github.com/Rfluid,github.com/Astervia
RUN mkdir -p ~/.ssh && chmod 700 ~/.ssh
RUN ssh-keyscan github.com >> ~/.ssh/known_hosts

# Force Go to use SSH instead of HTTPS for private repositories
RUN git config --global url."git@github.com:".insteadOf "https://github.com/"

# Copy Go dependencies separately for better caching
COPY go.mod go.sum /app/

# Use SSH agent forwarding to authenticate private repos
# 🚀 This allows Go modules to be downloaded without copying SSH keys
RUN --mount=type=ssh go mod download

# Copy the rest of the application code
COPY . /app/

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Build the Go application
RUN go build -o /bin/wacraft-server ./main.go

# 2️⃣ Final lightweight image
FROM alpine:latest

# Set working directory inside the container
WORKDIR /root/

# Copy only the compiled binary from the builder stage
COPY --from=builder /bin/wacraft-server .
RUN chmod +x wacraft-server
# 📄 Copy the Swagger docs
COPY --from=builder /app/docs /root/docs
# Copy migration folders
COPY --from=builder /app/src/database/migrations /root/src/database/migrations
COPY --from=builder /app/src/database/migrations-before /root/src/database/migrations-before

# Expose application port
EXPOSE 6900

# Run the Go server
CMD ["./wacraft-server"]
