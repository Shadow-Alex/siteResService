# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=service

all: proto release debug

proto:
	protoc -I. --micro_out=. --go_out=. ./src/proto/api.proto

release:
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -ldflags "-s -w" -o ./bin/release/$(BINARY_NAME) ./src/*.go

debug:
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -o ./bin/debug/$(BINARY_NAME) ./src/*.go
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -o ./bin/debug/clientTest ./src/test/clientTest/*.go
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -o ./bin/debug/serverTest ./src/test/serverTest/*.go
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -o ./bin/debug/readCSVTest ./src/test/readCSVTest/*.go

clean:
	rm -rf ./bin/release/* ./bin/debug/*
