# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BUILDPATH=$(CURDIR)
BINARY_NAME=pinbackup

all: test build

test: 
	$(GOTEST) -v ./...

clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run:
	$(GOBUILD) -o $(BINARY_NAME) .
	./$(BINARY_NAME) server

tidy:
	$(GOMOD) tidy

build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BINARY_NAME) .

build-docker:
	docker run --rm --env GO111MODULE=on -v "$(BUILDPATH)":/app -w /app golang:1.14 go build -v
