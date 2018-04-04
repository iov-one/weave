.PHONY: install protoc deps

GIT_VERSION := $(shell git describe --tags)
BUILD_FLAGS := -ldflags "-X github.com/confio/weave.Version=$(GIT_VERSION)"
TENDERMINT := ${GOBIN}/tendermint
BUILDOUT ?= mycoind

TM_VERSION := v0.17.1

install:
	go install $(BUILD_FLAGS) .

build:
	go build $(BUILD_FLAGS) -o $(BUILDOUT) .

protoc:
	protoc --gogofaster_out=. -I=. -I=$$GOPATH/src app/*.proto

# this is extra dependencies related to our example
# (integration tests)
deps: $(TENDERMINT)

$(TENDERMINT):
	@ #install tendermint binary for testing
	go get github.com/golang/dep/cmd/dep
	cd $$GOPATH/src/github.com/tendermint/tendermint && \
		git checkout $(TM_VERSION) && \
		make ensure_deps && make install && \
		git checkout -
