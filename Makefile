.PHONY: all install test tf cover deps prototools protoc govet
# make sure we turn on go modules
export GO111MODULE := on

EXAMPLES := examples/mycoind cmd/bcpd cmd/bnsd

# dont use `` in the makefile for windows compatibility
NOVENDOR := $(shell go list ./...)

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= set
GOPATH ?= $$HOME/go

# HAS_PROTOTOOL will be set to prototool path if available, "" if not
HAS_PROTOTOOL := $(shell which prototool)


all: deps test

dist:
	cd cmd/bnsd && $(MAKE) dist
	cd cmd/bcpd && $(MAKE) dist

install:
	for ex in $(EXAMPLES); do cd $$ex && make install && cd -; done

test:
	go vet ./...
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

deps:
	@ go mod vendor

lint:
ifeq (,$(HAS_PROTOTOOL))
	go get github.com/uber/prototool/cmd/prototool
endif
	prototool lint


PROTO_INCLUDE := -I=. -I $(shell go list -f '{{ .Dir }}' -m github.com/golang/protobuf)  -I $(shell go list -f '{{ .Dir }}' -m github.com/gogo/protobuf)  -I $(shell go list -f '{{ .Dir }}' -m github.com/gogo/protobuf)/protobuf -I $(shell go list -f '{{ .Dir }}' -m github.com/iov-one/weave)


protofmt:
	-find . -name '*proto' -exec prototool format -w {} \;

protoc: #protodocs protofmt
	find . \
		-iname '*.proto' \
		-path ./vendor -prune \
		-exec protoc --gogofaster_out=paths=source_relative:.  $(PROTO_INCLUDE) {} \;

	@# a bit of playing around to rename output, so it is only available for testcode
	-@mv x/gov/sample_test.pb.go x/gov/sample_test.go

protodocs:
	-protoc $(PROTO_INCLUDE) --doc_out="./docs/proto/" --doc_opt=html,index.html combined_proto_todo.proto

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


