all: install

build:
	go build -mod=readonly -ldflags "-X main.gitHash=$(shell git rev-list -1 HEAD)" .


clean:
	-rm bnscli


install:
	go install -mod=readonly -ldflags "-X main.gitHash=$(shell git rev-list -1 HEAD)" .


.PHONY: all build clean install
