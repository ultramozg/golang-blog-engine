#Go parameters
GOOS=linux
GOARCH=amd64
BINARY_NAME=webserver

GIT_COMMIT := $(shell git rev-list -1 HEAD)

build:
	rm -f $(BINARY_NAME)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -ldflags "-X main.gitCommit=$(GIT_COMMIT)" -o $(BINARY_NAME) *.go

build-arm:
	rm -f $(BINARY_NAME)
	CGO_ENABLED=1 CC=arm-linux-gnueabihf-gcc-6 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-X main.gitCommit=$(GIT_COMMIT)" -o $(BINARY_NAME) *.go

clean:
	rm $(BINARY_NAME)
