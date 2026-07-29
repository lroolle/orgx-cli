.PHONY: build install clean test lint fmt deps check

check:
	go vet ./...
	go test ./...
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/orgx
	@echo "All checks passed"

BINARY := orgx
VERSION := 0.2.0-dev
BUILD_DIR := bin
INSTALL_DIR := $(HOME)/.local/bin

build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY) ./cmd/orgx

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

lint:
	golangci-lint run

fmt:
	gofmt -s -w .
	goimports -w .

deps:
	go mod download
	go mod tidy
