# Bootstrap (pull) the build harness

# GITHUB_USER containing '@' char must be escaped with '%40'
GITHUB_USER := $(shell echo $(GITHUB_USER) | sed 's/@/%40/g')
GITHUB_TOKEN ?=

-include $(shell [ -f ".build-harness-bootstrap" ] || curl -sL -o .build-harness-bootstrap -H "Authorization: token $(GITHUB_TOKEN)" -H "Accept: application/vnd.github.v3.raw" "https://raw.github.com/stolostron/build-harness-extensions/main/templates/Makefile.build-harness-bootstrap"; echo .build-harness-bootstrap)

# Only use git commands if it exists
ifdef GIT
GIT_COMMIT      = $(shell git rev-parse --short HEAD)
GIT_REMOTE_URL  = $(shell git config --get remote.origin.url)
VCS_REF     = $(if $(shell git status --porcelain),$(GIT_COMMIT)-$(BUILD_DATE),$(GIT_COMMIT))
endif

SHORT_COMMIT_NAME := $(shell git rev-parse --short HEAD)
SEMVERSION ?= $(shell cat COMPONENT_VERSION)-${SHORT_COMMIT_NAME}

ifdef TRAVIS_PULL_REQUEST
	ifneq ($(TRAVIS_PULL_REQUEST),false)
		SEMVERSION = $(shell cat COMPONENT_VERSION)-PR${TRAVIS_PULL_REQUEST}-${SHORT_COMMIT_NAME}
	endif
endif

ifndef TRAVIS
	SEMVERSION = $(shell cat COMPONENT_VERSION)-$(shell whoami)-${SHORT_COMMIT_NAME}
endif

DOCKER_NAMESPACE := stolostron
DOCKER_REGISTRY := quay.io

.PHONY: build
build:
	make docker/info
	make docker/build

.PHONY: push
push:: docker/tag docker/login
	make docker/push

.PHONY: test-unit
test-unit:
	@echo "Running Unit Tests.."	
