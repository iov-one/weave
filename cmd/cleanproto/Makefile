.PHONY: build test

build:
	CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o cleanproto .

test:
	go test . -v -race
	./tests/test.sh
