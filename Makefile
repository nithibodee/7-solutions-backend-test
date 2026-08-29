.PHONY: help build run test test-integration cover lint mocks proto tidy docker-up docker-down

GOBIN ?= $(shell go env GOPATH)/bin

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

build: ## Build the server binary
	go build -o bin/server ./cmd/server

run: ## Run the server locally (needs a running MongoDB)
	go run ./cmd/server

test: ## Run unit tests
	go test ./... -race -count=1

test-integration: ## Run integration tests (needs MongoDB; MONGO_TEST_URI optional)
	go test -tags=integration ./internal/adapter/mongo/... -race -count=1

cover: ## Run tests with coverage report
	go test ./... -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | tail -1

lint: ## Run golangci-lint
	golangci-lint run

mocks: ## Regenerate mocks
	mockery

proto: ## Regenerate protobuf/gRPC code
	protoc \
		--go_out=. --go_opt=module=github.com/nithibodee/7-solutions-backend-test \
		--go-grpc_out=. --go-grpc_opt=module=github.com/nithibodee/7-solutions-backend-test \
		api/proto/user/v1/user.proto

tidy: ## go mod tidy
	go mod tidy

docker-up: ## Start API + MongoDB via docker compose
	docker compose up --build

docker-down: ## Stop and remove compose resources
	docker compose down -v
