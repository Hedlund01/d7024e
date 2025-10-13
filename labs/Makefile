all: build
.PHONY: all build

BINARY_NAME := kadlab
BUILD_IMAGE ?= kadlab:latest
NUM_NODES ?= 2
VERSION := $(shell git rev-parse --short HEAD)
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

GOLDFLAGS += -X 'd7024e/pkg/main.BuildVersion=$(VERSION)'
GOLDFLAGS += -X 'd7024e/pkg/build.BuildTime=$(BUILDTIME)'

build:
	@CGO_ENABLED=0 go build -v -ldflags="-s -w $(GOLDFLAGS)" -o ./bin/$(BINARY_NAME) ./cmd/main.go

container:
	docker build -t $(BUILD_IMAGE) .

test: 
	go test -cover --race ./...

coverage:
	go test -coverprofile=coverage.out -covermode=atomic --race ./...
	@echo "Overall coverage:"
	go tool cover -func=coverage.out | grep total

# up takes one argument and uses the docker-compose file
up:
	export NUMBER_OF_REPLICAS=$(NUM_NODES); docker stack deploy --compose-file docker-compose.yml KademliaStack
	docker service logs -f KademliaStack_kademliaNodes --raw

down:
	docker stack rm KademliaStack