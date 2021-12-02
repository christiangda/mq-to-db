.DELETE_ON_ERROR: clean

EXECUTABLES = go
K := $(foreach exec,$(EXECUTABLES),\
  $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH)))

#
APP_NAME 	       ?= mq-to-db
APP_DEPENDENCIES := $(shell go list -m -f '{{if not (or .Indirect .Main)}}{{.Path}}{{end}}' all)

GIT_VERSION  ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_REVISION ?= $(shell git rev-parse HEAD | tr -d '\040\011\012\015\n')
GIT_BRANCH   ?= $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n')
GIT_USER     ?= $(shell git config --get user.name | tr -d '\040\011\012\015\n')
BUILD_DATE   ?= $(shell date +'%Y-%m-%dT%H:%M:%S')
BUILD_DIR      := ./build
DIST_DIR       := ./dist

# Golang
GO_LDFLAGS     ?= -ldflags "-X github.com/christiangda/mq-to-db/internal/version.Version=$(GIT_VERSION) -X github.com/christiangda/mq-to-db/internal/version.Revision=$(GIT_REVISION) -X github.com/christiangda/mq-to-db/internal/version.Branch=$(GIT_BRANCH) -X github.com/christiangda/mq-to-db/internal/version.BuildUser=\"$(GIT_USER)\" -X github.com/christiangda/mq-to-db/internal/version.BuildDate=$(BUILD_DATE)"
GO_HOST_OS     ?= $(shell $(GO) env GOHOSTOS)
GO_HOST_ARCH   ?= $(shell $(GO) env GOHOSTARCH)
GO_OS          ?= darwin linux windows
GO_ARCH        ?= arm arm64 amd64
GO_OPTS        ?= -v
GO_CGO_ENABLED ?= 0
GO_FILES       := $(shell go list ./... | grep -v /mocks/)

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
all: clean test build

mod-update: tidy
	$(foreach dep, $(APP_DEPENDENCIES), $(shell go get -u $(dep)))
	go mod tidy

tidy:
	go mod tidy

fmt:
	@go fmt $(GO_FILES)

vet:
	go vet $(GO_FILES)

generate:
	go generate $(GO_FILES)

test: generate tidy fmt vet
	go test -race -covermode=atomic -coverprofile coverage.out -tags=unit $(GO_FILES)

test-coverage: test
	go tool cover -html=coverage.out

build:
	$(foreach proj_name, $(APP_NAME), \
		$(shell CGO_ENABLED=$(GO_CGO_ENABLED) go build $(GO_LDFLAGS) $(GO_OPTS) -o ./$(BUILD_DIR)/$(proj_name) ./cmd/$(proj_name)/ ))

install:
	GOOS=$(GO_HOST_OS) GOARCH=$(GO_HOST_ARCH) go install $(GO_LDFLAGS)

build-dist: build
	$(foreach GOOS, $(GO_OS), \
		$(foreach GOARCH, $(GO_ARCH), \
			$(foreach proj_name, $(APP_NAME), \
				$(shell GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(GO_CGO_ENABLED) go build $(GO_LDFLAGS) $(GO_OPTS) -o ./$(DIST_DIR)/$(proj_name)-$(GOOS)-$(GOARCH) ./cmd/$(proj_name)/ ))))

clean:
	go clean -n -x -i
	rm -rf $(BUILD_DIR) $(DIST_DIR) ./*.out

container-build: build-dist
	$(foreach OS, $(CONTAINER_OS), \
		$(foreach ARCH, $(CONTAINER_ARCH), \
			$(if $(findstring $(ARCH), arm64v8), $(eval BIN_ARCH = arm64),$(eval BIN_ARCH = $(ARCH)) ) \
				docker build \
					--build-arg ARCH=$(ARCH) \
					--build-arg BIN_ARCH=$(BIN_ARCH) \
					--build-arg OS=$(OS) \
					-t $(CONTAINER_REPO)/$(CONTAINER_IMAGE_NAME)-$(OS)-$(ARCH):latest \
					-t $(CONTAINER_REPO)/$(CONTAINER_IMAGE_NAME)-$(OS)-$(ARCH):$(GIT_VERSION) \
					./.; \
			))

container-publish-docker: container-build
	$(foreach OS, $(CONTAINER_OS), \
		$(foreach ARCH, $(CONTAINER_ARCH), \
			$(if $(findstring $(ARCH), arm64v8), $(eval BIN_ARCH = arm64),$(eval BIN_ARCH = $(ARCH)) ) \
			docker push "$(CONTAINER_REPO)/$(CONTAINER_IMAGE_NAME)-$(OS)-$(ARCH):latest";  \
			docker push "$(CONTAINER_REPO)/$(CONTAINER_IMAGE_NAME)-$(OS)-$(ARCH):$(GIT_VERSION)"; \
			))

container-publish-github: container-build
	$(foreach OS, $(CONTAINER_OS), \
		$(foreach ARCH, $(CONTAINER_ARCH), \
			$(if $(findstring $(ARCH), arm64v8), $(eval BIN_ARCH = arm64),$(eval BIN_ARCH = $(ARCH))) \
			docker push "ghcr.io/$(CONTAINER_REPO)/$(CONTAINER_IMAGE_NAME)-$(OS)-$(ARCH):latest"; \
			docker push "ghcr.io/$(CONTAINER_REPO)/$(CONTAINER_IMAGE_NAME)-$(OS)-$(ARCH):$(GIT_VERSION)"; \
			))