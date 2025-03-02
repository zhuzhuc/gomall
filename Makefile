.PHONY: all build test clean deps proto user-service

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
PROTOC=protoc

# Directories
PROTO_DIR=idl
SERVICE_DIR=internal/service
CMD_DIR=cmd

# Build targets
all: deps proto build test

# Install dependencies
deps:
	$(GOMOD) tidy
	$(GOMOD) download

# Generate protobuf files
proto:
	$(PROTOC) --go_out=. --go-grpc_out=. $(PROTO_DIR)/*.proto

# Build user service
user-service:
	$(GOBUILD) -o $(CMD_DIR)/user-service/main $(CMD_DIR)/user-service/main.go

# Run tests
test:
	$(GOTEST) ./... -v

# Run integration tests
integration-test:
	$(GOTEST) -tags=integration ./... -v

# Build Docker images
docker-build:
	docker build -t user-service -f Dockerfile.user .

# Clean build artifacts
clean:
	rm -f $(CMD_DIR)/user-service/main
	$(GOCMD) clean -cache

# Run user service locally
run-user-service:
	$(GOBUILD) -o $(CMD_DIR)/user-service/main $(CMD_DIR)/user-service/main.go
	$(CMD_DIR)/user-service/main

# Docker Compose commands
docker-up:
	docker-compose up --build

docker-down:
	docker-compose down

# Help target
help:
	@echo "Available targets:"
	@echo "  all            - Install deps, generate proto, build, and test"
	@echo "  deps           - Install Go dependencies"
	@echo "  proto          - Generate protobuf files"
	@echo "  user-service   - Build user service"
	@echo "  test           - Run unit tests"
	@echo "  integration-test - Run integration tests"
	@echo "  docker-build   - Build Docker image"
	@echo "  clean          - Clean build artifacts"
	@echo "  run-user-service - Build and run user service locally"
	@echo "  docker-up      - Start services with Docker Compose"
	@echo "  docker-down    - Stop services with Docker Compose"
