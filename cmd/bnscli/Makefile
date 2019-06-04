all: install

build:
	go build -ldflags "-X main.gitHash=$(shell git rev-list -1 HEAD)" .


clean:
	-rm bnscli


install:
	go install -ldflags "-X main.gitHash=$(shell git rev-list -1 HEAD)" .


.PHONY: all build clean install
