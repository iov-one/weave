.PHONY: all install test cover deps glide

GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_FLAGS := -ldflags "-X github.com/confio/weave.GitCommit=$(GIT_COMMIT)"


# dont use `` in the makefile for windows compatibility
NOVENDOR := $(shell glide novendor)

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= count

all: deps install test

install:
	# TODO: install cmd later... now just compile important dirs
	go install $(BUILD_FLAGS) .

test:
	go test $(NOVENDOR)

# TODO: test all packages... names on each
cover:
	go test -covermode=$(MODE) -coverprofile=coverage/$(MODE).out .
	go tool cover -html=coverage/$(MODE).out -o=coverage/$(MODE).html
	go tool cover -func=coverage/$(MODE).out

deps: glide
	@glide install

glide:
	go get github.com/tendermint/glide
