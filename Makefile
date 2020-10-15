# Bootstrap (pull) the build harness

# GITHUB_USER containing '@' char must be escaped with '%40'
GITHUB_USER := $(shell echo $(GITHUB_USER) | sed 's/@/%40/g')
GITHUB_TOKEN ?=


USE_VENDORIZED_BUILD_HARNESS = true 

ifndef USE_VENDORIZED_BUILD_HARNESS
-include $(shell curl -s -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/open-cluster-management/build-harness-extensions/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)
else
-include vbh/.build-harness-vendorized
endif

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

DOCKER_NAMESPACE := open-cluster-management
DOCKER_REGISTRY := quay.io

.PHONY: build
build:
	make docker/info
	make docker/build

.PHONY: push
push:: docker/tag docker/login
	make docker/push
