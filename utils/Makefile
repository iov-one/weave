.PHONY: test tf build dist

dist: test build

# test targets are to allow easy integration with root Makefile
test:
	go test -race ./...

# Test fast
tf:
	go test -short ./...

# build is just to verify it compiles
build:
	go build .
