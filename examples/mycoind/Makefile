.PHONY: install protoc deps

GIT_VERSION := $(shell git describe --tags)
BUILD_FLAGS := -ldflags "-X github.com/confio/weave.Version=$(GIT_VERSION)"
TENDERMINT := ${GOBIN}/tendermint

install:
	go install $(BUILD_FLAGS) .

protoc:
	protoc --gogofaster_out=. -I=. -I=$$GOPATH/src app/*.proto

# this is extra dependencies related to our example
# (integration tests)
deps: $(TENDERMINT)

$(TENDERMINT):
	@ #install tendermint binary for testing
	@ #go get -u github.com/tendermint/tendermint/cmd/tendermint
	@ # Use this if the above fails
	go get -u -d github.com/tendermint/tendermint || true
	cd $$GOPATH/src/github.com/tendermint/tendermint && make get_vendor_deps && make install
