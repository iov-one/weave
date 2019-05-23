.PHONY: all install test tf cover deps prototools protoc govet

# make sure we turn on go modules
export GO111MODULE := on

EXAMPLES := examples/mycoind cmd/bcpd cmd/bnsd

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= set
GOPATH ?= $$HOME/go

# for dockerized prototool
USER := $(shell id -u):$(shell id -g)
PROTOTOOL := docker run -it --rm --user $(USER) --mount type=bind,source="$(shell pwd)",target=/work --tmpfs /tmp:exec charithe/prototool-docker prototool

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
	$(PROTOTOOL) lint

protofmt:
	$(PROTOTOOL) format -w

protoc: protofmt #protodocs
	$(PROTOTOOL) generate
	# a bit of playing around to rename output, so it is only available for testcode
	@mv x/gov/sample_test.pb.go x/gov/sample_test.go

protodocs:
	@./scripts/build_protodocs.sh
