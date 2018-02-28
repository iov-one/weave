.PHONY: all install build test cover deps glide tools protoc

EXAMPLES := "examples/mycoind"

# dont use `` in the makefile for windows compatibility
NOVENDOR := $(shell go list ./...)

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= set

all: deps build test

install:
	for ex in $(EXAMPLES); do cd $$ex && make install; done

# This is to make sure it all compiles
build:
	go build ./...

test:
	go test -race ./...

# Test fast
tf:
	go test -short ./...

cover:
	@ #Note: 19 is the length of "github.com/confio/" prefix
	@ for pkg in $(NOVENDOR); do \
        file=`echo $$pkg | cut -c 19- | tr / _`; \
	    echo "Coverage on" $$pkg "as" $$file; \
		go test -covermode=$(MODE) -coverprofile=coverage/$$file.out $$pkg; \
		go tool cover -html=coverage/$$file.out -o=coverage/$$file.html; \
	done

deps: glide
	@glide install
	for ex in $(EXAMPLES); do cd $$ex && make deps; done

glide:
	@go get github.com/tendermint/glide

protoc:
	protoc --gogofaster_out=. crypto/*.proto
	protoc --gogofaster_out=. x/*.proto
	protoc --gogofaster_out=. -I=. -I=$$GOPATH/src x/cash/*.proto
	protoc --gogofaster_out=. -I=. -I=$$GOPATH/src x/sigs/*.proto
	for ex in $(EXAMPLES); do cd $$ex && make protoc; done

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
	# we only need one probably, choose wisely...
	@go install ./vendor/github.com/gogo/protobuf/protoc-gen-gogofast
	@go install ./vendor/github.com/gogo/protobuf/protoc-gen-gogofaster
	@go install ./vendor/github.com/gogo/protobuf/protoc-gen-gogoslick
	# these are for custom extensions
	@ # @go install ./vendor/github.com/gogo/protobuf/proto
	@ # @go install ./vendor/github.com/gogo/protobuf/jsonpb
	@ # @go install ./vendor/github.com/gogo/protobuf/protoc-gen-gogo
	@ # go get github.com/golang/protobuf/protoc-gen-go


