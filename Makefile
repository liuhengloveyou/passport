BUILD_TIME=$(shell date "+%FZ%T")
COMMIT_SHA1=$(shell git rev-parse HEAD)
flags="-X main.BuildTime=${BUILD_TIME} -X main.CommitID=${COMMIT_SHA1}"

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=passport
BINARY_UNIX=$(BINARY_NAME)_unix

all: build

build:
	$(GOBUILD) --ldflags ${flags} -o $(BINARY_NAME) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) --ldflags ${flags} -o $(BINARY_UNIX) -v
