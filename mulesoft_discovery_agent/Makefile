.PHONY: all clean lint build 

WORKSPACE ?= $$(pwd)

GO_PKG_LIST := $(shell go list ./... | grep -v /vendor/)

all: clean build

clean:
	@rm -rf dist

lint:
	@golint -set_exit_status ${GO_PKG_LIST}


update-sdk:
	@echo "Updating SDK dependencies"
	@export GOFLAGS="" && go get "github.com/Axway/agent-sdk@main"


${WORKSPACE}/mulesoft_discovery_agent: lint
	@go build -tags static_all -a -o ./bin/mulesoft_discovery_agent .
	@go test ./...

build:${WORKSPACE}/mulesoft_discovery_agent
	@echo "Build complete"