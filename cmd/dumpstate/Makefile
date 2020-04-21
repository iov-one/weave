.PHONY: build test

build:
	CGO_ENABLED=0 go build -mod=readonly -o dumpstate .

test:
	go test -mod=readonly . -v -race
