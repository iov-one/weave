.PHONY: all install build test cover deps glide tools protoc

GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_FLAGS := -ldflags "-X github.com/iov-one/bov-core.GitCommit=$(GIT_COMMIT)"
TENDERMINT := ${GOBIN}/tendermint

# dont use `` in the makefile for windows compatibility
NOVENDOR := $(shell go list ./...)

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= set

all: deps build test

install:
	go install $(BUILD_FLAGS) ./cmd/bov

# This is to make sure it all compiles
build:
	go build ./...

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

deps: glide $(TENDERMINT)
	@glide install

glide:
	@go get github.com/Masterminds/glide
	@glide mirror set https://github.com/tendermint/go-wire https://github.com/ethanfrey/go-wire

$(TENDERMINT):
	@ #install tendermint binary for testing
	@ #go get -u github.com/tendermint/tendermint/cmd/tendermint
	@ # Use this if the above fails
	go get -u -d github.com/tendermint/tendermint || true
	cd $$GOPATH/src/github.com/tendermint/tendermint && make get_vendor_deps && make install

protoc:
	protoc --gogofaster_out=. -I=. -I=./vendor x/namecoin/*.proto
	@ # $$GOPATH/src go we can import namecoin .proto
	protoc --gogofaster_out=. -I=. -I=./vendor -I=$$GOPATH/src app/*.proto

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

tools: /usr/local/bin/protoc deps
	# install all tools from our vendored dependencies
	@go install ./vendor/github.com/gogo/protobuf/proto
	@go install ./vendor/github.com/gogo/protobuf/gogoproto
	@go install ./vendor/github.com/gogo/protobuf/protoc-gen-gogofaster


