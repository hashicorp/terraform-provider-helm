TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
COVER_TEST?=$$(go list ./... |grep -v 'vendor')
WEBSITE_REPO=github.com/hashicorp/terraform-website
PKG_NAME=helm

PKG_OS ?= darwin linux
PKG_ARCH ?= amd64
BASE_PATH ?= $(shell pwd)
BUILD_PATH ?= $(BASE_PATH)/build
PROVIDER := $(shell basename $(BASE_PATH))
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
VERSION ?= v0.0.0
ifneq ($(origin TRAVIS_TAG), undefined)
	BRANCH := $(TRAVIS_TAG)
	VERSION := $(TRAVIS_TAG)
endif

default: build

build: fmtcheck
	go build -v .

test: fmtcheck
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m

testrace: fmtcheck
	TF_ACC= go test -race $(TEST) $(TESTARGS)

compile: fmtcheck
	@sh -c "'$(CURDIR)/scripts/compile.sh'"

cover:
	@go tool cover 2>/dev/null; if [ $$? -eq 3 ]; then \
		go get -u golang.org/x/tools/cmd/cover; \
	fi
	go test $(COVER_TEST) -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"

vendor-status:
	@govendor status

test-compile: fmtcheck
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./helm"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

packages:
	@for os in $(PKG_OS); do \
		for arch in $(PKG_ARCH); do \
			mkdir -p $(BUILD_PATH)/$(PROVIDER)_$${os}_$${arch} && \
			cd $(BASE_PATH) && \
			CGO_ENABLED=0 GOOS=$${os} GOARCH=$${arch} go build -o $(BUILD_PATH)/$(PROVIDER)_$${os}_$${arch}/$(PROVIDER)_$(VERSION) . && \
			cd $(BUILD_PATH) && \
			tar -cvzf $(BUILD_PATH)/$(PROVIDER)_$(BRANCH)_$${os}_$${arch}.tar.gz $(PROVIDER)_$${os}_$${arch}/; \
		done; \
	done;

clean:
	@rm -rf $(BUILD_PATH)

website:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), get-ting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	@$(MAKE) -C $(GOPATH)/src/$(WEBSITE_REPO) website-provider PROVIDER_PATH=$(shell pwd) PROVIDER_NAME=$(PKG_NAME)

website-test:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), get-ting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	@$(MAKE) -C $(GOPATH)/src/$(WEBSITE_REPO) website-provider-test PROVIDER_PATH=$(shell pwd) PROVIDER_NAME=$(PKG_NAME)

.PHONY: build test testacc testrace cover vet fmt fmtcheck errcheck vendor-status test-compile packages clean website website-test
