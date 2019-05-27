build: cleanproto

cleanproto:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o cleanproto .

test:
	go test . -v -race
	./tests/test.sh
