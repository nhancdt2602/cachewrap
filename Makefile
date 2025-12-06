# CacheWrap Makefile
.PHONY: build install clean test example

build:
	@echo "Building cachewrap CLI tool..."
	@go build -o cachewrap ./cmd/cachewrap

build-debug:
	@echo "Building cachewrap CLI tool with debug..."
	@go build -o cachewrap-debug ./cmd/cachewrap-debug

install:
	@echo "Installing cachewrap..."
	@go install ./cmd/cachewrap

install-debug:
	@echo "Installing cachewrap-debug..."
	@go install ./cmd/cachewrap-debug

example-basic:
	@echo "Building example with automatic cache instrumentation..."
	@cd example && go build -toolexec="$(CURDIR)/cachewrap  remix" -a -o basic_run ./basic
	@echo "\n=== Running basic example ===\n"
	@cd example && ./basic_run
	@rm example/basic_run


help:
	@echo "Usage:"
	@echo "  make build        		- Build the cachewrap CLI tool"
	@echo "  make build-debug  		- Build the cachewrap-debug CLI tool"
	@echo "  make install      		- Install cachewrap to GOPATH/bin"
	@echo "  make install-debug   	- Install cachewrap-debug to GOPATH/bin"
	@echo "  make example-basic  	- Build and run basic example"
	@echo "  make help           	- Show this help message"
