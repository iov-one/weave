.PHONY: build test

build:
	CGO_ENABLED=0 go build -mod=readonly -o cleanproto .

test:
	go test -mod=readonly . -v -race
	./tests/test.sh
