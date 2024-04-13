BIN="./bin"
SRC=$(shell find . -name "*.go")
CURRENT_TAG=$(shell git describe --tags --abbrev=0)

GOLANGCI := $(shell command -v golangci-lint 2>/dev/null)
RICHGO := $(shell command -v richgo 2>/dev/null)
GOTESTFMT := $(shell command -v gotestfmt 2>/dev/null)

.PHONY: fmt lint build test changelog

default: all

all: fmt lint build test changelog

fmt:
	$(info ******************** checking formatting ********************)
	@test -z $(shell gofmt -l $(SRC)) || (gofmt -d $(SRC); exit 1)

.PHONY: golangci-lint-check
golangci-lint-check:
ifndef GITHUB_ACTIONS
	$(info ******************** checking if golangci-lint is installed ********************)
	$(warning "ensuring latest version of golangci-lint installed, running: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
	go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@latest
endif

.PHONY: lint
lint: golangci-lint-check
	$(info ******************** running lint tools ********************)
	golangci-lint run -c .golangci-lint.yml -v ./... --timeout 10m

test:
	$(info ******************** running tests ********************)
    ifeq ($(GITHUB_ACTIONS), true)
        ifndef GOTESTFMT
			$(warning "could not find gotestfmt in $(PATH), running: go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest")
			$(shell go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest)
        endif
		go test -json -v ./... 2>&1 | tee coverage/gotest.log | gotestfmt
    else
        ifndef RICHGO
			$(warning "could not find richgo in $(PATH), running: go install github.com/kyoh86/richgo@latest")
			$(shell go install github.com/kyoh86/richgo@latest)
        endif
		richgo test -v ./...
    endif

changelog:
	$(info ******************** running git-cliff updating CHANGELOG.md ********************)
	git-cliff -o CHANGELOG.md

build:
	go env -w GOFLAGS=-mod=mod
	go mod tidy
	go build -v -o gofireprox -trimpath -ldflags="-s -w" ./cmd/gofireprox
