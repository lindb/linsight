.PHONY: help build test deps generate clean

# Ref: https://gist.github.com/prwhite/8168133
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

run: ## run local server for demo/debug
	go run github.com/lindb/linsight/cmd server run

GOMOCK_VERSION = "v1.6.0"
gomock: ## go generate mock file.
	go install "github.com/golang/mock/mockgen@$(GOMOCK_VERSION)"
	go list ./... |grep -v '/gomock' | xargs go generate -v

lint: ## run lint
ifeq (, $(shell which golangci-lint))
	# binary will be $(go env GOPATH)/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.51.2
else
	echo "Found golangci-lint"
endif
	golangci-lint run ./...

header: ## check and add license header.
	sh scripts/addlicense.sh

test-without-lint: ## Run test without lint
	go install "github.com/rakyll/gotest@v0.0.6"
	GIN_MODE=release
	LOG_LEVEL=fatal ## disable log for test
	gotest -v -race -coverprofile=coverage.out -covermode=atomic ./...

test: header lint test-without-lint ## Run test cases.

clean-frontend-build: ## Clean fontend build
	cd web/ && make web_clean

build-frontend: ## build frontend
	cd web/ && make web_build
deps:  ## Update vendor.
	go mod verify
	go mod tidy -v
