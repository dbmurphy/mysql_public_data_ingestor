# Variables
MODULE_NAME := mysql_public_data_ingestor
MAIN_DIR := .
PLUGIN_DIR := api_plugins
PLUGINS := $(wildcard $(PLUGIN_DIR)/*_plugin.go)
PLUGIN_SO := $(PLUGINS:.go=.so)
TEST_DIRS := $(shell go list ./... | grep -v /vendor/)

# Default target
.PHONY: all
all: build test

# Build main package
.PHONY: build
build: build-plugins build-image
	@echo "Building main package..."
	go build -o bin/$(MODULE_NAME) $(MAIN_DIR)

# Build Docker image if needed
.PHONY: build-image
build-image:
	@echo "Building Docker image if needed..."
	@./docker/build-if-needed-docker.sh

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v $(TEST_DIRS)

# Build plugins
.PHONY: build-plugins
build-plugins: $(PLUGIN_SO)

$(PLUGIN_DIR)/%.so: $(PLUGIN_DIR)/%.go
	@echo "Building plugin $<..."
	go build -buildmode=plugin -o $@ $<

# Test plugins
.PHONY: test-plugins
test-plugins:
	@echo "Running plugin tests..."
	@for plugin in $(PLUGINS); do \
		echo "Testing plugin $$plugin..."; \
		go test -v $$(dirname $$plugin); \
	done

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -f bin/$(MODULE_NAME)
	rm -f $(PLUGIN_DIR)/*.so
