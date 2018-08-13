.PHONY: install protoc deps

GIT_VERSION := $(shell git describe --tags)
BUILD_FLAGS := -ldflags "-X github.com/iov-one/weave.Version=$(GIT_VERSION)"
TENDERMINT := ${GOBIN}/tendermint
BUILDOUT ?= mycoind
GOPATH ?= $$HOME/go

TM_VERSION := v0.21.0

install:
	go install $(BUILD_FLAGS) .

build:
	go build $(BUILD_FLAGS) -o $(BUILDOUT) .

protoc:
	protoc --gogofaster_out=. -I=. -I=$(GOPATH)/src app/*.proto

# this is extra dependencies related to our example
# (integration tests)
deps: $(TENDERMINT)

$(TENDERMINT):
	@ #install tendermint binary for testing
	go get -u -d github.com/tendermint/tendermint/... || true
	cd $(GOPATH)/src/github.com/tendermint/tendermint && \
		git checkout $(TM_VERSION) && \
		make ensure_deps && make install && \
		git checkout . && \
		git checkout -
