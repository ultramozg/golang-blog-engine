#Go parameters
BINARY_NAME=webserver

build:
	rm -f $(BINARY_NAME)
	CGO_ENABLED=1 go build -o webserver *.go

build-arm:
	rm -f $(BINARY_NAME)
	CGO_ENABLED=1 CC=arm-linux-gnueabihf-gcc-6 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BINARY_NAME) *.go

clean:
	rm $(BINARY_NAME)
