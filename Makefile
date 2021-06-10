.PHONY: all clean lint build

WORKSPACE ?= $$(pwd)

GO_PKG_LIST := $(shell go list ./... | grep -v /vendor/ | grep -v *mock*.go)

download:
	@go mod tidy && go mod download

verify:
	@go mod verify

all: clean build

clean:
	@rm -rf dist

lint:
	@golint -set_exit_status ${GO_PKG_LIST}

format:
	@gofmt -w .
	@goimports -w .

update-sdk:
	@echo "Updating SDK dependencies"
	@export GOFLAGS="" && go get "github.com/Axway/agent-sdk@main"

run-discovery:
	@go run ./cmd/discovery/main.go

run-trace:
	@go run ./cmd/traceability/main.go

build-discovery:
	@go build -o bin/discovery ./cmd/discovery/main.go

build-trace:
	@go build -o /app/traceability ./cmd/traceability/main.go

test:
	mkdir -p coverage
	@go test -race -short -count=1 -coverprofile=coverage/coverage.cov ${GO_PKG_LIST}
