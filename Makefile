.PHONY: all install test cover deps glide

GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_FLAGS := -ldflags "-X github.com/confio/weave.GitCommit=$(GIT_COMMIT)"


# dont use `` in the makefile for windows compatibility
NOVENDOR := $(shell go list ./...)

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
	@ #Note: 19 is the length of "github.com/confio/" prefix
	for pkg in $(NOVENDOR); do \
        file=`echo $$pkg | cut -c 19- | tr / _`; \
		go test -covermode=$(MODE) -coverprofile=coverage/$$file.out $$pkg; \
		go tool cover -html=coverage/$$file.out -o=coverage/$$file.html; \
		go tool cover -func=coverage/$$file.out; \
	done

deps: glide
	@glide install

glide:
	go get github.com/tendermint/glide
