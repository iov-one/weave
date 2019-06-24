.PHONY: all dist install test tf cover lint protofmt protoc protodocs novendor

# make sure we turn on go modules
export GO111MODULE := on

TOOLS := cmd/bnsd cmd/bnscli

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= set

# Check if linter exists
LINT := $(shell command -v golangci-lint 2> /dev/null)

# for dockerized prototool
USER := $(shell id -u):$(shell id -g)
DOCKER_BASE := docker run --rm -v $(shell pwd):/work iov1/prototool:v0.2.2
PROTOTOOL := $(DOCKER_BASE) prototool
PROTOC := $(DOCKER_BASE) protoc


all: test lint

dist:
	cd cmd/bnsd && $(MAKE) dist

install:
	for ex in $(TOOLS); do cd $$ex && make install && cd -; done

test:
	@# bnscli binary is required by some tests. In order to not skip them, ensure bnscli binary is provided and in the latest version.
	go install -mod=readonly ./cmd/bnscli

	go vet -mod=readonly  ./...
	go test -mod=readonly -race ./...

lint:
	@go mod vendor
	docker run --rm -it -v $(shell pwd):/go/src/github.com/iov-one/weave -w="/go/src/github.com/iov-one/weave" golangci/golangci-lint:v1.17.1 golangci-lint run ./...
	@rm -rf vendor

# Test fast
tf:
	go test -short ./...

cover:
	@ go test -mod=readonly -covermode=$(MODE) -coverprofile=coverage/allpackages.out ./...
	@ go test -mod=readonly -covermode=$(MODE) \
		-coverpkg=github.com/iov-one/weave/cmd/bnsd/app,github.com/iov-one/weave/cmd/bnsd/client,github.com/iov-one/weave/app \
		-coverprofile=coverage/bnsd_scenarios.out \
		github.com/iov-one/weave/cmd/bnsd/scenarios
	@ go test -mod=readonly -covermode=$(MODE) \
		-coverpkg=github.com/iov-one/weave/cmd/bnsd/app,github.com/iov-one/weave/cmd/bnsd/client,github.com/iov-one/weave/app \
		-coverprofile=coverage/bnsd_app.out \
		github.com/iov-one/weave/cmd/bnsd/app
	@ go test -mod=readonly -covermode=$(MODE) \
		-coverpkg=github.com/iov-one/weave/cmd/bnsd/app,github.com/iov-one/weave/cmd/bnsd/client,github.com/iov-one/weave/app \
		-coverprofile=coverage/bnsd_client.out \
		github.com/iov-one/weave/cmd/bnsd/client
	cat coverage/*.out > coverage/coverage.txt

novendor:
	@rm -rf ./vendor

protolint: novendor
	$(PROTOTOOL) lint

protofmt: novendor
	$(PROTOTOOL) format -w

protoc: protofmt protolint protodocs protogen testvectors
	@# protoc will clean protobuf, generate new code, then create testvectors

protogen:
	$(PROTOTOOL) generate
	@# a bit of playing around to rename output, so it is only available for testcode
	@mv -f x/gov/sample_test.pb.go x/gov/sample_test.go

protodocs:
	./scripts/clean_protos.sh
	./scripts/build_protodocs_docker.sh

testvectors:
	@mkdir -p spec/testvectors
	go run ./cmd/bnsd/main.go testgen spec/testvectors > spec/testvectors/ADDRESS.txt
