GO111MODULE := on
BIN := bin/server

CLIENT_ID ?= 0YO3rhFC1cFvdcjG6vW3m0naHA5PVVvR

run: build
	$(BIN) -client-id $(CLIENT_ID)

build:
	GO111MODULE=on go mod tidy
	GO111MODULE=on go build -o $(BIN) .

clean:
	rm -rf $(shell dirname $(BIN))

.PHONY: run clean
