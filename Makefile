# Copyright (c) 2021 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

USE_VENDORIZED_BUILD_HARNESS = true

ifndef USE_VENDORIZED_BUILD_HARNESS
-include $(shell curl -s -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/open-cluster-management/build-harness-extensions/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)
else
-include vbh/.build-harness-vendorized
endif

-include /opt/build-harness/Makefile.prow

build:
	ginkgo build ./pkg/tests/

test-unit:
	@echo "Running Unit Tests.."

test-e2e:
	@echo "TODO: Running E2E Tests.."
