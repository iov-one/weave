BUILD_VERSION ?= manual
BUILD_FLAGS := -mod=readonly -ldflags "-X main.buildHash=`git log -1 --pretty=format:"%H"` -X main.buildVersion=${BUILD_VERSION}"
IMAGE_NAME = "iov1/bnsapi:${BUILD_VERSION}"

all: dist

dist: clean test build image

clean:
	rm -f bnsapi 2> /dev/null

build:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build $(BUILD_FLAGS) -o bnsapi .

image:
	docker build --pull -t $(IMAGE_NAME) .

test:
	go test -mod=readonly -race ./...

run:
	go run .
