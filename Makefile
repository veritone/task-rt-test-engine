# BINARY_NAME defaults to the name of the repository
BINARY_NAME := $(notdir $(shell pwd))
LIST_NO_VENDOR := $(go list ./... | grep -v /vendor/)
GOBIN := $(GOPATH)/bin

default: check fmt deps test build

.PHONY: build
build: deps
	# Build project
	go build -a -o $(BINARY_NAME) .

.PHONY: build-dev
build-dev: deps
	# Build Docker container
	env GOOS=linux GOARCH=amd64 go build -a -o $(BINARY_NAME) .
	docker build --build-arg APP_ENV=dev -t rt-test-engine .

.PHONY: build-stage
build-stage: deps
	# Build Docker container
	env GOOS=linux GOARCH=amd64 go build -a -o $(BINARY_NAME) .
	docker build --build-arg APP_ENV=stage -t rt-test-engine .

.PHONY: build-prod
build-prod: deps
	# Build Docker container
	env GOOS=linux GOARCH=amd64 go build -a -o $(BINARY_NAME) .
	docker build --build-arg APP_ENV=prod -t rt-test-engine .

.PHONY: run-docker
run-docker: build-docker
	# Run Docker container
	docker run rt-test-engine

.PHONY: check
check:
	# Only continue if go is installed
	go version || ( echo "Go not installed, exiting"; exit 1 )

.PHONY: clean
clean:
	go clean -i
	rm -rf ./vendor/*/
	rm -f $(BINARY_NAME)

deps:
	# Install or update govend
	go get -u github.com/govend/govend
	# Fetch vendored dependencies
	$(GOBIN)/govend -v

.PHONY: fmt
fmt:
	# Format all Go source files (excluding vendored packages)
	go fmt $(LIST_NO_VENDOR)

generate-deps:
	# Generate vendor.yml
	govend -v -l
	git checkout vendor/.gitignore

.PHONY: test
test:
	# Run all tests (excluding vendored packages)
	go test -a -v -cover $(LIST_NO_VENDOR)