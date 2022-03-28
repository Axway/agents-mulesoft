.PHONY: all clean lint build

WORKSPACE ?= $$(pwd)

GO_PKG_LIST := $(shell go list ./... | grep -v /vendor/ | grep -v *mock*.go)
SDK_VERSION := $(shell go list -m github.com/Axway/agent-sdk | cut -d ' ' -f 2 | cut -c 2-)

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

dep:
	@echo "Resolving go package dependencies"
	@go mod tidy
	@echo "Package dependencies completed"

dep-version:
	@export version=$(sdk) && make update-sdk && make dep

dep-sdk:
	@make sdk=main dep-version

update-sdk:
	@echo "Updating SDK dependencies"
	@export GOFLAGS="" && go get "github.com/Axway/agent-sdk@${version}"

sdk-version:
	@echo $(SDK_VERSION)

run-discovery:
	@go run ./cmd/discovery/main.go

run-trace:
	@go run ./cmd/traceability/main.go

build-discovery:
	@echo "building discovery agent with sdk version $(SDK_VERSION)"
	export CGO_ENABLED=0
	export TIME=`date +%Y%m%d%H%M%S`
	@go build \
		-ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=${TIME}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=${VERSION}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=${COMMIT_ID}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=${SDK_VERSION}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=ApigeeDiscoveryAgent'" \
		-o bin/discovery ./cmd/discovery/main.go
	@echo "discovery agent binary placed at bin/discovery"

build-trace:
	@echo "building traceability agent with sdk version $(SDK_VERSION)"
	export CGO_ENABLED=0
	export TIME=`date +%Y%m%d%H%M%S`
	@go build \
		-ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=${TIME}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=${VERSION}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=${COMMIT_ID}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=${SDK_VERSION}' \
			-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=ApigeeDiscoveryAgent'" \
		-o bin/traceability ./cmd/traceability/main.go
	@echo "traceability agent binary placed at bin/traceability"

build-trace-docker:
	@go build -o /app/traceability ./cmd/traceability/main.go

test:
	mkdir -p coverage
	@go test -race -short -count=1 -coverprofile=coverage/coverage.cov ${GO_PKG_LIST}

docker-build-disc:
	@docker build -t mulesoft_discovery_agent:latest -f ${WORKSPACE}/build/discovery.Dockerfile .
	@echo "Docker build complete"

docker-build-trace:
	@docker build -t mulesoft_traceability_agent:latest -f ${WORKSPACE}/build/traceability.Dockerfile .
	@echo "Docker build complete"
