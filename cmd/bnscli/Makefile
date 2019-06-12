all: install

build:
	CGO_ENABLED=0 go build -mod=readonly -ldflags "-X main.gitHash=$(shell git rev-list -1 HEAD) -extldflags \"-static\"" .


clean:
	-rm bnscli


install:
	CGO_ENABLED=0 go install -mod=readonly -ldflags "-X main.gitHash=$(shell git rev-list -1 HEAD) -extldflags \"-static\"" .


.PHONY: all build clean install
