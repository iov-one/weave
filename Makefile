.PHONY: all install build test tf cover deps tools prototools protoc

EXAMPLES := examples/mycoind cmd/bcpd cmd/bnsd

# dont use `` in the makefile for windows compatibility
NOVENDOR := $(shell go list ./...)

# MODE=count records heat map in test coverage
# MODE=set just records which lines were hit by one test
MODE ?= set
GOPATH ?= $$HOME/go

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

protoc:
	protoc --doc_out=docs/proto/ --doc_opt=html,app.html --gogofaster_out=. app/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,crypto.html --gogofaster_out=. crypto/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,orm.html --gogofaster_out=. orm/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,x.html --gogofaster_out=. x/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,nft.html --gogofaster_out=. -I=. -I=$(GOPATH)/src x/nft/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,bnsd.html --gogofaster_out=. -I=. -I=$(GOPATH)/src cmd/bnsd/x/nft/username/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,cash.html --gogofaster_out=. -I=. -I=$(GOPATH)/src x/cash/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,sigs.html --gogofaster_out=. -I=. -I=$(GOPATH)/src x/sigs/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,multisig.html --gogofaster_out=. -I=. -I=$(GOPATH)/src x/multisig/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,validators.html --gogofaster_out=. -I=. -I=$(GOPATH)/src x/validators/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,batch.html --gogofaster_out=. -I=. -I=$(GOPATH)/src x/batch/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,distribution.html --gogofaster_out=. -I=. -I=$(GOPATH)/src x/distribution/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,namecoin.html --gogofaster_out=. -I=. -I=$(GOPATH)/src -I=./vendor x/namecoin/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,escrow.html --gogofaster_out=. -I=. -I=$(GOPATH)/src -I=./vendor x/escrow/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,paychan.html --gogofaster_out=. -I=. -I=$(GOPATH)/src -I=./vendor x/paychan/*.proto
	protoc --doc_out=docs/proto/ --doc_opt=html,currency.html --gogofaster_out=. -I=. -I=$(GOPATH)/src x/currency/*.proto
	for ex in $(EXAMPLES); do cd $$ex && make protoc && cd -; done

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
	# these are for custom extensions
	@ # @go install ./vendor/github.com/gogo/protobuf/proto
	@ # @go install ./vendor/github.com/gogo/protobuf/jsonpb
	@ # @go install ./vendor/github.com/gogo/protobuf/protoc-gen-gogo
	@ # go get github.com/golang/protobuf/protoc-gen-go
	# docs
	@go get -u github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc


