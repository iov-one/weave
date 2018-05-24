.PHONY: all install build test cover deps tools prototools protoc

GIT_VERSION := $(shell git describe --tags)
BUILD_FLAGS := -ldflags "-X github.com/iov-one/bcp-demo.Version=$(GIT_VERSION)"
DOCKER_BUILD_FLAGS := -a -installsuffix cgo
TENDERMINT := ${GOBIN}/tendermint
BUILDOUT ?= bov
GOPATH ?= $$HOME/go

TM_VERSION := v0.17.1

# dont use `` in the makefile for windows compatibility
NOVENDOR := $(shell go list ./...)

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= set

all: deps build test

install:
	go install $(BUILD_FLAGS) ./cmd/bov

build:
	go build $(BUILD_FLAGS) -o $(BUILDOUT) ./cmd/bov

docker-build:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build $(BUILD_FLAGS) $(DOCKER_BUILD_FLAGS) -o $(BUILDOUT) ./cmd/bov
	docker build . -t "iov1/bov:$(GIT_VERSION)"
	rm -rf $(BUILDOUT)

test:
	go test -race ./...

# Test fast
tf:
	go test -short ./...

cover:
	@ #Note: 20 is the length of "github.com/iov-one/" prefix
	@ for pkg in $(NOVENDOR); do \
        file=`echo $$pkg | cut -c 20- | tr / _`; \
	    echo "Coverage on" $$pkg "as" $$file; \
		go test -covermode=$(MODE) -coverprofile=coverage/$$file.out $$pkg; \
		go tool cover -html=coverage/$$file.out -o=coverage/$$file.html; \
	done

deps: tools $(TENDERMINT)
	@rm -rf vendor/
	@dep ensure

tools:
	@go get github.com/golang/dep/cmd/dep

$(TENDERMINT):
	go get -d github.com/tendermint/tendermint/...
	cd $(GOPATH)/src/github.com/tendermint/tendermint && \
		git checkout $(TM_VERSION) && \
		make ensure_deps && make install && \
		git checkout -

protoc:
	protoc --gogofaster_out=. -I=. -I=./vendor x/namecoin/*.proto
	protoc --gogofaster_out=. -I=. -I=./vendor x/escrow/*.proto
	@ # $(GOPATH)/src go we can import namecoin .proto
	protoc --gogofaster_out=. -I=. -I=./vendor -I=$(GOPATH)/src app/*.proto

### cross-platform check for installing protoc ###

MYOS := $(shell uname -s)

ifeq ($(MYOS),Darwin)  # Mac OS X
	ZIP := protoc-3.4.0-osx-x86_64.zip
endif
ifeq ($(MYOS),Linux)
	ZIP := protoc-3.4.0-linux-x86_64.zip
endif

/usr/local/bin/protoc:
	@ curl -L https://github.com/google/protobuf/releases/download/v3.4.0/$(ZIP) > $(ZIP)
	@ unzip -q $(ZIP) -d protoc3
	@ rm $(ZIP)
	sudo mv protoc3/bin/protoc /usr/local/bin/
	@ sudo mv protoc3/include/* /usr/local/include/
	@ sudo chown `whoami` /usr/local/bin/protoc
	@ sudo chown -R `whoami` /usr/local/include/google
	@ rm -rf protoc3

prototools: /usr/local/bin/protoc deps
	# install all tools from our vendored dependencies
	@go install ./vendor/github.com/gogo/protobuf/proto
	@go install ./vendor/github.com/gogo/protobuf/gogoproto
	@go install ./vendor/github.com/gogo/protobuf/protoc-gen-gogofaster


