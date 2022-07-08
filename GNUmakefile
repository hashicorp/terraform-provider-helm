TEST?="./helm"
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

# For changelog generation, default the last release to the last tag on
# any branch, and this release to just be the current branch we're on.
LAST_RELEASE?=$$(git describe --tags $$(git rev-list --tags --max-count=1))
THIS_RELEASE?=$$(git rev-parse --abbrev-ref HEAD)

default: build

build: fmtcheck
	go build -v .

# expected to be invoked by make changelog LAST_RELEASE=gitref THIS_RELEASE=gitref
changelog:
	@echo "Generating changelog for $(THIS_RELEASE) from $(LAST_RELEASE)..."
	@echo
	@changelog-build -last-release $(LAST_RELEASE) \
		-entries-dir .changelog/ \
		-changelog-template .changelog/changelog.tmpl \
		-note-template .changelog/note.tmpl \
		-this-release $(THIS_RELEASE)

changelog-entry:
	@changelog-entry -dir .changelog/
	

test: fmtcheck
	go test $(TEST) -v || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 10m

testrace: fmtcheck
	TF_ACC= go test -race $(TEST) $(TESTARGS)

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

# The docker command and run options may be overridden using env variables DOCKER and DOCKER_RUN_OPTS.
# Example:
#   DOCKER="podman --cgroup-manager=cgroupfs" make website-lint
#   DOCKER_RUN_OPTS="--userns=keep-id" make website-lint
#   This option is needed for systems using SELinux and rootless containers.
#   DOCKER_VOLUME_OPTS="rw,Z"
# For more info, see https://docs.docker.com/storage/bind-mounts/#configure-the-selinux-label
DOCKER?=$(shell which docker)
ifeq ($(strip $(DOCKER)),)
$(error "Docker binary could not be found in PATH. Please install docker, or specify an alternative by setting DOCKER=/path/to/binary")
endif
DOCKER_VOLUME_OPTS?="rw"
DOCKER_SELINUX := $(shell which setenforce)
ifeq ($(.SHELLSTATUS),0)
DOCKER_VOLUME_OPTS="rw,Z"
endif
# PROVIDER_DIR_DOCKER is used instead of PWD since docker volume commands can be dangerous to run in $HOME.
# This ensures docker volumes are mounted from within provider directory instead.
PROVIDER_DIR_DOCKER := $(abspath $(lastword $(dir $(MAKEFILE_LIST))))

website-lint:
	@echo "==> Checking website against linters..."
	@echo "==> Running markdownlint-cli using DOCKER='$(DOCKER)', DOCKER_RUN_OPTS='$(DOCKER_RUN_OPTS)' and DOCKER_VOLUME_OPTS='$(DOCKER_VOLUME_OPTS)'"
	@$(DOCKER) run --rm $(DOCKER_RUN_OPTS) -v $(PROVIDER_DIR_DOCKER):/workspace:$(DOCKER_VOLUME_OPTS) -w /workspace 06kellyjac/markdownlint-cli ./website \
		&& (echo; echo "PASS - website markdown files pass linting"; echo ) \
		|| (echo; echo "FAIL - issues found in website markdown files"; echo ; exit 1)
	@echo "==> Checking for broken links..."
	@scripts/markdown-link-check.sh "$(DOCKER)" "$(DOCKER_RUN_OPTS)" "$(DOCKER_VOLUME_OPTS)" "$(PROVIDER_DIR_DOCKER)"

.PHONY: build test testacc testrace cover vet fmt fmtcheck errcheck test-compile packages clean website-lint changelog changelog-entry
