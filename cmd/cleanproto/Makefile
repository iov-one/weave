.PHONY: build test

build:
	CGO_ENABLED=0 go build -o cleanproto .

test:
	go test . -v -race
	./tests/test.sh
