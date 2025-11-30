.PHONY: help build test clean run-api run-cli install dev

help:
	@echo "Available targets:"
	@echo "  make build      - Build the CLI binary"
	@echo "  make test       - Run all tests"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make run-api    - Run the FastAPI backend"
	@echo "  make run-cli    - Run the CLI (requires args: ARGS='...')"
	@echo "  make install    - Install dependencies"
	@echo "  make dev        - Set up development environment"

build:
	@echo "Building CLI..."
	go build -o bin/chunk ./cmd/chunk

test:
	@echo "Running Go tests..."
	go test ./... -v
	@echo "Running Python tests..."
	cd api && uv run pytest

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf api/.pytest_cache
	rm -rf api/__pycache__
	go clean

run-api:
	@echo "Starting FastAPI server..."
	cd api && uv run uvicorn main:app --reload

run-cli:
	@echo "Running CLI..."
	go run ./cmd/chunk $(ARGS)

install:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing Python dependencies..."
	cd api && uv sync

dev: install
	@echo "Development environment ready!"
	@echo "Run 'make run-api' to start the API server"
	@echo "Run 'make build' to build the CLI"
