# Build the application
all: build test

build:
	@echo "Building..."
	@go build -o main cmd/cli/.

# Run the application
run:
	@go run ./cmd/cli/. pg --host localhost --port 5432 --username postgres --database postgres

playground:
	@go run cmd/playground/main.go

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Remove unused dep
tidy:
	@go mod tidy

db ?= "pg"
db-up:
	@docker compose -f docker-compose.$(db).yml up -d

db-down:
	@docker compose -f docker-compose.$(db).yml down --volumes

docker-nuke:
	docker system prune --all --force --volumes