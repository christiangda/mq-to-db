# #
# VERSION			:= $(shell git describe --tags)
# BUILD 			:= $(shell git rev-parse --short HEAD)
PROJECT_NAME 	:= $(shell basename "$(PWD)")

# Golang
GO ?= $(shell go)
GO_BUILD ?= $(shell $(GO) build)
GO_TEST ?= $(shell $(GO) test)

# Container
CONTAINER_BUILD_COMMAND ?= $(shell docker build)
CONTAINER_BUILD_FILE ?= ./Dockerfile
CONTAINER_IMAGE_NAME ?= mq-to-db
CONTAINER_IMAGE_TAGS ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))


#
.PHONY: all
all: go-lint go-tidy go-test go-build container-build

.PHONY: go-lint
go-lint:
	@echo "--> Linting"

.PHONY: go-build
go-build:
	@echo "--> Building"

.PHONY: go-tidy
go-tidy:
	@echo "--> Tidying"

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