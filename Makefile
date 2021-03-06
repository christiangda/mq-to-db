# Check for required command tools to build or stop immediately
EXECUTABLES = go find which
K := $(foreach exec,$(EXECUTABLES),\
  $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH)))

#
APP_NAME 	 ?= mq-to-db
GIT_VERSION  ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_REVISION ?= $(shell git rev-parse HEAD | tr -d '\040\011\012\015\n')
GIT_BRANCH   ?= $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n')
GIT_USER     ?= $(shell git config --get user.name | tr -d '\040\011\012\015\n')
BUILD_DATE   ?= $(shell date +'%Y-%m-%dT%H:%M:%S')

# Golang
GO_VERSION       ?= 1.16
GO               ?= go
GO_BUILD         ?= $(GO) build
GO_INSTALL       ?= $(GO) install
GO_TEST          ?= $(GO) test
GO_CLEAN         ?= $(GO) clean
GO_CLEAN_OPTS    ?= -n -x -i
GO_FMT           ?= $(GO)fmt
GO_FMT_OPTS      ?=
GO_MOD           ?= $(GO) mod
GO_OPTS          ?= -v
GO_HOST_OS       ?= $(shell $(GO) env GOHOSTOS)
GO_HOST_ARCH     ?= $(shell $(GO) env GOHOSTARCH)
GO_OS            ?= darwin linux windows
GO_ARCH          ?= arm arm64 amd64 386
GO_VENDOR_FOLDER ?= ./vendor
GO_PKGS_PATH     ?= ./...
GO_LDFLAGS       ?= -ldflags "-X github.com/christiangda/mq-to-db/internal/version.Version=$(GIT_VERSION) -X github.com/christiangda/mq-to-db/internal/version.Revision=$(GIT_REVISION) -X github.com/christiangda/mq-to-db/internal/version.Branch=$(GIT_BRANCH) -X github.com/christiangda/mq-to-db/internal/version.BuildUser=\"$(GIT_USER)\" -X github.com/christiangda/mq-to-db/internal/version.BuildDate=$(BUILD_DATE)"
GO_CGO_ENABLED    ?= 0

# Container
CONTAINER_BUILD_COMMAND ?= docker build
CONTAINER_PUBLISH_COMMAND ?= docker push
CONTAINER_TAG_COMMAND ?= docker tag
CONTAINER_BUILD_FILE ?= ./Dockerfile
CONTAINER_BUILD_CONTEXT ?= ./
CONTAINER_IMAGE_ARCH ?= amd64
CONTAINER_IMAGE_OS ?= linux
CONTAINER_IMAGE_NAME ?= $(APP_NAME)
CONTAINER_IMAGE_REPO ?= christiangda
CONTAINER_IMAGE_TAG ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

# Enable -race only on linux and no container
ifneq (,$(wildcard /.dockerenv))
	ifeq ($(GO_HOST_ARCH),amd64)
			ifeq ($(GO_HOST_OS),$(filter $(GO_HOST_OS),linux))
					GO_OPTS := $(GO_OPTS) -race
					GO_CGO_ENABLED := 1
			endif
	endif
endif

# compile all when cgo is disabled (slow)
ifeq ($(GO_CGO_ENABLED),0)
	GO_OPTS := $(GO_OPTS) -a
endif

#
.PHONY: all
all: clean go-lint go-tidy go-test go-build

.PHONY: full
full: clean go-lint go-tidy go-test go-build-all

.PHONY: go-lint
go-lint:
	@echo "--> Linting"

.PHONY: go-fmt
go-fmt:
	@echo "--> Checking formating"
	$(GO_FMT) $(GO_FMT_OPTS) -d $$(find . -path $(GO_VENDOR_FOLDER) -prune -o -name '*.go' -print)

.PHONY: go-build
go-build:
	@echo "--> Building native OS app"
	GOOS=$(GO_HOST_OS) GOARCH=$(GO_HOST_ARCH) CGO_ENABLED=$(GO_CGO_ENABLED) $(GO_BUILD) $(GO_OPTS) -o $(APP_NAME) $(GO_LDFLAGS) $$(find ./cmd -name '*.go' -print)

.PHONY: go-install
go-install:
	@echo "--> Installing"
	GOOS=$(GO_HOST_OS) GOARCH=$(GO_HOST_ARCH) $(GO_INSTALL) $(GO_LDFLAGS)

.PHONY: go-build-all
go-build-all: go-build
	@echo "--> Building for all OS and ARCH"
	$(foreach GOOS, $(GO_OS),\
	$(foreach GOARCH, $(GO_ARCH), $(shell GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(GO_CGO_ENABLED) $(GO_BUILD) $(GO_OPTS) -o $(APP_NAME)-$(GOOS)-$(GOARCH) $(GO_LDFLAGS) $$(find ./cmd -name '*.go' -print) )))

.PHONY: go-update-deps
go-update-deps:
	@echo "--> Updating Go dependencies"
	for dep in $$($(GO) list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all); do \
		$(GO) get $$dep; \
	done

.PHONY: go-tidy
go-tidy:
	@echo "--> Tidying"
	$(GO_MOD) tidy
ifneq (,$(wildcard $(GO_VENDOR_FOLDER)))
	@echo "--> Generating Vendor folder"
	$(GO_MOD) vendor
endif

.PHONY: go-test
go-test:
	@echo "--> Test"
	GOOS=$(GO_HOST_OS) GOARCH=$(GO_HOST_ARCH) CGO_ENABLED=$(GO_CGO_ENABLED) $(GO_TEST) $(GO_OPTS) $(GO_PKGS_PATH)

.PHONY: clean
clean:
	@echo "--> Cleaning"
	$(GO_CLEAN) $(GO_CLEAN_OPTS)
	rm -rf $(APP_NAME)*

.PHONY: container-build
container-build:
	@echo "--> Building container image"
	$(CONTAINER_BUILD_COMMAND) \
		--build-arg CONTAINER_ARCH="$(CONTAINER_IMAGE_ARCH)" \
		--build-arg CONTAINER_OS="$(CONTAINER_IMAGE_OS)" \
		--build-arg APP_NAME="$(CONTAINER_IMAGE_NAME)" \
		--build-arg GO_VERSION="$(GO_VERSION)" \
		--tag "$(CONTAINER_IMAGE_REPO)/$(CONTAINER_IMAGE_NAME):$(CONTAINER_IMAGE_TAG)" \
		--file $(CONTAINER_BUILD_FILE) \
		$(CONTAINER_BUILD_CONTEXT)

.PHONY: container-publish
container-publish:
	@echo "--> Publishing container image"
	$(CONTAINER_PUBLISH_COMMAND) "$(CONTAINER_IMAGE_REPO)/$(CONTAINER_IMAGE_NAME):$(CONTAINER_IMAGE_TAG)"

.PHONY: container-tag-latest
container-tag-latest:
	@echo "--> Tagging container as latest"
	$(CONTAINER_TAG_COMMAND) "$(CONTAINER_IMAGE_REPO)/$(CONTAINER_IMAGE_NAME):$(CONTAINER_IMAGE_TAG)" "$(CONTAINER_IMAGE_REPO)/$(CONTAINER_IMAGE_NAME):latest"

.PHONY: container-manifest
container-manifest:
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create "$(CONTAINER_IMAGE_REPO)/$(CONTAINER_IMAGE_NAME):$(CONTAINER_IMAGE_TAG)"
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push "$(CONTAINER_IMAGE_REPO)/$(CONTAINER_IMAGE_NAME):$(CONTAINER_IMAGE_TAG)"