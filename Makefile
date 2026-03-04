.PHONY: build run debug demo test clean deps generate docker-build docker-run docker-start docker-stop docker-restart docker-demo-build docker-demo-run

ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

# Go

deps:
	@echo "Installing Go dependencies..."
	@go install github.com/a-h/templ/cmd/templ@latest
	@go mod download

generate:
	@$$(go env GOPATH)/bin/templ generate

build: deps
	@echo "Generating templ files..."
	@$$(go env GOPATH)/bin/templ generate
	@echo "Building jumpgate..."
	@mkdir -p $(ROOT_DIR)/bin
	@go build -o bin/jumpgate ./cmd/jumpgate
	@echo "✓ Built bin/jumpgate"
	@echo "Building jumpgate-cli..."
	@go build -o bin/jumpgate-cli ./cmd/jumpgate-cli
	@echo "✓ Built bin/jumpgate-cli"

run: build
	@mkdir -p $(ROOT_DIR)/data
	@echo "Starting jumpgate server on :8080..."
	@$(ROOT_DIR)/bin/jumpgate server

debug: build
	@mkdir -p $(ROOT_DIR)/data
	@echo "Starting jumpgate server on :8080 (auth disabled)..."
	@$(ROOT_DIR)/bin/jumpgate server --config $(ROOT_DIR)/debug.yaml

demo: build
	@echo "Starting jumpgate demo server on :8080..."
	@$(ROOT_DIR)/bin/jumpgate server --config $(ROOT_DIR)/demo.yaml

demo-slow: build
	@echo "Starting jumpgate demo-slow server on :8080..."
	@$(ROOT_DIR)/bin/jumpgate server --config $(ROOT_DIR)/demo-slow.yaml

test: generate
	@echo "Running Go tests..."
	@go test ./...

clean:
	rm -rf bin

# Docker

IMAGE_NAME = jumpgate
CONTAINER_NAME = jumpgate
HOST_DATA_DIR = $(ROOT_DIR)/data
DOCKER_DATA_MOUNT = -v $(HOST_DATA_DIR):/app/data

docker-build:
	docker build -t $(IMAGE_NAME) .

docker-run: docker-build
	mkdir -p $(HOST_DATA_DIR)
	docker run --rm \
		--name $(CONTAINER_NAME) \
		-p 8080:8080 \
		$(DOCKER_DATA_MOUNT) \
		$(IMAGE_NAME)

docker-start: docker-build
	mkdir -p $(HOST_DATA_DIR)
	docker run -d \
		--name $(CONTAINER_NAME) \
		-p 8080:8080 \
		$(DOCKER_DATA_MOUNT) \
		$(IMAGE_NAME)

docker-stop:
	docker stop $(CONTAINER_NAME) || true
	docker rm $(CONTAINER_NAME) || true

docker-restart: docker-stop docker-start

# Docker (demo)

DEMO_IMAGE_NAME = jumpgate-demo
DEMO_CONTAINER_NAME = jumpgate-demo

docker-demo-build:
	docker build -t $(DEMO_IMAGE_NAME) -f Dockerfile.demo .

docker-demo-run: docker-demo-build
	docker run --rm \
		--name $(DEMO_CONTAINER_NAME) \
		-p 8080:8080 \
		$(DEMO_IMAGE_NAME)
