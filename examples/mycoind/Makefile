.PHONY: install protoc clean dist build

BUILD_VERSION ?= $(shell git describe --tags)
BUILD_FLAGS := -ldflags "-X github.com/iov-one/weave.Version=$(BUILD_VERSION)"
BUILDOUT ?= mycoind


dist: clean test build

clean:
	rm -f ${BUILDOUT}

test:
	go test -race ./...

install:
	go install $(BUILD_FLAGS) .

build:
	go build $(BUILD_FLAGS) -o $(BUILDOUT) .

protoc:
	protoc --gogofaster_out=. -I=. -I=$(GOPATH)/src app/*.proto
