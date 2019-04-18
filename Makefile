GO111MODULE := on
BIN := bin/server

SHELL := $(shell which bash)
VERBOSE ?= -verbose

run: build
	source .env && $(BIN) $(VERBOSE)

build:
	GO111MODULE=on go mod tidy
	GO111MODULE=on go build -o $(BIN) .

clean:
	rm -rf $(shell dirname $(BIN))

.PHONY: run clean
