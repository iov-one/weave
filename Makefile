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

# USER for dockerized prototool
USER := $(shell id -u):$(shell id -g)


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
	echo $(USER)
	# prototool lint


protofmt:
	-find . -name '*proto' -exec prototool format -w {} \;

protoc: protofmt #protodocs
	protoc --gogofaster_out=. $(PROTOC_FLAGS) codec.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) app/*.proto
	protoc --gogofaster_out=. $(PROTOC_FLAGS) migration/*.proto
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
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/aswap/*.proto
	# a bit of playing around to rename output, so it is only available for testcode
	rm -f x/gov/sample_test.go
	protoc --gogofaster_out=. $(PROTOC_FLAGS) x/gov/*.proto
	mv x/gov/sample_test.pb.go x/gov/sample_test.go
	# now build all examples
	for ex in $(EXAMPLES); do cd $$ex && make protoc && cd -; done

protodocs:
	@./scripts/build_protodocs.sh
