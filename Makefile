VERSION=$(shell git describe --tags)
COMMIT=$(shell git rev-parse HEAD)
DATE=$(shell date)

LDFLAGS=-ldflags "-w -s -X 'main.gitCommit=$(COMMIT)' -X 'main.gitDate=$(DATE)'"

.DEFAULT_GOAL := benchmark

%: cmd/%
	@echo "Building target: $@" 
	go build $(LDFLAGS) -o bin/$@ cmd/$@/*.go

clean:
	@rm bin/*

.PHONY: clean

all: benchmark reporter
