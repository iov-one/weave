.PHONY: all install build test tf cover deps tools prototools protoc

EXAMPLES := examples/mycoind cmd/bcpd cmd/bnsd

# dont use `` in the makefile for windows compatibility
NOVENDOR := $(shell go list ./...)

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= set
GOPATH ?= $$HOME/go

PROTOC_FLAGS := -I=. -I=./vendor -I=$(GOPATH)/src

all: deps build test

dist:
	cd cmd/bnsd ; make dist ; cd -
	cd cmd/bcpd ; make dist ; cd -

install:
	for ex in $(EXAMPLES); do cd $$ex && make install && cd -; done

# This is to make sure it all compiles
build:
	go build ./...

test:
	go vet -composites=false ./...
	go test -race ./...

# Test fast
tf:
	go test -short ./...

cover:
	@ go test -covermode=$(MODE) -coverprofile=coverage/allpackages.out ./...
	@ # most of the tests in the app package are in examples/mycoind/app...
	@ go test -covermode=$(MODE) \
	 	-coverpkg=github.com/iov-one/weave/app,github.com/iov-one/weave/examples/mycoind/app \
		-coverprofile=coverage/weave_examples_mycoind_app.out \
		github.com/iov-one/weave/examples/mycoind/app
	@ go test -covermode=$(MODE) \
	 	-coverpkg=github.com/iov-one/weave/commands/server \
		-coverprofile=coverage/weave_commands_server.out \
		github.com/iov-one/weave/examples/mycoind/commands
	@ go test -covermode=$(MODE) \
		-coverpkg=github.com/iov-one/weave/cmd/bnsd/app,github.com/iov-one/weave/cmd/bnsd/client,github.com/iov-one/weave/app \
		-coverprofile=coverage/bnsd_app.out \
		github.com/iov-one/weave/cmd/bnsd/scenarios
	cat coverage/*.out > coverage/coverage.txt

deps: tools
	@rm -rf vendor/
	dep ensure -vendor-only

tools:
	@go get github.com/golang/dep/cmd/dep

lint:
ifndef $(shell command -v prototool help > /dev/null)
	@go get github.com/uber/prototool/cmd/prototool
endif
	prototool lint

# protoc: protodocs
protoc:
	protoc --gogofaster_out=. $(PROTOC_FLAGS) app/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) coin/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) crypto/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) orm/*.proto
	# Note, you must include -I=./vendor when compiling files that use gogoprotobuf extensions
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/nft/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) cmd/bnsd/x/nft/username/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/cash/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/sigs/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/msgfee/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/multisig/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/validators/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/batch/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/distribution/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/namecoin/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/escrow/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/paychan/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/currency/*.proto
	for ex in $(EXAMPLES); do cd $$ex && make protoc && cd -; done

protodocs:
	@./scripts/build_protodocs.sh

### cross-platform check for installing protoc ###

MYOS := $(shell uname -s)

ifeq ($(MYOS),Darwin)  # Mac OS X
	ZIP := protoc-3.7.0-osx-x86_64.zip
endif
ifeq ($(MYOS),Linux)
	ZIP := protoc-3.7.0-linux-x86_64.zip
endif

/usr/local/bin/protoc:
	@ curl -L https://github.com/google/protobuf/releases/download/v3.7.0/$(ZIP) > $(ZIP)
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
	# these are for custom extensions
	@ # @go install ./vendor/github.com/gogo/protobuf/proto
	@ # @go install ./vendor/github.com/gogo/protobuf/jsonpb
	@ # @go install ./vendor/github.com/gogo/protobuf/protoc-gen-gogo
	@ # go get github.com/golang/protobuf/protoc-gen-go
	# docs
	@go get -u github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc


