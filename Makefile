# #
# VERSION			:= $(shell git describe --tags)
# BUILD 			:= $(shell git rev-parse --short HEAD)
#PROJECT_NAME 	:= $(shell basename "$(PWD)")
PROJECT_NAME 	:= mq-to-db
PROJECT_BIN_PATH := cmd

# Golang
GO ?= go
GO_BUILD ?= $(GO) build
GO_TEST ?= $(GO) test
GO_FMT ?= $(GO)fmt
GO_MOD ?= $(GO) mod
GO_OPTS ?= -race

GO_VENDOR_FOLDER ?= vendor

# Container
# CONTAINER_BUILD_COMMAND ?= docker build
# CONTAINER_BUILD_FILE ?= ./Dockerfile
# CONTAINER_IMAGE_NAME ?= mq-to-db
# CONTAINER_IMAGE_TAGS ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))


#
.PHONY: all
all: go-lint go-tidy go-test go-build

.PHONY: go-lint
go-lint:
	@echo "--> Linting"

.PHONY: go-fmt
go-fmt:
	@echo "--> Checking formating"
	@resp=$$($(GO_FMT) $(GO_OPTS) -d $$(find . -path cmd -name '*.go' -print) 2>&1); \
		if [ -n "$${resp}" ]; then \
			echo "-->--> $${resp}"; echo; \
			echo "-->--> [ERR] Use gofmt for formatting code."; \
			exit 1; \
		else \
			echo "--> [OK]"; \
		fi

.PHONY: go-build
go-build:
	@echo "--> Building"
	@resp=$$($(GO_BUILD) $(GO_OPTS) -o $(PROJECT_NAME) $$(find ./cmd -name '*.go' -print) 2>&1); \
		if [ -n "$${resp}" ]; then \
			echo "-->--> $${resp}"; echo; \
			echo "-->--> [ERR]"; \
			exit 1; \
		else \
			echo "--> [OK]"; \
		fi

.PHONY: go-update-deps
go-update-deps:
	@echo "--> Updating Go dependencies"
	@for dep in $$($(GO) list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all); do \
		$(GO) get $$dep; \
	done
	@echo "--> [OK]"

.PHONY: go-tidy
go-tidy:
	@echo "--> Tidying"
	@resp=$$($(GO_MOD) tidy); \
		if [ -n "$${resp}" ]; then \
			echo "-->--> $${resp}"; echo; \
			echo "-->--> [ERR]"; \
			exit 1; \
		else \
			echo "--> [OK]"; \
		fi

ifneq (,$(wildcard $(GO_VENDOR_FOLDER)))
	$(GO_MOD) vendor
endif

.PHONY: go-test
go-test:
	@echo "--> Test"

.PHONY: clean
clean:
	@echo "--> Cleaning"

.PHONY: container-build
container-build:
	@echo "--> Building container image"

	# $(CONTAINER_BUILD_COMMAND) \
	# 	--build-arg ARCH="$*" \
	# 	--build-arg OS="linux" \
	# 	--tag "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME)-linux-$*:$(DOCKER_IMAGE_TAG)" \
	# 	--file $(DOCKERFILE_PATH) \
	# 	$(DOCKERBUILD_CONTEXT)