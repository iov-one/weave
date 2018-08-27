.PHONY: all install build test cover deps tools prototools protoc clean dist

BUILD_VERSION ?= manual
BUILD_FLAGS := -ldflags "-X github.com/iov-one/weave.Version=${BUILD_VERSION}"
DOCKER_BUILD_FLAGS := -a -installsuffix cgo
TENDERMINT := ${GOBIN}/tendermint
BUILDOUT ?= bcpd
GOPATH ?= $$HOME/go
IMAGE_NAME = "iov1/bcpd:${BUILD_VERSION}"

TM_VERSION := v0.21.0

all: deps build test

dist: clean test build image

clean:
	rm -f ${BUILDOUT}

install:
	go install $(BUILD_FLAGS) .

build:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build $(BUILD_FLAGS) $(DOCKER_BUILD_FLAGS) -o $(BUILDOUT) .

image:
	docker build --pull -t $(IMAGE_NAME) .

test:
	go test -race ./...

# Test fast
tf:
	go test -short ./...

protoc:
	protoc --gogofaster_out=. -I=. -I=$(GOPATH)/src app/*.proto

deps: $(TENDERMINT)

$(TENDERMINT):
	go get -d github.com/tendermint/tendermint/...
	cd $(GOPATH)/src/github.com/tendermint/tendermint && \
		git checkout $(TM_VERSION) && \
		make ensure_deps && make install && \
		git checkout . && \
		git checkout -

